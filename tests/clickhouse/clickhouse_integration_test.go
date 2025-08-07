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
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"
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

	createParamTableStmt, insertParamTableStmt, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, paramTestParams := tests.GetClickHouseSQLParamToolInfo(tableNameParam)
	teardownTable1 := tests.SetupClickHouseSQLTable(t, ctx, pool, createParamTableStmt, insertParamTableStmt, tableNameParam, paramTestParams)
	defer teardownTable1(t)

	createAuthTableStmt, insertAuthTableStmt, authToolStmt, authTestParams := tests.GetClickHouseSQLAuthToolInfo(tableNameAuth)
	teardownTable2 := tests.SetupClickHouseSQLTable(t, ctx, pool, createAuthTableStmt, insertAuthTableStmt, tableNameAuth, authTestParams)
	defer teardownTable2(t)

	toolsFile := tests.GetToolsConfig(sourceConfig, ClickHouseToolKind, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, authToolStmt)
	toolsFile = AddClickHouseExecuteSqlConfig(t, toolsFile)
	tmplSelectCombined, tmplSelectFilterCombined := tests.GetClickHouseSQLTmplToolStatement()
	toolsFile = tests.AddTemplateParamConfig(t, toolsFile, ClickHouseToolKind, tmplSelectCombined, tmplSelectFilterCombined, "")

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

	select1Want, failInvocationWant, createTableStatement := GetClickHouseWants()
	invokeParamWant, invokeIdNullWant, nullWant, mcpInvokeParamWant := tests.GetNonSpannerInvokeParamWant()
	tests.RunToolInvokeTest(t, select1Want, invokeParamWant, invokeIdNullWant, nullWant, true, false)
	tests.RunExecuteSqlToolInvokeTest(t, createTableStatement, select1Want)
	tests.RunMCPToolCallMethod(t, mcpInvokeParamWant, failInvocationWant)
	tests.RunToolInvokeWithTemplateParameters(t, tableNameTemplateParam, tests.NewTemplateParameterTestConfig())
}

func AddClickHouseExecuteSqlConfig(t *testing.T, config map[string]any) map[string]any {
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

func GetClickHouseWants() (string, string, string) {
	select1Want := "[{\"1\":1}]"
	failInvocationWant := `{"jsonrpc":"2.0","id":"invoke-fail-tool","result":{"content":[{"type":"text","text":"unable to execute query: code: 62, message: Syntax error: failed at position 1 (line 1, col 1): SELEC 1;. Expected one of: EXPLAIN, SELECT, INSERT, DELETE, UPDATE, CREATE, ALTER, DROP, RENAME, SET, OPTIMIZE, USE, EXISTS, SHOW, DESCRIBE, DESC, WITH, SYSTEM, KILL, WATCH, CHECK"}],"isError":true}}`
	createTableStatement := `"CREATE TABLE t (id UInt32, name String) ENGINE = Memory"`
	return select1Want, failInvocationWant, createTableStatement
}
