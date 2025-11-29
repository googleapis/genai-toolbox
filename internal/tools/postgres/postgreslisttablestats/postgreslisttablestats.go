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

package postgreslisttablestats

import (
	"context"
	"fmt"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/alloydbpg"
	"github.com/googleapis/genai-toolbox/internal/sources/cloudsqlpg"
	"github.com/googleapis/genai-toolbox/internal/sources/postgres"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
	"github.com/jackc/pgx/v5/pgxpool"
)

const kind string = "postgres-list-table-stats"

const listTableStats = `
	WITH table_stats AS (
        SELECT
            s.schemaname AS schema_name,
            s.relname AS table_name,
            pg_catalog.pg_get_userbyid(c.relowner) AS owner,
            pg_total_relation_size(s.relid) AS total_size_bytes,
            s.seq_scan,
            s.idx_scan,
            -- Ratio of index scans to total scans
            CASE
                WHEN (s.seq_scan + s.idx_scan) = 0 THEN 0
                ELSE round((s.idx_scan * 100.0) / (s.seq_scan + s.idx_scan), 2)
            END AS idx_scan_ratio_percent,
            s.n_live_tup AS live_rows,
            s.n_dead_tup AS dead_rows,
            -- Percentage of rows that are "dead" (bloat)
            CASE
                WHEN (s.n_live_tup + s.n_dead_tup) = 0 THEN 0
                ELSE round((s.n_dead_tup * 100.0) / (s.n_live_tup + s.n_dead_tup), 2)
            END AS dead_row_ratio_percent,
            s.n_tup_ins,
            s.n_tup_upd,
            s.n_tup_del,
            s.last_vacuum,
            s.last_autovacuum,
            s.last_autoanalyze
        FROM pg_stat_all_tables s
        JOIN pg_catalog.pg_class c ON s.relid = c.oid
      )
      SELECT *
      FROM table_stats
      WHERE
        ($1::text IS NULL OR schema_name LIKE '%' || $1::text || '%')
        AND ($2::text IS NULL OR table_name LIKE '%' || $2::text || '%')
        AND ($3::text IS NULL OR owner LIKE '%' || $3::text || '%')
      ORDER BY
        CASE
          WHEN $4::text = 'size' THEN total_size_bytes
          WHEN $4::text = 'dead_rows' THEN dead_rows
          WHEN $4::text = 'seq_scan' THEN seq_scan
          WHEN $4::text = 'idx_scan' THEN idx_scan
          ELSE seq_scan
        END DESC
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
}

// validate compatible sources are still compatible
var _ compatibleSource = &alloydbpg.Source{}
var _ compatibleSource = &cloudsqlpg.Source{}
var _ compatibleSource = &postgres.Source{}

var compatibleSources = [...]string{alloydbpg.SourceKind, cloudsqlpg.SourceKind, postgres.SourceKind}

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

	allParameters := parameters.Parameters{
		parameters.NewStringParameterWithDefault("schema_name", "public", "Optional: A specific schema name to filter by"),
		parameters.NewStringParameterWithRequired("table_name", "Optional: A specific table name to filter by", false),
		parameters.NewStringParameterWithRequired("owner", "Optional: A specific owner to filter by", false),
		parameters.NewStringParameterWithRequired("sort_by", "Optional: The column to sort by", false),
		parameters.NewIntParameterWithDefault("limit", 50, "Optional: The maximum number of results to return"),
	}
	paramManifest := allParameters.Manifest()

	if cfg.Description == "" {
		cfg.Description = `Lists the user table statistics in the database ordered by number of
		sequential scans with a default limit of 50 rows. Returns the following
		columns: schema name, table name, table size in bytes, number of
		sequential scans, number of index scans, idx_scan_ratio_percent (showing
		the percentage of total scans that utilized an index, where a low ratio
		indicates missing or ineffective indexes), number of live rows, number
		of dead rows, dead_row_ratio_percent (indicating potential table bloat),
		total number of rows inserted, updated, and deleted, the timestamps
		for the last_vacuum, last_autovacuum, and last_autoanalyze operations.`
	}

	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, allParameters, nil)

	// finish tool setup
	return Tool{
		name:         cfg.Name,
		kind:         cfg.Kind,
		authRequired: cfg.AuthRequired,
		allParams:    allParameters,
		pool:         s.PostgresPool(),
		manifest: tools.Manifest{
			Description:  cfg.Description,
			Parameters:   paramManifest,
			AuthRequired: cfg.AuthRequired,
		},
		mcpManifest: mcpManifest,
	}, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Config
	name         string                `yaml:"name"`
	kind         string                `yaml:"kind"`
	authRequired []string              `yaml:"authRequired"`
	allParams    parameters.Parameters `yaml:"allParams"`
	pool         *pgxpool.Pool
	manifest     tools.Manifest
	mcpManifest  tools.McpManifest
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

func (t Tool) Invoke(ctx context.Context, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	paramsMap := params.AsMap()

	newParams, err := parameters.GetParams(t.allParams, paramsMap)
	if err != nil {
		return nil, fmt.Errorf("unable to extract standard params %w", err)
	}
	sliceParams := newParams.AsSlice()

	results, err := t.pool.Query(ctx, listTableStats, sliceParams...)
	if err != nil {
		return nil, fmt.Errorf("unable to execute query: %w", err)
	}
	defer results.Close()

	fields := results.FieldDescriptions()
	var out []map[string]any

	for results.Next() {
		values, err := results.Values()
		if err != nil {
			return nil, fmt.Errorf("unable to parse row: %w", err)
		}
		rowMap := make(map[string]any)
		for i, field := range fields {
			rowMap[string(field.Name)] = values[i]
		}
		out = append(out, rowMap)
	}

	return out, nil
}

func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (parameters.ParamValues, error) {
	return parameters.ParseParams(t.allParams, data, claims)
}

func (t Tool) Manifest() tools.Manifest {
	return t.manifest
}

func (t Tool) McpManifest() tools.McpManifest {
	return t.mcpManifest
}

func (t Tool) Authorized(verifiedAuthServices []string) bool {
	return tools.IsAuthorized(t.authRequired, verifiedAuthServices)
}

func (t Tool) RequiresClientAuthorization() bool {
	return false
}

func (t Tool) GetAuthTokenHeaderName() string {
	return "Authorization"
}
