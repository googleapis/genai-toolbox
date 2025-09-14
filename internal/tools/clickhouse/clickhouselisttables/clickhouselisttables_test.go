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

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

func TestListTablesConfigToolConfigKind(t *testing.T) {
	cfg := Config{}
	if cfg.ToolConfigKind() != listTablesKind {
		t.Errorf("expected %q, got %q", listTablesKind, cfg.ToolConfigKind())
	}
}

func TestListTablesConfigInitializeMissingSource(t *testing.T) {
	cfg := Config{
		Name:        "test-list-tables",
		Kind:        listTablesKind,
		Source:      "missing-source",
		Description: "Test list tables tool",
	}

	srcs := map[string]sources.Source{}
	_, err := cfg.Initialize(srcs)
	if err == nil {
		t.Error("expected error for missing source")
	}
}

func TestParseFromYamlClickHouseListTables(t *testing.T) {
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
					kind: clickhouse-list-tables
					source: my-instance
					description: some description
			`,
			want: server.ToolConfigs{
				"example_tool": Config{
					Name:         "example_tool",
					Kind:         "clickhouse-list-tables",
					Source:       "my-instance",
					Description:  "some description",
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

func TestListTablesToolParseParams(t *testing.T) {
	databaseParam := tools.NewStringParameter("database", "The database to list tables from.")
	tool := Tool{
		Parameters: tools.Parameters{databaseParam},
		AllParams:  tools.Parameters{databaseParam},
	}

	params, err := tool.ParseParams(map[string]any{"database": "test_db"}, map[string]map[string]any{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(params) != 1 {
		t.Errorf("expected 1 parameter, got %d", len(params))
	}

	mapParams := params.AsMap()
	if mapParams["database"] != "test_db" {
		t.Errorf("expected database parameter to be 'test_db', got %v", mapParams["database"])
	}
}