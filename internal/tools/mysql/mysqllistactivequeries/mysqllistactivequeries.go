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

package mysqllistactivequeries

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

const kind string = "mysql-list-active-queries"

const listActiveQueriesStatementMySQL = `
	SELECT
		p.id AS processlist_id,
		substring(IFNULL(p.info, t.trx_query), 1, 100) AS query,
		t.trx_started AS trx_started,
		(UNIX_TIMESTAMP(UTC_TIMESTAMP()) - UNIX_TIMESTAMP(t.trx_started)) AS trx_duration_seconds,
		(UNIX_TIMESTAMP(UTC_TIMESTAMP()) - UNIX_TIMESTAMP(t.trx_wait_started)) AS trx_wait_duration_seconds,
		p.time AS query_time,
		t.trx_state AS trx_state,
		p.state AS process_state,
		IF(p.host IS NULL OR p.host = '', p.user, concat(p.user, '@', SUBSTRING_INDEX(p.host, ':', 1))) AS user,
		t.trx_rows_locked AS trx_rows_locked,
		t.trx_rows_modified AS trx_rows_modified,
		p.db AS db
	FROM
		information_schema.processlist p
		LEFT OUTER JOIN
		information_schema.innodb_trx t
		ON p.id = t.trx_mysql_thread_id
	WHERE
		(? IS NULL OR p.time >= ?)
		AND p.id != CONNECTION_ID()
		AND Command NOT IN ('Binlog Dump', 'Binlog Dump GTID', 'Connect', 'Connect Out', 'Register Slave')
		AND User NOT IN ('system user', 'event_scheduler')
		AND (t.trx_id is NOT NULL OR command != 'Sleep')
	ORDER BY
		t.trx_started
	LIMIT ?;
`

const listActiveQueriesStatementCloudSQLMySQL = `
	SELECT
		p.id AS processlist_id,
		substring(IFNULL(p.info, t.trx_query), 1, 100) AS query,
		t.trx_started AS trx_started,
		(UNIX_TIMESTAMP(UTC_TIMESTAMP()) - UNIX_TIMESTAMP(t.trx_started)) AS trx_duration_seconds,
		(UNIX_TIMESTAMP(UTC_TIMESTAMP()) - UNIX_TIMESTAMP(t.trx_wait_started)) AS trx_wait_duration_seconds,
		p.time AS query_time,
		t.trx_state AS trx_state,
		p.state AS process_state,
		IF(p.host IS NULL OR p.host = '', p.user, concat(p.user, '@', SUBSTRING_INDEX(p.host, ':', 1))) AS user,
		t.trx_rows_locked AS trx_rows_locked,
		t.trx_rows_modified AS trx_rows_modified,
		p.db AS db
	FROM
		information_schema.processlist p
		LEFT OUTER JOIN
		information_schema.innodb_trx t
		ON p.id = t.trx_mysql_thread_id
	WHERE
		(? IS NULL OR p.time >= ?)
		AND p.id != CONNECTION_ID()
		AND SUBSTRING_INDEX(IFNULL(p.host,''), ':', 1) NOT IN ('localhost', '127.0.0.1')
		AND IFNULL(p.host,'') NOT LIKE '::1%'
		AND Command NOT IN ('Binlog Dump', 'Binlog Dump GTID', 'Connect', 'Connect Out', 'Register Slave')
		AND User NOT IN ('system user', 'event_scheduler')
		AND (t.trx_id is NOT NULL OR command != 'sleep')
	ORDER BY
		t.trx_started
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
		parameters.NewIntParameterWithDefault("min_duration_secs", 0, "Optional: Only show queries running for at least this long in seconds"),
		parameters.NewIntParameterWithDefault("limit", 100, "Optional: The maximum number of rows to return."),
	}
	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, allParameters, nil)

	var statement string
	sourceKind := rawS.SourceKind()

	switch sourceKind {
	case mysql.SourceKind:
		statement = listActiveQueriesStatementMySQL
	case cloudsqlmysql.SourceKind:
		statement = listActiveQueriesStatementCloudSQLMySQL
	default:
		return nil, fmt.Errorf("unsupported source kind kind: %q", sourceKind)
	}
	// finish tool setup
	t := Tool{
		Config:      cfg,
		Pool:        s.MySQLPool(),
		allParams:   allParameters,
		manifest:    tools.Manifest{Description: cfg.Description, Parameters: allParameters.Manifest(), AuthRequired: cfg.AuthRequired},
		mcpManifest: mcpManifest,
		statement:   statement,
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
	statement   string
}

func (t Tool) Invoke(ctx context.Context, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	paramsMap := params.AsMap()

	duration, ok := paramsMap["min_duration_secs"].(int)
	if !ok {
		return nil, fmt.Errorf("invalid 'min_duration_secs' parameter; expected an integer")
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
	logger.DebugContext(ctx, fmt.Sprintf("executing `%s` tool query: %s", kind, t.statement))

	results, err := t.Pool.QueryContext(ctx, t.statement, duration, duration, limit)
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
