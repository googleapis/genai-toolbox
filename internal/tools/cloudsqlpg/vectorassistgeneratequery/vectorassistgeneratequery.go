// Copyright 2026 Google LLC
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

package vectorassistgeneratequery

import (
	"context"
	"fmt"
	"net/http"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/mcp-toolbox/internal/embeddingmodels"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const resourceType string = "vector-assist-generate-query"

const generateQueryStatement = `
    SELECT vector_assist.generate_query(
        spec_id => @spec_id::TEXT, table_name => @table_name::TEXT,
        schema_name => @schema_name::TEXT, column_name => @column_name::TEXT,
        search_text => @search_text::TEXT, search_vector => @search_vector::vector,
        output_column_names => @output_column_names,
        top_k => @top_k::INTEGER,
        filter_expressions => @filter_expressions,
        target_recall => @target_recall::FLOAT,
        iterative_index_search => @iterative_index_search::BOOLEAN
      );
`

func init() {
	if !tools.Register(resourceType, newConfig) {
		panic(fmt.Sprintf("tool type %q already registered", resourceType))
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
	PostgresPool() *pgxpool.Pool
	RunSQL(context.Context, string, []any) (any, error)
}

type Config struct {
	Name         string   `yaml:"name" validate:"required"`
	Type         string   `yaml:"type" validate:"required"`
	Source       string   `yaml:"source" validate:"required"`
	Description  string   `yaml:"description"`
	AuthRequired []string `yaml:"authRequired"`

	ScopesRequired []string `yaml:"scopesRequired"`
}

var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigType() string {
	return resourceType
}

func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	// parameters are marked required/ optional based on the vector assist function defintions
	allParameters := parameters.Parameters{
		parameters.NewStringParameterWithRequired("spec_id", "Generate the vector query corresponding to this vector spec.", false),
		parameters.NewStringParameterWithRequired("table_name", "Generate the vector query corresponding to this table (in case of a single spec defined on the table).", false),
		parameters.NewStringParameterWithRequired("schema_name", "Schema name for the table related to the vector query generation.", false),
		parameters.NewStringParameterWithRequired("column_name", "text_column_name or vector_column_name of the spec to identify the exact spec in case there are multiple specs defined on a table.", false),
		parameters.NewStringParameterWithRequired("search_text", "Text search for which query needs to be generated. Embeddings are generated using the model defined in the vector spec.", false),
		parameters.NewStringParameterWithRequired("search_vector", "Vector for which query needs to be generated. Only one of search_text or search_vector must be populated.", false),
		parameters.NewArrayParameterWithRequired("output_column_names", "Column names to retrieve in the output search query. Defaults to retrieving all columns.", false, parameters.NewStringParameter("output_column_name", "Output column name")),
		parameters.NewIntParameterWithRequired("top_k", "Number of nearest neighbors to be returned in the vector search query. Defaults to 10.", false),
		parameters.NewArrayParameterWithRequired("filter_expressions", "Any filter expressions to be applied on the vector search query.", false, parameters.NewStringParameter("filter_expression", "Filter expression")),
		parameters.NewFloatParameterWithRequired("target_recall", "The recall that the user would like to target with the given query. Overrides the spec-level target_recall.", false),
		parameters.NewBooleanParameterWithRequired("iterative_index_search", "Perform iterative index search for filtered queries to ensure enough results are returned.", false),
	}
	paramManifest := allParameters.Manifest()

	if cfg.Description == "" {
		cfg.Description = "This tool generates optimized SQL queries for vector search by leveraging the metadata and vector specifications defined in a specific spec_id. It may return a single query or a sequence of multiple SQL queries that can be executed sequentially. Use this tool when a user wants to perform semantic or similarity searches on their data. It serves as the primary actionable tool to invoke for generating the executable SQL required to retrieve relevant results based on vector similarity. The 'execute_sql' tool can be used as a follow-up action after invoking this tool."
	}

	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, allParameters, nil)

	return Tool{
		Config:    cfg,
		allParams: allParameters,
		manifest: tools.Manifest{
			Description:  cfg.Description,
			Parameters:   paramManifest,
			AuthRequired: cfg.AuthRequired,
		},
		mcpManifest: mcpManifest,
	}, nil
}

var _ tools.Tool = Tool{}

type Tool struct {
	Config
	allParams   parameters.Parameters `yaml:"allParams"`
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, util.ToolboxError) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Source, t.Name, t.Type)
	if err != nil {
		return nil, util.NewClientServerError("source used is not compatible with the tool", http.StatusInternalServerError, err)
	}
	paramsMap := params.AsMap()

	// Convert our parsed parameters directly into pgx.NamedArgs
	namedArgs := pgx.NamedArgs{}
	for key, value := range paramsMap {
		namedArgs[key] = value
	}

	// As long as source.RunSQL unwraps args into pgx.Query(ctx, sql, args...), pgx handles the mapping of @param to the named parameter.
	resp, err := source.RunSQL(ctx, generateQueryStatement, []any{namedArgs})
	if err != nil {
		return nil, util.ProcessGeneralError(err)
	}
	return resp, nil
}

func (t Tool) EmbedParams(ctx context.Context, paramValues parameters.ParamValues, embeddingModelsMap map[string]embeddingmodels.EmbeddingModel) (parameters.ParamValues, error) {
	return parameters.EmbedParams(ctx, t.allParams, paramValues, embeddingModelsMap, nil)
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

func (t Tool) GetParameters() parameters.Parameters {
	return t.allParams
}

func (t Tool) GetScopesRequired() []string {
	return t.ScopesRequired
}
