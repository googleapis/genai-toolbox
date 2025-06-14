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

package runtime

import (
	"context"
	"testing"
	"time"

	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"go.opentelemetry.io/otel/trace/noop"
)

// MockSource implements sources.Source for testing
type MockSource struct {
	kind string
	name string
}

func (m *MockSource) SourceKind() string {
	return m.kind
}

// MockConfigManager implements ConfigManager for testing
type MockConfigManager struct{}

func (m *MockConfigManager) SaveConfiguration(ctx context.Context, config DynamicConfig) error {
	return nil
}

func (m *MockConfigManager) LoadConfiguration(ctx context.Context) (DynamicConfig, error) {
	return DynamicConfig{}, nil
}

func (m *MockConfigManager) ValidateConfiguration(ctx context.Context, config DynamicConfig) error {
	return nil
}

func (m *MockConfigManager) ExportConfiguration(ctx context.Context, format string) ([]byte, error) {
	return []byte("{}"), nil
}

func (m *MockConfigManager) ImportConfiguration(ctx context.Context, data []byte, format string) error {
	return nil
}

// MockLogger implements a minimal logger for testing
type MockLogger struct{}

func (m *MockLogger) DebugContext(ctx context.Context, msg string, args ...any) {}
func (m *MockLogger) InfoContext(ctx context.Context, msg string, args ...any)  {}
func (m *MockLogger) WarnContext(ctx context.Context, msg string, args ...any)  {}
func (m *MockLogger) ErrorContext(ctx context.Context, msg string, args ...any) {}

func createTestManager() *DefaultManager {
	sources := map[string]sources.Source{
		"test-source": &MockSource{kind: "postgres", name: "test-source"},
	}
	
	return NewDefaultManager(
		sources,
		&MockConfigManager{},
		noop.NewTracerProvider().Tracer("test"),
		&MockLogger{},
	)
}

func TestDefaultManager_CreateDynamicTool(t *testing.T) {
	manager := createTestManager()
	defer manager.Stop()
	
	ctx := context.Background()
	
	tests := []struct {
		name    string
		spec    DynamicToolSpec
		wantErr bool
	}{
		{
			name: "valid tool creation",
			spec: DynamicToolSpec{
				Name:        "test-tool",
				Description: "A test tool",
				SourceID:    "test-source",
				Query:       "SELECT * FROM users WHERE id = $1",
				Parameters: []ParameterSpec{
					{
						Name:        "user_id",
						Type:        "integer",
						Description: "User ID to fetch",
						Required:    true,
					},
				},
				Timeout: 30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "missing name",
			spec: DynamicToolSpec{
				Description: "A test tool",
				SourceID:    "test-source",
				Query:       "SELECT * FROM users",
			},
			wantErr: true,
		},
		{
			name: "missing source",
			spec: DynamicToolSpec{
				Name:        "test-tool",
				Description: "A test tool",
				SourceID:    "non-existent-source",
				Query:       "SELECT * FROM users",
			},
			wantErr: true,
		},
		{
			name: "missing query",
			spec: DynamicToolSpec{
				Name:        "test-tool",
				Description: "A test tool",
				SourceID:    "test-source",
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool, err := manager.CreateDynamicTool(ctx, tt.spec)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("CreateDynamicTool() expected error, got nil")
				}
				return
			}
			
			if err != nil {
				t.Errorf("CreateDynamicTool() unexpected error: %v", err)
				return
			}
			
			if tool == nil {
				t.Errorf("CreateDynamicTool() returned nil tool")
				return
			}
			
			// Verify tool was stored
			retrievedTool, err := manager.GetDynamicTool(ctx, tt.spec.Name)
			if err != nil {
				t.Errorf("GetDynamicTool() error: %v", err)
				return
			}
			
			if retrievedTool == nil {
				t.Errorf("GetDynamicTool() returned nil")
			}
		})
	}
}

