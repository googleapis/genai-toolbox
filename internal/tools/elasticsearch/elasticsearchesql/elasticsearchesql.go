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

	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/googleapis/genai-toolbox/internal/util"

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
	Name         string           `yaml:"name" validate:"required"`
	Kind         string           `yaml:"kind" validate:"required"`
	Source       string           `yaml:"source" validate:"required"`
	Description  string           `yaml:"description" validate:"required"`
	AuthRequired []string         `yaml:"authRequired"`
	Query        string           `yaml:"query" validate:"required"`
	Format       string           `yaml:"format"`
	Timeout      int              `yaml:"timeout"`
	Parameters   tools.Parameters `yaml:"parameters"`
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
	Name         string           `yaml:"name"`
	Kind         string           `yaml:"kind"`
	AuthRequired []string         `yaml:"authRequired"`
	Parameters   tools.Parameters `yaml:"parameters"`
	Query        string           `yaml:"query"`
	Format       string           `yaml:"format" default:"json"`
	Timeout      int              `yaml:"timeout"`

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

	mcpManifest := tools.McpManifest{
		Name:        c.Name,
		Description: c.Description,
		InputSchema: c.Parameters.McpManifest(),
	}

	return Tool{
		Name:         c.Name,
		Kind:         kind,
		Parameters:   c.Parameters,
		Query:        c.Query,
		Format:       c.Format,
		Timeout:      c.Timeout,
		AuthRequired: c.AuthRequired,
		EsClient:     s.ElasticsearchClient(),
		manifest:     tools.Manifest{Description: c.Description, Parameters: c.Parameters.Manifest(), AuthRequired: c.AuthRequired},
		mcpManifest:  mcpManifest,
	}, nil
}

func (t Tool) Invoke(ctx context.Context, params tools.ParamValues, accessToken tools.AccessToken) (any, error) {
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
		Instrument: t.EsClient.InstrumentationEnabled(),
	}.Do(ctx, t.EsClient)

	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var result any
	err = util.DecodeJSON(res.Body, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response body: %w", err)
	}

	return result, nil
}

func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (tools.ParamValues, error) {
	return tools.ParseParams(t.Parameters, data, claims)
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
