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
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/clickhouse"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

func TestConfigToolConfigKind(t *testing.T) {
	config := Config{}
	if config.ToolConfigKind() != sqlKind {
		t.Errorf("Expected %s, got %s", sqlKind, config.ToolConfigKind())
	}
}

func TestParseFromYamlClickHouseSQL(t *testing.T) {
	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	tcs := []struct {
		desc string
		in   string
		want server.ToolConfigs
	}{
		{
			desc: "basic example",
			in: `
			tools:
				example_tool:
					kind: clickhouse-sql
					source: my-instance
					description: some description
					statement: SELECT 1
			`,
			want: server.ToolConfigs{
				"example_tool": Config{
					Name:         "example_tool",
					Kind:         "clickhouse-sql",
					Source:       "my-instance",
					Description:  "some description",
					Statement:    "SELECT 1",
					AuthRequired: []string{},
				},
			},
		},
		{
			desc: "with parameters",
			in: `
			tools:
				param_tool:
					kind: clickhouse-sql
					source: test-source
					description: Test ClickHouse tool
					statement: SELECT * FROM test_table WHERE id = $1
					parameters:
					  - name: id
					    type: string
					    description: Test ID
			`,
			want: server.ToolConfigs{
				"param_tool": Config{
					Name:        "param_tool",
					Kind:        "clickhouse-sql",
					Source:      "test-source",
					Description: "Test ClickHouse tool",
					Statement:   "SELECT * FROM test_table WHERE id = $1",
					Parameters: tools.Parameters{
						tools.NewStringParameter("id", "Test ID"),
					},
					AuthRequired: []string{},
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := struct {
				Tools server.ToolConfigs `yaml:"tools"`
			}{}
			err := yaml.UnmarshalContext(ctx, testutils.FormatYaml(tc.in), &got)
			if err != nil {
				t.Fatalf("unable to unmarshal: %s", err)
			}
			if diff := cmp.Diff(tc.want, got.Tools); diff != "" {
				t.Fatalf("incorrect parse: diff %v", diff)
			}
		})
	}
}

func TestSQLConfigInitializeValidSource(t *testing.T) {
	config := Config{
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
	config := Config{
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
	config := Config{
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
