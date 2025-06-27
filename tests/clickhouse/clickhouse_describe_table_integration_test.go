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
	"testing"
	"time"

	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/clickhouse"
	"github.com/googleapis/genai-toolbox/internal/tools"
	clickhousetools "github.com/googleapis/genai-toolbox/internal/tools/clickhouse"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestDescribeTableBasic(t *testing.T) {
	// Test tool configuration and basic functionality without requiring actual ClickHouse connection
	// Create a mock source configuration
	mockSourceConfig := clickhouse.Config{
		Name:     "test-clickhouse",
		Kind:     "clickhouse",
		Host:     "localhost",
		Port:     "9000",
		User:     "default",
		Password: "",
		Database: "system",
		Protocol: "native",
		Secure:   false,
	}

	// Test tool configuration creation
	toolConfig := clickhousetools.DescribeTableConfig{
		Name:        "describe_table",
		Kind:        "clickhouse-describe-table",
		Source:      "test-clickhouse",
		Description: "Test describe table tool",
	}

	// Verify tool config kind
	if toolConfig.ToolConfigKind() != "clickhouse-describe-table" {
		t.Errorf("Expected tool kind 'clickhouse-describe-table', got %s", toolConfig.ToolConfigKind())
	}

	// Test parameter validation
	expectedParams := []string{"table_name"}
	sourcesMap := map[string]sources.Source{
		"test-clickhouse": &mockClickHouseSource{config: mockSourceConfig},
	}

	tool, err := toolConfig.Initialize(sourcesMap)
	if err != nil {
		t.Fatalf("Failed to initialize tool: %v", err)
	}

	manifest := tool.Manifest()
	if len(manifest.Parameters) != len(expectedParams) {
		t.Errorf("Expected %d parameters, got %d", len(expectedParams), len(manifest.Parameters))
	}

	// Verify parameter names
	for i, param := range manifest.Parameters {
		if param.Name != expectedParams[i] {
			t.Errorf("Expected parameter %s, got %s", expectedParams[i], param.Name)
		}
	}

	t.Logf("✅ Tool configuration and parameters validated successfully")
}

func TestDescribeTableWithRealConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Only run if ClickHouse environment variables are set
	sourceConfig := getClickHouseVars(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Initialize ClickHouse source
	chConfig := clickhouse.Config{
		Name:        "test-clickhouse",
		Kind:        "clickhouse",
		Host:        sourceConfig["host"].(string),
		Port:        sourceConfig["port"].(string),
		User:        sourceConfig["user"].(string),
		Password:    sourceConfig["password"].(string),
		Database:    sourceConfig["database"].(string),
		Protocol:    sourceConfig["protocol"].(string),
		Secure:      sourceConfig["secure"].(bool),
		Compression: sourceConfig["compression"].(string),
	}

	// Create tracer
	tracer := noop.NewTracerProvider().Tracer("test")

	// Initialize source
	source, err := chConfig.Initialize(ctx, tracer)
	if err != nil {
		t.Skipf("Cannot connect to ClickHouse: %v", err)
	}

	// Create sources map
	sourcesMap := map[string]sources.Source{
		"test-clickhouse": source,
	}

	// Initialize describe_table tool
	toolConfig := clickhousetools.DescribeTableConfig{
		Name:        "describe_table",
		Kind:        "clickhouse-describe-table",
		Source:      "test-clickhouse",
		Description: "Test describe table tool",
	}

	tool, err := toolConfig.Initialize(sourcesMap)
	if err != nil {
		t.Fatalf("Failed to initialize describe_table tool: %v", err)
	}

	// Test with system.tables (should exist in any ClickHouse instance)
	params := tools.ParamValues{
		{Name: "database", Value: "system"},
		{Name: "table", Value: "tables"},
	}

	result, err := tool.Invoke(ctx, params)
	if err != nil {
		t.Fatalf("Failed to describe system.tables: %v", err)
	}

	if len(result) == 0 {
		t.Fatal("Expected at least one result")
	}

	// Verify the result structure
	resultMap, ok := result[0].(map[string]any)
	if !ok {
		t.Fatalf("Expected result to be a map, got %T", result[0])
	}

	// Verify expected fields in the result
	expectedFields := []string{"database", "table", "engine", "columns"}
	for _, field := range expectedFields {
		if _, exists := resultMap[field]; !exists {
			t.Errorf("Expected field %s not found in result", field)
		}
	}

	t.Logf("✅ Successfully described system.tables table with real connection")
}

func TestDescribeTableNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Only run if ClickHouse environment variables are set
	sourceConfig := getClickHouseVars(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Initialize ClickHouse source
	chConfig := clickhouse.Config{
		Name:        "test-clickhouse",
		Kind:        "clickhouse",
		Host:        sourceConfig["host"].(string),
		Port:        sourceConfig["port"].(string),
		User:        sourceConfig["user"].(string),
		Password:    sourceConfig["password"].(string),
		Database:    sourceConfig["database"].(string),
		Protocol:    sourceConfig["protocol"].(string),
		Secure:      sourceConfig["secure"].(bool),
		Compression: sourceConfig["compression"].(string),
	}

	// Create tracer
	tracer := noop.NewTracerProvider().Tracer("test")

	// Initialize source
	source, err := chConfig.Initialize(ctx, tracer)
	if err != nil {
		t.Skipf("Cannot connect to ClickHouse: %v", err)
	}

	// Create sources map
	sourcesMap := map[string]sources.Source{
		"test-clickhouse": source,
	}

	// Initialize describe_table tool
	toolConfig := clickhousetools.DescribeTableConfig{
		Name:        "describe_table",
		Kind:        "clickhouse-describe-table",
		Source:      "test-clickhouse",
		Description: "Test describe table tool",
	}

	tool, err := toolConfig.Initialize(sourcesMap)
	if err != nil {
		t.Fatalf("Failed to initialize describe_table tool: %v", err)
	}

	// Test with non-existent table
	params := tools.ParamValues{
		{Name: "database", Value: "system"},
		{Name: "table", Value: "nonexistent_table_12345"},
	}

	_, err = tool.Invoke(ctx, params)
	if err == nil {
		t.Fatal("Expected error for non-existent table")
	}

	// Verify the error contains information about the missing table
	if !contains(err.Error(), "nonexistent_table_12345") {
		t.Errorf("Expected error to mention the table name, got: %v", err)
	}

	t.Logf("✅ Successfully handled non-existent table error: %v", err)
}

// Mock source for testing without actual ClickHouse connection
type mockClickHouseSource struct {
	config clickhouse.Config
}

func (m *mockClickHouseSource) SourceKind() string {
	return "clickhouse"
}

func (m *mockClickHouseSource) ClickHousePool() *sql.DB {
	// Return nil for mock - real connection not needed for basic tests
	return nil
}

// Ensure mockClickHouseSource implements the sources.Source interface
var _ sources.Source = &mockClickHouseSource{}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr || 
		   (len(s) > len(substr) && contains(s[1:], substr))
}