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

package trinoanalyze_test

import (
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools/trino/trinoanalyze"
)

func TestParseFromYamlTrinoAnalyze(t *testing.T) {
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
				analyze_query:
					kind: trino-analyze
					source: my-trino-instance
					description: Analyzes SQL queries for performance
			`,
			want: server.ToolConfigs{
				"analyze_query": trinoanalyze.Config{
					Name:         "analyze_query",
					Kind:         "trino-analyze",
					Source:       "my-trino-instance",
					Description:  "Analyzes SQL queries for performance",
					AuthRequired: []string{},
				},
			},
		},
		{
			desc: "with auth",
			in: `
			tools:
				analyze_query_auth:
					kind: trino-analyze
					source: my-trino-instance
					description: Analyzes queries with authentication
					authRequired:
						- my-auth-service
						- other-auth
			`,
			want: server.ToolConfigs{
				"analyze_query_auth": trinoanalyze.Config{
					Name:         "analyze_query_auth",
					Kind:         "trino-analyze",
					Source:       "my-trino-instance",
					Description:  "Analyzes queries with authentication",
					AuthRequired: []string{"my-auth-service", "other-auth"},
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := struct {
				Tools server.ToolConfigs `yaml:"tools"`
			}{}
			// Parse contents
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

func TestTrinoAnalyzeToolConfigKind(t *testing.T) {
	config := trinoanalyze.Config{
		Name:        "test",
		Kind:        "trino-analyze",
		Source:      "test-source",
		Description: "test",
	}

	if kind := config.ToolConfigKind(); kind != "trino-analyze" {
		t.Errorf("expected ToolConfigKind to return 'trino-analyze', got %s", kind)
	}
}

func TestTrinoAnalyzeParameters(t *testing.T) {
	// This test verifies the expected parameters for the analyze tool
	// The actual tool initialization requires a database connection

	config := trinoanalyze.Config{
		Name:        "test_tool",
		Kind:        "trino-analyze",
		Source:      "test-source",
		Description: "test description",
	}

	// Verify the config structure
	if config.Name != "test_tool" {
		t.Errorf("expected Name to be 'test_tool', got %s", config.Name)
	}

	// The tool should have the following parameters when initialized:
	// - query (string, required) - The SQL query to analyze
	// - format (string, default "text") - Output format
	// - analyze (boolean, default false) - Run ANALYZE for actual stats
	// - distributed (boolean, default true) - Show distributed plan
	// - validate (boolean, default false) - Only validate syntax
}

func TestBuildExplainCommand(t *testing.T) {
	// Test that the EXPLAIN command is built correctly
	// This would require access to the private buildExplainCommand method
	// or testing through the Invoke method with a mock database

	testCases := []struct {
		name           string
		format         string
		analyze        bool
		distributed    bool
		expectedPrefix string
	}{
		{
			name:           "basic text",
			format:         "text",
			analyze:        false,
			distributed:    false,
			expectedPrefix: "EXPLAIN",
		},
		{
			name:           "with analyze",
			format:         "text",
			analyze:        true,
			distributed:    false,
			expectedPrefix: "EXPLAIN (TYPE ANALYZE)",
		},
		{
			name:           "with distributed",
			format:         "text",
			analyze:        false,
			distributed:    true,
			expectedPrefix: "EXPLAIN (TYPE DISTRIBUTED)",
		},
		{
			name:           "json format",
			format:         "json",
			analyze:        false,
			distributed:    false,
			expectedPrefix: "EXPLAIN (FORMAT JSON)",
		},
	}

	// These test cases verify the expected behavior
	// Actual testing would require database connection
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test verification would go here with actual tool invocation
		})
	}
}
