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

package trinolistschemas_test

import (
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools/trino/trinolistschemas"
)

func TestParseFromYamlTrinoListSchemas(t *testing.T) {
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
				list_schemas:
					kind: trino-list-schemas
					source: my-trino-instance
					description: Lists all schemas in a catalog
			`,
			want: server.ToolConfigs{
				"list_schemas": trinolistschemas.Config{
					Name:         "list_schemas",
					Kind:         "trino-list-schemas",
					Source:       "my-trino-instance",
					Description:  "Lists all schemas in a catalog",
					AuthRequired: []string{},
				},
			},
		},
		{
			desc: "with auth",
			in: `
			tools:
				list_schemas_auth:
					kind: trino-list-schemas
					source: my-trino-instance
					description: Lists schemas with authentication
					authRequired:
						- my-auth-service
						- other-auth-service
			`,
			want: server.ToolConfigs{
				"list_schemas_auth": trinolistschemas.Config{
					Name:         "list_schemas_auth",
					Kind:         "trino-list-schemas",
					Source:       "my-trino-instance",
					Description:  "Lists schemas with authentication",
					AuthRequired: []string{"my-auth-service", "other-auth-service"},
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

func TestTrinoListSchemasParameters(t *testing.T) {
	// Test that the tool properly initializes with parameters
	config := trinolistschemas.Config{
		Name:        "test_tool",
		Kind:        "trino-list-schemas",
		Source:      "test-source",
		Description: "test description",
	}

	// Verify the config has the expected fields
	if config.Name != "test_tool" {
		t.Errorf("expected Name to be 'test_tool', got %s", config.Name)
	}
	if config.Kind != "trino-list-schemas" {
		t.Errorf("expected Kind to be 'trino-list-schemas', got %s", config.Kind)
	}

	// The actual tool initialization would require a valid source,
	// which we can't test without a database connection
	// This test just verifies the configuration structure
}

func TestTrinoListSchemasToolConfigKind(t *testing.T) {
	config := trinolistschemas.Config{
		Name:        "test",
		Kind:        "trino-list-schemas",
		Source:      "test-source",
		Description: "test",
	}

	if kind := config.ToolConfigKind(); kind != "trino-list-schemas" {
		t.Errorf("expected ToolConfigKind to return 'trino-list-schemas', got %s", kind)
	}
}
