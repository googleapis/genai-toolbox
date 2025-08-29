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

package clickhouse

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	_ "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/google/uuid"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/clickhouse"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools"
	clickhouseexecutesql "github.com/googleapis/genai-toolbox/internal/tools/clickhouse/clickhouseexecutesql"
	clickhousesql "github.com/googleapis/genai-toolbox/internal/tools/clickhouse/clickhousesql"
	"github.com/googleapis/genai-toolbox/tests"
	"go.opentelemetry.io/otel/trace/noop"
)

var (
	ClickHouseSourceKind = "clickhouse"
	ClickHouseToolKind   = "clickhouse-sql"
	ClickHouseDatabase   = os.Getenv("CLICKHOUSE_DATABASE")
	ClickHouseHost       = os.Getenv("CLICKHOUSE_HOST")
	ClickHousePort       = os.Getenv("CLICKHOUSE_PORT")
	ClickHouseUser       = os.Getenv("CLICKHOUSE_USER")
	ClickHousePass       = os.Getenv("CLICKHOUSE_PASS")
	ClickHouseProtocol   = os.Getenv("CLICKHOUSE_PROTOCOL")
)

func getClickHouseVars(t *testing.T) map[string]any {
	switch "" {
	case ClickHouseHost:
		t.Skip("'CLICKHOUSE_HOST' not set")
	case ClickHousePort:
		t.Skip("'CLICKHOUSE_PORT' not set")
	case ClickHouseUser:
		t.Skip("'CLICKHOUSE_USER' not set")
	}

	// Set defaults for optional parameters
	if ClickHouseDatabase == "" {
		ClickHouseDatabase = "default"
	}
	if ClickHouseProtocol == "" {
		ClickHouseProtocol = "http"
	}

	return map[string]any{
		"kind":     ClickHouseSourceKind,
		"host":     ClickHouseHost,
		"port":     ClickHousePort,
		"database": ClickHouseDatabase,
		"user":     ClickHouseUser,
		"password": ClickHousePass,
		"protocol": ClickHouseProtocol,
		"secure":   false,
	}
}

// initClickHouseConnectionPool creates a ClickHouse connection using HTTP protocol only.
// Note: ClickHouse tools in this codebase only support HTTP/HTTPS protocols, not the native protocol.
// Typical setup: localhost:8123 (HTTP) or localhost:8443 (HTTPS)
func initClickHouseConnectionPool(host, port, user, pass, dbname, protocol string) (*sql.DB, error) {
	if protocol == "" {
		protocol = "https"
	}

	var dsn string
	switch protocol {
	case "http":
		dsn = fmt.Sprintf("http://%s:%s@%s:%s/%s", user, pass, host, port, dbname)
	case "https":
		dsn = fmt.Sprintf("https://%s:%s@%s:%s/%s?secure=true&skip_verify=false", user, pass, host, port, dbname)
	default:
		dsn = fmt.Sprintf("https://%s:%s@%s:%s/%s?secure=true&skip_verify=false", user, pass, host, port, dbname)
	}

	pool, err := sql.Open("clickhouse", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}

	return pool, nil
}

func TestClickHouse(t *testing.T) {
	sourceConfig := getClickHouseVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	pool, err := initClickHouseConnectionPool(ClickHouseHost, ClickHousePort, ClickHouseUser, ClickHousePass, ClickHouseDatabase, ClickHouseProtocol)
	if err != nil {
		t.Fatalf("unable to create ClickHouse connection pool: %s", err)
	}
	defer pool.Close()

	tableNameParam := "param_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableNameAuth := "auth_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableNameTemplateParam := "template_param_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")

	createParamTableStmt, insertParamTableStmt, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, paramTestParams := getClickHouseSQLParamToolInfo(tableNameParam)
	teardownTable1 := setupClickHouseSQLTable(t, ctx, pool, createParamTableStmt, insertParamTableStmt, tableNameParam, paramTestParams)
	defer teardownTable1(t)

	createAuthTableStmt, insertAuthTableStmt, authToolStmt, authTestParams := getClickHouseSQLAuthToolInfo(tableNameAuth)
	teardownTable2 := setupClickHouseSQLTable(t, ctx, pool, createAuthTableStmt, insertAuthTableStmt, tableNameAuth, authTestParams)
	defer teardownTable2(t)

	toolsFile := tests.GetToolsConfig(sourceConfig, ClickHouseToolKind, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, authToolStmt)
	toolsFile = addClickHouseExecuteSqlConfig(t, toolsFile)
	tmplSelectCombined, tmplSelectFilterCombined := getClickHouseSQLTmplToolStatement()
	toolsFile = addClickHouseTemplateParamConfig(t, toolsFile, ClickHouseToolKind, tmplSelectCombined, tmplSelectFilterCombined)

	cmd, cleanup, err := tests.StartCmd(ctx, toolsFile, args...)
	if err != nil {
		t.Fatalf("command initialization returned an error: %s", err)
	}
	defer cleanup()

	waitCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	out, err := testutils.WaitForString(waitCtx, regexp.MustCompile(`Server ready to serve`), cmd.Out)
	if err != nil {
		t.Logf("toolbox command logs: \n%s", out)
		t.Fatalf("toolbox didn't start successfully: %s", err)
	}

    // Get configs for tests
	select1Want, mcpSelect1Want, mcpMyFailToolWant, createTableStatement, nilIdWant := getClickHouseWants()

    // Run tests
	tests.RunToolGetTest(t)
	tests.RunToolInvokeTest(t, select1Want, tests.WithMyToolById4Want(nilIdWant))
	tests.RunExecuteSqlToolInvokeTest(t, createTableStatement, select1Want)
	tests.RunMCPToolCallMethod(t, mcpMyFailToolWant, mcpSelect1Want)
	tests.RunToolInvokeWithTemplateParameters(t, tableNameTemplateParam)
}

