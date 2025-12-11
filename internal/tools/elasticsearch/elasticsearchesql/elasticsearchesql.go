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

package elasticsearchesql

import (
	"context"
	"fmt"
	"time"

	"github.com/googleapis/genai-toolbox/internal/embeddingmodels"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	es "github.com/googleapis/genai-toolbox/internal/sources/elasticsearch"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

const kind string = "elasticsearch-esql"

func init() {
	if !tools.Register(kind, newConfig) {
		panic(fmt.Sprintf("tool kind %q already registered", kind))
	}
}

type compatibleSource interface {
	ElasticsearchClient() es.EsClient
	RunSQL(ctx context.Context, format, query string, params []map[string]any) (any, error)
}

type Config struct {
	Name         string                `yaml:"name" validate:"required"`
	Kind         string                `yaml:"kind" validate:"required"`
	Source       string                `yaml:"source" validate:"required"`
	Description  string                `yaml:"description" validate:"required"`
	AuthRequired []string              `yaml:"authRequired" validate:"required"`
	Query        string                `yaml:"query"`
	Format       string                `yaml:"format"`
	Timeout      int                   `yaml:"timeout"`
	Parameters   parameters.Parameters `yaml:"parameters"`
}

var _ tools.ToolConfig = Config{}

func (c Config) ToolConfigKind() string {
	return kind
}

func newConfig(ctx context.Context, name string, decoder *yaml.Decoder) (tools.ToolConfig, error) {
	actual := Config{Name: name}
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return nil, err
	}
	return actual, nil
}

type Tool struct {
	Config
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

var _ tools.Tool = Tool{}

func (c Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	mcpManifest := tools.GetMcpManifest(c.Name, c.Description, c.AuthRequired, c.Parameters, nil)

	return Tool{
		Config:      c,
		manifest:    tools.Manifest{Description: c.Description, Parameters: c.Parameters.Manifest(), AuthRequired: c.AuthRequired},
		mcpManifest: mcpManifest,
	}, nil
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Source, t.Name, t.Kind)
	if err != nil {
		return nil, err
	}

	var cancel context.CancelFunc
	if t.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(t.Timeout)*time.Second)
		defer cancel()
	} else {
		ctx, cancel = context.WithTimeout(ctx, time.Minute)
		defer cancel()
	}

	query := t.Query
	sqlParams := make([]map[string]any, 0, len(params))
	paramMap := params.AsMap()
	// If a query is provided in the params and not already set in the tool, use it.
	if queryVal, ok := paramMap["query"]; ok {
		if str, ok := queryVal.(string); ok && t.Query == "" {
			query = str
		}

		// Drop the query param if not a string or if the tool already has a query.
		delete(paramMap, "query")
	}

	for _, param := range t.Parameters {
		if param.GetType() == "array" {
			return nil, fmt.Errorf("array parameters are not supported yet")
		}
		sqlParams = append(sqlParams, map[string]any{param.GetName(): paramMap[param.GetName()]})
	}
	return source.RunSQL(ctx, t.Format, query, sqlParams)
}

func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (parameters.ParamValues, error) {
	return parameters.ParseParams(t.Parameters, data, claims)
}

func (t Tool) EmbedParams(ctx context.Context, paramValues parameters.ParamValues, embeddingModelsMap map[string]embeddingmodels.EmbeddingModel) (parameters.ParamValues, error) {
	return parameters.EmbedParams(ctx, t.Parameters, paramValues, embeddingModelsMap, nil)
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
	return false, nil
}

func (t Tool) GetAuthTokenHeaderName(resourceMgr tools.SourceProvider) (string, error) {
	return "Authorization", nil
}
