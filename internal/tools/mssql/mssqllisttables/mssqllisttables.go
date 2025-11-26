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

package mssqllisttables

import (
	"context"
	"database/sql"
	"fmt"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/cloudsqlmssql"
	"github.com/googleapis/genai-toolbox/internal/sources/mssql"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
)

const kind string = "mssql-list-tables"

const listTablesStatement = `
	WITH table_info AS (
		SELECT
			t.object_id AS table_oid,
			s.name AS schema_name,
			t.name AS table_name,
			dp.name AS table_owner, -- Schema's owner principal name
			CAST(ep.value AS NVARCHAR(MAX)) AS table_comment, -- Cast for JSON compatibility
			CASE
				WHEN EXISTS ( -- Check if the table has more than one partition for any of its indexes or heap
					SELECT 1 FROM sys.partitions p
					WHERE p.object_id = t.object_id AND p.partition_number > 1
				) THEN 'PARTITIONED TABLE'
				ELSE 'TABLE'
			END AS object_type_detail
		FROM
			sys.tables t
		INNER JOIN
			sys.schemas s ON t.schema_id = s.schema_id
		LEFT JOIN
			sys.database_principals dp ON s.principal_id = dp.principal_id
		LEFT JOIN
			sys.extended_properties ep ON ep.major_id = t.object_id AND ep.minor_id = 0 AND ep.class = 1 AND ep.name = 'MS_Description'
		WHERE
			t.type = 'U' -- User tables
			AND s.name NOT IN ('sys', 'INFORMATION_SCHEMA', 'guest', 'db_owner', 'db_accessadmin', 'db_backupoperator', 'db_datareader', 'db_datawriter', 'db_ddladmin', 'db_denydatareader', 'db_denydatawriter', 'db_securityadmin')
			AND (@table_names IS NULL OR LTRIM(RTRIM(@table_names)) = '' OR t.name IN (SELECT LTRIM(RTRIM(value)) FROM STRING_SPLIT(@table_names, ',')))
	),
	columns_info AS (
		SELECT
			c.object_id AS table_oid,
			c.name AS column_name,
			CONCAT(
				UPPER(TY.name), -- Base type name
				CASE
					WHEN TY.name IN ('char', 'varchar', 'nchar', 'nvarchar', 'binary', 'varbinary') THEN
						CONCAT('(', IIF(c.max_length = -1, 'MAX', CAST(c.max_length / CASE WHEN TY.name IN ('nchar', 'nvarchar') THEN 2 ELSE 1 END AS VARCHAR(10))), ')')
					WHEN TY.name IN ('decimal', 'numeric') THEN
						CONCAT('(', c.precision, ',', c.scale, ')')
					WHEN TY.name IN ('datetime2', 'datetimeoffset', 'time') THEN
						CONCAT('(', c.scale, ')')
					ELSE ''
				END
			) AS data_type,
			c.column_id AS column_ordinal_position,
			IIF(c.is_nullable = 0, CAST(1 AS BIT), CAST(0 AS BIT)) AS is_not_nullable,
			dc.definition AS column_default,
			CAST(epc.value AS NVARCHAR(MAX)) AS column_comment
		FROM
			sys.columns c
		JOIN
			table_info ti ON c.object_id = ti.table_oid
		JOIN
			sys.types TY ON c.user_type_id = TY.user_type_id AND TY.is_user_defined = 0 -- Ensure we get base types
		LEFT JOIN
			sys.default_constraints dc ON c.object_id = dc.parent_object_id AND c.column_id = dc.parent_column_id
		LEFT JOIN
			sys.extended_properties epc ON epc.major_id = c.object_id AND epc.minor_id = c.column_id AND epc.class = 1 AND epc.name = 'MS_Description'
	),
	constraints_info AS (
		-- Primary Keys & Unique Constraints
		SELECT
			kc.parent_object_id AS table_oid,
			kc.name AS constraint_name,
			REPLACE(kc.type_desc, '_CONSTRAINT', '') AS constraint_type, -- 'PRIMARY_KEY', 'UNIQUE'
			STUFF((SELECT ', ' + col.name
				FROM sys.index_columns ic
				JOIN sys.columns col ON ic.object_id = col.object_id AND ic.column_id = col.column_id
				WHERE ic.object_id = kc.parent_object_id AND ic.index_id = kc.unique_index_id
				ORDER BY ic.key_ordinal
				FOR XML PATH(''), TYPE).value('.', 'NVARCHAR(MAX)'), 1, 2, '') AS constraint_columns,
			NULL AS foreign_key_referenced_table,
			NULL AS foreign_key_referenced_columns,
			CASE kc.type
				WHEN 'PK' THEN 'PRIMARY KEY (' + STUFF((SELECT ', ' + col.name FROM sys.index_columns ic JOIN sys.columns col ON ic.object_id = col.object_id AND ic.column_id = col.column_id WHERE ic.object_id = kc.parent_object_id AND ic.index_id = kc.unique_index_id ORDER BY ic.key_ordinal FOR XML PATH(''), TYPE).value('.', 'NVARCHAR(MAX)'), 1, 2, '') + ')'
				WHEN 'UQ' THEN 'UNIQUE (' + STUFF((SELECT ', ' + col.name FROM sys.index_columns ic JOIN sys.columns col ON ic.object_id = col.object_id AND ic.column_id = col.column_id WHERE ic.object_id = kc.parent_object_id AND ic.index_id = kc.unique_index_id ORDER BY ic.key_ordinal FOR XML PATH(''), TYPE).value('.', 'NVARCHAR(MAX)'), 1, 2, '') + ')'
			END AS constraint_definition
		FROM sys.key_constraints kc
		JOIN table_info ti ON kc.parent_object_id = ti.table_oid
		UNION ALL
		-- Foreign Keys
		SELECT
			fk.parent_object_id AS table_oid,
			fk.name AS constraint_name,
			'FOREIGN KEY' AS constraint_type,
			STUFF((SELECT ', ' + pc.name
				FROM sys.foreign_key_columns fkc
				JOIN sys.columns pc ON fkc.parent_object_id = pc.object_id AND fkc.parent_column_id = pc.column_id
				WHERE fkc.constraint_object_id = fk.object_id
				ORDER BY fkc.constraint_column_id
				FOR XML PATH(''), TYPE).value('.', 'NVARCHAR(MAX)'), 1, 2, '') AS constraint_columns,
			SCHEMA_NAME(rt.schema_id) + '.' + OBJECT_NAME(fk.referenced_object_id) AS foreign_key_referenced_table,
			STUFF((SELECT ', ' + rc.name
				FROM sys.foreign_key_columns fkc
				JOIN sys.columns rc ON fkc.referenced_object_id = rc.object_id AND fkc.referenced_column_id = rc.column_id
				WHERE fkc.constraint_object_id = fk.object_id
				ORDER BY fkc.constraint_column_id
				FOR XML PATH(''), TYPE).value('.', 'NVARCHAR(MAX)'), 1, 2, '') AS foreign_key_referenced_columns,
			OBJECT_DEFINITION(fk.object_id) AS constraint_definition
		FROM sys.foreign_keys fk
		JOIN sys.tables rt ON fk.referenced_object_id = rt.object_id
		JOIN table_info ti ON fk.parent_object_id = ti.table_oid
		UNION ALL
		-- Check Constraints
		SELECT
			cc.parent_object_id AS table_oid,
			cc.name AS constraint_name,
			'CHECK' AS constraint_type,
			NULL AS constraint_columns, -- Definition includes column context
			NULL AS foreign_key_referenced_table,
			NULL AS foreign_key_referenced_columns,
			cc.definition AS constraint_definition
		FROM sys.check_constraints cc
		JOIN table_info ti ON cc.parent_object_id = ti.table_oid
	),
	indexes_info AS (
		SELECT
			i.object_id AS table_oid,
			i.name AS index_name,
			i.type_desc AS index_method, -- CLUSTERED, NONCLUSTERED, XML, etc.
			i.is_unique,
			i.is_primary_key AS is_primary,
			STUFF((SELECT ', ' + c.name
				FROM sys.index_columns ic
				JOIN sys.columns c ON i.object_id = c.object_id AND ic.column_id = c.column_id
				WHERE ic.object_id = i.object_id AND ic.index_id = i.index_id AND ic.is_included_column = 0
				ORDER BY ic.key_ordinal
				FOR XML PATH(''), TYPE).value('.', 'NVARCHAR(MAX)'), 1, 2, '') AS index_columns,
			(
				'COLUMNS: (' + ISNULL(STUFF((SELECT ', ' + c.name + CASE WHEN ic.is_descending_key = 1 THEN ' DESC' ELSE '' END
										FROM sys.index_columns ic
										JOIN sys.columns c ON i.object_id = c.object_id AND ic.column_id = c.column_id
										WHERE ic.object_id = i.object_id AND ic.index_id = i.index_id AND ic.is_included_column = 0
										ORDER BY ic.key_ordinal FOR XML PATH(''), TYPE).value('.', 'NVARCHAR(MAX)'), 1, 2, ''), 'N/A') + ')' +
				ISNULL(CHAR(13)+CHAR(10) + 'INCLUDE: (' + STUFF((SELECT ', ' + c.name
										FROM sys.index_columns ic
										JOIN sys.columns c ON i.object_id = c.object_id AND ic.column_id = c.column_id
										WHERE ic.object_id = i.object_id AND ic.index_id = i.index_id AND ic.is_included_column = 1
										ORDER BY ic.index_column_id FOR XML PATH(''), TYPE).value('.', 'NVARCHAR(MAX)'), 1, 2, '') + ')', '') +
				ISNULL(CHAR(13)+CHAR(10) + 'FILTER: (' + i.filter_definition + ')', '')
			) AS index_definition_details
		FROM
			sys.indexes i
		JOIN
			table_info ti ON i.object_id = ti.table_oid
		WHERE i.type <> 0 -- Exclude Heaps
		AND i.name IS NOT NULL -- Exclude unnamed heap indexes; named indexes (PKs are often named) are preferred.
	),
	triggers_info AS (
		SELECT
			tr.parent_id AS table_oid,
			tr.name AS trigger_name,
			OBJECT_DEFINITION(tr.object_id) AS trigger_definition,
			CASE tr.is_disabled WHEN 0 THEN 'ENABLED' ELSE 'DISABLED' END AS trigger_enabled_state
		FROM
			sys.triggers tr
		JOIN
			table_info ti ON tr.parent_id = ti.table_oid
		WHERE
			tr.is_ms_shipped = 0
			AND tr.parent_class_desc = 'OBJECT_OR_COLUMN' -- DML Triggers on tables/views
	)
	SELECT
		ti.schema_name,
		ti.table_name AS object_name,
		CASE
			WHEN @output_format = 'simple' THEN
				(SELECT ti.table_name AS name FOR JSON PATH, WITHOUT_ARRAY_WRAPPER)
			ELSE
				(
					SELECT
						ti.schema_name AS schema_name,
						ti.table_name AS object_name,
						ti.object_type_detail AS object_type,
						ti.table_owner AS owner,
						ti.table_comment AS comment,
						JSON_QUERY(ISNULL((
							SELECT
								ci.column_name,
								ci.data_type,
								ci.column_ordinal_position,
								ci.is_not_nullable,
								ci.column_default,
								ci.column_comment
							FROM columns_info ci
							WHERE ci.table_oid = ti.table_oid
							ORDER BY ci.column_ordinal_position
							FOR JSON PATH
						), '[]')) AS columns,
						JSON_QUERY(ISNULL((
							SELECT
								cons.constraint_name,
								cons.constraint_type,
								cons.constraint_definition,
								JSON_QUERY(
									CASE
										WHEN cons.constraint_columns IS NOT NULL AND LTRIM(RTRIM(cons.constraint_columns)) <> ''
										THEN '[' + (SELECT STRING_AGG('"' + LTRIM(RTRIM(value)) + '"', ',') FROM STRING_SPLIT(cons.constraint_columns, ',')) + ']'
										ELSE '[]'
									END
								) AS constraint_columns,
								cons.foreign_key_referenced_table,
								JSON_QUERY(
									CASE
										WHEN cons.foreign_key_referenced_columns IS NOT NULL AND LTRIM(RTRIM(cons.foreign_key_referenced_columns)) <> ''
										THEN '[' + (SELECT STRING_AGG('"' + LTRIM(RTRIM(value)) + '"', ',') FROM STRING_SPLIT(cons.foreign_key_referenced_columns, ',')) + ']'
										ELSE '[]'
									END
								) AS foreign_key_referenced_columns
							FROM constraints_info cons
							WHERE cons.table_oid = ti.table_oid
							FOR JSON PATH
						), '[]')) AS constraints,
						JSON_QUERY(ISNULL((
							SELECT
								ii.index_name,
								ii.index_definition_details AS index_definition,
								ii.is_unique,
								ii.is_primary,
								ii.index_method,
								JSON_QUERY(
									CASE
										WHEN ii.index_columns IS NOT NULL AND LTRIM(RTRIM(ii.index_columns)) <> ''
										THEN '[' + (SELECT STRING_AGG('"' + LTRIM(RTRIM(value)) + '"', ',') FROM STRING_SPLIT(ii.index_columns, ',')) + ']'
										ELSE '[]'
									END
								) AS index_columns
							FROM indexes_info ii
							WHERE ii.table_oid = ti.table_oid
							FOR JSON PATH
						), '[]')) AS indexes,
						JSON_QUERY(ISNULL((
							SELECT
								tri.trigger_name,
								tri.trigger_definition,
								tri.trigger_enabled_state
							FROM triggers_info tri
							WHERE tri.table_oid = ti.table_oid
							FOR JSON PATH
						), '[]')) AS triggers
					FOR JSON PATH, WITHOUT_ARRAY_WRAPPER -- Creates a single JSON object for this table's details
				)
		END AS object_details
	FROM
		table_info ti
	ORDER BY
		ti.schema_name, ti.table_name;
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
	MSSQLDB() *sql.DB
}

// validate compatible sources are still compatible
var _ compatibleSource = &cloudsqlmssql.Source{}
var _ compatibleSource = &mssql.Source{}

var compatibleSources = [...]string{cloudsqlmssql.SourceKind, mssql.SourceKind}

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

	allParameters := parameters.Parameters{
		parameters.NewStringParameterWithDefault("table_names", "", "Optional: A comma-separated list of table names. If empty, details for all tables will be listed."),
		parameters.NewStringParameterWithDefault("output_format", "detailed", "Optional: Use 'simple' for names only or 'detailed' for full info."),
	}
	paramManifest := allParameters.Manifest()
	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, allParameters, nil)

	// finish tool setup
	t := Tool{
		Config:      cfg,
		AllParams:   allParameters,
		Db:          s.MSSQLDB(),
		manifest:    tools.Manifest{Description: cfg.Description, Parameters: paramManifest, AuthRequired: cfg.AuthRequired},
		mcpManifest: mcpManifest,
	}
	return t, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Config
	AllParams parameters.Parameters `yaml:"allParams"`

	Db          *sql.DB
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) Invoke(ctx context.Context, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	paramsMap := params.AsMap()

	outputFormat, _ := paramsMap["output_format"].(string)
	if outputFormat != "simple" && outputFormat != "detailed" {
		return nil, fmt.Errorf("invalid value for output_format: must be 'simple' or 'detailed', but got %q", outputFormat)
	}

	namedArgs := []any{
		sql.Named("table_names", paramsMap["table_names"]),
		sql.Named("output_format", outputFormat),
	}

	rows, err := t.Db.QueryContext(ctx, listTablesStatement, namedArgs...)
	if err != nil {
		return nil, fmt.Errorf("unable to execute query: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("unable to fetch column names: %w", err)
	}

	// create an array of values for each column, which can be re-used to scan each row
	rawValues := make([]any, len(cols))
	values := make([]any, len(cols))
	for i := range rawValues {
		values[i] = &rawValues[i]
	}

	var out []any
	for rows.Next() {
		err = rows.Scan(values...)
		if err != nil {
			return nil, fmt.Errorf("unable to parse row: %w", err)
		}
		vMap := make(map[string]any)
		for i, name := range cols {
			vMap[name] = rawValues[i]
		}
		out = append(out, vMap)
	}

	// Check if error occurred during iteration
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("errors encountered during row iteration: %w", err)
	}

	return out, nil
}

func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (parameters.ParamValues, error) {
	return parameters.ParseParams(t.AllParams, data, claims)
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

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

func (t Tool) GetAuthTokenHeaderName() string {
	return "Authorization"
}
