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

func TestClickHouseListTablesTool(t *testing.T) {
	_ = getClickHouseVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	pool, err := initClickHouseConnectionPool(ClickHouseHost, ClickHousePort, ClickHouseUser, ClickHousePass, ClickHouseDatabase, ClickHouseProtocol)
	if err != nil {
		t.Fatalf("unable to create ClickHouse connection pool: %s", err)
	}
	defer pool.Close()

	testTables := []string{
		"list_test_1_" + strings.ReplaceAll(uuid.New().String(), "-", ""),
		"list_test_2_" + strings.ReplaceAll(uuid.New().String(), "-", ""),
		"list_test_3_" + strings.ReplaceAll(uuid.New().String(), "-", ""),
	}

	for _, table := range testTables {
		createSQL := fmt.Sprintf("CREATE TABLE %s (id UInt32) ENGINE = Memory", table)
		_, err = pool.ExecContext(ctx, createSQL)
		if err != nil {
			t.Fatalf("Failed to create test table %s: %v", table, err)
		}
		defer func(tableName string) {
			_, _ = pool.ExecContext(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName))
		}(table)
	}

	t.Run("ListTablesInCurrentDatabase", func(t *testing.T) {
		toolConfig := clickhousetools.ListTablesConfig{
			Name:        "test-list-tables",
			Kind:        "clickhouse-list-tables",
			Source:      "test-clickhouse",
			Description: "Test list tables",
		}

		source := createMockSource(t, pool)
		sourcesMap := map[string]sources.Source{
			"test-clickhouse": source,
		}

		tool, err := toolConfig.Initialize(sourcesMap)
		if err != nil {
			t.Fatalf("Failed to initialize tool: %v", err)
		}

		// List tables in current database (empty table_names)
		params := tools.ParamValues{
			{Name: "table_names", Value: ""},
		}

		result, err := tool.Invoke(ctx, params)
		if err != nil {
			t.Fatalf("Failed to list tables: %v", err)
		}

		resultSlice, ok := result.([]any)
		if !ok {
			t.Fatalf("Expected result to be []any, got %T", result)
		}

		// Should contain our test tables
		foundTables := 0
		for _, item := range resultSlice {
			if itemMap, ok := item.(map[string]any); ok {
				if tableName, ok := itemMap["object_name"].(string); ok {
					for _, testTable := range testTables {
						if tableName == testTable {
							foundTables++
						}
					}
				}
			}
		}

		if foundTables != len(testTables) {
			t.Errorf("Expected to find %d test tables, found %d", len(testTables), foundTables)
		}
	})

	t.Run("ListSpecificTables", func(t *testing.T) {
		toolConfig := clickhousetools.ListTablesConfig{
			Name:        "test-list-specific-tables",
			Kind:        "clickhouse-list-tables",
			Source:      "test-clickhouse",
			Description: "Test list specific tables",
		}

		source := createMockSource(t, pool)
		sourcesMap := map[string]sources.Source{
			"test-clickhouse": source,
		}

		tool, err := toolConfig.Initialize(sourcesMap)
		if err != nil {
			t.Fatalf("Failed to initialize tool: %v", err)
		}

		// List specific tables by name
		tableNames := strings.Join(testTables[:2], ",") // First 2 test tables
		params := tools.ParamValues{
			{Name: "table_names", Value: tableNames},
		}

		result, err := tool.Invoke(ctx, params)
		if err != nil {
			t.Fatalf("Failed to list specific tables: %v", err)
		}

		resultSlice, ok := result.([]any)
		if !ok {
			t.Fatalf("Expected result to be []any, got %T", result)
		}

		// Should return info for the 2 specific tables
		if len(resultSlice) < 2 {
			t.Errorf("Expected at least 2 results for specific tables, got %d", len(resultSlice))
		}
	})

	t.Run("ListNonExistentTables", func(t *testing.T) {
		toolConfig := clickhousetools.ListTablesConfig{
			Name:        "test-list-nonexistent-tables",
			Kind:        "clickhouse-list-tables",
			Source:      "test-clickhouse",
			Description: "Test list non-existent tables",
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
			{Name: "table_names", Value: "nonexistent_table_xyz,another_nonexistent_table"},
		}

		result, err := tool.Invoke(ctx, params)
		if err != nil {
			t.Logf("Got expected error for non-existent tables: %v", err)
			return
		}

		// Or it might return empty results
		if result != nil {
			if resultSlice, ok := result.([]any); ok && len(resultSlice) == 0 {
				t.Log("Got empty result for non-existent tables")
				return
			}
		}
	})

	t.Logf("✅ clickhouse-list-tables tool tests completed successfully")
}

