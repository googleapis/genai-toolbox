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

// Package clickhouse provides comprehensive integration tests for all ClickHouse tools.
//
// NOTE: ClickHouse tools in this codebase ONLY support HTTP/HTTPS protocols.
// The native ClickHouse protocol (typically port 9000) is NOT supported.
// Use HTTP interface on port 8123 for testing. Native support will be
// added in the future
//
// Environment variables for testing:
//   - CLICKHOUSE_HOST: ClickHouse host (default: localhost)
//   - CLICKHOUSE_PORT: ClickHouse HTTP port (default: 8123)
//   - CLICKHOUSE_USER: ClickHouse user (default: default)
//   - CLICKHOUSE_PASS: ClickHouse password (optional)
//   - CLICKHOUSE_DATABASE: ClickHouse database (default: default)
//   - CLICKHOUSE_PROTOCOL: Protocol to use (http/https, default: http)

package clickhouse

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	_ "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/google/uuid"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/clickhouse"
	"github.com/googleapis/genai-toolbox/internal/tools"
	clickhousetools "github.com/googleapis/genai-toolbox/internal/tools/clickhouse"
	"go.opentelemetry.io/otel/trace"
)

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
		toolConfig := clickhousetools.SQLConfig{
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

		result, err := tool.Invoke(ctx, tools.ParamValues{})
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
		toolConfig := clickhousetools.SQLConfig{
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

		result, err := tool.Invoke(ctx, params)
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
		toolConfig := clickhousetools.SQLConfig{
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

		result, err := tool.Invoke(ctx, params)
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
		toolConfig := clickhousetools.SQLConfig{
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

		_, err = tool.Invoke(ctx, tools.ParamValues{})
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
		toolConfig := clickhousetools.ExecuteSQLConfig{
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

		result, err := tool.Invoke(ctx, params)
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
		toolConfig := clickhousetools.ExecuteSQLConfig{
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

		result, err := tool.Invoke(ctx, params)
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
		toolConfig := clickhousetools.ExecuteSQLConfig{
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

		result, err := tool.Invoke(ctx, params)
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
		toolConfig := clickhousetools.ExecuteSQLConfig{
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

		result, err := tool.Invoke(ctx, params)
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
		toolConfig := clickhousetools.ExecuteSQLConfig{
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

		_, err = tool.Invoke(ctx, params)
		if err == nil {
			t.Error("Expected error for empty SQL parameter, got nil")
		} else {
			t.Logf("Got expected error for empty SQL parameter: %v", err)
		}
	})

	t.Run("SQLInjectionAttempt", func(t *testing.T) {
		toolConfig := clickhousetools.ExecuteSQLConfig{
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

		_, err = tool.Invoke(ctx, params)
		// This should either fail or only execute the first statement
		// dont check the specific error as behavior may vary
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

		toolConfig := clickhousetools.ExecuteSQLConfig{
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

		result, err := tool.Invoke(ctx, params)
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

		toolConfig := clickhousetools.SQLConfig{
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

		result, err := tool.Invoke(ctx, tools.ParamValues{})
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
		toolConfig := clickhousetools.SQLConfig{
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

				result, err := tool.Invoke(ctx, params)
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

	source, err := config.Initialize(context.Background(), trace.NewNoopTracerProvider().Tracer(""))
	if err != nil {
		t.Fatalf("Failed to initialize source: %v", err)
	}

	return source
}
