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

package clickhousedescribetable

import (
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/clickhouse"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

func TestConfig_ToolConfigKind(t *testing.T) {
	config := Config{}
	if config.ToolConfigKind() != kind {
		t.Errorf("Expected %s, got %s", kind, config.ToolConfigKind())
	}
}

func TestNewConfig(t *testing.T) {
	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("Failed to create context with logger: %v", err)
	}

	yamlContent := `
name: test-clickhouse-describe-table
kind: clickhouse-describe-table
source: test-source
description: Test ClickHouse describe table tool
`

	decoder := yaml.NewDecoder(strings.NewReader(yamlContent))
	config, err := newConfig(ctx, "test-clickhouse-describe-table", decoder)
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	clickhouseConfig, ok := config.(Config)
	if !ok {
		t.Fatalf("Expected Config type, got %T", config)
	}

	if clickhouseConfig.Name != "test-clickhouse-describe-table" {
		t.Errorf("Expected name 'test-clickhouse-describe-table', got %s", clickhouseConfig.Name)
	}
	if clickhouseConfig.Source != "test-source" {
		t.Errorf("Expected source 'test-source', got %s", clickhouseConfig.Source)
	}
	if clickhouseConfig.Description != "Test ClickHouse describe table tool" {
		t.Errorf("Expected description 'Test ClickHouse describe table tool', got %s", clickhouseConfig.Description)
	}
}

func TestConfig_Initialize_ValidSource(t *testing.T) {
	config := Config{
		Name:        "test-tool",
		Kind:        kind,
		Source:      "test-clickhouse",
		Description: "Test tool",
	}

	// Create a mock ClickHouse source
	mockSource := &clickhouse.Source{}

	sources := map[string]sources.Source{
		"test-clickhouse": mockSource,
	}

	tool, err := config.Initialize(sources)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	clickhouseTool, ok := tool.(Tool)
	if !ok {
		t.Fatalf("Expected Tool type, got %T", tool)
	}

	if clickhouseTool.Name != "test-tool" {
		t.Errorf("Expected name 'test-tool', got %s", clickhouseTool.Name)
	}

	// Verify parameters are correctly set
	if len(clickhouseTool.Parameters) != 1 {
		t.Errorf("Expected 1 parameter, got %d", len(clickhouseTool.Parameters))
	}

	// Check parameter names
	paramNames := make([]string, 0, len(clickhouseTool.Parameters))
	for _, param := range clickhouseTool.Parameters {
		paramNames = append(paramNames, param.GetName())
	}

	expectedParamNames := []string{"table_name"}
	for i, expected := range expectedParamNames {
		if i >= len(paramNames) || paramNames[i] != expected {
			t.Errorf("Expected parameter %d to be %s, got %v", i, expected, paramNames)
		}
	}
}

func TestConfig_Initialize_MissingSource(t *testing.T) {
	config := Config{
		Name:        "test-tool",
		Kind:        kind,
		Source:      "missing-source",
		Description: "Test tool",
	}

	sources := map[string]sources.Source{}

	_, err := config.Initialize(sources)
	if err == nil {
		t.Fatal("Expected error for missing source, got nil")
	}

	expectedErr := `no source named "missing-source" configured`
	if err.Error() != expectedErr {
		t.Errorf("Expected error %q, got %q", expectedErr, err.Error())
	}
}

func TestConfig_Initialize_IncompatibleSource(t *testing.T) {
	config := Config{
		Name:        "test-tool",
		Kind:        kind,
		Source:      "incompatible-source",
		Description: "Test tool",
	}

	// Create a mock incompatible source
	mockSource := &mockIncompatibleSource{}

	sources := map[string]sources.Source{
		"incompatible-source": mockSource,
	}

	_, err := config.Initialize(sources)
	if err == nil {
		t.Fatal("Expected error for incompatible source, got nil")
	}

	if err.Error() == "" {
		t.Error("Expected non-empty error message")
	}
}

func TestTool_Manifest(t *testing.T) {
	tool := Tool{
		manifest: tools.Manifest{
			Description: "Test description",
			Parameters:  []tools.ParameterManifest{},
		},
	}

	manifest := tool.Manifest()
	if manifest.Description != "Test description" {
		t.Errorf("Expected description 'Test description', got %s", manifest.Description)
	}
}

func TestTool_McpManifest(t *testing.T) {
	tool := Tool{
		mcpManifest: tools.McpManifest{
			Name:        "test-tool",
			Description: "Test description",
		},
	}

	manifest := tool.McpManifest()
	if manifest.Name != "test-tool" {
		t.Errorf("Expected name 'test-tool', got %s", manifest.Name)
	}
	if manifest.Description != "Test description" {
		t.Errorf("Expected description 'Test description', got %s", manifest.Description)
	}
}

func TestTool_Authorized(t *testing.T) {
	tests := []struct {
		name                 string
		authRequired         []string
		verifiedAuthServices []string
		expectedAuthorized   bool
	}{
		{
			name:                 "no auth required",
			authRequired:         []string{},
			verifiedAuthServices: []string{},
			expectedAuthorized:   true,
		},
		{
			name:                 "auth required and verified",
			authRequired:         []string{"google"},
			verifiedAuthServices: []string{"google"},
			expectedAuthorized:   true,
		},
		{
			name:                 "auth required but not verified",
			authRequired:         []string{"google"},
			verifiedAuthServices: []string{},
			expectedAuthorized:   false,
		},
		{
			name:                 "auth required but different service verified",
			authRequired:         []string{"google"},
			verifiedAuthServices: []string{"aws"},
			expectedAuthorized:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool := Tool{
				AuthRequired: tt.authRequired,
			}

			authorized := tool.Authorized(tt.verifiedAuthServices)
			if authorized != tt.expectedAuthorized {
				t.Errorf("Expected authorized %t, got %t", tt.expectedAuthorized, authorized)
			}
		})
	}
}

// Mock incompatible source for testing
type mockIncompatibleSource struct{}

func (m *mockIncompatibleSource) SourceKind() string {
	return "incompatible"
}

// This source doesn't implement ClickHousePool() method, making it incompatible