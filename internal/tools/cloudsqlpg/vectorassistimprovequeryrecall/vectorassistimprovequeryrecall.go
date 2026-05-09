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

package vectorassistimprovequeryrecall

import (
	"context"
	"encoding/json"
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

const resourceType string = "vector-assist-improve-query-recall"

const listAllIndexesQuery =  `
SELECT indexname, indexdef
FROM pg_indexes
WHERE tablename = @table_name;
`

// Query to check if the index exists and if it is an HNSW index.
const checkIndexQuery = `
	SELECT 
		(COUNT(1) > 0) AS index_present,
		COALESCE(BOOL_OR(indexdef ILIKE '%USING hnsw%'), false) AS is_hnsw
	FROM pg_indexes 
	WHERE schemaname = @schema_name::TEXT
		AND tablename = @table_name::TEXT
		AND indexname = @index_name::TEXT
		AND indexdef ILIKE '%' || @vector_column_name::TEXT || '%';
`

// Query to find the optimal index parameters
const improveRecallQuery = `
  SELECT output_ef_search AS ef_search
  FROM vector_assist.find_ef_search_for_target_recall(
    table_name => @table_name::TEXT,
    schema_name => @schema_name::TEXT,
    column_name => @vector_column_name::TEXT,
    top_k => @top_k::INT,
    target_recall => @target_recall::FLOAT,
    distance_func => @distance_func::TEXT
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
}

var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigType() string {
	return resourceType
}

func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	allParameters := parameters.Parameters{
		parameters.NewStringParameterWithDefault("schema_name", "public", "Optional parameter: Schema name of the table."),
		parameters.NewStringParameterWithRequired("table_name", "Table name experiencing degraded vector search recall.", true),
		parameters.NewStringParameterWithRequired("vector_column_name", "Column name containing the vector embeddings.", true),
		parameters.NewStringParameterWithRequired("index_name", "Name of the vector index to tune.", true),
		parameters.NewIntParameterWithDefault("top_k", 10, "Optional parameter: Top k value for the vector search."),
		parameters.NewFloatParameterWithDefault("target_recall", 0.95, "Optional parameter: Target recall value for search results."),
		parameters.NewStringParameterWithDefault("distance_func", "cosine", "Optional parameter: Distance function used for the vector search similarity."),
	}
	paramManifest := allParameters.Manifest()

	if cfg.Description == "" {
		cfg.Description = "Use this tool to troubleshoot and optimize existing vector search workloads when a user reports irrelevant results, poor accuracy, or degraded recall. It determines the optimal tuning parameter (such as ef_search) for an active vector index to improve the search results. The tool outputs an actionable SQL query recommendation to be executed as a next action using the 'execute_sql' tool."
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

	namedArgs := pgx.NamedArgs{}
	for key, value := range paramsMap {
		namedArgs[key] = value
		fmt.Println("key", key, "value", value)
	}

	listAllIndexesResp, err := source.RunSQL(ctx, listAllIndexesQuery, []any{namedArgs})
	
	fmt.Println("listAllIndexesResp", listAllIndexesResp)
	// print all indexes

	listAllIndexesBytes, marshalErr := json.Marshal(listAllIndexesResp)
	if marshalErr != nil {
		return nil, util.NewClientServerError("failed to process index check response", http.StatusInternalServerError, marshalErr)
	}

	var listAllIndexesRows []map[string]interface{}
	if unmarshalErr := json.Unmarshal(listAllIndexesBytes, &listAllIndexesRows); unmarshalErr != nil || len(listAllIndexesRows) == 0 {
		return nil, util.NewClientServerError("unexpected empty response from database", http.StatusInternalServerError, unmarshalErr)
	}

	for _, index := range listAllIndexesRows {
		fmt.Println("index", index)
	}

	// Check if the index exists and if it is an HNSW index.
	checkResp, err := source.RunSQL(ctx, checkIndexQuery, []any{namedArgs})
	if err != nil {
		return nil, util.ProcessGeneralError(err)
	}

	checkBytes, marshalErr := json.Marshal(checkResp)
	if marshalErr != nil {
		return nil, util.NewClientServerError("failed to process index check response", http.StatusInternalServerError, marshalErr)
	}

	var checkRows []map[string]interface{}
	if unmarshalErr := json.Unmarshal(checkBytes, &checkRows); unmarshalErr != nil || len(checkRows) == 0 {
		return nil, util.NewClientServerError("unexpected empty response from database", http.StatusInternalServerError, unmarshalErr)
	}

	row := checkRows[0]
	
	indexPresent, _ := row["index_present"].(bool)
	if !indexPresent {
		return nil, util.NewClientServerError("Index not found for the given table and vector column. If the table lacks an existing vector setup, use the 'define-spec' tool to configure the database.", http.StatusBadRequest, nil)
	}

	isHnsw, _ := row["is_hnsw"].(bool)
	if !isHnsw {
		return nil, util.NewClientServerError("Unsupported index type for recall optimization. Only HNSW index is supported.", http.StatusBadRequest, nil)
	}

	// Calculate the optimal index parameters to achieve the target recall.
	tuningResp, err := source.RunSQL(ctx, improveRecallQuery, []any{namedArgs})
	if err != nil {
		return nil, util.ProcessGeneralError(err)
	}

	tuningBytes, marshalErr := json.Marshal(tuningResp)
	if marshalErr != nil {
		return nil, util.NewClientServerError("failed to process tuning response", http.StatusInternalServerError, marshalErr)
	}

	var tuningRows []map[string]interface{}
	if unmarshalErr := json.Unmarshal(tuningBytes, &tuningRows); unmarshalErr != nil || len(tuningRows) == 0 {
		return nil, util.NewClientServerError("unexpected empty tuning response from database", http.StatusInternalServerError, unmarshalErr)
	}

	// Extract ef_search (JSON decoder defaults numbers to float64)
	efSearchVal, ok := tuningRows[0]["ef_search"].(float64)
	if !ok {
		return nil, util.NewClientServerError("Failed to calculate appropriate efSearch value", http.StatusInternalServerError, nil)
	}

	queryRecommendation := fmt.Sprintf("SET hnsw.ef_search TO %d;", int(efSearchVal))
	return queryRecommendation, nil
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