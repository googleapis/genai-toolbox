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

package bigquerysearchcatalog

import (
	"context"
	"fmt"
	"net/http"

	dataplexapi "cloud.google.com/go/dataplex/apiv1"
	"github.com/goccy/go-yaml"
	bigqueryds "github.com/googleapis/mcp-toolbox/internal/sources/bigquery"
	"github.com/googleapis/mcp-toolbox/internal/sources/dataplex/searchcatalog"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
)

const resourceType string = "bigquery-search-catalog"

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
	MakeDataplexCatalogClient() func() (*dataplexapi.CatalogClient, bigqueryds.DataplexClientCreator, error)
	BigQueryProject() string
	UseClientAuthorization() bool
	GetAuthTokenHeaderName() string
	InvokeSearchCatalog(ctx context.Context, params map[string]any, tokenStr string) ([]searchcatalog.DataplexSearchResponse, error)
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
	prompt := parameters.NewStringParameter("prompt", "Prompt representing search intention. Do not rewrite the prompt.")
	datasetIds := parameters.NewArrayParameter("datasetIds", "Array of dataset IDs.", parameters.NewStringParameter("datasetId", "The IDs of the bigquery dataset."), parameters.WithArrayDefault([]any{}))
	projectIds := parameters.NewArrayParameter("projectIds", "Array of project IDs.", parameters.NewStringParameter("projectId", "The IDs of the bigquery project."), parameters.WithArrayDefault([]any{}))
	types := parameters.NewArrayParameter("types", "Array of data types to filter by.", parameters.NewStringParameter("type", "The type of the data. Accepted values are: CONNECTION, POLICY, DATASET, MODEL, ROUTINE, TABLE, VIEW."), parameters.WithArrayDefault([]any{}))
	pageSize := parameters.NewIntParameter("pageSize", "Number of results in the search page.", parameters.WithIntDefault(5))
	params := parameters.Parameters{prompt, datasetIds, projectIds, types, pageSize}

	if cfg.Description == "" {
		cfg.Description = "Use this tool to find tables, views, models, routines or connections."
	}

	return Tool{
		BaseTool: tools.NewBaseTool(
			cfg,
			tools.GetAnnotationsOrDefault(cfg.Annotations, tools.NewReadOnlyAnnotations),
			tools.Manifest{Description: cfg.Description, Parameters: params.Manifest(), AuthRequired: cfg.AuthRequired},
			params,
		),
	}, nil
}

type Tool struct {
	tools.BaseTool[Config]
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Cfg
}

func (t Tool) RequiresClientAuthorization(primitiveMgr tools.SourceProvider) (bool, error) {
	source, err := tools.GetCompatibleSource[compatibleSource](primitiveMgr, t.Cfg.Source, t.Cfg.Name, t.Cfg.Type)
	if err != nil {
		return false, err
	}
	return source.UseClientAuthorization(), nil
}

func (t Tool) Invoke(ctx context.Context, primitiveMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, util.ToolboxError) {
	source, err := tools.GetCompatibleSource[compatibleSource](primitiveMgr, t.Cfg.Source, t.Cfg.Name, t.Cfg.Type)
	if err != nil {
		return nil, util.NewClientServerError("source used is not compatible with the tool", http.StatusInternalServerError, err)
	}

	var tokenStr string
	if source.UseClientAuthorization() {
		tokenStr, err = accessToken.ParseBearerToken()
		if err != nil {
			return nil, util.NewClientServerError("error parsing access token", http.StatusUnauthorized, err)
		}
	}

	results, err := source.InvokeSearchCatalog(ctx, params.AsMap(), tokenStr)
	if err != nil {
		return nil, util.ProcessGcpError(err)
	}

	return results, nil
}

func (t Tool) GetAuthTokenHeaderName(primitiveMgr tools.SourceProvider) (string, error) {
	source, err := tools.GetCompatibleSource[compatibleSource](primitiveMgr, t.Cfg.Source, t.Cfg.Name, t.Cfg.Type)
	if err != nil {
		return "", err
	}
	return source.GetAuthTokenHeaderName(), nil
}
