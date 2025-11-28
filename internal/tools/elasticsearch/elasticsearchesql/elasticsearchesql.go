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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/elastic/go-elasticsearch/v9/esapi"
	"github.com/googleapis/genai-toolbox/internal/util"
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
}

var _ compatibleSource = &es.Source{}

var compatibleSources = [...]string{es.SourceKind}

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
	EsClient    es.EsClient
}

var _ tools.Tool = Tool{}

func (c Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	// verify source exists
	src, ok := srcs[c.Source]
	if !ok {
		return nil, fmt.Errorf("source %q not found", c.Source)
	}

	// verify the source is compatible
	s, ok := src.(compatibleSource)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be one of %q", kind, compatibleSources)
	}

	mcpManifest := tools.GetMcpManifest(c.Name, c.Description, c.AuthRequired, c.Parameters, nil)

	return Tool{
		Config:      c,
		EsClient:    s.ElasticsearchClient(),
		manifest:    tools.Manifest{Description: c.Description, Parameters: c.Parameters.Manifest(), AuthRequired: c.AuthRequired},
		mcpManifest: mcpManifest,
	}, nil
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

type esqlColumn struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type esqlResult struct {
	Columns []esqlColumn `json:"columns"`
	Values  [][]any      `json:"values"`
}

func (t Tool) Invoke(ctx context.Context, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	var cancel context.CancelFunc
	if t.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(t.Timeout)*time.Second)
		defer cancel()
	} else {
		ctx, cancel = context.WithTimeout(ctx, time.Minute)
		defer cancel()
	}

	bodyStruct := struct {
		Query  string           `json:"query"`
		Params []map[string]any `json:"params,omitempty"`
	}{
		Query:  t.Query,
		Params: make([]map[string]any, 0, len(params)),
	}

	paramMap := params.AsMap()

	// If a query is provided in the params and not already set in the tool, use it.
	if query, ok := paramMap["query"]; ok {
		if str, ok := query.(string); ok && bodyStruct.Query == "" {
			bodyStruct.Query = str
		}

		// Drop the query param if not a string or if the tool already has a query.
		delete(paramMap, "query")
	}

	for _, param := range t.Parameters {
		if param.GetType() == "array" {
			return nil, fmt.Errorf("array parameters are not supported yet")
		}
		bodyStruct.Params = append(bodyStruct.Params, map[string]any{param.GetName(): paramMap[param.GetName()]})
	}

	body, err := json.Marshal(bodyStruct)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query body: %w", err)
	}
	res, err := esapi.EsqlQueryRequest{
		Body:       bytes.NewReader(body),
		Format:     t.Format,
		FilterPath: []string{"columns", "values"},
		Instrument: t.EsClient.InstrumentationEnabled(),
	}.Do(ctx, t.EsClient)

	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		// Try to extract error message from response
		var esErr json.RawMessage
		err = util.DecodeJSON(res.Body, &esErr)
		if err != nil {
			return nil, fmt.Errorf("elasticsearch error: status %s", res.Status())
		}
		return esErr, nil
	}

	var result esqlResult
	err = util.DecodeJSON(res.Body, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response body: %w", err)
	}

	output := t.esqlToMap(result)

	return output, nil
}

// esqlToMap converts the esqlResult to a slice of maps.
func (t Tool) esqlToMap(result esqlResult) []map[string]any {
	output := make([]map[string]any, 0, len(result.Values))
	for _, value := range result.Values {
		row := make(map[string]any)
		if value == nil {
			output = append(output, row)
			continue
		}
		for i, col := range result.Columns {
			if i < len(value) {
				row[col.Name] = value[i]
			} else {
				row[col.Name] = nil
			}
		}
		output = append(output, row)
	}
	return output
}

func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (parameters.ParamValues, error) {
	return parameters.ParseParams(t.Parameters, data, claims)
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

func (t Tool) RequiresClientAuthorization() bool {
	return false
}

func (t Tool) GetAuthTokenHeaderName() string {
	return "Authorization"
}
