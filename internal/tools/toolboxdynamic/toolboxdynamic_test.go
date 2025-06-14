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

package toolboxdynamic

import (
	"context"
	"testing"
	"time"

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
	createResult tools.Tool
	createError  error
	lastSpec     runtime.DynamicToolSpec
}

func (m *MockDynamicToolManager) CreateDynamicTool(ctx context.Context, spec runtime.DynamicToolSpec) (tools.Tool, error) {
	m.lastSpec = spec
	return m.createResult, m.createError
}

func (m *MockDynamicToolManager) ExecuteArbitrarySQL(ctx context.Context, req runtime.ArbitrarySQLRequest) ([]any, error) {
	return nil, nil
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

// MockDynamicTool implements tools.Tool for testing
type MockDynamicTool struct {
	name     string
	manifest runtime.ToolManifest
}

func (m *MockDynamicTool) Invoke(ctx context.Context, params tools.ParamValues) ([]any, error) {
	return nil, nil
}

func (m *MockDynamicTool) ParseParams(data map[string]any, claims map[string]map[string]any) (tools.ParamValues, error) {
	return tools.ParamValues{}, nil
}

func (m *MockDynamicTool) Manifest() tools.Manifest {
	return tools.Manifest{}
}

func (m *MockDynamicTool) McpManifest() tools.McpManifest {
	return tools.McpManifest{}
}

func (m *MockDynamicTool) Authorized(verifiedAuthServices []string) bool {
	return true
}

func (m *MockDynamicTool) GetManifest() runtime.ToolManifest {
	return m.manifest
}

func createTestTool() (*Tool, *MockDynamicToolManager) {
	sources := map[string]sources.Source{
		"test-source": &MockSource{kind: "postgres"},
		"mysql-source": &MockSource{kind: "mysql"},
	}
	
	manifest := runtime.ToolManifest{
		ID:          "test-tool-id",
		Name:        "test-dynamic-tool",
		Description: "Test dynamic tool",
		SourceID:    "test-source",
		Kind:        "dynamic-sql",
		CreatedAt:   time.Now(),
		Status:      "active",
	}
	
	mockTool := &MockDynamicTool{
		name:     "test-dynamic-tool",
		manifest: manifest,
	}
	
	manager := &MockDynamicToolManager{
		createResult: mockTool,
	}
	
	config := Config{
		Name:        "test-create-dynamic-tool",
		Kind:        kind,
		Description: "Test dynamic tool creation",
		Manager:     manager,
	}
	
	tool, _ := config.Initialize(sources)
	toolImpl := tool.(*Tool)
	
	return toolImpl, manager
}

func TestTool_Invoke_Success(t *testing.T) {
	tool, manager := createTestTool()
	ctx := context.Background()
	
	params := tools.ParamValues{
		{Name: "name", Value: "user-lookup"},
		{Name: "description", Value: "Look up user by ID"},
		{Name: "sourceId", Value: "test-source"},
		{Name: "query", Value: "SELECT * FROM users WHERE id = $1"},
		{Name: "parameters", Value: []interface{}{
			map[string]interface{}{
				"name":        "user_id",
				"type":        "integer",
				"description": "User ID to lookup",
				"required":    true,
			},
		}},
		{Name: "timeout", Value: 30},
		{Name: "tags", Value: []interface{}{"users", "lookup"}},
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
	
	if resultMap["status"] != "success" {
		t.Errorf("Expected status=success, got %v", resultMap["status"])
	}
	
	if resultMap["toolName"] != "user-lookup" {
		t.Errorf("Expected toolName=user-lookup, got %v", resultMap["toolName"])
	}
	
	// Verify the spec was passed correctly to the manager
	if manager.lastSpec.Name != "user-lookup" {
		t.Errorf("Expected spec name=user-lookup, got %s", manager.lastSpec.Name)
	}
	
	if manager.lastSpec.SourceID != "test-source" {
		t.Errorf("Expected spec sourceId=test-source, got %s", manager.lastSpec.SourceID)
	}
	
	if len(manager.lastSpec.Parameters) != 1 {
		t.Errorf("Expected 1 parameter in spec, got %d", len(manager.lastSpec.Parameters))
	}
	
	if len(manager.lastSpec.Tags) != 2 {
		t.Errorf("Expected 2 tags in spec, got %d", len(manager.lastSpec.Tags))
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
			name: "missing name",
			params: map[string]any{
				"description": "Test tool",
				"sourceId":    "test-source",
				"query":       "SELECT 1",
			},
		},
		{
			name: "empty name",
			params: map[string]any{
				"name":        "",
				"description": "Test tool",
				"sourceId":    "test-source",
				"query":       "SELECT 1",
			},
		},
		{
			name: "missing description",
			params: map[string]any{
				"name":     "test-tool",
				"sourceId": "test-source",
				"query":    "SELECT 1",
			},
		},
		{
			name: "missing sourceId",
			params: map[string]any{
				"name":        "test-tool",
				"description": "Test tool",
				"query":       "SELECT 1",
			},
		},
		{
			name: "missing query",
			params: map[string]any{
				"name":        "test-tool",
				"description": "Test tool",
				"sourceId":    "test-source",
			},
		},
		{
			name: "non-existent source",
			params: map[string]any{
				"name":        "test-tool",
				"description": "Test tool",
				"sourceId":    "non-existent-source",
				"query":       "SELECT 1",
			},
		},
		{
			name: "invalid timeout",
			params: map[string]any{
				"name":        "test-tool",
				"description": "Test tool",
				"sourceId":    "test-source",
				"query":       "SELECT 1",
				"timeout":     500, // Too high
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

func TestTool_Invoke_WithAuth(t *testing.T) {
	tool, manager := createTestTool()
	ctx := context.Background()
	
	params := tools.ParamValues{
		{Name: "name", Value: "secure-tool"},
		{Name: "description", Value: "Tool with auth requirements"},
		{Name: "sourceId", Value: "test-source"},
		{Name: "query", Value: "SELECT * FROM sensitive_data"},
		{Name: "authRequired", Value: []interface{}{"google", "azure"}},
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
	
	// Verify auth requirements were set correctly
	if !manager.lastSpec.Auth.Required {
		t.Errorf("Expected auth to be required")
	}
	
	if len(manager.lastSpec.Auth.Services) != 2 {
		t.Errorf("Expected 2 auth services, got %d", len(manager.lastSpec.Auth.Services))
	}
	
	expectedServices := map[string]bool{"google": true, "azure": true}
	for _, service := range manager.lastSpec.Auth.Services {
		if !expectedServices[service] {
			t.Errorf("Unexpected auth service: %s", service)
		}
	}
}

func TestTool_Invoke_WithComplexParameters(t *testing.T) {
	tool, manager := createTestTool()
	ctx := context.Background()
	
	params := tools.ParamValues{
		{Name: "name", Value: "complex-tool"},
		{Name: "description", Value: "Tool with complex parameters"},
		{Name: "sourceId", Value: "test-source"},
		{Name: "query", Value: "SELECT * FROM users WHERE name = $1 AND age = $2 AND active = $3"},
		{Name: "parameters", Value: []interface{}{
			map[string]interface{}{
				"name":        "user_name",
				"type":        "string",
				"description": "User name to search",
				"required":    true,
			},
			map[string]interface{}{
				"name":        "age",
				"type":        "integer",
				"description": "User age",
				"required":    false,
				"default":     18,
			},
			map[string]interface{}{
				"name":        "active",
				"type":        "boolean",
				"description": "Whether user is active",
				"required":    true,
			},
		}},
		{Name: "metadata", Value: map[string]interface{}{
			"version": "1.0",
			"author":  "test",
		}},
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
	
	// Verify parameters were parsed correctly
	if len(manager.lastSpec.Parameters) != 3 {
		t.Errorf("Expected 3 parameters, got %d", len(manager.lastSpec.Parameters))
		return
	}
	
	// Check first parameter
	param1 := manager.lastSpec.Parameters[0]
	if param1.Name != "user_name" {
		t.Errorf("Expected parameter name 'user_name', got %s", param1.Name)
	}
	if param1.Type != "string" {
		t.Errorf("Expected parameter type 'string', got %s", param1.Type)
	}
	if !param1.Required {
		t.Errorf("Expected parameter to be required")
	}
	
	// Check second parameter with default
	param2 := manager.lastSpec.Parameters[1]
	if param2.Default != 18 {
		t.Errorf("Expected parameter default 18, got %v", param2.Default)
	}
	if param2.Required {
		t.Errorf("Expected parameter to not be required")
	}
	
	// Check metadata
	if len(manager.lastSpec.Metadata) != 2 {
		t.Errorf("Expected 2 metadata items, got %d", len(manager.lastSpec.Metadata))
	}
}

func TestTool_Invoke_InvalidParameterSpecification(t *testing.T) {
	tool, _ := createTestTool()
	ctx := context.Background()
	
	tests := []struct {
		name       string
		parameters interface{}
	}{
		{
			name: "parameter missing name",
			parameters: []interface{}{
				map[string]interface{}{
					"type":        "string",
					"description": "Parameter without name",
				},
			},
		},
		{
			name: "parameter missing type",
			parameters: []interface{}{
				map[string]interface{}{
					"name":        "param1",
					"description": "Parameter without type",
				},
			},
		},
		{
			name: "parameter not an object",
			parameters: []interface{}{
				"invalid-parameter-spec",
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := tools.ParamValues{
				{Name: "name", Value: "test-tool"},
				{Name: "description", Value: "Test tool"},
				{Name: "sourceId", Value: "test-source"},
				{Name: "query", Value: "SELECT 1"},
				{Name: "parameters", Value: tt.parameters},
			}
			
			_, err := tool.Invoke(ctx, params)
			if err == nil {
				t.Errorf("Invoke() expected error for %s", tt.name)
			}
		})
	}
}

func TestTool_Invoke_NoManager(t *testing.T) {
	tool, _ := createTestTool()
	tool.Manager = nil // Remove manager
	ctx := context.Background()
	
	params := tools.ParamValues{
		{Name: "name", Value: "test-tool"},
		{Name: "description", Value: "Test tool"},
		{Name: "sourceId", Value: "test-source"},
		{Name: "query", Value: "SELECT 1"},
	}
	
	_, err := tool.Invoke(ctx, params)
	if err == nil {
		t.Errorf("Invoke() expected error when manager is nil")
	}
	
	if err.Error() != "dynamic tool manager not available" {
		t.Errorf("Expected specific error message, got: %v", err)
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
	
	requiredParams := []string{"name", "description", "sourceId", "query", "parameters", "timeout", "tags", "authRequired", "metadata"}
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
	
	if _, exists := properties["name"]; !exists {
		t.Errorf("Expected name in input schema properties")
	}
	
	if _, exists := properties["description"]; !exists {
		t.Errorf("Expected description in input schema properties")
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

func TestConfig_Initialize_DefaultDescription(t *testing.T) {
	sources := map[string]sources.Source{
		"test-source": &MockSource{kind: "postgres"},
	}
	
	config := Config{
		Name: "test-tool",
		Kind: kind,
		// No description provided
	}
	
	tool, err := config.Initialize(sources)
	if err != nil {
		t.Errorf("Initialize() error: %v", err)
		return
	}
	
	toolImpl := tool.(*Tool)
	manifest := toolImpl.Manifest()
	
	if manifest.Description == "" {
		t.Errorf("Expected default description to be set")
	}
	
	expectedDesc := "Create new database tools at runtime with custom SQL queries and parameters"
	if manifest.Description != expectedDesc {
		t.Errorf("Expected default description, got: %s", manifest.Description)
	}
}

func TestParseParameterSpec_Success(t *testing.T) {
	paramMap := map[string]interface{}{
		"name":        "test_param",
		"type":        "string",
		"description": "Test parameter",
		"required":    true,
		"default":     "default_value",
		"validation":  map[string]interface{}{"min_length": 5},
	}
	
	spec, err := parseParameterSpec(paramMap)
	if err != nil {
		t.Errorf("parseParameterSpec() error: %v", err)
		return
	}
	
	if spec.Name != "test_param" {
		t.Errorf("Expected name 'test_param', got %s", spec.Name)
	}
	
	if spec.Type != "string" {
		t.Errorf("Expected type 'string', got %s", spec.Type)
	}
	
	if spec.Description != "Test parameter" {
		t.Errorf("Expected description 'Test parameter', got %s", spec.Description)
	}
	
	if !spec.Required {
		t.Errorf("Expected required to be true")
	}
	
	if spec.Default != "default_value" {
		t.Errorf("Expected default 'default_value', got %v", spec.Default)
	}
	
	if spec.Validation == nil {
		t.Errorf("Expected validation to be set")
	}
}

func TestParseParameterSpec_ValidationErrors(t *testing.T) {
	tests := []struct {
		name     string
		paramMap map[string]interface{}
	}{
		{
			name: "missing name",
			paramMap: map[string]interface{}{
				"type": "string",
			},
		},
		{
			name: "empty name",
			paramMap: map[string]interface{}{
				"name": "",
				"type": "string",
			},
		},
		{
			name: "missing type",
			paramMap: map[string]interface{}{
				"name": "test_param",
			},
		},
		{
			name: "empty type",
			paramMap: map[string]interface{}{
				"name": "test_param",
				"type": "",
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseParameterSpec(tt.paramMap)
			if err == nil {
				t.Errorf("parseParameterSpec() expected error for %s", tt.name)
			}
		})
	}
}

func TestParseParameterSpec_OptionalFields(t *testing.T) {
	paramMap := map[string]interface{}{
		"name": "test_param",
		"type": "string",
		// All other fields optional
	}
	
	spec, err := parseParameterSpec(paramMap)
	if err != nil {
		t.Errorf("parseParameterSpec() error: %v", err)
		return
	}
	
	if spec.Description != "" {
		t.Errorf("Expected empty description, got %s", spec.Description)
	}
	
	if spec.Required {
		t.Errorf("Expected required to be false by default")
	}
	
	if spec.Default != nil {
		t.Errorf("Expected default to be nil, got %v", spec.Default)
	}
	
	if spec.Validation != nil {
		t.Errorf("Expected validation to be nil, got %v", spec.Validation)
	}
}