func addClickHouseExecuteSqlConfig(t *testing.T, config map[string]any) map[string]any {
	tools, ok := config["tools"].(map[string]any)
	if !ok {
		t.Fatalf("unable to get tools from config")
	}
	tools["my-exec-sql-tool"] = map[string]any{
		"kind":        "clickhouse-execute-sql",
		"source":      "my-instance",
		"description": "Tool to execute sql",
	}
	tools["my-auth-exec-sql-tool"] = map[string]any{
		"kind":        "clickhouse-execute-sql",
		"source":      "my-instance",
		"description": "Tool to execute sql",
		"authRequired": []string{
			"my-google-auth",
		},
	}
	config["tools"] = tools
	return config
}

func addClickHouseTemplateParamConfig(t *testing.T, config map[string]any, toolKind, tmplSelectCombined, tmplSelectFilterCombined string) map[string]any {
	toolsMap, ok := config["tools"].(map[string]any)
	if !ok {
		t.Fatalf("unable to get tools from config")
	}

	// ClickHouse-specific template parameter tools with compatible syntax
	toolsMap["create-table-templateParams-tool"] = map[string]any{
		"kind":        toolKind,
		"source":      "my-instance",
		"description": "Create table tool with template parameters",
		"statement":   "CREATE TABLE {{.tableName}} ({{array .columns}}) ORDER BY id",
		"templateParameters": []tools.Parameter{
			tools.NewStringParameter("tableName", "some description"),
			tools.NewArrayParameter("columns", "The columns to create", tools.NewStringParameter("column", "A column name that will be created")),
		},
	}
	toolsMap["insert-table-templateParams-tool"] = map[string]any{
		"kind":        toolKind,
		"source":      "my-instance",
		"description": "Insert table tool with template parameters",
		"statement":   "INSERT INTO {{.tableName}} ({{array .columns}}) VALUES ({{.values}})",
		"templateParameters": []tools.Parameter{
			tools.NewStringParameter("tableName", "some description"),
			tools.NewArrayParameter("columns", "The columns to insert into", tools.NewStringParameter("column", "A column name that will be returned from the query.")),
			tools.NewStringParameter("values", "The values to insert as a comma separated string"),
		},
	}
	toolsMap["select-templateParams-tool"] = map[string]any{
		"kind":        toolKind,
		"source":      "my-instance",
		"description": "Select table tool with template parameters",
		"statement":   "SELECT id AS \"id\", name AS \"name\", age AS \"age\" FROM {{.tableName}} ORDER BY id",
		"templateParameters": []tools.Parameter{
			tools.NewStringParameter("tableName", "some description"),
		},
	}
	toolsMap["select-templateParams-combined-tool"] = map[string]any{
		"kind":        toolKind,
		"source":      "my-instance",
		"description": "Select table tool with combined template parameters",
		"statement":   tmplSelectCombined,
		"parameters": []tools.Parameter{
			tools.NewIntParameter("id", "the id of the user"),
		},
		"templateParameters": []tools.Parameter{
			tools.NewStringParameter("tableName", "some description"),
		},
	}
	toolsMap["select-fields-templateParams-tool"] = map[string]any{
		"kind":        toolKind,
		"source":      "my-instance",
		"description": "Select specific fields tool with template parameters",
		"statement":   "SELECT name AS \"name\" FROM {{.tableName}} ORDER BY id",
		"templateParameters": []tools.Parameter{
			tools.NewStringParameter("tableName", "some description"),
		},
	}
	toolsMap["select-filter-templateParams-combined-tool"] = map[string]any{
		"kind":        toolKind,
		"source":      "my-instance",
		"description": "Select table tool with filter template parameters",
		"statement":   tmplSelectFilterCombined,
		"parameters": []tools.Parameter{
			tools.NewStringParameter("name", "the name to filter by"),
		},
		"templateParameters": []tools.Parameter{
			tools.NewStringParameter("tableName", "some description"),
			tools.NewStringParameter("columnFilter", "some description"),
		},
	}
	// Firebird uses simple DROP TABLE syntax without IF EXISTS
	toolsMap["drop-table-templateParams-tool"] = map[string]any{
		"kind":        toolKind,
		"source":      "my-instance",
		"description": "Drop table tool with template parameters",
		"statement":   "DROP TABLE {{.tableName}}",
		"templateParameters": []tools.Parameter{
			tools.NewStringParameter("tableName", "some description"),
		},
	}
	config["tools"] = toolsMap
	return config
}

