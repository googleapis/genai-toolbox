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

package cockroachdblistschemas_test

import (
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools/cockroachdb/cockroachdblistschemas"
)

func TestParseFromYamlCockroachDBListSchemas(t *testing.T) {
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
				list_schemas_tool:
					kind: cockroachdb-list-schemas
					source: my-crdb-instance
					description: List schemas in CockroachDB
			`,
			want: server.ToolConfigs{
				"list_schemas_tool": cockroachdblistschemas.Config{
					Name:         "list_schemas_tool",
					Kind:         "cockroachdb-list-schemas",
					Source:       "my-crdb-instance",
					Description:  "List schemas in CockroachDB",
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

func TestCockroachDBListSchemasToolConfigKind(t *testing.T) {
	cfg := cockroachdblistschemas.Config{
		Name:        "test-tool",
		Kind:        "cockroachdb-list-schemas",
		Source:      "test-source",
		Description: "test description",
	}

	if cfg.ToolConfigKind() != "cockroachdb-list-schemas" {
		t.Errorf("expected ToolConfigKind 'cockroachdb-list-schemas', got %q", cfg.ToolConfigKind())
	}
}
