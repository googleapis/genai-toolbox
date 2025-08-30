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

//go:build integration
// +build integration

package trino_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/trino"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/tools/trino/trinoanalyze"
	"github.com/googleapis/genai-toolbox/internal/tools/trino/trinoexecutesql"
	"github.com/googleapis/genai-toolbox/internal/tools/trino/trinogettableinfo"
	"github.com/googleapis/genai-toolbox/internal/tools/trino/trinolistcatalogs"
	"github.com/googleapis/genai-toolbox/internal/tools/trino/trinolistschemas"
	"github.com/googleapis/genai-toolbox/internal/tools/trino/trinolisttables"
	"github.com/googleapis/genai-toolbox/internal/tools/trino/trinoschema"
	"github.com/googleapis/genai-toolbox/internal/tools/trino/trinosql"
	"github.com/googleapis/genai-toolbox/internal/tools/trino/trinotablestatistics"
	"go.opentelemetry.io/otel/trace"
)

// getTrinoSource returns a test Trino source
// The database instance should be available for tests
func getTrinoSource(t *testing.T) sources.Source {
	t.Helper()

	// Create Trino source config for testing
	config := trino.Config{
		Name:    "test-trino",
		Host:    "localhost",
		Port:    "8080",
		Catalog: "memory",
		Schema:  "default",
		User:    "test",
	}

	ctx := context.Background()
	var tracer trace.Tracer // Use a no-op tracer for tests

	source, err := config.Initialize(ctx, tracer)
	if err != nil {
		t.Fatalf("Failed to initialize Trino source: %v", err)
	}

	return source
}

// TestTrinoExecuteSQL tests the trino-execute-sql tool
func TestTrinoExecuteSQL(t *testing.T) {
	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}

	source := getTrinoSource(t)
	sources := map[string]sources.Source{
		"test-trino": source,
	}

	// Create and initialize the tool
	config := trinoexecutesql.Config{
		Name:        "test-execute-sql",
		Kind:        "trino-execute-sql",
		Source:      "test-trino",
		Description: "Test execute SQL tool",
	}

	tool, err := config.Initialize(sources)
	if err != nil {
		t.Fatalf("Failed to initialize tool: %v", err)
	}

	// Test a simple query
	params := tools.ParamValues{
		{Name: "sql", Value: "SELECT 1 as test_column"},
	}

	result, err := tool.Invoke(ctx, params, "")
	if err != nil {
		t.Fatalf("Failed to invoke tool: %v", err)
	}

	// Verify result
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Check if result is an array with at least one row
	rows, ok := result.([]any)
	if !ok {
		t.Fatalf("Expected result to be []any, got %T", result)
	}

	if len(rows) != 1 {
		t.Fatalf("Expected 1 row, got %d", len(rows))
	}
}

// TestTrinoListCatalogs tests the trino-list-catalogs tool
func TestTrinoListCatalogs(t *testing.T) {
	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}

	source := getTrinoSource(t)
	sources := map[string]sources.Source{
		"test-trino": source,
	}

	// Create and initialize the tool
	config := trinolistcatalogs.Config{
		Name:        "test-list-catalogs",
		Kind:        "trino-list-catalogs",
		Source:      "test-trino",
		Description: "Test list catalogs tool",
	}

	tool, err := config.Initialize(sources)
	if err != nil {
		t.Fatalf("Failed to initialize tool: %v", err)
	}

	// Test listing catalogs
	result, err := tool.Invoke(ctx, tools.ParamValues{}, "")
	if err != nil {
		t.Fatalf("Failed to invoke tool: %v", err)
	}

	// Verify result structure
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	response, ok := result.(trinolistcatalogs.CatalogsResponse)
	if !ok {
		t.Fatalf("Expected result to be CatalogsResponse, got %T", result)
	}

	if response.TotalCount == 0 {
		t.Fatal("Expected at least one catalog")
	}

	if len(response.Catalogs) != response.TotalCount {
		t.Fatalf("Catalog count mismatch: got %d catalogs but TotalCount is %d",
			len(response.Catalogs), response.TotalCount)
	}
}

