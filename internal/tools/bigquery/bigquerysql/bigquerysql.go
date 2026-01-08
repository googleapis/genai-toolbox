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
	"reflect"
	"strings"

	bigqueryapi "cloud.google.com/go/bigquery"
	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/embeddingmodels"
	"github.com/googleapis/genai-toolbox/internal/sources"
	bigqueryds "github.com/googleapis/genai-toolbox/internal/sources/bigquery"
	"github.com/googleapis/genai-toolbox/internal/tools"
	bqutil "github.com/googleapis/genai-toolbox/internal/tools/bigquery/bigquerycommon"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
	bigqueryrestapi "google.golang.org/api/bigquery/v2"
)

const kind string = "bigquery-sql"

func init() {
	if !tools.Register(kind, newConfig) {
		panic(fmt.Sprintf("tool kind %q already registered", kind))
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
	RetrieveClientAndService(tools.AccessToken) (*bigqueryapi.Client, *bigqueryrestapi.Service, error)
	RunSQL(context.Context, *bigqueryapi.Client, string, string, []bigqueryapi.QueryParameter, []*bigqueryapi.ConnectionProperty) (any, error)
}

type Config struct {
	Name               string                `yaml:"name" validate:"required"`
	Kind               string                `yaml:"kind" validate:"required"`
	Source             string                `yaml:"source" validate:"required"`
	Description        string                `yaml:"description" validate:"required"`
	Statement          string                `yaml:"statement" validate:"required"`
	AuthRequired       []string              `yaml:"authRequired"`
	Parameters         parameters.Parameters `yaml:"parameters"`
	TemplateParameters parameters.Parameters `yaml:"templateParameters"`
}

// validate interface
var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigKind() string {
	return kind
}

func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	allParameters, paramManifest, err := parameters.ProcessParameters(cfg.TemplateParameters, cfg.Parameters)
	if err != nil {
		return nil, err
	}

	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, allParameters, nil)

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

func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Source, t.Name, t.Kind)
	if err != nil {
		return nil, err
	}

	highLevelParams := make([]bigqueryapi.QueryParameter, 0, len(t.Parameters))
	lowLevelParams := make([]*bigqueryrestapi.QueryParameter, 0, len(t.Parameters))

	paramsMap := params.AsMap()
	newStatement, err := parameters.ResolveTemplateParams(t.TemplateParameters, t.Statement, paramsMap)
	if err != nil {
		return nil, fmt.Errorf("unable to extract template params %w", err)
	}

	for _, p := range t.Parameters {
		name := p.GetName()
		value := paramsMap[name]

		// This block for converting []any to typed slices is still necessary and correct.
		if arrayParam, ok := p.(*parameters.ArrayParameter); ok {
			arrayParamValue, ok := value.([]any)
			if !ok {
				return nil, fmt.Errorf("unable to convert parameter `%s` to []any", name)
			}
			itemType := arrayParam.GetItems().GetType()
			var err error
			value, err = parameters.ConvertAnySliceToTyped(arrayParamValue, itemType)
			if err != nil {
				return nil, fmt.Errorf("unable to convert parameter `%s` from []any to typed slice: %w", name, err)
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

		// 2. Create the low-level parameter for the dry run, using the defined type from `p`.
		lowLevelParam := &bigqueryrestapi.QueryParameter{
			Name:           paramNameForHighLevel,
			ParameterType:  &bigqueryrestapi.QueryParameterType{},
			ParameterValue: &bigqueryrestapi.QueryParameterValue{},
		}

		if arrayParam, ok := p.(*parameters.ArrayParameter); ok {
			// Handle array types based on their defined item type.
			lowLevelParam.ParameterType.Type = "ARRAY"
			itemType, err := bqutil.BQTypeStringFromToolType(arrayParam.GetItems().GetType())
			if err != nil {
				return nil, err
			}
			lowLevelParam.ParameterType.ArrayType = &bigqueryrestapi.QueryParameterType{Type: itemType}

			// Build the array values.
			sliceVal := reflect.ValueOf(value)
			arrayValues := make([]*bigqueryrestapi.QueryParameterValue, sliceVal.Len())
			for i := 0; i < sliceVal.Len(); i++ {
				arrayValues[i] = &bigqueryrestapi.QueryParameterValue{
					Value: fmt.Sprintf("%v", sliceVal.Index(i).Interface()),
				}
			}
			lowLevelParam.ParameterValue.ArrayValues = arrayValues
		} else {
			// Handle scalar types based on their defined type.
			bqType, err := bqutil.BQTypeStringFromToolType(p.GetType())
			if err != nil {
				return nil, err
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
			return nil, fmt.Errorf("failed to get BigQuery session: %w", err)
		}
		if session != nil {
			// Add session ID to the connection properties for subsequent calls.
			connProps = append(connProps, &bigqueryapi.ConnectionProperty{Key: "session_id", Value: session.ID})
		}
	}

	bqClient, restService, err := source.RetrieveClientAndService(accessToken)
	if err != nil {
		return nil, err
	}

	dryRunJob, err := bqutil.DryRunQuery(ctx, restService, bqClient.Project(), bqClient.Location, newStatement, lowLevelParams, connProps)
	if err != nil {
		return nil, fmt.Errorf("query validation failed: %w", err)
	}

	statementType := dryRunJob.Statistics.Query.StatementType

	return source.RunSQL(ctx, bqClient, newStatement, statementType, highLevelParams, connProps)
}

func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (parameters.ParamValues, error) {
	return parameters.ParseParams(t.AllParams, data, claims)
}

func (t Tool) EmbedParams(ctx context.Context, paramValues parameters.ParamValues, embeddingModelsMap map[string]embeddingmodels.EmbeddingModel) (parameters.ParamValues, error) {
	return parameters.EmbedParams(ctx, t.AllParams, paramValues, embeddingModelsMap, nil)
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
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Source, t.Name, t.Kind)
	if err != nil {
		return false, err
	}
	return source.UseClientAuthorization(), nil
}

func (t Tool) GetAuthTokenHeaderName(resourceMgr tools.SourceProvider) (string, error) {
	return "Authorization", nil
}
