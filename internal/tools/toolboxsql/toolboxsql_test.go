// Copyright 2024 Google LLC
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

package toolboxsql

import (
	"context"
	"testing"

	"github.com/googleapis/genai-toolbox/internal/runtime"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

// MockSource implements sources.Source for testing
type MockSource struct {
	kind string
}

func (m *MockSource) SourceKind() string {
	return m.kind
}

// MockDynamicToolManager implements runtime.DynamicToolManager for testing
type MockDynamicToolManager struct {
	executeResult []any
	executeError  error
}

func (m *MockDynamicToolManager) CreateDynamicTool(ctx context.Context, spec runtime.DynamicToolSpec) (tools.Tool, error) {
	return nil, nil
}

func (m *MockDynamicToolManager) ExecuteArbitrarySQL(ctx context.Context, req runtime.ArbitrarySQLRequest) ([]any, error) {
	return m.executeResult, m.executeError
}

func (m *MockDynamicToolManager) ListDynamicTools(ctx context.Context) ([]runtime.ToolManifest, error) {
	return nil, nil
}

func (m *MockDynamicToolManager) RemoveDynamicTool(ctx context.Context, toolID string) error {
	return nil
}

func (m *MockDynamicToolManager) GetDynamicTool(ctx context.Context, toolID string) (tools.Tool, error) {
	return nil, nil
}

func (m *MockDynamicToolManager) Cleanup(ctx context.Context) error {
	return nil
}

func createTestTool() (*Tool, *MockDynamicToolManager) {
	sources := map[string]sources.Source{
		"test-source": &MockSource{kind: "postgres"},
	}
	
	manager := &MockDynamicToolManager{
		executeResult: []any{
			map[string]any{
				"id":   1,
				"name": "test",
			},
		},
	}
	
	config := Config{
		Name:        "test-arbitrary-sql",
		Kind:        kind,
		Description: "Test arbitrary SQL execution",
		Manager:     manager,
	}
	
	tool, _ := config.Initialize(sources)
	toolImpl := tool.(*Tool)
	
	return toolImpl, manager
}

func TestTool_Invoke_Success(t *testing.T) {
	tool, _ := createTestTool()
	ctx := context.Background()
	
	// Test successful execution
	params := tools.ParamValues{
		{Name: "sourceId", Value: "test-source"},
		{Name: "query", Value: "SELECT * FROM users"},
		{Name: "timeout", Value: 30},
		{Name: "maxRows", Value: 100},
		{Name: "dryRun", Value: false},
	}
	
	result, err := tool.Invoke(ctx, params)
	if err != nil {
		t.Errorf("Invoke() error: %v", err)
		return
	}
	
	if len(result) != 1 {
		t.Errorf("Expected 1 result, got %d", len(result))
		return
	}
	
	resultMap, ok := result[0].(map[string]any)
	if !ok {
		t.Errorf("Expected result to be map[string]any")
		return
	}
	
	if resultMap["id"] != 1 {
		t.Errorf("Expected id=1, got %v", resultMap["id"])
	}
}

func TestTool_Invoke_DryRun(t *testing.T) {
	tool, manager := createTestTool()
	ctx := context.Background()
	
	// Set up mock to return dry run result
	manager.executeResult = []any{
		map[string]any{
			"status":  "valid",
			"message": "Query syntax is valid",
		},
	}
	
	params := tools.ParamValues{
		{Name: "sourceId", Value: "test-source"},
		{Name: "query", Value: "SELECT * FROM users WHERE id = $1"},
		{Name: "dryRun", Value: true},
	}
	
	result, err := tool.Invoke(ctx, params)
	if err != nil {
		t.Errorf("Invoke() error: %v", err)
		return
	}
	
	if len(result) != 1 {
		t.Errorf("Expected 1 result, got %d", len(result))
		return
	}
	
	resultMap, ok := result[0].(map[string]any)
	if !ok {
		t.Errorf("Expected result to be map[string]any")
		return
	}
	
	if resultMap["status"] != "valid" {
		t.Errorf("Expected status=valid, got %v", resultMap["status"])
	}
}

func TestTool_Invoke_ValidationErrors(t *testing.T) {
	tool, _ := createTestTool()
	ctx := context.Background()
	
	tests := []struct {
		name   string
		params map[string]any
	}{
		{
			name: "missing sourceId",
			params: map[string]any{
				"query": "SELECT 1",
			},
		},
		{
			name: "missing query",
			params: map[string]any{
				"sourceId": "test-source",
			},
		},
		{
			name: "invalid timeout",
			params: map[string]any{
				"sourceId": "test-source",
				"query":    "SELECT 1",
				"timeout":  500, // Too high
			},
		},
		{
			name: "invalid maxRows",
			params: map[string]any{
				"sourceId": "test-source",
				"query":    "SELECT 1",
				"maxRows":  20000, // Too high
			},
		},
		{
			name: "non-existent source",
			params: map[string]any{
				"sourceId": "non-existent",
				"query":    "SELECT 1",
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := tools.ParamValues{}
			for k, v := range tt.params {
				params = append(params, tools.ParamValue{Name: k, Value: v})
			}
			
			_, err := tool.Invoke(ctx, params)
			if err == nil {
				t.Errorf("Invoke() expected error for %s", tt.name)
			}
		})
	}
}

func TestTool_WithQueryParameters(t *testing.T) {
	tool, _ := createTestTool()
	ctx := context.Background()
	
	params := tools.ParamValues{
		{Name: "sourceId", Value: "test-source"},
		{Name: "query", Value: "SELECT * FROM users WHERE id = $1 AND active = $2"},
		{Name: "parameters", Value: map[string]interface{}{
			"id":     123,
			"active": true,
		}},
	}
	
	result, err := tool.Invoke(ctx, params)
	if err != nil {
		t.Errorf("Invoke() error with parameters: %v", err)
		return
	}
	
	if len(result) == 0 {
		t.Errorf("Expected non-empty result")
	}
}

func TestTool_Manifest(t *testing.T) {
	tool, _ := createTestTool()
	
	manifest := tool.Manifest()
	
	if manifest.Description == "" {
		t.Errorf("Expected non-empty description")
	}
	
	if len(manifest.Parameters) == 0 {
		t.Errorf("Expected parameters in manifest")
	}
	
	// Check that required parameters are present
	paramNames := make(map[string]bool)
	for _, param := range manifest.Parameters {
		paramNames[param.Name] = true
	}
	
	requiredParams := []string{"sourceId", "query", "parameters", "timeout", "maxRows", "dryRun"}
	for _, required := range requiredParams {
		if !paramNames[required] {
			t.Errorf("Expected parameter %s in manifest", required)
		}
	}
}

func TestTool_McpManifest(t *testing.T) {
	tool, _ := createTestTool()
	
	mcpManifest := tool.McpManifest()
	
	if mcpManifest.Name == "" {
		t.Errorf("Expected non-empty name in MCP manifest")
	}
	
	if mcpManifest.Description == "" {
		t.Errorf("Expected non-empty description in MCP manifest")
	}
	
	if mcpManifest.InputSchema.Type != "object" {
		t.Errorf("Expected input schema type to be 'object'")
	}
	
	// Check that required parameters are in the schema
	properties := mcpManifest.InputSchema.Properties
	if properties == nil {
		t.Errorf("Expected properties in input schema")
		return
	}
	
	if _, exists := properties["sourceId"]; !exists {
		t.Errorf("Expected sourceId in input schema properties")
	}
	
	if _, exists := properties["query"]; !exists {
		t.Errorf("Expected query in input schema properties")
	}
}

func TestTool_Authorized(t *testing.T) {
	tool, _ := createTestTool()
	
	// Tool has no auth requirements by default
	if !tool.Authorized([]string{}) {
		t.Errorf("Expected tool to be authorized with no auth requirements")
	}
	
	if !tool.Authorized([]string{"google"}) {
		t.Errorf("Expected tool to be authorized regardless of provided services when no auth required")
	}
}

func TestTool_SetManager(t *testing.T) {
	tool, _ := createTestTool()
	
	newManager := &MockDynamicToolManager{}
	tool.SetManager(newManager)
	
	if tool.Manager != newManager {
		t.Errorf("SetManager() did not set the manager correctly")
	}
}

func TestConfig_Initialize(t *testing.T) {
	sources := map[string]sources.Source{
		"test-source": &MockSource{kind: "postgres"},
	}
	
	config := Config{
		Name:        "test-tool",
		Kind:        kind,
		Description: "Test description",
	}
	
	tool, err := config.Initialize(sources)
	if err != nil {
		t.Errorf("Initialize() error: %v", err)
		return
	}
	
	if tool == nil {
		t.Errorf("Initialize() returned nil tool")
		return
	}
	
	toolImpl, ok := tool.(*Tool)
	if !ok {
		t.Errorf("Initialize() returned wrong tool type")
		return
	}
	
	if toolImpl.Name != "test-tool" {
		t.Errorf("Expected tool name 'test-tool', got %s", toolImpl.Name)
	}
	
	if toolImpl.Kind != kind {
		t.Errorf("Expected tool kind %s, got %s", kind, toolImpl.Kind)
	}
}