// TestTrinoListSchemas tests the trino-list-schemas tool
func TestTrinoListSchemas(t *testing.T) {
	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}

	source := getTrinoSource(t)
	sources := map[string]sources.Source{
		"test-trino": source,
	}

	// Create and initialize the tool
	config := trinolistschemas.Config{
		Name:        "test-list-schemas",
		Kind:        "trino-list-schemas",
		Source:      "test-trino",
		Description: "Test list schemas tool",
	}

	tool, err := config.Initialize(sources)
	if err != nil {
		t.Fatalf("Failed to initialize tool: %v", err)
	}

	// Test listing schemas in current catalog
	params := tools.ParamValues{
		{Name: "include_system", Value: false},
	}

	result, err := tool.Invoke(ctx, params, "")
	if err != nil {
		t.Fatalf("Failed to invoke tool: %v", err)
	}

	// Verify result structure
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	response, ok := result.(trinolistschemas.SchemasResponse)
	if !ok {
		t.Fatalf("Expected result to be SchemasResponse, got %T", result)
	}

	if response.Catalog == "" {
		t.Fatal("Expected catalog name to be set")
	}
}

// TestTrinoListTables tests the trino-list-tables tool
func TestTrinoListTables(t *testing.T) {
	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}

	source := getTrinoSource(t)
	sources := map[string]sources.Source{
		"test-trino": source,
	}

	// First create a test table
	db := source.(*trino.Source).TrinoDB()
	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS test_table (
			id INTEGER,
			name VARCHAR(100)
		)
	`)
	if err != nil {
		t.Skipf("Could not create test table: %v", err)
	}
	defer func() {
		_, _ = db.ExecContext(ctx, "DROP TABLE IF EXISTS test_table")
	}()

	// Create and initialize the tool
	config := trinolisttables.Config{
		Name:        "test-list-tables",
		Kind:        "trino-list-tables",
		Source:      "test-trino",
		Description: "Test list tables tool",
	}

	tool, err := config.Initialize(sources)
	if err != nil {
		t.Fatalf("Failed to initialize tool: %v", err)
	}

	// Test listing tables
	params := tools.ParamValues{
		{Name: "include_views", Value: true},
		{Name: "include_details", Value: false},
	}

	result, err := tool.Invoke(ctx, params, "")
	if err != nil {
		t.Fatalf("Failed to invoke tool: %v", err)
	}

	// Verify result structure
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	response, ok := result.(trinolisttables.TablesResponse)
	if !ok {
		t.Fatalf("Expected result to be TablesResponse, got %T", result)
	}

	if response.TotalCount != len(response.Tables) {
		t.Fatalf("Table count mismatch: got %d tables but TotalCount is %d",
			len(response.Tables), response.TotalCount)
	}
}

// TestTrinoGetTableInfo tests the trino-get-table-info tool
func TestTrinoGetTableInfo(t *testing.T) {
	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}

	source := getTrinoSource(t)
	sources := map[string]sources.Source{
		"test-trino": source,
	}

	// First create a test table
	db := source.(*trino.Source).TrinoDB()
	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS test_info_table (
			id INTEGER,
			name VARCHAR(100),
			created_date DATE
		)
	`)
	if err != nil {
		t.Skipf("Could not create test table: %v", err)
	}
	defer func() {
		_, _ = db.ExecContext(ctx, "DROP TABLE IF EXISTS test_info_table")
	}()

	// Insert some test data
	_, err = db.ExecContext(ctx, `
		INSERT INTO test_info_table VALUES 
		(1, 'Test 1', DATE '2024-01-01'),
		(2, 'Test 2', DATE '2024-01-02')
	`)
	if err != nil {
		t.Logf("Warning: Could not insert test data: %v", err)
	}

	// Create and initialize the tool
	config := trinogettableinfo.Config{
		Name:        "test-get-table-info",
		Kind:        "trino-get-table-info",
		Source:      "test-trino",
		Description: "Test get table info tool",
	}

	tool, err := config.Initialize(sources)
	if err != nil {
		t.Fatalf("Failed to initialize tool: %v", err)
	}

	// Test getting table info
	params := tools.ParamValues{
		{Name: "table_name", Value: "test_info_table"},
		{Name: "include_stats", Value: false},
		{Name: "include_sample", Value: true},
		{Name: "sample_size", Value: 2},
	}

	result, err := tool.Invoke(ctx, params, "")
	if err != nil {
		t.Fatalf("Failed to invoke tool: %v", err)
	}

	// Verify result structure
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	metadata, ok := result.(*trinogettableinfo.TableMetadata)
	if !ok {
		t.Fatalf("Expected result to be *TableMetadata, got %T", result)
	}

	if metadata.TableName != "test_info_table" {
		t.Fatalf("Expected table name 'test_info_table', got '%s'", metadata.TableName)
	}

	if metadata.ColumnCount != 3 {
		t.Fatalf("Expected 3 columns, got %d", metadata.ColumnCount)
	}

	if len(metadata.Columns) != 3 {
		t.Fatalf("Expected 3 column definitions, got %d", len(metadata.Columns))
	}
}

