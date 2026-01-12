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

package postgreslistindexes

import (
	"context"
	"fmt"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/embeddingmodels"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
	"github.com/jackc/pgx/v5/pgxpool"
)

const kind string = "postgres-list-indexes"

const listIndexesStatement = `
	WITH IndexDetails AS (
		SELECT
			s.schemaname AS schema_name,
			t.relname AS table_name,
			i.relname AS index_name,
			am.amname AS index_type,
			ix.indisunique AS is_unique,
			ix.indisprimary AS is_primary,
			pg_get_indexdef(i.oid) AS index_definition,
			pg_relation_size(i.oid) AS index_size_bytes,
			s.idx_scan AS index_scans,
			s.idx_tup_read AS tuples_read,
			s.idx_tup_fetch AS tuples_fetched,
			CASE 
				WHEN s.idx_scan > 0 THEN true 
				ELSE false 
			END AS is_used
		FROM pg_catalog.pg_class t
		JOIN pg_catalog.pg_index ix
			ON t.oid = ix.indrelid
		JOIN pg_catalog.pg_class i
			ON i.oid = ix.indexrelid
		JOIN pg_catalog.pg_am am
			ON i.relam = am.oid
		JOIN pg_catalog.pg_stat_all_indexes s
			ON i.oid = s.indexrelid
		WHERE
			t.relkind = 'r'
			AND s.schemaname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
			AND s.schemaname NOT LIKE 'pg_temp_%'
	)
	SELECT *
	FROM IndexDetails
	WHERE
		($1::text IS NULL OR schema_name LIKE '%' || $1 || '%')
		AND ($2::text IS NULL OR table_name LIKE '%' || $2 || '%')
		AND ($3::text IS NULL OR index_name LIKE '%' || $3 || '%')
		AND ($4::boolean IS NOT TRUE OR is_used IS FALSE)
	ORDER BY
		schema_name,
		table_name,
		index_name
	LIMIT COALESCE($5::int, 50);
`

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
	PostgresPool() *pgxpool.Pool
	RunSQL(context.Context, string, []any) (any, error)
}

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
	allParameters := parameters.Parameters{
		parameters.NewStringParameterWithDefault("schema_name", "", "Optional: a text to filter results by schema name. The input is used within a LIKE clause."),
		parameters.NewStringParameterWithDefault("table_name", "", "Optional: a text to filter results by table name. The input is used within a LIKE clause."),
		parameters.NewStringParameterWithDefault("index_name", "", "Optional: a text to filter results by index name. The input is used within a LIKE clause."),
		parameters.NewBooleanParameterWithDefault("only_unused", false, "Optional: If true, only returns indexes that have never been used."),
		parameters.NewIntParameterWithDefault("limit", 50, "Optional: The maximum number of rows to return. Default is 50"),
	}

	if cfg.Description == "" {
		cfg.Description = "Lists available user indexes in the database, excluding system schemas (pg_catalog, information_schema). For each index, the following properties are returned: schema name, table name, index name, index type (access method), a boolean indicating if it's a unique index, a boolean indicating if it's for a primary key, the index definition, index size in bytes, the number of index scans, the number of index tuples read, the number of table tuples fetched via index scans, and a boolean indicating if the index has been used at least once."
	}
	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, allParameters, nil)

	// finish tool setup
	return Tool{
		Config:    cfg,
		allParams: allParameters,
		manifest: tools.Manifest{
			Description:  cfg.Description,
			Parameters:   allParameters.Manifest(),
			AuthRequired: cfg.AuthRequired,
		},
		mcpManifest: mcpManifest,
	}, nil
}

// validate interface
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

func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Source, t.Name, t.Kind)
	if err != nil {
		return nil, err
	}

	paramsMap := params.AsMap()

	newParams, err := parameters.GetParams(t.allParams, paramsMap)
	if err != nil {
		return nil, fmt.Errorf("unable to extract standard params %w", err)
	}
	sliceParams := newParams.AsSlice()

	return source.RunSQL(ctx, listIndexesStatement, sliceParams)
}

func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (parameters.ParamValues, error) {
	return parameters.ParseParams(t.allParams, data, claims)
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
