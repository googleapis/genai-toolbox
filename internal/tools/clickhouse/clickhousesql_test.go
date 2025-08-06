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
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/clickhouse"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

func TestConfigToolConfigKind(t *testing.T) {
	config := SQLConfig{}
	if config.ToolConfigKind() != sqlKind {
		t.Errorf("Expected %s, got %s", sqlKind, config.ToolConfigKind())
	}
}

func TestNewSQLConfig(t *testing.T) {
	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("Failed to create context with logger: %v", err)
	}

	yamlContent := `
name: test-clickhouse-tool
kind: clickhouse-sql
source: test-source
description: Test ClickHouse tool
statement: SELECT * FROM test_table WHERE id = $1
parameters:
  - name: id
    type: string
    description: Test ID
`

	decoder := yaml.NewDecoder(strings.NewReader(yamlContent))
	config, err := newSQLConfig(ctx, "test-clickhouse-tool", decoder)
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	clickhouseConfig, ok := config.(SQLConfig)
	if !ok {
		t.Fatalf("Expected Config type, got %T", config)
	}

	if clickhouseConfig.Name != "test-clickhouse-tool" {
		t.Errorf("Expected name 'test-clickhouse-tool', got %s", clickhouseConfig.Name)
	}
	if clickhouseConfig.Source != "test-source" {
		t.Errorf("Expected source 'test-source', got %s", clickhouseConfig.Source)
	}
	if clickhouseConfig.Description != "Test ClickHouse tool" {
		t.Errorf("Expected description 'Test ClickHouse tool', got %s", clickhouseConfig.Description)
	}
	if clickhouseConfig.Statement != "SELECT * FROM test_table WHERE id = $1" {
		t.Errorf("Expected statement 'SELECT * FROM test_table WHERE id = $1', got %s", clickhouseConfig.Statement)
	}
}

func TestSQLConfigInitializeValidSource(t *testing.T) {
	config := SQLConfig{
		Name:        "test-tool",
		Kind:        sqlKind,
		Source:      "test-clickhouse",
		Description: "Test tool",
		Statement:   "SELECT 1",
		Parameters:  tools.Parameters{},
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

	clickhouseTool, ok := tool.(SQLTool)
	if !ok {
		t.Fatalf("Expected Tool type, got %T", tool)
	}

	if clickhouseTool.Name != "test-tool" {
		t.Errorf("Expected name 'test-tool', got %s", clickhouseTool.Name)
	}
}

func TestSQLConfig_Initialize_MissingSource(t *testing.T) {
	config := SQLConfig{
		Name:        "test-tool",
		Kind:        sqlKind,
		Source:      "missing-source",
		Description: "Test tool",
		Statement:   "SELECT 1",
		Parameters:  tools.Parameters{},
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

func TestSQLConfig_Initialize_IncompatibleSource(t *testing.T) {
	config := SQLConfig{
		Name:        "test-tool",
		Kind:        sqlKind,
		Source:      "incompatible-source",
		Description: "Test tool",
		Statement:   "SELECT 1",
		Parameters:  tools.Parameters{},
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

func TestSQLTool_Manifest(t *testing.T) {
	tool := SQLTool{
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

func TestSQLTool_McpManifest(t *testing.T) {
	tool := SQLTool{
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

func TestSQLTool_Authorized(t *testing.T) {
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
			tool := SQLTool{
				AuthRequired: tt.authRequired,
			}

			authorized := tool.Authorized(tt.verifiedAuthServices)
			if authorized != tt.expectedAuthorized {
				t.Errorf("Expected authorized %t, got %t", tt.expectedAuthorized, authorized)
			}
		})
	}
}
