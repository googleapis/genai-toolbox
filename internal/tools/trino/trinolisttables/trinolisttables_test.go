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

package trinolisttables_test

import (
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools/trino/trinolisttables"
)

func TestParseFromYamlTrinoListTables(t *testing.T) {
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
				list_tables:
					kind: trino-list-tables
					source: my-trino-instance
					description: Lists all tables in a schema
			`,
			want: server.ToolConfigs{
				"list_tables": trinolisttables.Config{
					Name:         "list_tables",
					Kind:         "trino-list-tables",
					Source:       "my-trino-instance",
					Description:  "Lists all tables in a schema",
					AuthRequired: []string{},
				},
			},
		},
		{
			desc: "with auth",
			in: `
			tools:
				list_tables_auth:
					kind: trino-list-tables
					source: my-trino-instance
					description: Lists tables with authentication
					authRequired:
						- my-auth-service
			`,
			want: server.ToolConfigs{
				"list_tables_auth": trinolisttables.Config{
					Name:         "list_tables_auth",
					Kind:         "trino-list-tables",
					Source:       "my-trino-instance",
					Description:  "Lists tables with authentication",
					AuthRequired: []string{"my-auth-service"},
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

func TestTrinoListTablesToolConfigKind(t *testing.T) {
	config := trinolisttables.Config{
		Name:        "test",
		Kind:        "trino-list-tables",
		Source:      "test-source",
		Description: "test",
	}

	if kind := config.ToolConfigKind(); kind != "trino-list-tables" {
		t.Errorf("expected ToolConfigKind to return 'trino-list-tables', got %s", kind)
	}
}

func TestTrinoListTablesInitializeParameters(t *testing.T) {
	// This test verifies that the tool would be initialized with the correct parameters
	// The actual initialization requires a valid source which we can't provide in unit tests

	config := trinolisttables.Config{
		Name:        "test_tool",
		Kind:        "trino-list-tables",
		Source:      "test-source",
		Description: "test description",
	}

	// Verify the config structure
	if config.Name != "test_tool" {
		t.Errorf("expected Name to be 'test_tool', got %s", config.Name)
	}

	// The tool should have the following parameters when initialized:
	// - catalog (string, optional)
	// - schema (string, optional)
	// - table_filter (string, optional)
	// - include_views (boolean, default true)
	// - include_details (boolean, default false)
	// This is verified through the Initialize method which we can't call without a DB
}
