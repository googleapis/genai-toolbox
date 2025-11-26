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

package dataplexsearchaspecttypes

import (
	"context"
	"fmt"

	dataplexapi "cloud.google.com/go/dataplex/apiv1"
	dataplexpb "cloud.google.com/go/dataplex/apiv1/dataplexpb"
	"github.com/cenkalti/backoff/v5"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	dataplexds "github.com/googleapis/genai-toolbox/internal/sources/dataplex"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
)

const kind string = "dataplex-search-aspect-types"

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

	query := parameters.NewStringParameter("query", "The query against which aspect type should be matched.")
	pageSize := parameters.NewIntParameterWithDefault("pageSize", 5, "Number of returned aspect types in the search page.")
	orderBy := parameters.NewStringParameterWithDefault("orderBy", "relevance", "Specifies the ordering of results. Supported values are: relevance, last_modified_timestamp, last_modified_timestamp asc")
	params := parameters.Parameters{query, pageSize, orderBy}

	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, params, nil)

	t := Tool{
		Config:        cfg,
		Parameters:    params,
		CatalogClient: s.CatalogClient(),
		ProjectID:     s.ProjectID(),
		manifest: tools.Manifest{
			Description:  cfg.Description,
			Parameters:   params.Manifest(),
			AuthRequired: cfg.AuthRequired,
		},
		mcpManifest: mcpManifest,
	}
	return t, nil
}

type Tool struct {
	Config
	Parameters    parameters.Parameters
	CatalogClient *dataplexapi.CatalogClient
	ProjectID     string
	manifest      tools.Manifest
	mcpManifest   tools.McpManifest
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

func (t Tool) Invoke(ctx context.Context, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	// Invoke the tool with the provided parameters
	paramsMap := params.AsMap()
	query, _ := paramsMap["query"].(string)
	pageSize := int32(paramsMap["pageSize"].(int))
	orderBy, _ := paramsMap["orderBy"].(string)

	// Create SearchEntriesRequest with the provided parameters
	req := &dataplexpb.SearchEntriesRequest{
		Query:          query + " type=projects/dataplex-types/locations/global/entryTypes/aspecttype",
		Name:           fmt.Sprintf("projects/%s/locations/global", t.ProjectID),
		PageSize:       pageSize,
		OrderBy:        orderBy,
		SemanticSearch: true,
	}

	// Perform the search using the CatalogClient - this will return an iterator
	it := t.CatalogClient.SearchEntries(ctx, req)
	if it == nil {
		return nil, fmt.Errorf("failed to create search entries iterator for project %q", t.ProjectID)
	}

	// Create an instance of exponential backoff with default values for retrying GetAspectType calls
	// InitialInterval, RandomizationFactor, Multiplier, MaxInterval = 500 ms, 0.5, 1.5, 60 s
	getAspectBackOff := backoff.NewExponentialBackOff()

	// Iterate through the search results and call GetAspectType for each result using the resource name
	var results []*dataplexpb.AspectType
	for {
		entry, err := it.Next()
		if err != nil {
			break
		}
		resourceName := entry.DataplexEntry.GetEntrySource().Resource
		getAspectTypeReq := &dataplexpb.GetAspectTypeRequest{
			Name: resourceName,
		}

		operation := func() (*dataplexpb.AspectType, error) {
			aspectType, err := t.CatalogClient.GetAspectType(ctx, getAspectTypeReq)
			if err != nil {
				return nil, fmt.Errorf("failed to get aspect type for entry %q: %w", resourceName, err)
			}
			return aspectType, nil
		}

		// Retry the GetAspectType operation with exponential backoff
		aspectType, err := backoff.Retry(ctx, operation, backoff.WithBackOff(getAspectBackOff))
		if err != nil {
			return nil, fmt.Errorf("failed to get aspect type after retries for entry %q: %w", resourceName, err)
		}

		results = append(results, aspectType)
	}
	return results, nil
}

func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (parameters.ParamValues, error) {
	// Parse parameters from the provided data
	return parameters.ParseParams(t.Parameters, data, claims)
}

func (t Tool) Manifest() tools.Manifest {
	// Returns the tool manifest
	return t.manifest
}

func (t Tool) McpManifest() tools.McpManifest {
	// Returns the tool MCP manifest
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
