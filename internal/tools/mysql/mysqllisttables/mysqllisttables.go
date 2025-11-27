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

package mysqllisttables

import (
	"context"
	"database/sql"
	"fmt"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/cloudsqlmysql"
	"github.com/googleapis/genai-toolbox/internal/sources/mysql"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/tools/mysql/mysqlcommon"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
)

const kind string = "mysql-list-tables"

const listTablesStatement = `
    SELECT
        T.TABLE_SCHEMA AS schema_name,
        T.TABLE_NAME AS object_name,
        CASE
            WHEN @output_format = 'simple' THEN
                JSON_OBJECT('name', T.TABLE_NAME)
            ELSE
                CONVERT(
                    JSON_OBJECT(
                        'schema_name', T.TABLE_SCHEMA,
                        'object_name', T.TABLE_NAME,
                        'object_type', 'TABLE',
                        'owner', (
                            SELECT
                                IFNULL(U.GRANTEE, 'N/A')
                            FROM
                                INFORMATION_SCHEMA.SCHEMA_PRIVILEGES U
                            WHERE
                                U.TABLE_SCHEMA = T.TABLE_SCHEMA
                            LIMIT 1
                        ),
                        'comment', IFNULL(T.TABLE_COMMENT, ''),
                        'columns', (
                            SELECT
                                IFNULL(
                                    JSON_ARRAYAGG(
                                        JSON_OBJECT(
                                            'column_name', C.COLUMN_NAME,
                                            'data_type', C.COLUMN_TYPE,
                                            'ordinal_position', C.ORDINAL_POSITION,
                                            'is_not_nullable', IF(C.IS_NULLABLE = 'NO', TRUE, FALSE),
                                            'column_default', C.COLUMN_DEFAULT,
                                            'column_comment', IFNULL(C.COLUMN_COMMENT, '')
                                        )
                                    ),
                                    JSON_ARRAY()
                                )
                            FROM
                                INFORMATION_SCHEMA.COLUMNS C
                            WHERE
                                C.TABLE_SCHEMA = T.TABLE_SCHEMA AND C.TABLE_NAME = T.TABLE_NAME
                            ORDER BY C.ORDINAL_POSITION
                        ),
                        'constraints', (
                            SELECT
                                IFNULL(
                                    JSON_ARRAYAGG(
                                        JSON_OBJECT(
                                            'constraint_name', TC.CONSTRAINT_NAME,
                                            'constraint_type',
                                            CASE TC.CONSTRAINT_TYPE
                                                WHEN 'PRIMARY KEY' THEN 'PRIMARY KEY'
                                                WHEN 'FOREIGN KEY' THEN 'FOREIGN KEY'
                                                WHEN 'UNIQUE' THEN 'UNIQUE'
                                                ELSE TC.CONSTRAINT_TYPE
                                            END,
                                            'constraint_definition', '',
                                            'constraint_columns', (
                                                SELECT
                                                    IFNULL(JSON_ARRAYAGG(KCU.COLUMN_NAME), JSON_ARRAY())
                                                FROM
                                                    INFORMATION_SCHEMA.KEY_COLUMN_USAGE KCU
                                                WHERE
                                                    KCU.CONSTRAINT_SCHEMA = TC.CONSTRAINT_SCHEMA
                                                    AND KCU.CONSTRAINT_NAME = TC.CONSTRAINT_NAME
                                                    AND KCU.TABLE_NAME = TC.TABLE_NAME
                                                ORDER BY KCU.ORDINAL_POSITION
                                            ),
                                            'foreign_key_referenced_table', IF(TC.CONSTRAINT_TYPE = 'FOREIGN KEY', RC.REFERENCED_TABLE_NAME, NULL),
                                            'foreign_key_referenced_columns', IF(TC.CONSTRAINT_TYPE = 'FOREIGN KEY',
                                                (SELECT IFNULL(JSON_ARRAYAGG(FKCU.REFERENCED_COLUMN_NAME), JSON_ARRAY())
                                                FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE FKCU
                                                WHERE FKCU.CONSTRAINT_SCHEMA = TC.CONSTRAINT_SCHEMA
                                                    AND FKCU.CONSTRAINT_NAME = TC.CONSTRAINT_NAME
                                                    AND FKCU.TABLE_NAME = TC.TABLE_NAME
                                                    AND FKCU.REFERENCED_TABLE_NAME IS NOT NULL
                                                ORDER BY FKCU.ORDINAL_POSITION),
                                                NULL
                                            )
                                        )
                                    ),
                                    JSON_ARRAY()
                                )
                            FROM
                                INFORMATION_SCHEMA.TABLE_CONSTRAINTS TC
                            LEFT JOIN
                                INFORMATION_SCHEMA.REFERENTIAL_CONSTRAINTS RC
                                ON TC.CONSTRAINT_SCHEMA = RC.CONSTRAINT_SCHEMA
                                AND TC.CONSTRAINT_NAME = RC.CONSTRAINT_NAME
                                AND TC.TABLE_NAME = RC.TABLE_NAME
                            WHERE
                                TC.TABLE_SCHEMA = T.TABLE_SCHEMA AND TC.TABLE_NAME = T.TABLE_NAME
                        ),
                        'indexes', (
                            SELECT
                                IFNULL(
                                    JSON_ARRAYAGG(
                                        JSON_OBJECT(
                                            'index_name', IndexData.INDEX_NAME,
                                            'is_unique', IF(IndexData.NON_UNIQUE = 0, TRUE, FALSE),
                                            'is_primary', IF(IndexData.INDEX_NAME = 'PRIMARY', TRUE, FALSE),
                                            'index_columns', IFNULL(IndexData.INDEX_COLUMNS_ARRAY, JSON_ARRAY())
                                        )
                                    ),
                                    JSON_ARRAY()
                                )
                            FROM (
                                SELECT
                                    S.TABLE_SCHEMA,
                                    S.TABLE_NAME,
                                    S.INDEX_NAME,
                                    MIN(S.NON_UNIQUE) AS NON_UNIQUE,
                                    JSON_ARRAYAGG(S.COLUMN_NAME) AS INDEX_COLUMNS_ARRAY
                                FROM
                                    INFORMATION_SCHEMA.STATISTICS S
                                GROUP BY
                                    S.TABLE_SCHEMA, S.TABLE_NAME, S.INDEX_NAME
                            ) AS IndexData
                            WHERE IndexData.TABLE_SCHEMA = T.TABLE_SCHEMA AND IndexData.TABLE_NAME = T.TABLE_NAME
                            ORDER BY IndexData.INDEX_NAME
                        ),
                        'triggers', (
                            SELECT
                                IFNULL(
                                    JSON_ARRAYAGG(
                                        JSON_OBJECT(
                                            'trigger_name', TR.TRIGGER_NAME,
                                            'trigger_definition', TR.ACTION_STATEMENT
                                        )
                                    ),
                                    JSON_ARRAY()
                                )
                            FROM
                                INFORMATION_SCHEMA.TRIGGERS TR
                            WHERE
                                TR.EVENT_OBJECT_SCHEMA = T.TABLE_SCHEMA AND TR.EVENT_OBJECT_TABLE = T.TABLE_NAME
                            ORDER BY TR.TRIGGER_NAME
                        )
                    )
                USING utf8mb4)
        END AS object_details
    FROM
        INFORMATION_SCHEMA.TABLES T
    CROSS JOIN (SELECT @table_names := ?, @output_format := ?) AS variables
    WHERE
        T.TABLE_SCHEMA NOT IN ('mysql', 'information_schema', 'performance_schema', 'sys')
        AND (NULLIF(TRIM(@table_names), '') IS NULL OR FIND_IN_SET(T.TABLE_NAME, @table_names))
        AND T.TABLE_TYPE = 'BASE TABLE'
    ORDER BY
        T.TABLE_SCHEMA, T.TABLE_NAME;
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
	MySQLPool() *sql.DB
}

// validate compatible sources are still compatible
var _ compatibleSource = &cloudsqlmysql.Source{}
var _ compatibleSource = &mysql.Source{}

var compatibleSources = [...]string{cloudsqlmysql.SourceKind, mysql.SourceKind}

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
		Pool:        s.MySQLPool(),
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

	Pool        *sql.DB
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) Invoke(ctx context.Context, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	paramsMap := params.AsMap()

	tableNames, ok := paramsMap["table_names"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid '%s' parameter; expected a string", tableNames)
	}
	outputFormat, _ := paramsMap["output_format"].(string)
	if outputFormat != "simple" && outputFormat != "detailed" {
		return nil, fmt.Errorf("invalid value for output_format: must be 'simple' or 'detailed', but got %q", outputFormat)
	}

	results, err := t.Pool.QueryContext(ctx, listTablesStatement, tableNames, outputFormat)
	if err != nil {
		return nil, fmt.Errorf("unable to execute query: %w", err)
	}

	cols, err := results.Columns()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve rows column name: %w", err)
	}

	// create an array of values for each column, which can be re-used to scan each row
	rawValues := make([]any, len(cols))
	values := make([]any, len(cols))
	for i := range rawValues {
		values[i] = &rawValues[i]
	}
	defer results.Close()

	colTypes, err := results.ColumnTypes()
	if err != nil {
		return nil, fmt.Errorf("unable to get column types: %w", err)
	}

	var out []any
	for results.Next() {
		err := results.Scan(values...)
		if err != nil {
			return nil, fmt.Errorf("unable to parse row: %w", err)
		}
		vMap := make(map[string]any)
		for i, name := range cols {
			val := rawValues[i]
			if val == nil {
				vMap[name] = nil
				continue
			}

			vMap[name], err = mysqlcommon.ConvertToType(colTypes[i], val)
			if err != nil {
				return nil, fmt.Errorf("errors encountered when converting values: %w", err)
			}
		}
		out = append(out, vMap)
	}

	if err := results.Err(); err != nil {
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
