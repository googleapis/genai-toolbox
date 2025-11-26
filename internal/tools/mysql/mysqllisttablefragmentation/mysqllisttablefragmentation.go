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

package mysqllisttablefragmentation

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
	"github.com/googleapis/genai-toolbox/internal/util"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
)

const kind string = "mysql-list-table-fragmentation"

const listTableFragmentationStatement = `
	SELECT
		table_schema,
		table_name,
		data_length AS data_size,
		index_length AS index_size,
		data_free AS data_free,
		ROUND((data_free / (data_length + index_length)) * 100, 2) AS fragmentation_percentage
	FROM
		information_schema.tables
	WHERE
		table_schema NOT IN ('sys', 'performance_schema', 'mysql', 'information_schema')
		AND (COALESCE(?, '') = '' OR table_schema = ?)
		AND (COALESCE(?, '') = '' OR table_name = ?)
		AND data_free >= ?
	ORDER BY
		fragmentation_percentage DESC,
		table_schema,
		table_name
	LIMIT ?;
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
var _ compatibleSource = &mysql.Source{}
var _ compatibleSource = &cloudsqlmysql.Source{}

var compatibleSources = [...]string{mysql.SourceKind, cloudsqlmysql.SourceKind}

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
		parameters.NewStringParameterWithDefault("table_schema", "", "(Optional) The database where fragmentation check is to be executed. Check all tables visible to the current user if not specified"),
		parameters.NewStringParameterWithDefault("table_name", "", "(Optional) Name of the table to be checked. Check all tables visible to the current user if not specified."),
		parameters.NewIntParameterWithDefault("data_free_threshold_bytes", 1, "(Optional) Only show tables with at least this much free space in bytes. Default is 1"),
		parameters.NewIntParameterWithDefault("limit", 10, "(Optional) Max rows to return, default is 10"),
	}
	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, allParameters, nil)

	// finish tool setup
	t := Tool{
		Config:      cfg,
		Pool:        s.MySQLPool(),
		allParams:   allParameters,
		manifest:    tools.Manifest{Description: cfg.Description, Parameters: allParameters.Manifest(), AuthRequired: cfg.AuthRequired},
		mcpManifest: mcpManifest,
	}
	return t, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Config
	allParams   parameters.Parameters `yaml:"parameters"`
	Pool        *sql.DB
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) Invoke(ctx context.Context, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	paramsMap := params.AsMap()

	table_schema, ok := paramsMap["table_schema"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid 'table_schema' parameter; expected a string")
	}
	table_name, ok := paramsMap["table_name"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid 'table_name' parameter; expected a string")
	}
	data_free_threshold_bytes, ok := paramsMap["data_free_threshold_bytes"].(int)
	if !ok {
		return nil, fmt.Errorf("invalid 'data_free_threshold_bytes' parameter; expected an integer")
	}
	limit, ok := paramsMap["limit"].(int)
	if !ok {
		return nil, fmt.Errorf("invalid 'limit' parameter; expected an integer")
	}

	// Log the query executed for debugging.
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting logger: %s", err)
	}
	logger.DebugContext(ctx, fmt.Sprintf("executing `%s` tool query: %s", kind, listTableFragmentationStatement))

	results, err := t.Pool.QueryContext(ctx, listTableFragmentationStatement, table_schema, table_schema, table_name, table_name, data_free_threshold_bytes, limit)
	if err != nil {
		return nil, fmt.Errorf("unable to execute query: %w", err)
	}
	defer results.Close()

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

func (t Tool) RequiresClientAuthorization() bool {
	return false
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

func (t Tool) GetAuthTokenHeaderName() string {
	return "Authorization"
}