// TestTrinoSchema tests the trino-schema tool
func TestTrinoSchema(t *testing.T) {
	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}

	source := getTrinoSource(t)
	sources := map[string]sources.Source{
		"test-trino": source,
	}

	// Create and initialize the tool
	cacheMinutes := 5
	config := trinoschema.Config{
		Name:               "test-schema",
		Kind:               "trino-schema",
		Source:             "test-trino",
		Description:        "Test schema tool",
		CacheExpireMinutes: &cacheMinutes,
	}

	tool, err := config.Initialize(sources)
	if err != nil {
		t.Fatalf("Failed to initialize tool: %v", err)
	}

	// Test getting schema
	result, err := tool.Invoke(ctx, tools.ParamValues{}, "")
	if err != nil {
		t.Fatalf("Failed to invoke tool: %v", err)
	}

	// Verify result structure
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	schema, ok := result.(*trinoschema.SchemaInfo)
	if !ok {
		t.Fatalf("Expected result to be *SchemaInfo, got %T", result)
	}

	if len(schema.Catalogs) == 0 {
		t.Fatal("Expected at least one catalog")
	}

	if schema.Statistics.TotalCatalogs == 0 {
		t.Fatal("Expected TotalCatalogs to be greater than 0")
	}

	// Test caching - second call should be faster
	result2, err := tool.Invoke(ctx, tools.ParamValues{}, "")
	if err != nil {
		t.Fatalf("Failed to invoke tool second time: %v", err)
	}

	if result2 == nil {
		t.Fatal("Expected non-nil result on second call")
	}
}

// TestTrinoAnalyze tests the trino-analyze tool
func TestTrinoAnalyze(t *testing.T) {
	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}

	source := getTrinoSource(t)
	sources := map[string]sources.Source{
		"test-trino": source,
	}

	// Create and initialize the tool
	config := trinoanalyze.Config{
		Name:        "test-analyze",
		Kind:        "trino-analyze",
		Source:      "test-trino",
		Description: "Test analyze tool",
	}

	tool, err := config.Initialize(sources)
	if err != nil {
		t.Fatalf("Failed to initialize tool: %v", err)
	}

	// Test analyzing a simple query
	params := tools.ParamValues{
		{Name: "query", Value: "SELECT 1 as test_column"},
		{Name: "format", Value: "text"},
		{Name: "analyze", Value: false},
		{Name: "distributed", Value: false},
		{Name: "validate", Value: false},
	}

	result, err := tool.Invoke(ctx, params, "")
	if err != nil {
		t.Fatalf("Failed to invoke tool: %v", err)
	}

	// Verify result structure
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	plan, ok := result.(*trinoanalyze.QueryPlan)
	if !ok {
		t.Fatalf("Expected result to be *QueryPlan, got %T", result)
	}

	if !plan.IsValid {
		t.Fatalf("Expected query to be valid, got validation errors: %v", plan.ValidationErrors)
	}

	if plan.Query != "SELECT 1 as test_column" {
		t.Fatalf("Expected query to match input, got '%s'", plan.Query)
	}

	// Test query validation
	params2 := tools.ParamValues{
		{Name: "query", Value: "SELECT * FROM non_existent_table"},
		{Name: "validate", Value: true},
	}

	result2, err := tool.Invoke(ctx, params2, "")
	if err != nil {
		// This might fail if the table doesn't exist, which is expected
		t.Logf("Query validation returned error (expected): %v", err)
	}

	if result2 != nil {
		plan2, ok := result2.(*trinoanalyze.QueryPlan)
		if ok && !plan2.IsValid {
			t.Logf("Query correctly identified as invalid: %v", plan2.ValidationErrors)
		}
	}
}

