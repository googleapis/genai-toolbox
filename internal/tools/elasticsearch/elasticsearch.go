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

package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	elasticsearchsrc "github.com/googleapis/genai-toolbox/internal/sources/elasticsearch"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

const kind string = "elasticsearch"

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
	ElasticsearchClient() *elasticsearch.Client
}

// validate compatible sources are still compatible
var _ compatibleSource = &elasticsearchsrc.Source{}

var compatibleSources = [...]string{elasticsearchsrc.SourceKind}

type Config struct {
	Name         string           `yaml:"name" validate:"required"`
	Kind         string           `yaml:"kind" validate:"required"`
	Source       string           `yaml:"source" validate:"required"`
	Description  string           `yaml:"description" validate:"required"`
	Operation    string           `yaml:"operation" validate:"required"` // search, index, delete, bulk
	Parameters   tools.Parameters `yaml:"parameters"`
	AuthRequired []string         `yaml:"authRequired"`
}

// validate interface
var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigKind() string {
	return kind
}

func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	// verify source exists
	rawS, ok := srcs[cfg.Source]
	if !ok {
		return nil, fmt.Errorf("no source named %q configured", cfg.Source)
	}

	// verify the source is compatible
	s, ok := rawS.(compatibleSource)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be one of %q", kind, compatibleSources)
	}

	mcpManifest := tools.McpManifest{
		Name:        cfg.Name,
		Description: cfg.Description,
		InputSchema: cfg.Parameters.McpManifest(),
	}

	// finish tool setup
	t := Tool{
		Name:         cfg.Name,
		Kind:         kind,
		Parameters:   cfg.Parameters,
		Operation:    cfg.Operation,
		AuthRequired: cfg.AuthRequired,
		Client:       s.ElasticsearchClient(),
		manifest:     tools.Manifest{Description: cfg.Description, Parameters: cfg.Parameters.Manifest(), AuthRequired: cfg.AuthRequired},
		mcpManifest:  mcpManifest,
	}
	return t, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Name         string           `yaml:"name"`
	Kind         string           `yaml:"kind"`
	AuthRequired []string         `yaml:"authRequired"`
	Parameters   tools.Parameters `yaml:"parameters"`

	Client      *elasticsearch.Client
	Operation   string
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) Invoke(ctx context.Context, params tools.ParamValues) (any, error) {
	switch strings.ToLower(t.Operation) {
	case "search":
		return t.performSearch(ctx, params)
	case "index":
		return t.performIndex(ctx, params)
	case "delete":
		return t.performDelete(ctx, params)
	case "bulk":
		return t.performBulk(ctx, params)
	case "info":
		return t.performInfo(ctx, params)
	default:
		return nil, fmt.Errorf("unsupported operation: %s", t.Operation)
	}
}

func (t Tool) performSearch(ctx context.Context, params tools.ParamValues) (any, error) {
	paramMap := params.AsMap()

	index, ok := paramMap["index"].(string)
	if !ok {
		return nil, fmt.Errorf("index parameter is required for search operation")
	}

	query, ok := paramMap["query"]
	if !ok {
		return nil, fmt.Errorf("query parameter is required for search operation")
	}

	var queryBytes []byte
	var err error

	switch q := query.(type) {
	case string:
		queryBytes = []byte(q)
	case map[string]interface{}:
		queryBytes, err = json.Marshal(q)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal query: %w", err)
		}
	default:
		return nil, fmt.Errorf("query must be a string or object")
	}

	res, err := t.Client.Search(
		t.Client.Search.WithContext(ctx),
		t.Client.Search.WithIndex(index),
		t.Client.Search.WithBody(bytes.NewReader(queryBytes)),
		t.Client.Search.WithTrackTotalHits(true),
		t.Client.Search.WithPretty(),
	)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("search error: %s", res.Status())
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

func (t Tool) performIndex(ctx context.Context, params tools.ParamValues) (any, error) {
	paramMap := params.AsMap()

	index, ok := paramMap["index"].(string)
	if !ok {
		return nil, fmt.Errorf("index parameter is required for index operation")
	}

	document, ok := paramMap["document"]
	if !ok {
		return nil, fmt.Errorf("document parameter is required for index operation")
	}

	var docBytes []byte
	var err error

	switch doc := document.(type) {
	case string:
		docBytes = []byte(doc)
	case map[string]interface{}:
		docBytes, err = json.Marshal(doc)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal document: %w", err)
		}
	default:
		return nil, fmt.Errorf("document must be a string or object")
	}

	// Check if ID is provided
	var res *esapi.Response
	if id, hasID := paramMap["id"].(string); hasID && id != "" {
		res, err = t.Client.Index(
			index,
			bytes.NewReader(docBytes),
			t.Client.Index.WithContext(ctx),
			t.Client.Index.WithDocumentID(id),
			t.Client.Index.WithPretty(),
		)
	} else {
		res, err = t.Client.Index(
			index,
			bytes.NewReader(docBytes),
			t.Client.Index.WithContext(ctx),
			t.Client.Index.WithPretty(),
		)
	}

	if err != nil {
		return nil, fmt.Errorf("index failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("index error: %s", res.Status())
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

func (t Tool) performDelete(ctx context.Context, params tools.ParamValues) (any, error) {
	paramMap := params.AsMap()

	index, ok := paramMap["index"].(string)
	if !ok {
		return nil, fmt.Errorf("index parameter is required for delete operation")
	}

	id, ok := paramMap["id"].(string)
	if !ok {
		return nil, fmt.Errorf("id parameter is required for delete operation")
	}

	res, err := t.Client.Delete(
		index,
		id,
		t.Client.Delete.WithContext(ctx),
		t.Client.Delete.WithPretty(),
	)
	if err != nil {
		return nil, fmt.Errorf("delete failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("delete error: %s", res.Status())
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

func (t Tool) performBulk(ctx context.Context, params tools.ParamValues) (any, error) {
	paramMap := params.AsMap()

	operations, ok := paramMap["operations"]
	if !ok {
		return nil, fmt.Errorf("operations parameter is required for bulk operation")
	}

	var bulkBody []byte
	var err error

	switch ops := operations.(type) {
	case string:
		bulkBody = []byte(ops)
	case []interface{}:
		var bulkLines []string
		for _, op := range ops {
			opBytes, err := json.Marshal(op)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal operation: %w", err)
			}
			bulkLines = append(bulkLines, string(opBytes))
		}
		bulkBody = []byte(strings.Join(bulkLines, "\n") + "\n")
	default:
		return nil, fmt.Errorf("operations must be a string or array")
	}

	res, err := t.Client.Bulk(
		bytes.NewReader(bulkBody),
		t.Client.Bulk.WithContext(ctx),
		t.Client.Bulk.WithPretty(),
	)
	if err != nil {
		return nil, fmt.Errorf("bulk failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("bulk error: %s", res.Status())
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

func (t Tool) performInfo(ctx context.Context, params tools.ParamValues) (any, error) {
	res, err := t.Client.Info(
		t.Client.Info.WithContext(ctx),
	)
	if err != nil {
		return nil, fmt.Errorf("info failed: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("info error: %s", res.Status())
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
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