func TestDefaultManager_ExecuteArbitrarySQL(t *testing.T) {
	manager := createTestManager()
	defer manager.Stop()
	
	ctx := context.Background()
	
	tests := []struct {
		name    string
		req     ArbitrarySQLRequest
		wantErr bool
	}{
		{
			name: "valid SQL execution",
			req: ArbitrarySQLRequest{
				SourceID:   "test-source",
				Query:      "SELECT 1",
				Parameters: map[string]interface{}{},
				Timeout:    30 * time.Second,
				MaxRows:    1000,
				DryRun:     false,
			},
			wantErr: false,
		},
		{
			name: "dry run execution",
			req: ArbitrarySQLRequest{
				SourceID:   "test-source",
				Query:      "SELECT * FROM users WHERE active = $1",
				Parameters: map[string]interface{}{"active": true},
				Timeout:    30 * time.Second,
				MaxRows:    100,
				DryRun:     true,
			},
			wantErr: false,
		},
		{
			name: "missing source",
			req: ArbitrarySQLRequest{
				SourceID: "non-existent-source",
				Query:    "SELECT 1",
			},
			wantErr: true,
		},
		{
			name: "missing query",
			req: ArbitrarySQLRequest{
				SourceID: "test-source",
			},
			wantErr: true,
		},
		{
			name: "too many rows",
			req: ArbitrarySQLRequest{
				SourceID: "test-source",
				Query:    "SELECT 1",
				MaxRows:  20000,
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := manager.ExecuteArbitrarySQL(ctx, tt.req)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("ExecuteArbitrarySQL() expected error, got nil")
				}
				return
			}
			
			if err != nil {
				t.Errorf("ExecuteArbitrarySQL() unexpected error: %v", err)
				return
			}
			
			if result == nil {
				t.Errorf("ExecuteArbitrarySQL() returned nil result")
				return
			}
			
			if len(result) == 0 {
				t.Errorf("ExecuteArbitrarySQL() returned empty result")
			}
		})
	}
}

func TestDefaultManager_ListDynamicTools(t *testing.T) {
	manager := createTestManager()
	defer manager.Stop()
	
	ctx := context.Background()
	
	// Initially should be empty
	tools, err := manager.ListDynamicTools(ctx)
	if err != nil {
		t.Errorf("ListDynamicTools() error: %v", err)
		return
	}
	
	if len(tools) != 0 {
		t.Errorf("ListDynamicTools() expected 0 tools, got %d", len(tools))
	}
	
	// Create a tool
	spec := DynamicToolSpec{
		Name:        "test-tool-1",
		Description: "First test tool",
		SourceID:    "test-source",
		Query:       "SELECT 1",
	}
	
	_, err = manager.CreateDynamicTool(ctx, spec)
	if err != nil {
		t.Errorf("CreateDynamicTool() error: %v", err)
		return
	}
	
	// Should now have one tool
	tools, err = manager.ListDynamicTools(ctx)
	if err != nil {
		t.Errorf("ListDynamicTools() error: %v", err)
		return
	}
	
	if len(tools) != 1 {
		t.Errorf("ListDynamicTools() expected 1 tool, got %d", len(tools))
		return
	}
	
	if tools[0].Name != "test-tool-1" {
		t.Errorf("ListDynamicTools() expected tool name 'test-tool-1', got %s", tools[0].Name)
	}
}

func TestDefaultManager_RemoveDynamicTool(t *testing.T) {
	manager := createTestManager()
	defer manager.Stop()
	
	ctx := context.Background()
	
	// Try to remove non-existent tool
	err := manager.RemoveDynamicTool(ctx, "non-existent")
	if err == nil {
		t.Errorf("RemoveDynamicTool() expected error for non-existent tool")
		return
	}
	
	// Create a tool
	spec := DynamicToolSpec{
		Name:        "test-tool-remove",
		Description: "Tool to be removed",
		SourceID:    "test-source",
		Query:       "SELECT 1",
	}
	
	_, err = manager.CreateDynamicTool(ctx, spec)
	if err != nil {
		t.Errorf("CreateDynamicTool() error: %v", err)
		return
	}
	
	// Verify tool exists
	_, err = manager.GetDynamicTool(ctx, "test-tool-remove")
	if err != nil {
		t.Errorf("GetDynamicTool() error before removal: %v", err)
		return
	}
	
	// Wait a moment for reference count to decrease (in a real implementation)
	// For now, we'll manually set the reference count to 0
	if dynamicTool, exists := manager.dynamicTools["test-tool-remove"]; exists {
		dynamicTool.mu.Lock()
		dynamicTool.refCount = 0
		dynamicTool.mu.Unlock()
	}
	
	// Remove the tool
	err = manager.RemoveDynamicTool(ctx, "test-tool-remove")
	if err != nil {
		t.Errorf("RemoveDynamicTool() error: %v", err)
		return
	}
	
	// Verify tool is gone
	_, err = manager.GetDynamicTool(ctx, "test-tool-remove")
	if err == nil {
		t.Errorf("GetDynamicTool() expected error after removal")
	}
}

