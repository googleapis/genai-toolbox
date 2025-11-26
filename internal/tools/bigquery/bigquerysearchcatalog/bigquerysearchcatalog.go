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
	"strings"

	dataplexapi "cloud.google.com/go/dataplex/apiv1"
	dataplexpb "cloud.google.com/go/dataplex/apiv1/dataplexpb"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	bigqueryds "github.com/googleapis/genai-toolbox/internal/sources/bigquery"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
	"google.golang.org/api/iterator"
)

const kind string = "bigquery-search-catalog"

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
	MakeDataplexCatalogClient() func() (*dataplexapi.CatalogClient, bigqueryds.DataplexClientCreator, error)
	BigQueryProject() string
	UseClientAuthorization() bool
}

// validate compatible sources are still compatible
var _ compatibleSource = &bigqueryds.Source{}

var compatibleSources = [...]string{bigqueryds.SourceKind}

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

	// Get the Dataplex client using the method from the source
	makeCatalogClient := s.MakeDataplexCatalogClient()

	prompt := parameters.NewStringParameter("prompt", "Prompt representing search intention. Do not rewrite the prompt.")
	datasetIds := parameters.NewArrayParameterWithDefault("datasetIds", []any{}, "Array of dataset IDs.", parameters.NewStringParameter("datasetId", "The IDs of the bigquery dataset."))
	projectIds := parameters.NewArrayParameterWithDefault("projectIds", []any{}, "Array of project IDs.", parameters.NewStringParameter("projectId", "The IDs of the bigquery project."))
	types := parameters.NewArrayParameterWithDefault("types", []any{}, "Array of data types to filter by.", parameters.NewStringParameter("type", "The type of the data. Accepted values are: CONNECTION, POLICY, DATASET, MODEL, ROUTINE, TABLE, VIEW."))
	pageSize := parameters.NewIntParameterWithDefault("pageSize", 5, "Number of results in the search page.")
	params := parameters.Parameters{prompt, datasetIds, projectIds, types, pageSize}

	description := "Use this tool to find tables, views, models, routines or connections."
	if cfg.Description != "" {
		description = cfg.Description
	}
	mcpManifest := tools.GetMcpManifest(cfg.Name, description, cfg.AuthRequired, params, nil)

	t := Tool{
		Config:            cfg,
		Parameters:        params,
		UseClientOAuth:    s.UseClientAuthorization(),
		MakeCatalogClient: makeCatalogClient,
		ProjectID:         s.BigQueryProject(),
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
	Parameters        parameters.Parameters
	UseClientOAuth    bool
	MakeCatalogClient func() (*dataplexapi.CatalogClient, bigqueryds.DataplexClientCreator, error)
	ProjectID         string
	manifest          tools.Manifest
	mcpManifest       tools.McpManifest
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

func (t Tool) Authorized(verifiedAuthServices []string) bool {
	return tools.IsAuthorized(t.AuthRequired, verifiedAuthServices)
}

func (t Tool) RequiresClientAuthorization() bool {
	return t.UseClientOAuth
}

func constructSearchQueryHelper(predicate string, operator string, items []string) string {
	if len(items) == 0 {
		return ""
	}

	if len(items) == 1 {
		return predicate + operator + items[0]
	}

	var builder strings.Builder
	builder.WriteString("(")
	for i, item := range items {
		if i > 0 {
			builder.WriteString(" OR ")
		}
		builder.WriteString(predicate)
		builder.WriteString(operator)
		builder.WriteString(item)
	}
	builder.WriteString(")")
	return builder.String()
}

func constructSearchQuery(projectIds []string, datasetIds []string, types []string) string {
	queryParts := []string{}

	if clause := constructSearchQueryHelper("projectid", "=", projectIds); clause != "" {
		queryParts = append(queryParts, clause)
	}

	if clause := constructSearchQueryHelper("parent", "=", datasetIds); clause != "" {
		queryParts = append(queryParts, clause)
	}

	if clause := constructSearchQueryHelper("type", "=", types); clause != "" {
		queryParts = append(queryParts, clause)
	}
	queryParts = append(queryParts, "system=bigquery")

	return strings.Join(queryParts, " AND ")
}

type Response struct {
	DisplayName   string
	Description   string
	Type          string
	Resource      string
	DataplexEntry string
}

var typeMap = map[string]string{
	"bigquery-connection":  "CONNECTION",
	"bigquery-data-policy": "POLICY",
	"bigquery-dataset":     "DATASET",
	"bigquery-model":       "MODEL",
	"bigquery-routine":     "ROUTINE",
	"bigquery-table":       "TABLE",
	"bigquery-view":        "VIEW",
}

func ExtractType(resourceString string) string {
	lastIndex := strings.LastIndex(resourceString, "/")
	if lastIndex == -1 {
		// No "/" found, return the original string
		return resourceString
	}
	return typeMap[resourceString[lastIndex+1:]]
}

func (t Tool) Invoke(ctx context.Context, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	paramsMap := params.AsMap()
	pageSize := int32(paramsMap["pageSize"].(int))
	prompt, _ := paramsMap["prompt"].(string)
	projectIdSlice, err := parameters.ConvertAnySliceToTyped(paramsMap["projectIds"].([]any), "string")
	if err != nil {
		return nil, fmt.Errorf("can't convert projectIds to array of strings: %s", err)
	}
	projectIds := projectIdSlice.([]string)
	datasetIdSlice, err := parameters.ConvertAnySliceToTyped(paramsMap["datasetIds"].([]any), "string")
	if err != nil {
		return nil, fmt.Errorf("can't convert datasetIds to array of strings: %s", err)
	}
	datasetIds := datasetIdSlice.([]string)
	typesSlice, err := parameters.ConvertAnySliceToTyped(paramsMap["types"].([]any), "string")
	if err != nil {
		return nil, fmt.Errorf("can't convert types to array of strings: %s", err)
	}
	types := typesSlice.([]string)

	req := &dataplexpb.SearchEntriesRequest{
		Query:          fmt.Sprintf("%s %s", prompt, constructSearchQuery(projectIds, datasetIds, types)),
		Name:           fmt.Sprintf("projects/%s/locations/global", t.ProjectID),
		PageSize:       pageSize,
		SemanticSearch: true,
	}

	catalogClient, dataplexClientCreator, _ := t.MakeCatalogClient()

	if t.UseClientOAuth {
		tokenStr, err := accessToken.ParseBearerToken()
		if err != nil {
			return nil, fmt.Errorf("error parsing access token: %w", err)
		}
		catalogClient, err = dataplexClientCreator(tokenStr)
		if err != nil {
			return nil, fmt.Errorf("error creating client from OAuth access token: %w", err)
		}
	}

	it := catalogClient.SearchEntries(ctx, req)
	if it == nil {
		return nil, fmt.Errorf("failed to create search entries iterator for project %q", t.ProjectID)
	}

	var results []Response
	for {
		entry, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			break
		}
		entrySource := entry.DataplexEntry.GetEntrySource()
		resp := Response{
			DisplayName:   entrySource.GetDisplayName(),
			Description:   entrySource.GetDescription(),
			Type:          ExtractType(entry.DataplexEntry.GetEntryType()),
			Resource:      entrySource.GetResource(),
			DataplexEntry: entry.DataplexEntry.GetName(),
		}
		results = append(results, resp)
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

func (t Tool) GetAuthTokenHeaderName() string {
	return "Authorization"
}
