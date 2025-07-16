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

package duckdb_test

import (
	"testing"

	"github.com/googleapis/genai-toolbox/internal/tools/duckdb"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

func TestParseFromYamlDuckDb(t *testing.T) {
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
					kind: duckdb-sql
					source: my-duckdb-instance
					description: some description
					statement: |
						select * from hotel WHERE name = $hotel;
					parameters:
						- name: hotel
						  type: string
						  description: hotel parameter description
			`,
			want: server.ToolConfigs{
				"example_tool": duckdb.Config{
					Name:         "example_tool",
					Kind:         "duckdb-sql",
					Source:       "my-duckdb-instance",
					Description:  "some description",
					Statement:    "select * from hotel WHERE name = $hotel;\n",
					AuthRequired: []string{},
					Parameters: []tools.Parameter{
						tools.NewStringParameter("hotel", "hotel parameter description"),
					},
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := struct {
				Tools server.ToolConfigs `yaml:"tools"`
			}{}

			// Create a context with a logger
			ctx, err := testutils.ContextWithNewLogger()
			if err != nil {
				t.Fatalf("unable to create context with logger: %s", err)
			}

			// Parse contents with context
			err = yaml.UnmarshalContext(ctx, testutils.FormatYaml(tc.in), &got)
			if err != nil {
				t.Fatalf("unable to unmarshal: %s", err)
			}
			if diff := cmp.Diff(tc.want, got.Tools); diff != "" {
				t.Fatalf("incorrect parse: diff %v", diff)
			}
		})
	}
}