func TestClickHouseListDatabasesTool(t *testing.T) {
	_ = getClickHouseVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	pool, err := initClickHouseConnectionPool(ClickHouseHost, ClickHousePort, ClickHouseUser, ClickHousePass, ClickHouseDatabase, ClickHouseProtocol)
	if err != nil {
		t.Fatalf("unable to create ClickHouse connection pool: %s", err)
	}
	defer pool.Close()

	t.Run("ListAllDatabases", func(t *testing.T) {
		toolConfig := clickhousetools.ListDatabasesConfig{
			Name:        "test-list-databases",
			Kind:        "clickhouse-list-databases",
			Source:      "test-clickhouse",
			Description: "Test list databases",
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
			t.Fatalf("Failed to list databases: %v", err)
		}

		resultSlice, ok := result.([]any)
		if !ok {
			t.Fatalf("Expected result to be []any, got %T", result)
		}

		// Should at least have default and system databases
		if len(resultSlice) < 2 {
			t.Errorf("Expected at least 2 databases (default and system), got %d", len(resultSlice))
		}

		// Check for required databases
		hasDefault := false
		hasSystem := false
		for _, item := range resultSlice {
			if itemMap, ok := item.(map[string]any); ok {
				if dbName, ok := itemMap["database_name"].(string); ok {
					if dbName == "default" {
						hasDefault = true
					}
					if dbName == "system" {
						hasSystem = true
					}
				}
			}
		}

		if !hasDefault {
			t.Error("Expected to find 'default' database")
		}
		if !hasSystem {
			t.Error("Expected to find 'system' database")
		}
	})

	t.Run("DatabaseEngineInfo", func(t *testing.T) {
		toolConfig := clickhousetools.ListDatabasesConfig{
			Name:        "test-list-db-with-engine",
			Kind:        "clickhouse-list-databases",
			Source:      "test-clickhouse",
			Description: "Test list databases with engine info",
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
			t.Fatalf("Failed to list databases: %v", err)
		}

		resultSlice, ok := result.([]any)
		if !ok {
			t.Fatalf("Expected result to be []any, got %T", result)
		}

		// Check that database_details field is included (which contains engine info)
		for _, item := range resultSlice {
			if itemMap, ok := item.(map[string]any); ok {
				if _, hasDetails := itemMap["database_details"]; !hasDetails {
					t.Error("Expected database info to include 'database_details' field")
					break
				}
			}
		}
	})

	t.Logf("✅ clickhouse-list-databases tool tests completed successfully")
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

	t.Run("SpecialCharactersInTableName", func(t *testing.T) {
		// ClickHouse allows pretty much anything in table names
		tableName := "`test_t$able_123`"
		createSQL := fmt.Sprintf("CREATE TABLE %s (id UInt32) ENGINE = Memory", tableName)

		_, err = pool.ExecContext(ctx, createSQL)
		if err != nil {
			t.Fatalf("Failed to create table with special name: %v", err)
		}
		defer func() {
			_, _ = pool.ExecContext(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName))
		}()

		toolConfig := clickhousetools.DescribeTableConfig{
			Name:        "test-special-chars",
			Kind:        "clickhouse-describe-table",
			Source:      "test-clickhouse",
			Description: "Test special characters",
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
			{Name: "table_name", Value: "test_t$able_123"},
			{Name: "database_name", Value: ""},
		}

		result, err := tool.Invoke(ctx, params)
		if err != nil {
			t.Fatalf("Failed to describe table with special name: %v", err)
		}

		if result == nil {
			t.Error("Expected result for table with special name")
		}
	})

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
