// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package bigquerysql

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	bigqueryapi "cloud.google.com/go/bigquery"
	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/mcp-toolbox/internal/embeddingmodels"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	bigqueryds "github.com/googleapis/mcp-toolbox/internal/sources/bigquery"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	bqutil "github.com/googleapis/mcp-toolbox/internal/tools/bigquery/bigquerycommon"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
	bigqueryrestapi "google.golang.org/api/bigquery/v2"
)

const resourceType string = "bigquery-sql"

func init() {
	if !tools.Register(resourceType, newConfig) {
		panic(fmt.Sprintf("tool type %q already registered", resourceType))
	}
}

func newConfig(ctx context.Context, name string, decoder *yaml.Decoder) (tools.ToolConfig, error) {
	actual := Config{Name: name}
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return nil, err
	}
	return actual, nil
}

type compatibleSource interface {
	BigQuerySession() bigqueryds.BigQuerySessionProvider
	UseClientAuthorization() bool
	GetAuthTokenHeaderName() string
	GetMaximumBytesBilled() int64
	RetrieveClientAndService(tools.AccessToken) (*bigqueryapi.Client, *bigqueryrestapi.Service, error)
	RunSQL(context.Context, *bigqueryapi.Client, string, string, []bigqueryapi.QueryParameter, []*bigqueryapi.ConnectionProperty) (any, error)
}

type Config struct {
	Name               string                 `yaml:"name" validate:"required"`
	Type               string                 `yaml:"type" validate:"required"`
	Source             string                 `yaml:"source" validate:"required"`
	Description        string                 `yaml:"description" validate:"required"`
	Statement          string                 `yaml:"statement" validate:"required"`
	AuthRequired       []string               `yaml:"authRequired"`
	Parameters         parameters.Parameters  `yaml:"parameters"`
	TemplateParameters parameters.Parameters  `yaml:"templateParameters"`
	Annotations        *tools.ToolAnnotations `yaml:"annotations,omitempty"`
}

// validate interface
var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigType() string {
	return resourceType
}

