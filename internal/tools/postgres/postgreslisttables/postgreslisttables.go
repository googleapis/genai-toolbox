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

package postgreslisttables

import (
	"context"
	"fmt"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/alloydbpg"
	"github.com/googleapis/genai-toolbox/internal/sources/cloudsqlpg"
	"github.com/googleapis/genai-toolbox/internal/sources/postgres"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/jackc/pgx/v5/pgxpool"
)

const kind string = "postgres-list-tables"

const listTablesStatement = `
	WITH desired_relkinds AS (
		SELECT ARRAY['r', 'p']::char[] AS kinds -- Always consider both 'TABLE' and 'PARTITIONED TABLE'
	),
	table_info AS (
		SELECT
			t.oid AS table_oid,
			ns.nspname AS schema_name,
			t.relname AS table_name,
			pg_get_userbyid(t.relowner) AS table_owner,
			obj_description(t.oid, 'pg_class') AS table_comment,
			t.relkind AS object_kind
		FROM
			pg_class t
		JOIN
			pg_namespace ns ON ns.oid = t.relnamespace
		CROSS JOIN desired_relkinds dk
		WHERE
			t.relkind = ANY(dk.kinds) -- Filter by selected table relkinds ('r', 'p')
			AND (NULLIF(TRIM($1), '') IS NULL OR t.relname = ANY(string_to_array($1,','))) -- $1 is object_names
			AND ns.nspname NOT IN ('pg_catalog', 'information_schema', 'pg_toast')
			AND ns.nspname NOT LIKE 'pg_temp_%' AND ns.nspname NOT LIKE 'pg_toast_temp_%'
	),
	columns_info AS (
		SELECT
			att.attrelid AS table_oid, att.attname AS column_name, format_type(att.atttypid, att.atttypmod) AS data_type,
			att.attnum AS column_ordinal_position, att.attnotnull AS is_not_nullable,
			pg_get_expr(ad.adbin, ad.adrelid) AS column_default, col_description(att.attrelid, att.attnum) AS column_comment
		FROM pg_attribute att LEFT JOIN pg_attrdef ad ON att.attrelid = ad.adrelid AND att.attnum = ad.adnum
		JOIN table_info ti ON att.attrelid = ti.table_oid WHERE att.attnum > 0 AND NOT att.attisdropped
	),
	constraints_info AS (
		SELECT
			con.conrelid AS table_oid, con.conname AS constraint_name, pg_get_constraintdef(con.oid) AS constraint_definition,
			CASE con.contype WHEN 'p' THEN 'PRIMARY KEY' WHEN 'f' THEN 'FOREIGN KEY' WHEN 'u' THEN 'UNIQUE' WHEN 'c' THEN 'CHECK' ELSE con.contype::text END AS constraint_type,
			(SELECT array_agg(att.attname ORDER BY u.attposition) FROM unnest(con.conkey) WITH ORDINALITY AS u(attnum, attposition) JOIN pg_attribute att ON att.attrelid = con.conrelid AND att.attnum = u.attnum) AS constraint_columns,
			NULLIF(con.confrelid, 0)::regclass AS foreign_key_referenced_table,
			(SELECT array_agg(att.attname ORDER BY u.attposition) FROM unnest(con.confkey) WITH ORDINALITY AS u(attnum, attposition) JOIN pg_attribute att ON att.attrelid = con.confrelid AND att.attnum = u.attnum WHERE con.contype = 'f') AS foreign_key_referenced_columns
		FROM pg_constraint con JOIN table_info ti ON con.conrelid = ti.table_oid
	),
	indexes_info AS (
		SELECT
			idx.indrelid AS table_oid, ic.relname AS index_name, pg_get_indexdef(idx.indexrelid) AS index_definition,
			idx.indisunique AS is_unique, idx.indisprimary AS is_primary, am.amname AS index_method,
			(SELECT array_agg(att.attname ORDER BY u.ord) FROM unnest(idx.indkey::int[]) WITH ORDINALITY AS u(colidx, ord) LEFT JOIN pg_attribute att ON att.attrelid = idx.indrelid AND att.attnum = u.colidx WHERE u.colidx <> 0) AS index_columns
		FROM pg_index idx JOIN pg_class ic ON ic.oid = idx.indexrelid JOIN pg_am am ON am.oid = ic.relam JOIN table_info ti ON idx.indrelid = ti.table_oid
	),
	triggers_info AS (
		SELECT tg.tgrelid AS table_oid, tg.tgname AS trigger_name, pg_get_triggerdef(tg.oid) AS trigger_definition, tg.tgenabled AS trigger_enabled_state
		FROM pg_trigger tg JOIN table_info ti ON tg.tgrelid = ti.table_oid WHERE NOT tg.tgisinternal
	)
	SELECT
		ti.schema_name,
		ti.table_name AS object_name,
		CASE
		  WHEN $2 = 'simple' THEN
			  -- IF format is 'simple', return basic JSON
			  json_build_object('name', ti.table_name)
		  ELSE
			json_build_object(
				'schema_name', ti.schema_name,
				'object_name', ti.table_name,
				'object_type', CASE ti.object_kind
								WHEN 'r' THEN 'TABLE'
								WHEN 'p' THEN 'PARTITIONED TABLE'
								ELSE ti.object_kind::text -- Should not happen due to WHERE clause
							END,
				'owner', ti.table_owner,
				'comment', ti.table_comment,
				'columns', COALESCE((SELECT json_agg(json_build_object('column_name',ci.column_name,'data_type',ci.data_type,'ordinal_position',ci.column_ordinal_position,'is_not_nullable',ci.is_not_nullable,'column_default',ci.column_default,'column_comment',ci.column_comment) ORDER BY ci.column_ordinal_position) FROM columns_info ci WHERE ci.table_oid = ti.table_oid), '[]'::json),
				'constraints', COALESCE((SELECT json_agg(json_build_object('constraint_name',cons.constraint_name,'constraint_type',cons.constraint_type,'constraint_definition',cons.constraint_definition,'constraint_columns',cons.constraint_columns,'foreign_key_referenced_table',cons.foreign_key_referenced_table,'foreign_key_referenced_columns',cons.foreign_key_referenced_columns)) FROM constraints_info cons WHERE cons.table_oid = ti.table_oid), '[]'::json),
				'indexes', COALESCE((SELECT json_agg(json_build_object('index_name',ii.index_name,'index_definition',ii.index_definition,'is_unique',ii.is_unique,'is_primary',ii.is_primary,'index_method',ii.index_method,'index_columns',ii.index_columns)) FROM indexes_info ii WHERE ii.table_oid = ti.table_oid), '[]'::json),
				'triggers', COALESCE((SELECT json_agg(json_build_object('trigger_name',tri.trigger_name,'trigger_definition',tri.trigger_definition,'trigger_enabled_state',tri.trigger_enabled_state)) FROM triggers_info tri WHERE tri.table_oid = ti.table_oid), '[]'::json)
			) 
		END AS object_details
	FROM table_info ti ORDER BY ti.schema_name, ti.table_name;
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
	Description  string   `yaml:"description" validate:"required"`
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

	allParameters := tools.Parameters{
		tools.NewStringParameterWithDefault("table_names", "", "Optional: A comma-separated list of table names. If empty, details for all tables will be listed."),
		tools.NewStringParameterWithDefault("output_format", "detailed", "Optional: Use 'simple' for names only or 'detailed' for full info."),
	}
	paramManifest := allParameters.Manifest()
	inputSchema := allParameters.McpManifest()

	mcpManifest := tools.McpManifest{
		Name:        cfg.Name,
		Description: cfg.Description,
		InputSchema: inputSchema,
	}

	t := Tool{
		Name:         cfg.Name,
		Kind:         kind,
		AuthRequired: cfg.AuthRequired,
		AllParams:    allParameters,
		Pool:         s.PostgresPool(),
		manifest:     tools.Manifest{Description: cfg.Description, Parameters: paramManifest, AuthRequired: cfg.AuthRequired},
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
	AllParams    tools.Parameters `yaml:"allParams"`

	Pool        *pgxpool.Pool
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) Invoke(ctx context.Context, params tools.ParamValues, accessToken tools.AccessToken) (any, error) {
	paramsMap := params.AsMap()

	tableNames, ok := paramsMap["table_names"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid 'table_names' parameter; expected a string")
	}
	outputFormat, _ := paramsMap["output_format"].(string)
	if outputFormat != "simple" && outputFormat != "detailed" {
		return nil, fmt.Errorf("invalid value for output_format: must be 'simple' or 'detailed', but got %q", outputFormat)
	}

	results, err := t.Pool.Query(ctx, listTablesStatement, tableNames, outputFormat)
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

	if err := results.Err(); err != nil {
		return nil, fmt.Errorf("error reading query results: %w", err)
	}

	return out, nil
}

func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (tools.ParamValues, error) {
	return tools.ParseParams(t.AllParams, data, claims)
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