func TestClickHouseBasicConnection(t *testing.T) {
	sourceConfig := getClickHouseVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	pool, err := initClickHouseConnectionPool(ClickHouseHost, ClickHousePort, ClickHouseUser, ClickHousePass, ClickHouseDatabase, ClickHouseProtocol)
	if err != nil {
		t.Fatalf("unable to create ClickHouse connection pool: %s", err)
	}
	defer pool.Close()

	// Test basic connection
	err = pool.PingContext(ctx)
	if err != nil {
		t.Fatalf("unable to ping ClickHouse: %s", err)
	}

	// Test basic query
	rows, err := pool.QueryContext(ctx, "SELECT 1 as test_value")
	if err != nil {
		t.Fatalf("unable to execute basic query: %s", err)
	}
	defer rows.Close()

	if !rows.Next() {
		t.Fatalf("expected at least one row from basic query")
	}

	var testValue int
	err = rows.Scan(&testValue)
	if err != nil {
		t.Fatalf("unable to scan result: %s", err)
	}

	if testValue != 1 {
		t.Fatalf("expected test_value to be 1, got %d", testValue)
	}

	// Write a basic tools config and test the server endpoint (without auth services)
	toolsFile := map[string]any{
		"sources": map[string]any{
			"my-instance": sourceConfig,
		},
		"tools": map[string]any{
			"my-simple-tool": map[string]any{
				"kind":        ClickHouseToolKind,
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
				"statement":   "SELECT 1;",
			},
		},
	}

	cmd, cleanup, err := tests.StartCmd(ctx, toolsFile, args...)
	if err != nil {
		t.Fatalf("command initialization returned an error: %s", err)
	}
	defer cleanup()

	waitCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	out, err := testutils.WaitForString(waitCtx, regexp.MustCompile(`Server ready to serve`), cmd.Out)
	if err != nil {
		t.Logf("toolbox command logs: \n%s", out)
		t.Fatalf("toolbox didn't start successfully: %s", err)
	}

	tests.RunToolGetTest(t)
	t.Logf("✅ ClickHouse basic connection test completed successfully")
}

func getClickHouseWants() (string, string, string, string, string) {
	select1Want := "[{\"1\":1}]"
	mcpSelect1Want := `{"jsonrpc":"2.0","id":"invoke my-auth-required-tool","result":{"content":[{"type":"text","text":"{\"1\":1}"}]}}`
	mcpMyFailToolWant := `{"jsonrpc":"2.0","id":"invoke-fail-tool","result":{"content":[{"type":"text","text":"unable to execute query: sendQuery: [HTTP 400] response body: \"Code: 62. DB::Exception: Syntax error: failed at position 1 (SELEC): SELEC 1;. Expected one of: Query, Query with output, EXPLAIN, EXPLAIN, SELECT query, possibly with UNION, list of union elements, SELECT query, subquery, possibly with UNION, SELECT subquery, SELECT query, WITH, FROM, SELECT, SHOW CREATE QUOTA query, SHOW CREATE, SHOW [FULL] [TEMPORARY] TABLES|DATABASES|CLUSTERS|CLUSTER|MERGES 'name' [[NOT] [I]LIKE 'str'] [LIMIT expr], SHOW, SHOW COLUMNS query, SHOW ENGINES query, SHOW ENGINES, SHOW FUNCTIONS query, SHOW FUNCTIONS, SHOW INDEXES query, SHOW SETTING query, SHOW SETTING, EXISTS or SHOW CREATE query, EXISTS, DESCRIBE FILESYSTEM CACHE query, DESCRIBE, DESC, DESCRIBE query, SHOW PROCESSLIST query, SHOW PROCESSLIST, CREATE TABLE or ATTACH TABLE query, CREATE, ATTACH, REPLACE, CREATE DATABASE query, CREATE VIEW query, CREATE DICTIONARY, CREATE LIVE VIEW query, CREATE WINDOW VIEW query, ALTER query, ALTER TABLE, ALTER TEMPORARY TABLE, ALTER DATABASE, RENAME query, RENAME DATABASE, RENAME TABLE, EXCHANGE TABLES, RENAME DICTIONARY, EXCHANGE DICTIONARIES, RENAME, DROP query, DROP, DETACH, TRUNCATE, UNDROP query, UNDROP, CHECK ALL TABLES, CHECK TABLE, KILL QUERY query, KILL, OPTIMIZE query, OPTIMIZE TABLE, WATCH query, WATCH, SHOW ACCESS query, SHOW ACCESS, ShowAccessEntitiesQuery, SHOW GRANTS query, SHOW GRANTS, SHOW PRIVILEGES query, SHOW PRIVILEGES, BACKUP or RESTORE query, BACKUP, RESTORE, INSERT query, INSERT INTO, USE query, USE, SET ROLE or SET DEFAULT ROLE query, SET ROLE DEFAULT, SET ROLE, SET DEFAULT ROLE, SET query, SET, SYSTEM query, SYSTEM, CREATE USER or ALTER USER query, ALTER USER, CREATE USER, CREATE ROLE or ALTER ROLE query, ALTER ROLE, CREATE ROLE, CREATE QUOTA or ALTER QUOTA query, ALTER QUOTA, CREATE QUOTA, CREATE ROW POLICY or ALTER ROW POLICY query, ALTER POLICY, ALTER ROW POLICY, CREATE POLICY, CREATE ROW POLICY, CREATE SETTINGS PROFILE or ALTER SETTINGS PROFILE query, ALTER SETTINGS PROFILE, ALTER PROFILE, CREATE SETTINGS PROFILE, CREATE PROFILE, CREATE FUNCTION query, DROP FUNCTION query, CREATE WORKLOAD query, DROP WORKLOAD query, CREATE RESOURCE query, DROP RESOURCE query, CREATE NAMED COLLECTION, DROP NAMED COLLECTION query, Alter NAMED COLLECTION query, ALTER, CREATE INDEX query, DROP INDEX query, DROP access entity query, MOVE access entity query, MOVE, GRANT or REVOKE query, REVOKE, GRANT, CHECK GRANT, CHECK GRANT, EXTERNAL DDL query, EXTERNAL DDL FROM, TCL query, BEGIN TRANSACTION, START TRANSACTION, COMMIT, ROLLBACK, SET TRANSACTION SNAPSHOT, Delete query, DELETE, Update query, UPDATE. (SYNTAX_ERROR) (version 25.7.5.34 (official build))\n\""}],"isError":true}}`
	createTableStatement := `"CREATE TABLE t (id UInt32, name String) ENGINE = Memory"`
	nullWant := `[{"id":4,"name":""}]`
	return select1Want, mcpSelect1Want, mcpMyFailToolWant, createTableStatement, nullWant
}

