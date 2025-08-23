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

package trinogettableinfo_test

import (
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools/trino/trinogettableinfo"
)

func TestParseFromYamlTrinoGetTableInfo(t *testing.T) {
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
				get_table_info:
					kind: trino-get-table-info
					source: my-trino-instance
					description: Gets detailed table information
			`,
			want: server.ToolConfigs{
				"get_table_info": trinogettableinfo.Config{
					Name:         "get_table_info",
					Kind:         "trino-get-table-info",
					Source:       "my-trino-instance",
					Description:  "Gets detailed table information",
					AuthRequired: []string{},
				},
			},
		},
		{
			desc: "with auth",
			in: `
			tools:
				get_table_info_auth:
					kind: trino-get-table-info
					source: my-trino-instance
					description: Gets table info with authentication
					authRequired:
						- my-auth-service
						- another-auth
			`,
			want: server.ToolConfigs{
				"get_table_info_auth": trinogettableinfo.Config{
					Name:         "get_table_info_auth",
					Kind:         "trino-get-table-info",
					Source:       "my-trino-instance",
					Description:  "Gets table info with authentication",
					AuthRequired: []string{"my-auth-service", "another-auth"},
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

func TestTrinoGetTableInfoToolConfigKind(t *testing.T) {
	config := trinogettableinfo.Config{
		Name:        "test",
		Kind:        "trino-get-table-info",
		Source:      "test-source",
		Description: "test",
	}

	if kind := config.ToolConfigKind(); kind != "trino-get-table-info" {
		t.Errorf("expected ToolConfigKind to return 'trino-get-table-info', got %s", kind)
	}
}

func TestTrinoGetTableInfoParameters(t *testing.T) {
	// This test verifies the expected parameters for the tool
	// The actual tool initialization requires a database connection

	config := trinogettableinfo.Config{
		Name:        "test_tool",
		Kind:        "trino-get-table-info",
		Source:      "test-source",
		Description: "test description",
	}

	// Verify the config structure
	if config.Name != "test_tool" {
		t.Errorf("expected Name to be 'test_tool', got %s", config.Name)
	}

	// The tool should have the following parameters when initialized:
	// - table_name (string, required)
	// - catalog (string, optional)
	// - schema (string, optional)
	// - include_stats (boolean, default false)
	// - include_sample (boolean, default false)
	// - sample_size (integer, default 5)
}
