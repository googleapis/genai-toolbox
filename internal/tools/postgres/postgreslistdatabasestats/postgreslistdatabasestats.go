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

package postgreslistdatabasestats

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

const kind string = "postgres-list-database-stats"

// SQL query to list database statistics
const listDatabaseStats = `
	WITH database_stats AS (
		SELECT
			s.datname AS database_name, 
			-- Database Metadata 
			d.datallowconn AS is_connectable,
			pg_get_userbyid(d.datdba) AS database_owner, 
			ts.spcname AS default_tablespace,             

			-- Cache Performance
			CASE
				WHEN (s.blks_hit + s.blks_read) = 0 THEN 0
				ELSE round((s.blks_hit * 100.0) / (s.blks_hit + s.blks_read), 2)
			END AS cache_hit_ratio_percent,
			s.blks_read AS blocks_read_from_disk,
			s.blks_hit AS blocks_hit_in_cache,

			-- Transaction Throughput
			s.xact_commit,
			s.xact_rollback,
			round(s.xact_rollback * 100.0 / (s.xact_commit + s.xact_rollback + 1), 2) AS rollback_ratio_percent,

			-- Tuple Activity
			s.tup_returned AS rows_returned_by_queries,
			s.tup_fetched AS rows_fetched_by_scans,
			s.tup_inserted,
			s.tup_updated,
			s.tup_deleted,

			-- Temporary File Usage
			s.temp_files,
			s.temp_bytes AS temp_size_bytes,

			-- Conflicts & Deadlocks
			s.conflicts,
			s.deadlocks,

			-- General Info
			s.numbackends AS active_connections,
			s.stats_reset AS statistics_last_reset,
			pg_database_size(s.datid) AS database_size_bytes 
		FROM
			pg_stat_database s
		JOIN
			pg_database d ON d.oid = s.datid
		JOIN
			pg_tablespace ts ON ts.oid = d.dattablespace
		WHERE
			-- Exclude cloudsql internal databases
			s.datname NOT IN ('cloudsqladmin')
			-- Exclude template databases if not requested
			AND ( $2::boolean IS TRUE OR d.datistemplate IS FALSE )
	)
	SELECT *
	FROM database_stats
	WHERE
		($1::text IS NULL OR database_name LIKE '%' || $1::text || '%')
		AND ($3::text IS NULL OR database_owner LIKE '%' || $3::text || '%')
		AND ($4::text IS NULL OR default_tablespace LIKE '%' || $4::text || '%')
	ORDER BY
		CASE WHEN $5::text = 'size' THEN database_size_bytes END DESC,
		CASE WHEN $5::text = 'commit' THEN xact_commit END DESC,
		database_name
	LIMIT COALESCE($6::int, 10);
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
		parameters.NewStringParameterWithDefault("database_name", "", "Optional: A specific database name pattern to search for."),
		parameters.NewBooleanParameterWithDefault("include_templates", false, "Optional: Whether to include template databases in the results."),
		parameters.NewStringParameterWithDefault("database_owner", "", "Optional: A specific database owner name pattern to search for."),
		parameters.NewStringParameterWithDefault("default_tablespace", "", "Optional: A specific default tablespace name pattern to search for."),
		parameters.NewStringParameterWithDefault("order_by", "", "Optional: The field to order the results by. Valid values are 'size' and 'commit'."),
		parameters.NewIntParameterWithDefault("limit", 10, "Optional: The maximum number of rows to return."),
	}
	description := cfg.Description
	if description == "" {
		description =
			"Lists the key performance and activity statistics for each PostgreSQL database" +
				"in the instance, offering insights into cache efficiency, transaction throughput" +
				"row-level activity, temporary file " +
				"usage, and contention. " +
				"It returns: the database name, whether the database is connectable,  " +
				"database owner, default tablespace name, the percentage of data blocks " +
				"found in the buffer cache rather than being read from disk (a higher " +
				"value indicates better cache performance), the total number of disk " +
				"blocks read from disk, the total number of times disk blocks were found " +
				"already in the cache; the total number of committed transactions, the " +
				"total number of rolled back transactions, the percentage of rolled back " +
				"transactions compared to the total number of completed transactions, the " +
				"total number of rows returned by queries, the total number of live rows " +
				"fetched by scans, the total number of rows inserted, the total number " +
				"of rows updated, the total number of rows deleted, the number of " +
				"temporary files created by queries, the total size of all temporary " +
				"files created by queries in bytes, the number of query cancellations due " +
				"to conflicts with recovery, the number of deadlocks detected, the current " +
				"number of active connections to the database, the timestamp of the " +
				"last statistics reset, and total database size in bytes."
	}
	mcpManifest := tools.GetMcpManifest(cfg.Name, description, cfg.AuthRequired, allParameters, nil)

	// finish tool setup
	return Tool{
		Config:    cfg,
		allParams: allParameters,
		pool:      s.PostgresPool(),
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
	pool        *pgxpool.Pool
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	paramsMap := params.AsMap()

	newParams, err := parameters.GetParams(t.allParams, paramsMap)
	if err != nil {
		return nil, fmt.Errorf("unable to extract standard params %w", err)
	}
	sliceParams := newParams.AsSlice()

	results, err := t.pool.Query(ctx, listDatabaseStats, sliceParams...)
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

	// this will catch actual query execution errors
	if err := results.Err(); err != nil {
		return nil, fmt.Errorf("unable to execute query: %w", err)
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
	return tools.IsAuthorized(t.AuthRequired, verifiedAuthServices)
}

func (t Tool) RequiresClientAuthorization(resourceMgr tools.SourceProvider) bool {
	return false
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

func (t Tool) GetAuthTokenHeaderName() string {
	return "Authorization"
}
