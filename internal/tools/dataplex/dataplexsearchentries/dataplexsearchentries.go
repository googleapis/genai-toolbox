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

package dataplexsearchentries

import (
	"context"
	"fmt"

	dataplexapi "cloud.google.com/go/dataplex/apiv1"
	dataplexpb "cloud.google.com/go/dataplex/apiv1/dataplexpb"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	dataplexds "github.com/googleapis/genai-toolbox/internal/sources/dataplex"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

const kind string = "dataplex-search-entries"

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
	CatalogClient() *dataplexapi.CatalogClient
	ProjectID() string
}

// validate compatible sources are still compatible
var _ compatibleSource = &dataplexds.Source{}

var compatibleSources = [...]string{dataplexds.SourceKind}

type Config struct {
	Name         string   `yaml:"name" validate:"required"`
	Kind         string   `yaml:"kind" validate:"required"`
	Source       string   `yaml:"source" validate:"required"`
	Description  string   `yaml:"description"`
	AuthRequired []string `yaml:"authRequired"`
}

// validate interface
var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigKind() string {
	return kind
}

func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	// Initialize the search configuration with the provided sources
	rawS, ok := srcs[cfg.Source]
	if !ok {
		return nil, fmt.Errorf("no source named %q configured", cfg.Source)
	}
	// verify the source is compatible
	s, ok := rawS.(compatibleSource)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be one of %q", kind, compatibleSources)
	}

	query := tools.NewStringParameter("query", "The query against which entries in scope should be matched.")
	name := tools.NewStringParameterWithDefault("name", fmt.Sprintf("projects/%s/locations/global", s.ProjectID()), "The project to which the request should be attributed in the following form: projects/{project}/locations/global")
	pageSize := tools.NewIntParameterWithDefault("pageSize", 100, "Number of results in the search page.")
	pageToken := tools.NewStringParameterWithDefault("pageToken", "", "Page token received from a previous locations.searchEntries call. Provide this to retrieve the subsequent page.")
	orderBy := tools.NewStringParameterWithDefault("orderBy", "relevance", "Specifies the ordering of results. Supported values are: relevance, last_modified_timestamp, last_modified_timestamp asc")
	semanticSearch := tools.NewBooleanParameterWithDefault("semanticSearch", false, "Whether to use semantic search for the query. If true, the query will be processed using semantic search capabilities.")
	parameters := tools.Parameters{query, name, pageSize, pageToken, orderBy, semanticSearch}

	mcpManifest := tools.McpManifest{
		Name:        cfg.Name,
		Description: cfg.Description,
		InputSchema: parameters.McpManifest(),
	}

	t := &SearchTool{
		Name:          cfg.Name,
		Kind:          kind,
		Parameters:    parameters,
		AuthRequired:  cfg.AuthRequired,
		CatalogClient: s.CatalogClient(),
		ProjectID:     s.ProjectID(),
		manifest: tools.Manifest{
			Description:  cfg.Description,
			Parameters:   parameters.Manifest(),
			AuthRequired: cfg.AuthRequired,
		},
		mcpManifest: mcpManifest,
	}
	return t, nil
}

type SearchTool struct {
	Name          string
	Kind          string
	Parameters    tools.Parameters
	AuthRequired  []string
	CatalogClient *dataplexapi.CatalogClient
	ProjectID     string
	manifest      tools.Manifest
	mcpManifest   tools.McpManifest
}

func (t *SearchTool) Authorized(verifiedAuthServices []string) bool {
	return tools.IsAuthorized(t.AuthRequired, verifiedAuthServices)
}

func (t *SearchTool) Invoke(ctx context.Context, params tools.ParamValues) (any, error) {
	paramsMap := params.AsMap()
	query, _ := paramsMap["query"].(string)
	name, _ := paramsMap["name"].(string)
	pageSize, _ := paramsMap["pageSize"].(int32)
	pageToken, _ := paramsMap["pageToken"].(string)
	orderBy, _ := paramsMap["orderBy"].(string)
	semanticSearch, _ := paramsMap["semanticSearch"].(bool)

	req := &dataplexpb.SearchEntriesRequest{
		Query:          query,
		Name:           name,
		PageSize:       pageSize,
		PageToken:      pageToken,
		OrderBy:        orderBy,
		SemanticSearch: semanticSearch,
	}

	it := t.CatalogClient.SearchEntries(ctx, req)
	if it == nil {
		return nil, fmt.Errorf("failed to create search entries iterator for project %q", t.ProjectID)
	}
	var results []*dataplexpb.SearchEntriesResult
	for {
		entry, err := it.Next()
		if err != nil {
			break
		}
		results = append(results, entry)
	}
	return results, nil
}

func (t *SearchTool) ParseParams(data map[string]any, claims map[string]map[string]any) (tools.ParamValues, error) {
	// Parse parameters from the provided data
	return tools.ParseParams(t.Parameters, data, claims)
}

func (t *SearchTool) Manifest() tools.Manifest {
	// Returns the tool manifest
	return t.manifest
}

func (t *SearchTool) McpManifest() tools.McpManifest {
	// Returns the tool MCP manifest
	return t.mcpManifest
}