func TestDefaultManager_Cleanup(t *testing.T) {
	manager := createTestManager()
	defer manager.Stop()
	
	ctx := context.Background()
	
	// Create a tool
	spec := DynamicToolSpec{
		Name:        "test-tool-cleanup",
		Description: "Tool for cleanup test",
		SourceID:    "test-source",
		Query:       "SELECT 1",
	}
	
	_, err := manager.CreateDynamicTool(ctx, spec)
	if err != nil {
		t.Errorf("CreateDynamicTool() error: %v", err)
		return
	}
	
	// Verify tool exists
	tools, err := manager.ListDynamicTools(ctx)
	if err != nil {
		t.Errorf("ListDynamicTools() error: %v", err)
		return
	}
	
	if len(tools) != 1 {
		t.Errorf("Expected 1 tool before cleanup, got %d", len(tools))
		return
	}
	
	// Run cleanup (should not remove recently created tool)
	err = manager.Cleanup(ctx)
	if err != nil {
		t.Errorf("Cleanup() error: %v", err)
		return
	}
	
	// Tool should still exist
	tools, err = manager.ListDynamicTools(ctx)
	if err != nil {
		t.Errorf("ListDynamicTools() error after cleanup: %v", err)
		return
	}
	
	if len(tools) != 1 {
		t.Errorf("Expected 1 tool after cleanup, got %d", len(tools))
	}
}

func TestHybridToolRegistry(t *testing.T) {
	// Create manager
	manager := createTestManager()
	defer manager.Stop()
	
	// Create registry
	registry := NewHybridToolRegistry(
		make(map[string]tools.Tool), // Empty static tools map
		manager,
		noop.NewTracerProvider().Tracer("test"),
	)
	
	// Test that registry is created
	if registry == nil {
		t.Errorf("NewHybridToolRegistry() returned nil")
	}
	
	// Test listing tools (should be empty initially)
	toolsList := registry.ListTools()
	if len(toolsList) != 0 {
		t.Errorf("ListTools() expected 0 tools, got %d", len(toolsList))
	}
	
	// Test cleanup of unused tools
	removed := registry.CleanupUnusedTools(1 * time.Hour)
	if removed != 0 {
		t.Errorf("CleanupUnusedTools() expected 0 removed, got %d", removed)
	}
}

// MockTool implements a basic tool for testing
type MockTool struct {
	name string
}

func TestDynamicToolBuilder(t *testing.T) {
	builder := NewDynamicToolBuilder()
	
	spec, err := builder.
		WithName("test-tool").
		WithDescription("Test tool description").
		WithSourceID("test-source").
		WithQuery("SELECT * FROM users WHERE id = $1").
		WithParameter(ParameterSpec{
			Name:        "user_id",
			Type:        "integer",
			Description: "User ID",
			Required:    true,
		}).
		WithTimeout(45 * time.Second).
		WithTags([]string{"test", "users"}).
		Build()
	
	if err != nil {
		t.Errorf("Build() error: %v", err)
		return
	}
	
	if spec.Name != "test-tool" {
		t.Errorf("Expected name 'test-tool', got %s", spec.Name)
	}
	
	if spec.Timeout != 45*time.Second {
		t.Errorf("Expected timeout 45s, got %v", spec.Timeout)
	}
	
	if len(spec.Parameters) != 1 {
		t.Errorf("Expected 1 parameter, got %d", len(spec.Parameters))
	}
	
	if len(spec.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(spec.Tags))
	}
}

func TestDynamicToolBuilder_ValidationErrors(t *testing.T) {
	builder := NewDynamicToolBuilder()
	
	// Try to build with missing required fields
	_, err := builder.Build()
	if err == nil {
		t.Errorf("Build() expected validation error")
		return
	}
	
	// Try with empty name
	builder = NewDynamicToolBuilder()
	_, err = builder.
		WithName(""). // Empty name should cause error
		WithDescription("test").
		WithSourceID("test").
		WithQuery("SELECT 1").
		Build()
	
	if err == nil {
		t.Errorf("Build() expected name validation error")
	}
	
	// Try with invalid timeout
	builder = NewDynamicToolBuilder()
	_, err = builder.
		WithName("test").
		WithDescription("test").
		WithSourceID("test").
		WithQuery("SELECT 1").
		WithTimeout(10 * time.Minute). // Too long
		Build()
	
	if err == nil {
		t.Errorf("Build() expected timeout validation error")
	}
}