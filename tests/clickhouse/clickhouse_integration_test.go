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
	"testing"
	"time"

	_ "github.com/ClickHouse/clickhouse-go/v2"
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
	CLICKHOUSE_PROTOCOL    = os.Getenv("CLICKHOUSE_PROTOCOL") // native, http, https
)

func getClickHouseVars(t *testing.T) map[string]any {
	switch "" {
	case CLICKHOUSE_DATABASE:
		t.Fatal("'CLICKHOUSE_DATABASE' not set")
	case CLICKHOUSE_HOST:
		t.Fatal("'CLICKHOUSE_HOST' not set")
	case CLICKHOUSE_PORT:
		t.Fatal("'CLICKHOUSE_PORT' not set")
	case CLICKHOUSE_USER:
		t.Fatal("'CLICKHOUSE_USER' not set")
	}

	// Set default protocol if not specified
	if CLICKHOUSE_PROTOCOL == "" {
		CLICKHOUSE_PROTOCOL = "native"
	}

	return map[string]any{
		"kind":        CLICKHOUSE_SOURCE_KIND,
		"host":        CLICKHOUSE_HOST,
		"port":        CLICKHOUSE_PORT,
		"database":    CLICKHOUSE_DATABASE,
		"user":        CLICKHOUSE_USER,
		"password":    CLICKHOUSE_PASS,
		"protocol":    CLICKHOUSE_PROTOCOL,
		"secure":      false,
		"compression": "lz4",
	}
}

// Copied over from clickhouse.go
func initClickHouseConnectionPool(host, port, user, pass, dbname, protocol string) (*sql.DB, error) {
	// Set default protocol if not specified
	if protocol == "" {
		protocol = "native"
	}

	// Build DSN based on protocol
	var dsn string
	switch protocol {
	case "http":
		dsn = fmt.Sprintf("http://%s:%s@%s:%s/%s?compress=lz4", user, pass, host, port, dbname)
	case "https":
		dsn = fmt.Sprintf("https://%s:%s@%s:%s/%s?compress=lz4", user, pass, host, port, dbname)
	case "native":
		fallthrough
	default:
		dsn = fmt.Sprintf("clickhouse://%s:%s@%s:%s/%s?compress=lz4", user, pass, host, port, dbname)
	}

	pool, err := sql.Open("clickhouse", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}

	return pool, nil
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

	// Write a basic tools config and test the server endpoint
	toolsFile := tests.GetToolsConfig(sourceConfig, CLICKHOUSE_TOOL_KIND, "SELECT 1;", "SELECT 1;")

	_, cleanup, err := tests.StartCmd(ctx, toolsFile, args...)
	if err != nil {
		t.Fatalf("command initialization returned an error: %s", err)
	}
	defer cleanup()

	// Give the server a moment to start
	time.Sleep(2 * time.Second)

	tests.RunToolGetTest(t)
}