func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	allParameters, paramManifest, err := parameters.ProcessParameters(cfg.TemplateParameters, cfg.Parameters)
	if err != nil {
		return nil, err
	}

	annotations := tools.GetAnnotationsOrDefault(cfg.Annotations, tools.NewDestructiveAnnotations)
	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, allParameters, annotations)

	// finish tool setup
	t := Tool{
		Config:      cfg,
		AllParams:   allParameters,
		manifest:    tools.Manifest{Description: cfg.Description, Parameters: paramManifest, AuthRequired: cfg.AuthRequired},
		mcpManifest: mcpManifest,
	}
	return t, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Config
	AllParams   parameters.Parameters `yaml:"allParams"`
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, util.ToolboxError) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Source, t.Name, t.Type)
	if err != nil {
		return nil, util.NewClientServerError("source used is not compatible with the tool", http.StatusInternalServerError, err)
	}

	highLevelParams := make([]bigqueryapi.QueryParameter, 0, len(t.Parameters))
	lowLevelParams := make([]*bigqueryrestapi.QueryParameter, 0, len(t.Parameters))

	paramsMap := params.AsMap()
	newStatement, err := parameters.ResolveTemplateParams(t.TemplateParameters, t.Statement, paramsMap)
	if err != nil {
		return nil, util.NewAgentError("unable to extract template params", err)
	}

	for _, p := range t.Parameters {
		name := p.GetName()
		value := paramsMap[name]

		// This block for converting []any to typed slices is still necessary and correct.
		if arrayParam, ok := p.(*parameters.ArrayParameter); ok {
			arrayParamValue, ok := value.([]any)
			if !ok {
				return nil, util.NewAgentError(fmt.Sprintf("unable to convert parameter `%s` to []any", name), nil)
			}
			itemType := arrayParam.GetItems().GetType()
			var err error
			value, err = parameters.ConvertAnySliceToTyped(arrayParamValue, itemType)
			if err != nil {
				return nil, util.NewAgentError(fmt.Sprintf("unable to convert parameter `%s` from []any to typed slice", name), err)
			}
		}

		// Determine if the parameter is named or positional for the high-level client.
		var paramNameForHighLevel string
		if strings.Contains(newStatement, "@"+name) {
			paramNameForHighLevel = name
		}

		// 1. Create the high-level parameter for the final query execution.
		highLevelParams = append(highLevelParams, bigqueryapi.QueryParameter{
			Name:  paramNameForHighLevel,
			Value: value,
		})

		// 2. Create the low-level parameter for the dry run.
		lowLevelParam := &bigqueryrestapi.QueryParameter{
			Name:           paramNameForHighLevel,
			ParameterType:  &bigqueryrestapi.QueryParameterType{},
			ParameterValue: &bigqueryrestapi.QueryParameterValue{},
		}

		rv := reflect.ValueOf(value)
		if rv.Kind() == reflect.Slice && rv.Type().Elem().Kind() != reflect.Uint8 {
			lowLevelParam.ParameterType.Type = "ARRAY"

			// Default item type to FLOAT64 for embeddings, or use config if available.
			itemType := "FLOAT64"
			if arrayParam, ok := p.(*parameters.ArrayParameter); ok {
				if bqType, err := bqutil.BQTypeStringFromToolType(arrayParam.GetItems().GetType()); err == nil {
					itemType = bqType
				}
			}
			lowLevelParam.ParameterType.ArrayType = &bigqueryrestapi.QueryParameterType{Type: itemType}

			// Build the array values.
			arrayValues := make([]*bigqueryrestapi.QueryParameterValue, rv.Len())
			for i := 0; i < rv.Len(); i++ {
				val := rv.Index(i).Interface()

				// Prevent precision loss and scientific notation issues
				var valStr string
				switch v := val.(type) {
				case float64:
					valStr = strconv.FormatFloat(v, 'f', -1, 64)
				case float32:
					valStr = strconv.FormatFloat(float64(v), 'f', -1, 32)
				default:
					valStr = fmt.Sprintf("%v", val)
				}

				arrayValues[i] = &bigqueryrestapi.QueryParameterValue{
					Value: valStr,
				}
			}
			lowLevelParam.ParameterValue.ArrayValues = arrayValues
		} else {
			// Handle scalar types based on their defined type.
			bqType, err := bqutil.BQTypeStringFromToolType(p.GetType())
			if err != nil {
				return nil, util.NewAgentError("unable to get BigQuery type from tool parameter type", err)
			}
			lowLevelParam.ParameterType.Type = bqType
			lowLevelParam.ParameterValue.Value = fmt.Sprintf("%v", value)
		}
		lowLevelParams = append(lowLevelParams, lowLevelParam)
	}

	connProps := []*bigqueryapi.ConnectionProperty{}
	if source.BigQuerySession() != nil {
		session, err := source.BigQuerySession()(ctx)
		if err != nil {
			return nil, util.NewClientServerError("failed to get BigQuery session", http.StatusInternalServerError, err)
		}
		if session != nil {
			// Add session ID to the connection properties for subsequent calls.
			connProps = append(connProps, &bigqueryapi.ConnectionProperty{Key: "session_id", Value: session.ID})
		}
	}

	bqClient, restService, err := source.RetrieveClientAndService(accessToken)
	if err != nil {
		return nil, util.NewClientServerError("failed to retrieve BigQuery client", http.StatusInternalServerError, err)
	}

	dryRunJob, err := bqutil.DryRunQuery(ctx, restService, bqClient.Project(), bqClient.Location, newStatement, lowLevelParams, connProps, source.GetMaximumBytesBilled())
	if err != nil {
		return nil, util.ProcessGcpError(err)
	}

	statementType := dryRunJob.Statistics.Query.StatementType
	resp, err := source.RunSQL(ctx, bqClient, newStatement, statementType, highLevelParams, connProps)
	if err != nil {
		return nil, util.ProcessGcpError(err)
	}
	return resp, nil
}

func formatVectorForBigQuery(vectorFloats []float32) any {
	if len(vectorFloats) == 0 {
		return []float64{}
	}

	res := make([]float64, len(vectorFloats))
	for i, f := range vectorFloats {
		// Convert to float64
		res[i] = float64(f)
	}
	return res
}

func (t Tool) EmbedParams(ctx context.Context, paramValues parameters.ParamValues, embeddingModelsMap map[string]embeddingmodels.EmbeddingModel) (parameters.ParamValues, error) {
	return parameters.EmbedParams(ctx, t.AllParams, paramValues, embeddingModelsMap, formatVectorForBigQuery)
}

func (t Tool) Manifest() tools.Manifest {
	return t.manifest
}

func (t Tool) McpManifest() tools.McpManifest {
	return t.mcpManifest
}

func (t Tool) Authorized(verifiedAuthServices []string) bool {
	return tools.IsAuthorized(t.AuthRequired, verifiedAuthServices)
}

func (t Tool) RequiresClientAuthorization(resourceMgr tools.SourceProvider) (bool, error) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Source, t.Name, t.Type)
	if err != nil {
		return false, err
	}
	return source.UseClientAuthorization(), nil
}

func (t Tool) GetAuthTokenHeaderName(resourceMgr tools.SourceProvider) (string, error) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Source, t.Name, t.Type)
	if err != nil {
		return "", err
	}
	return source.GetAuthTokenHeaderName(), nil
}

func (t Tool) GetParameters() parameters.Parameters {
	return t.AllParams
}
