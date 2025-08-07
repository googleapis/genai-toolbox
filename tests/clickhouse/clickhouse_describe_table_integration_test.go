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
	mockSourceConfig := clickhouse.Config{
		Name:     "test-clickhouse",
		Kind:     "clickhouse",
		Host:     "localhost",
		Port:     "9000",
		User:     "default",
		Password: "",
		Database: "system",
		Protocol: "https",
		Secure:   false,
	}

	toolConfig := clickhousetools.DescribeTableConfig{
		Name:        "describe_table",
		Kind:        "clickhouse-describe-table",
		Source:      "test-clickhouse",
		Description: "Test describe table tool",
	}

	if toolConfig.ToolConfigKind() != "clickhouse-describe-table" {
		t.Errorf("Expected tool kind 'clickhouse-describe-table', got %s", toolConfig.ToolConfigKind())
	}

	expectedParams := []string{"table_name", "database_name"}
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

	sourceConfig := getClickHouseVars(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	chConfig := clickhouse.Config{
		Name:     "test-clickhouse",
		Kind:     "clickhouse",
		Host:     sourceConfig["host"].(string),
		Port:     sourceConfig["port"].(string),
		User:     sourceConfig["user"].(string),
		Password: sourceConfig["password"].(string),
		Database: "system",
		Protocol: sourceConfig["protocol"].(string),
		Secure:   sourceConfig["secure"].(bool),
	}

	tracer := noop.NewTracerProvider().Tracer("test")

	source, err := chConfig.Initialize(ctx, tracer)
	if err != nil {
		t.Skipf("Cannot connect to ClickHouse: %v", err)
	}

	sourcesMap := map[string]sources.Source{
		"test-clickhouse": source,
	}

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

	params := tools.ParamValues{
		{Name: "table_name", Value: "tables"},
		{Name: "database_name", Value: "system"},
	}

	result, err := tool.Invoke(ctx, params)
	if err != nil {
		t.Fatalf("Failed to describe system.tables: %v", err)
	}

	resultSlice, ok := result.([]any)
	if !ok {
		t.Fatalf("Expected result to be []any, got %T", result)
	}

	if len(resultSlice) == 0 {
		t.Fatal("Expected at least one result")
	}

	resultMap, ok := resultSlice[0].(map[string]any)
	if !ok {
		t.Fatalf("Expected result to be a map, got %T", resultSlice[0])
	}

	expectedFields := []string{"column_name", "data_type", "table_name", "table_engine"}
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

	sourceConfig := getClickHouseVars(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	chConfig := clickhouse.Config{
		Name:     "test-clickhouse",
		Kind:     "clickhouse",
		Host:     sourceConfig["host"].(string),
		Port:     sourceConfig["port"].(string),
		User:     sourceConfig["user"].(string),
		Password: sourceConfig["password"].(string),
		Database: sourceConfig["database"].(string),
		Protocol: sourceConfig["protocol"].(string),
		Secure:   sourceConfig["secure"].(bool),
	}

	tracer := noop.NewTracerProvider().Tracer("test")
	source, err := chConfig.Initialize(ctx, tracer)
	if err != nil {
		t.Skipf("Cannot connect to ClickHouse: %v", err)
	}

	sourcesMap := map[string]sources.Source{
		"test-clickhouse": source,
	}

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

	params := tools.ParamValues{
		{Name: "table_name", Value: "nonexistent_table_12345"},
		{Name: "database_name", Value: ""},
	}

	_, err = tool.Invoke(ctx, params)
	if err == nil {
		t.Fatal("Expected error for non-existent table")
	}

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
	// nil for mock - real connection not needed these tests
	return nil
}

// Ensure mockClickHouseSource implements the sources.Source interface
var _ sources.Source = &mockClickHouseSource{}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr ||
		(len(s) > len(substr) && contains(s[1:], substr))
}
