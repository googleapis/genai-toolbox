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
	"testing"
	"time"

	_ "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"
)

var (
	CLICKHOUSE_SOURCE_KIND = "clickhouse"
	CLICKHOUSE_TOOL_KIND   = "clickhouse-sql"
	CLICKHOUSE_DATABASE    = os.Getenv("CLICKHOUSE_DATABASE")
	CLICKHOUSE_HOST        = os.Getenv("CLICKHOUSE_HOST")
	CLICKHOUSE_PORT        = os.Getenv("CLICKHOUSE_PORT")
	CLICKHOUSE_USER        = os.Getenv("CLICKHOUSE_USER")
	CLICKHOUSE_PASS        = os.Getenv("CLICKHOUSE_PASS")
	CLICKHOUSE_PROTOCOL    = os.Getenv("CLICKHOUSE_PROTOCOL")
)

func getClickHouseVars(t *testing.T) map[string]any {
	switch "" {
	case CLICKHOUSE_HOST:
		t.Skip("'CLICKHOUSE_HOST' not set")
	case CLICKHOUSE_PORT:
		t.Skip("'CLICKHOUSE_PORT' not set")
	case CLICKHOUSE_USER:
		t.Skip("'CLICKHOUSE_USER' not set")
	}

	// Set defaults for optional parameters
	if CLICKHOUSE_DATABASE == "" {
		CLICKHOUSE_DATABASE = "default"
	}
	if CLICKHOUSE_PROTOCOL == "" {
		CLICKHOUSE_PROTOCOL = "http"
	}

	return map[string]any{
		"kind":     CLICKHOUSE_SOURCE_KIND,
		"host":     CLICKHOUSE_HOST,
		"port":     CLICKHOUSE_PORT,
		"database": CLICKHOUSE_DATABASE,
		"user":     CLICKHOUSE_USER,
		"password": CLICKHOUSE_PASS,
		"protocol": CLICKHOUSE_PROTOCOL,
		"secure":   false,
	}
}

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

func TestClickHouseIntegration(t *testing.T) {
	sourceConfig := getClickHouseVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	pool, err := initClickHouseConnectionPool(CLICKHOUSE_HOST, CLICKHOUSE_PORT, CLICKHOUSE_USER, CLICKHOUSE_PASS, CLICKHOUSE_DATABASE, CLICKHOUSE_PROTOCOL)
	if err != nil {
		t.Fatalf("unable to create ClickHouse connection pool: %s", err)
	}
	defer pool.Close()

	err = pool.PingContext(ctx)
	if err != nil {
		t.Fatalf("unable to ping ClickHouse: %s", err)
	}

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

	toolsFile := map[string]any{
		"sources": map[string]any{
			"my-instance": sourceConfig,
		},
		"tools": map[string]any{
			"my-simple-tool": map[string]any{
				"kind":        CLICKHOUSE_TOOL_KIND,
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

	t.Logf("✅ ClickHouse integration test completed successfully (auth tests skipped)")
}

func TestClickHouseBasicConnection(t *testing.T) {
	sourceConfig := getClickHouseVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	pool, err := initClickHouseConnectionPool(CLICKHOUSE_HOST, CLICKHOUSE_PORT, CLICKHOUSE_USER, CLICKHOUSE_PASS, CLICKHOUSE_DATABASE, CLICKHOUSE_PROTOCOL)
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
				"kind":        CLICKHOUSE_TOOL_KIND,
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
}

func GetClickHouseWants() (string, string, string) {
	select1Want := "[{\"1\":1}]"
	failInvocationWant := `{"jsonrpc":"2.0","id":"invoke-fail-tool","result":{"content":[{"type":"text","text":"unable to execute query: code: 62, message: Syntax error: failed at position 1 (line 1, col 1): SELEC 1;. Expected one of: EXPLAIN, SELECT, INSERT, DELETE, UPDATE, CREATE, ALTER, DROP, RENAME, SET, OPTIMIZE, USE, EXISTS, SHOW, DESCRIBE, DESC, WITH, SYSTEM, KILL, WATCH, CHECK"}],"isError":true}}`
	createTableStatement := `"CREATE TABLE t (id UInt32, name String) ENGINE = Memory"`
	return select1Want, failInvocationWant, createTableStatement
}