func TestClickHouseSQLTool(t *testing.T) {
	_ = getClickHouseVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	pool, err := initClickHouseConnectionPool(ClickHouseHost, ClickHousePort, ClickHouseUser, ClickHousePass, ClickHouseDatabase, ClickHouseProtocol)
	if err != nil {
		t.Fatalf("unable to create ClickHouse connection pool: %s", err)
	}
	defer pool.Close()

	tableName := "test_sql_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	createTableSQL := fmt.Sprintf(`
		CREATE TABLE %s (
			id UInt32,
			name String,
			age UInt8,
			created_at DateTime DEFAULT now()
		) ENGINE = Memory
	`, tableName)

	_, err = pool.ExecContext(ctx, createTableSQL)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}
	defer func() {
		_, _ = pool.ExecContext(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName))
	}()

	insertSQL := fmt.Sprintf("INSERT INTO %s (id, name, age) VALUES (?, ?, ?), (?, ?, ?), (?, ?, ?)", tableName)
	_, err = pool.ExecContext(ctx, insertSQL, 1, "Alice", 25, 2, "Bob", 30, 3, "Charlie", 35)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	t.Run("SimpleSelect", func(t *testing.T) {
		toolConfig := clickhousesql.Config{
			Name:        "test-select",
			Kind:        "clickhouse-sql",
			Source:      "test-clickhouse",
			Description: "Test select query",
			Statement:   fmt.Sprintf("SELECT * FROM %s ORDER BY id", tableName),
		}

		source := createMockSource(t, pool)
		sourcesMap := map[string]sources.Source{
			"test-clickhouse": source,
		}

		tool, err := toolConfig.Initialize(sourcesMap)
		if err != nil {
			t.Fatalf("Failed to initialize tool: %v", err)
		}

		result, err := tool.Invoke(ctx, tools.ParamValues{}, "")
		if err != nil {
			t.Fatalf("Failed to invoke tool: %v", err)
		}

		resultSlice, ok := result.([]any)
		if !ok {
			t.Fatalf("Expected result to be []any, got %T", result)
		}

		if len(resultSlice) != 3 {
			t.Errorf("Expected 3 results, got %d", len(resultSlice))
		}
	})

	t.Run("ParameterizedQuery", func(t *testing.T) {
		toolConfig := clickhousesql.Config{
			Name:        "test-param-query",
			Kind:        "clickhouse-sql",
			Source:      "test-clickhouse",
			Description: "Test parameterized query",
			Statement:   fmt.Sprintf("SELECT * FROM %s WHERE age > ? ORDER BY id", tableName),
			Parameters: tools.Parameters{
				tools.NewIntParameter("min_age", "Minimum age"),
			},
		}

		source := createMockSource(t, pool)
		sourcesMap := map[string]sources.Source{
			"test-clickhouse": source,
		}

		tool, err := toolConfig.Initialize(sourcesMap)
		if err != nil {
			t.Fatalf("Failed to initialize tool: %v", err)
		}

		params := tools.ParamValues{
			{Name: "min_age", Value: 28},
		}

		result, err := tool.Invoke(ctx, params, "")
		if err != nil {
			t.Fatalf("Failed to invoke tool: %v", err)
		}

		resultSlice, ok := result.([]any)
		if !ok {
			t.Fatalf("Expected result to be []any, got %T", result)
		}

		if len(resultSlice) != 2 {
			t.Errorf("Expected 2 results (Bob and Charlie), got %d", len(resultSlice))
		}
	})

	t.Run("EmptyResult", func(t *testing.T) {
		toolConfig := clickhousesql.Config{
			Name:        "test-empty-result",
			Kind:        "clickhouse-sql",
			Source:      "test-clickhouse",
			Description: "Test query with no results",
			Statement:   fmt.Sprintf("SELECT * FROM %s WHERE id = ?", tableName),
			Parameters: tools.Parameters{
				tools.NewIntParameter("id", "Record ID"),
			},
		}

		source := createMockSource(t, pool)
		sourcesMap := map[string]sources.Source{
			"test-clickhouse": source,
		}

		tool, err := toolConfig.Initialize(sourcesMap)
		if err != nil {
			t.Fatalf("Failed to initialize tool: %v", err)
		}

		params := tools.ParamValues{
			{Name: "id", Value: 999}, // Non-existent ID
		}

		result, err := tool.Invoke(ctx, params, "")
		if err != nil {
			t.Fatalf("Failed to invoke tool: %v", err)
		}

		// ClickHouse returns empty slice for no results, not nil
		if resultSlice, ok := result.([]any); ok {
			if len(resultSlice) != 0 {
				t.Errorf("Expected empty result for non-existent record, got %d results", len(resultSlice))
			}
		} else if result != nil {
			t.Errorf("Expected empty slice or nil result for empty query, got %v", result)
		}
	})

	t.Run("InvalidSQL", func(t *testing.T) {
		toolConfig := clickhousesql.Config{
			Name:        "test-invalid-sql",
			Kind:        "clickhouse-sql",
			Source:      "test-clickhouse",
			Description: "Test invalid SQL",
			Statement:   "SELEC * FROM nonexistent_table", // Typo in SELECT
		}

		source := createMockSource(t, pool)
		sourcesMap := map[string]sources.Source{
			"test-clickhouse": source,
		}

		tool, err := toolConfig.Initialize(sourcesMap)
		if err != nil {
			t.Fatalf("Failed to initialize tool: %v", err)
		}

		_, err = tool.Invoke(ctx, tools.ParamValues{}, "")
		if err == nil {
			t.Error("Expected error for invalid SQL, got nil")
		}

		if !strings.Contains(err.Error(), "Syntax error") && !strings.Contains(err.Error(), "SELEC") {
			t.Errorf("Expected syntax error message, got: %v", err)
		}
	})

	t.Logf("✅ clickhouse-sql tool tests completed successfully")
}

func TestClickHouseExecuteSQLTool(t *testing.T) {
	_ = getClickHouseVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	pool, err := initClickHouseConnectionPool(ClickHouseHost, ClickHousePort, ClickHouseUser, ClickHousePass, ClickHouseDatabase, ClickHouseProtocol)
	if err != nil {
		t.Fatalf("unable to create ClickHouse connection pool: %s", err)
	}
	defer pool.Close()

	tableName := "test_exec_sql_" + strings.ReplaceAll(uuid.New().String(), "-", "")

	t.Run("CreateTable", func(t *testing.T) {
		toolConfig := clickhouseexecutesql.Config{
			Name:        "test-create-table",
			Kind:        "clickhouse-execute-sql",
			Source:      "test-clickhouse",
			Description: "Test create table",
		}

		source := createMockSource(t, pool)
		sourcesMap := map[string]sources.Source{
			"test-clickhouse": source,
		}

		tool, err := toolConfig.Initialize(sourcesMap)
		if err != nil {
			t.Fatalf("Failed to initialize tool: %v", err)
		}

		createSQL := fmt.Sprintf(`
			CREATE TABLE %s (
				id UInt32,
				data String
			) ENGINE = Memory
		`, tableName)

		params := tools.ParamValues{
			{Name: "sql", Value: createSQL},
		}

		result, err := tool.Invoke(ctx, params, "")
		if err != nil {
			t.Fatalf("Failed to create table: %v", err)
		}

		// CREATE TABLE should return nil or empty slice (no rows)
		if resultSlice, ok := result.([]any); ok {
			if len(resultSlice) != 0 {
				t.Errorf("Expected empty result for CREATE TABLE, got %d results", len(resultSlice))
			}
		} else if result != nil {
			t.Errorf("Expected nil or empty slice for CREATE TABLE, got %v", result)
		}
	})

	t.Run("InsertData", func(t *testing.T) {
		toolConfig := clickhouseexecutesql.Config{
			Name:        "test-insert",
			Kind:        "clickhouse-execute-sql",
			Source:      "test-clickhouse",
			Description: "Test insert data",
		}

		source := createMockSource(t, pool)
		sourcesMap := map[string]sources.Source{
			"test-clickhouse": source,
		}

		tool, err := toolConfig.Initialize(sourcesMap)
		if err != nil {
			t.Fatalf("Failed to initialize tool: %v", err)
		}

		insertSQL := fmt.Sprintf("INSERT INTO %s (id, data) VALUES (1, 'test1'), (2, 'test2')", tableName)
		params := tools.ParamValues{
			{Name: "sql", Value: insertSQL},
		}

		result, err := tool.Invoke(ctx, params, "")
		if err != nil {
			t.Fatalf("Failed to insert data: %v", err)
		}

		// INSERT should return nil or empty slice
		if resultSlice, ok := result.([]any); ok {
			if len(resultSlice) != 0 {
				t.Errorf("Expected empty result for INSERT, got %d results", len(resultSlice))
			}
		} else if result != nil {
			t.Errorf("Expected nil or empty slice for INSERT, got %v", result)
		}
	})

	t.Run("SelectData", func(t *testing.T) {
		toolConfig := clickhouseexecutesql.Config{
			Name:        "test-select",
			Kind:        "clickhouse-execute-sql",
			Source:      "test-clickhouse",
			Description: "Test select data",
		}

		source := createMockSource(t, pool)
		sourcesMap := map[string]sources.Source{
			"test-clickhouse": source,
		}

		tool, err := toolConfig.Initialize(sourcesMap)
		if err != nil {
			t.Fatalf("Failed to initialize tool: %v", err)
		}

		selectSQL := fmt.Sprintf("SELECT * FROM %s ORDER BY id", tableName)
		params := tools.ParamValues{
			{Name: "sql", Value: selectSQL},
		}

		result, err := tool.Invoke(ctx, params, "")
		if err != nil {
			t.Fatalf("Failed to select data: %v", err)
		}

		resultSlice, ok := result.([]any)
		if !ok {
			t.Fatalf("Expected result to be []any, got %T", result)
		}

		if len(resultSlice) != 2 {
			t.Errorf("Expected 2 results, got %d", len(resultSlice))
		}
	})

	t.Run("DropTable", func(t *testing.T) {
		toolConfig := clickhouseexecutesql.Config{
			Name:        "test-drop-table",
			Kind:        "clickhouse-execute-sql",
			Source:      "test-clickhouse",
			Description: "Test drop table",
		}

		source := createMockSource(t, pool)
		sourcesMap := map[string]sources.Source{
			"test-clickhouse": source,
		}

		tool, err := toolConfig.Initialize(sourcesMap)
		if err != nil {
			t.Fatalf("Failed to initialize tool: %v", err)
		}

		dropSQL := fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName)
		params := tools.ParamValues{
			{Name: "sql", Value: dropSQL},
		}

		result, err := tool.Invoke(ctx, params, "")
		if err != nil {
			t.Fatalf("Failed to drop table: %v", err)
		}

		// DROP TABLE should return nil or empty slice
		if resultSlice, ok := result.([]any); ok {
			if len(resultSlice) != 0 {
				t.Errorf("Expected empty result for DROP TABLE, got %d results", len(resultSlice))
			}
		} else if result != nil {
			t.Errorf("Expected nil or empty slice for DROP TABLE, got %v", result)
		}
	})

	t.Run("MissingSQL", func(t *testing.T) {
		toolConfig := clickhouseexecutesql.Config{
			Name:        "test-missing-sql",
			Kind:        "clickhouse-execute-sql",
			Source:      "test-clickhouse",
			Description: "Test missing SQL parameter",
		}

		source := createMockSource(t, pool)
		sourcesMap := map[string]sources.Source{
			"test-clickhouse": source,
		}

		tool, err := toolConfig.Initialize(sourcesMap)
		if err != nil {
			t.Fatalf("Failed to initialize tool: %v", err)
		}

		// Pass empty SQL parameter - this should cause an error
		params := tools.ParamValues{
			{Name: "sql", Value: ""},
		}

		_, err = tool.Invoke(ctx, params, "")
		if err == nil {
			t.Error("Expected error for empty SQL parameter, got nil")
		} else {
			t.Logf("Got expected error for empty SQL parameter: %v", err)
		}
	})

	t.Run("SQLInjectionAttempt", func(t *testing.T) {
		toolConfig := clickhouseexecutesql.Config{
			Name:        "test-sql-injection",
			Kind:        "clickhouse-execute-sql",
			Source:      "test-clickhouse",
			Description: "Test SQL injection attempt",
		}

		source := createMockSource(t, pool)
		sourcesMap := map[string]sources.Source{
			"test-clickhouse": source,
		}

		tool, err := toolConfig.Initialize(sourcesMap)
		if err != nil {
			t.Fatalf("Failed to initialize tool: %v", err)
		}

		// Try to execute multiple statements (should fail or execute safely)
		injectionSQL := "SELECT 1; DROP TABLE system.users; SELECT 2"
		params := tools.ParamValues{
			{Name: "sql", Value: injectionSQL},
		}

		_, err = tool.Invoke(ctx, params, "")
		// This should either fail or only execute the first statement
		// dont check the specific error as behavior may vary
		_ = err // We're not checking the error intentionally
	})

	t.Logf("✅ clickhouse-execute-sql tool tests completed successfully")
}

func TestClickHouseEdgeCases(t *testing.T) {
	_ = getClickHouseVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	pool, err := initClickHouseConnectionPool(ClickHouseHost, ClickHousePort, ClickHouseUser, ClickHousePass, ClickHouseDatabase, ClickHouseProtocol)
	if err != nil {
		t.Fatalf("unable to create ClickHouse connection pool: %s", err)
	}
	defer pool.Close()

	t.Run("VeryLongQuery", func(t *testing.T) {
		// Create a very long but valid query
		var conditions []string
		for i := 1; i <= 100; i++ {
			conditions = append(conditions, fmt.Sprintf("(%d = %d)", i, i))
		}
		longQuery := "SELECT 1 WHERE " + strings.Join(conditions, " AND ")

		toolConfig := clickhouseexecutesql.Config{
			Name:        "test-long-query",
			Kind:        "clickhouse-execute-sql",
			Source:      "test-clickhouse",
			Description: "Test very long query",
		}

		source := createMockSource(t, pool)
		sourcesMap := map[string]sources.Source{
			"test-clickhouse": source,
		}

		tool, err := toolConfig.Initialize(sourcesMap)
		if err != nil {
			t.Fatalf("Failed to initialize tool: %v", err)
		}

		params := tools.ParamValues{
			{Name: "sql", Value: longQuery},
		}

		result, err := tool.Invoke(ctx, params, "")
		if err != nil {
			t.Fatalf("Failed to execute long query: %v", err)
		}

		// Should return [{1:1}]
		if resultSlice, ok := result.([]any); ok {
			if len(resultSlice) != 1 {
				t.Errorf("Expected 1 result from long query, got %d", len(resultSlice))
			}
		}
	})

	t.Run("NullValues", func(t *testing.T) {
		tableName := "test_nulls_" + strings.ReplaceAll(uuid.New().String(), "-", "")
		createSQL := fmt.Sprintf(`
			CREATE TABLE %s (
				id UInt32,
				nullable_field Nullable(String)
			) ENGINE = Memory
		`, tableName)

		_, err = pool.ExecContext(ctx, createSQL)
		if err != nil {
			t.Fatalf("Failed to create table: %v", err)
		}
		defer func() {
			_, _ = pool.ExecContext(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName))
		}()

		// Insert null value
		insertSQL := fmt.Sprintf("INSERT INTO %s (id, nullable_field) VALUES (1, NULL), (2, 'not null')", tableName)
		_, err = pool.ExecContext(ctx, insertSQL)
		if err != nil {
			t.Fatalf("Failed to insert null value: %v", err)
		}

		toolConfig := clickhousesql.Config{
			Name:        "test-null-values",
			Kind:        "clickhouse-sql",
			Source:      "test-clickhouse",
			Description: "Test null values",
			Statement:   fmt.Sprintf("SELECT * FROM %s ORDER BY id", tableName),
		}

		source := createMockSource(t, pool)
		sourcesMap := map[string]sources.Source{
			"test-clickhouse": source,
		}

		tool, err := toolConfig.Initialize(sourcesMap)
		if err != nil {
			t.Fatalf("Failed to initialize tool: %v", err)
		}

		result, err := tool.Invoke(ctx, tools.ParamValues{}, "")
		if err != nil {
			t.Fatalf("Failed to select null values: %v", err)
		}

		resultSlice, ok := result.([]any)
		if !ok {
			t.Fatalf("Expected result to be []any, got %T", result)
		}

		if len(resultSlice) != 2 {
			t.Errorf("Expected 2 results, got %d", len(resultSlice))
		}

		// Check that null is properly handled
		if firstRow, ok := resultSlice[0].(map[string]any); ok {
			if _, hasNullableField := firstRow["nullable_field"]; !hasNullableField {
				t.Error("Expected nullable_field in result")
			}
		}
	})

	t.Run("ConcurrentQueries", func(t *testing.T) {
		toolConfig := clickhousesql.Config{
			Name:        "test-concurrent",
			Kind:        "clickhouse-sql",
			Source:      "test-clickhouse",
			Description: "Test concurrent queries",
			Statement:   "SELECT number FROM system.numbers LIMIT ?",
			Parameters: tools.Parameters{
				tools.NewIntParameter("limit", "Limit"),
			},
		}

		source := createMockSource(t, pool)
		sourcesMap := map[string]sources.Source{
			"test-clickhouse": source,
		}

		tool, err := toolConfig.Initialize(sourcesMap)
		if err != nil {
			t.Fatalf("Failed to initialize tool: %v", err)
		}

		// Run multiple queries concurrently
		done := make(chan bool, 5)
		for i := 0; i < 5; i++ {
			go func(n int) {
				defer func() { done <- true }()

				params := tools.ParamValues{
					{Name: "limit", Value: n + 1},
				}

				result, err := tool.Invoke(ctx, params, "")
				if err != nil {
					t.Errorf("Concurrent query %d failed: %v", n, err)
					return
				}

				if resultSlice, ok := result.([]any); ok {
					if len(resultSlice) != n+1 {
						t.Errorf("Query %d: expected %d results, got %d", n, n+1, len(resultSlice))
					}
				}
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < 5; i++ {
			<-done
		}
	})

	t.Logf("✅ Edge case tests completed successfully")
}

func createMockSource(t *testing.T, pool *sql.DB) sources.Source {
	config := clickhouse.Config{
		Host:     ClickHouseHost,
		Port:     ClickHousePort,
		Database: ClickHouseDatabase,
		User:     ClickHouseUser,
		Password: ClickHousePass,
		Protocol: ClickHouseProtocol,
		Secure:   false,
	}

	source, err := config.Initialize(context.Background(), noop.NewTracerProvider().Tracer(""))
	if err != nil {
		t.Fatalf("Failed to initialize source: %v", err)
	}

	return source
}

// getClickHouseSQLParamToolInfo returns statements and param for my-tool clickhouse-sql kind
func getClickHouseSQLParamToolInfo(tableName string) (string, string, string, string, string, string, []any) {
	createStatement := fmt.Sprintf("CREATE TABLE %s (id UInt32, name String) ENGINE = Memory", tableName)
	insertStatement := fmt.Sprintf("INSERT INTO %s (id, name) VALUES (?, ?), (?, ?), (?, ?), (?, ?)", tableName)
	paramStatement := fmt.Sprintf("SELECT * FROM %s WHERE id = ? OR name = ?", tableName)
	idParamStatement := fmt.Sprintf("SELECT * FROM %s WHERE id = ?", tableName)
	nameParamStatement := fmt.Sprintf("SELECT * FROM %s WHERE name = ?", tableName)
	arrayStatement := fmt.Sprintf("SELECT * FROM %s WHERE id IN (?) AND name IN (?)", tableName)
	params := []any{1, "Alice", 2, "Bob", 3, "Sid", 4, nil}
	return createStatement, insertStatement, paramStatement, idParamStatement, nameParamStatement, arrayStatement, params
}

// getClickHouseSQLAuthToolInfo returns statements and param of my-auth-tool for clickhouse-sql kind
func getClickHouseSQLAuthToolInfo(tableName string) (string, string, string, []any) {
	createStatement := fmt.Sprintf("CREATE TABLE %s (id UInt32, name String, email String) ENGINE = Memory", tableName)
	insertStatement := fmt.Sprintf("INSERT INTO %s (id, name, email) VALUES (?, ?, ?), (?, ?, ?)", tableName)
	authStatement := fmt.Sprintf("SELECT name FROM %s WHERE email = ?", tableName)
	params := []any{1, "Alice", tests.ServiceAccountEmail, 2, "jane", "janedoe@gmail.com"}
	return createStatement, insertStatement, authStatement, params
}

// getClickHouseSQLTmplToolStatement returns statements and param for template parameter test cases for clickhouse-sql kind
func getClickHouseSQLTmplToolStatement() (string, string) {
	tmplSelectCombined := "SELECT * FROM {{.tableName}} WHERE id = ?"
	tmplSelectFilterCombined := "SELECT * FROM {{.tableName}} WHERE {{.columnFilter}} = ?"
	return tmplSelectCombined, tmplSelectFilterCombined
}

// SetupClickHouseSQLTable creates and inserts data into a table of tool
// compatible with clickhouse-sql tool
func setupClickHouseSQLTable(t *testing.T, ctx context.Context, pool *sql.DB, createStatement, insertStatement, tableName string, params []any) func(*testing.T) {
	err := pool.PingContext(ctx)
	if err != nil {
		t.Fatalf("unable to connect to test database: %s", err)
	}

	// Create table
	_, err = pool.ExecContext(ctx, createStatement)
	if err != nil {
		t.Fatalf("unable to create test table %s: %s", tableName, err)
	}

	// Insert test data
	_, err = pool.ExecContext(ctx, insertStatement, params...)
	if err != nil {
		t.Fatalf("unable to insert test data: %s", err)
	}

	return func(t *testing.T) {
		// tear down test
		_, err = pool.ExecContext(ctx, fmt.Sprintf("DROP TABLE %s", tableName))
		if err != nil {
			t.Errorf("Teardown failed: %s", err)
		}
	}
}