// TestTrinoTableStatistics tests the trino-table-statistics tool
func TestTrinoTableStatistics(t *testing.T) {
	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}

	source := getTrinoSource(t)
	sources := map[string]sources.Source{
		"test-trino": source,
	}

	// First create a test table
	db := source.(*trino.Source).TrinoDB()
	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS test_stats_table (
			id INTEGER,
			value DOUBLE,
			description VARCHAR(200)
		)
	`)
	if err != nil {
		t.Skipf("Could not create test table: %v", err)
	}
	defer func() {
		_, _ = db.ExecContext(ctx, "DROP TABLE IF EXISTS test_stats_table")
	}()

	// Insert some test data
	for i := 1; i <= 10; i++ {
		_, err = db.ExecContext(ctx,
			fmt.Sprintf("INSERT INTO test_stats_table VALUES (%d, %f, 'Description %d')",
				i, float64(i)*1.5, i))
		if err != nil {
			t.Logf("Warning: Could not insert test data: %v", err)
		}
	}

	// Create and initialize the tool
	config := trinotablestatistics.Config{
		Name:        "test-table-statistics",
		Kind:        "trino-table-statistics",
		Source:      "test-trino",
		Description: "Test table statistics tool",
	}

	tool, err := config.Initialize(sources)
	if err != nil {
		t.Fatalf("Failed to initialize tool: %v", err)
	}

	// Test getting table statistics
	params := tools.ParamValues{
		{Name: "table_name", Value: "test_stats_table"},
		{Name: "include_columns", Value: true},
		{Name: "include_partitions", Value: false},
		{Name: "analyze_table", Value: false},
	}

	result, err := tool.Invoke(ctx, params, "")
	if err != nil {
		t.Fatalf("Failed to invoke tool: %v", err)
	}

	// Verify result structure
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	stats, ok := result.(*trinotablestatistics.TableStatistics)
	if !ok {
		t.Fatalf("Expected result to be *TableStatistics, got %T", result)
	}

	if stats.TableName != "test_stats_table" {
		t.Fatalf("Expected table name 'test_stats_table', got '%s'", stats.TableName)
	}

	// The tool should have retrieved column statistics
	if len(stats.ColumnStatistics) > 0 {
		t.Logf("Retrieved statistics for %d columns", len(stats.ColumnStatistics))
	}
}

// TestTrinoSQL tests the trino-sql tool
func TestTrinoSQL(t *testing.T) {
	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("Failed to create context: %v", err)
	}

	source := getTrinoSource(t)
	sources := map[string]sources.Source{
		"test-trino": source,
	}

	// Create and initialize the tool
	config := trinosql.Config{
		Name:        "test-sql",
		Kind:        "trino-sql",
		Source:      "test-trino",
		Description: "Test SQL tool",
		Statement:   "SELECT ? as param1, ? as param2",
		Parameters: tools.Parameters{
			tools.NewStringParameter("param1", "First parameter"),
			tools.NewIntParameter("param2", "Second parameter"),
		},
	}

	tool, err := config.Initialize(sources)
	if err != nil {
		t.Fatalf("Failed to initialize tool: %v", err)
	}

	// Test executing parameterized query
	params := tools.ParamValues{
		{Name: "param1", Value: "test_value"},
		{Name: "param2", Value: 42},
	}

	result, err := tool.Invoke(ctx, params, "")
	if err != nil {
		t.Fatalf("Failed to invoke tool: %v", err)
	}

	// Verify result
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	rows, ok := result.([]any)
	if !ok {
		t.Fatalf("Expected result to be []any, got %T", result)
	}

	if len(rows) != 1 {
		t.Fatalf("Expected 1 row, got %d", len(rows))
	}

	row, ok := rows[0].(map[string]any)
	if !ok {
		t.Fatalf("Expected row to be map[string]any, got %T", rows[0])
	}

	if row["param1"] != "test_value" {
		t.Fatalf("Expected param1 to be 'test_value', got '%v'", row["param1"])
	}

	// Check param2 - it might be returned as int64 or float64 depending on the driver
	param2Val := row["param2"]
	switch v := param2Val.(type) {
	case int64:
		if v != 42 {
			t.Fatalf("Expected param2 to be 42, got %d", v)
		}
	case float64:
		if v != 42.0 {
			t.Fatalf("Expected param2 to be 42, got %f", v)
		}
	case int:
		if v != 42 {
			t.Fatalf("Expected param2 to be 42, got %d", v)
		}
	default:
		t.Fatalf("Expected param2 to be numeric, got %T: %v", param2Val, param2Val)
	}
}

// TestMain runs the integration tests
func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
