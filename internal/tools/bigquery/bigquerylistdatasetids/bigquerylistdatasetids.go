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

package bigquerylistdatasetids

import (
	"context"
	"fmt"
	"net/http"

	bigqueryapi "cloud.google.com/go/bigquery"
	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
	bigqueryrestapi "google.golang.org/api/bigquery/v2"
	"google.golang.org/api/iterator"
)

const resourceType string = "bigquery-list-dataset-ids"
const projectKey string = "project"

func init() {
	if !tools.Register(resourceType, newConfig) {
		panic(fmt.Sprintf("tool type %q already registered", resourceType))
	}
}

func newConfig(ctx context.Context, name string, decoder *yaml.Decoder) (tools.ToolConfig, error) {
	actual := Config{ConfigBase: tools.ConfigBase{Name: name}}
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return nil, err
	}
	return actual, nil
}

type compatibleSource interface {
	BigQueryProject() string
	UseClientAuthorization() bool
	GetAuthTokenHeaderName() string
	BigQueryAllowedDatasets() []string
	RetrieveClientAndService(tools.AccessToken) (*bigqueryapi.Client, *bigqueryrestapi.Service, error)
}

type Config struct {
	tools.ConfigBase `yaml:",inline"`
	Type             string                 `yaml:"type" validate:"required"`
	Source           string                 `yaml:"source" validate:"required"`
	Annotations      *tools.ToolAnnotations `yaml:"annotations,omitempty"`
}

// validate interface
var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigType() string {
	return resourceType
}

func (cfg Config) Initialize(context.Context) (tools.Tool, error) {
	if cfg.Description == "" {
		return nil, fmt.Errorf("description is required for tool %q", cfg.Name)
	}

	params := buildParams(nil, "")
	return Tool{
		BaseTool: tools.NewBaseTool(
			cfg,
			tools.GetAnnotationsOrDefault(cfg.Annotations, tools.NewReadOnlyAnnotations),
			tools.Manifest{Description: cfg.Description, Parameters: params.Manifest(), AuthRequired: cfg.AuthRequired},
			params,
		),
	}, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	tools.BaseTool[Config]
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Cfg
}

func (t Tool) Invoke(ctx context.Context, primitiveMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, util.ToolboxError) {
	source, err := tools.GetCompatibleSource[compatibleSource](primitiveMgr, t.Cfg.Source, t.Cfg.Name, t.Cfg.Type)
	if err != nil {
		return nil, util.NewClientServerError("source used is not compatible with the tool", http.StatusInternalServerError, err)
	}

	if len(source.BigQueryAllowedDatasets()) > 0 {
		return source.BigQueryAllowedDatasets(), nil
	}
	mapParams := params.AsMap()
	projectId, ok := mapParams[projectKey].(string)
	if !ok {
		return nil, util.NewAgentError(fmt.Sprintf("invalid or missing '%s' parameter; expected a string", projectKey), nil)
	}

	bqClient, _, err := source.RetrieveClientAndService(accessToken)
	if err != nil {
		return nil, util.NewClientServerError("failed to retrieve BigQuery client", http.StatusInternalServerError, err)
	}
	datasetIterator := bqClient.Datasets(ctx)
	datasetIterator.ProjectID = projectId

	var datasetIds []any
	for {
		dataset, err := datasetIterator.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, util.ProcessGcpError(err)
		}

		// Remove leading and trailing quotes
		id := dataset.DatasetID
		if len(id) >= 2 && id[0] == '"' && id[len(id)-1] == '"' {
			id = id[1 : len(id)-1]
		}
		datasetIds = append(datasetIds, id)
	}

	return datasetIds, nil
}

func (t Tool) RequiresClientAuthorization(primitiveMgr tools.SourceProvider) (bool, error) {
	source, err := tools.GetCompatibleSource[compatibleSource](primitiveMgr, t.Cfg.Source, t.Cfg.Name, t.Cfg.Type)
	if err != nil {
		return false, err
	}
	return source.UseClientAuthorization(), nil
}

func (t Tool) GetAuthTokenHeaderName(primitiveMgr tools.SourceProvider) (string, error) {
	source, err := tools.GetCompatibleSource[compatibleSource](primitiveMgr, t.Cfg.Source, t.Cfg.Name, t.Cfg.Type)
	if err != nil {
		return "", err
	}
	return source.GetAuthTokenHeaderName(), nil
}

// buildParams builds the tool's parameters from the source's allowed-dataset configuration.
// A nil allow-list and empty default project yield the plain skeleton.
func buildParams(allowedDatasets []string, defaultProject string) parameters.Parameters {
	projectParameterDescription := "The Google Cloud project to list dataset ids."
	if len(allowedDatasets) > 0 {
		projectParameterDescription = "This parameter will be ignored. The list of datasets is restricted to a pre-configured list; No need to provide a project ID."
	}
	projectParameter := parameters.NewStringParameter(projectKey, projectParameterDescription, parameters.WithStringDefault(defaultProject))
	return parameters.Parameters{projectParameter}
}

// resolveParams builds the tool's parameters using the source's allowed-dataset configuration.
func (t Tool) resolveParams(srcs map[string]sources.Source) (parameters.Parameters, error) {
	s, err := tools.GetCompatibleSourceFromMap[compatibleSource](srcs, t.Cfg.Source, t.Cfg.Name, t.Cfg.Type)
	if err != nil {
		return nil, err
	}
	return buildParams(s.BigQueryAllowedDatasets(), s.BigQueryProject()), nil
}

// GetParameters returns the tool's parameters, resolved against the source.
func (t Tool) GetParameters(srcs map[string]sources.Source) (parameters.Parameters, error) {
	return t.resolveParams(srcs)
}

// Manifest returns the tool's manifest, resolved against the source.
func (t Tool) Manifest(srcs map[string]sources.Source) (tools.Manifest, error) {
	params, err := t.resolveParams(srcs)
	if err != nil {
		return tools.Manifest{}, err
	}
	return tools.Manifest{Description: t.Cfg.Description, Parameters: params.Manifest(), AuthRequired: t.Cfg.AuthRequired}, nil
}
