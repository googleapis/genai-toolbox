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
	"github.com/google/go-cmp/cmp"
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

	tests := []struct {
		name    string
		yaml    string
		want    SQLConfig
		wantErr bool
	}{
		{
			name: "valid config with parameters",
			yaml: `
name: test-clickhouse-tool
kind: clickhouse-sql
source: test-source
description: Test ClickHouse tool
statement: SELECT * FROM test_table WHERE id = $1
parameters:
  - name: id
    type: string
    description: Test ID
`,
			want: SQLConfig{
				Name:        "test-clickhouse-tool",
				Kind:        "clickhouse-sql",
				Source:      "test-source",
				Description: "Test ClickHouse tool",
				Statement:   "SELECT * FROM test_table WHERE id = $1",
				Parameters: tools.Parameters{
					tools.NewStringParameter("id", "Test ID"),
				},
			},
			wantErr: false,
		},
		{
			name: "valid config without parameters",
			yaml: `
name: simple-tool
kind: clickhouse-sql
source: ch-source
description: Simple query
statement: SELECT 1
`,
			want: SQLConfig{
				Name:        "simple-tool",
				Kind:        "clickhouse-sql",
				Source:      "ch-source",
				Description: "Simple query",
				Statement:   "SELECT 1",
				Parameters:  nil,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoder := yaml.NewDecoder(strings.NewReader(tt.yaml))
			got, err := newSQLConfig(ctx, tt.want.Name, decoder)
			
			if (err != nil) != tt.wantErr {
				t.Fatalf("newSQLConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
			
			if err != nil {
				return
			}

			gotConfig, ok := got.(SQLConfig)
			if !ok {
				t.Fatalf("Expected SQLConfig type, got %T", got)
			}

			if diff := cmp.Diff(tt.want, gotConfig); diff != "" {
				t.Errorf("newSQLConfig() mismatch (-want +got):\n%s", diff)
			}
		})
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

	clickhouseTool, ok := tool.(Tool)
	if !ok {
		t.Fatalf("Expected Tool type, got %T", tool)
	}

	if clickhouseTool.Name != "test-tool" {
		t.Errorf("Expected name 'test-tool', got %s", clickhouseTool.Name)
	}
}

func TestSQLConfigInitializeMissingSource(t *testing.T) {
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

// mockIncompatibleSource is a mock source that doesn't implement the compatibleSource interface
type mockIncompatibleSource struct{}

func (m *mockIncompatibleSource) SourceKind() string {
	return "mock"
}

func TestSQLConfigInitializeIncompatibleSource(t *testing.T) {
	config := SQLConfig{
		Name:        "test-tool",
		Kind:        sqlKind,
		Source:      "incompatible-source",
		Description: "Test tool",
		Statement:   "SELECT 1",
		Parameters:  tools.Parameters{},
	}

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

func TestToolManifest(t *testing.T) {
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

func TestToolMcpManifest(t *testing.T) {
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

func TestToolAuthorized(t *testing.T) {
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
