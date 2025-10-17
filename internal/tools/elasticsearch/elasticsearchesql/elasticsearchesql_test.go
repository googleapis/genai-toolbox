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

package elasticsearchesql

import (
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

func TestParseFromYamlElasticsearchEsql(t *testing.T) {
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
			desc: "basic search example",
			in: `
		tools:
			example_tool:
				kind: elasticsearch-esql
				source: my-elasticsearch-instance
				description: Elasticsearch ES|QL tool
				query: |
				  FROM my-index
				  | KEEP first_name, last_name
		`,
			want: server.ToolConfigs{
				"example_tool": Config{
					Name:         "example_tool",
					Kind:         "elasticsearch-esql",
					Source:       "my-elasticsearch-instance",
					Description:  "Elasticsearch ES|QL tool",
					AuthRequired: []string{},
					Query:        "FROM my-index\n| KEEP first_name, last_name\n",
				},
			},
		},
		{
			desc: "search with customizable limit parameter",
			in: `
			tools:
				example_tool:
					kind: elasticsearch-esql
					source: my-elasticsearch-instance
					description: Elasticsearch ES|QL tool with customizable limit
					parameters:
						- name: limit
						  type: integer
						  description: Limit the number of results
					query: |
					  FROM my-index
					  | LIMIT ?limit
			`,
			want: server.ToolConfigs{
				"example_tool": Config{
					Name:         "example_tool",
					Kind:         "elasticsearch-esql",
					Source:       "my-elasticsearch-instance",
					Description:  "Elasticsearch ES|QL tool with customizable limit",
					AuthRequired: []string{},
					Parameters: tools.Parameters{
						tools.NewIntParameter("limit", "Limit the number of results"),
					},
					Query: "FROM my-index\n| LIMIT ?limit\n",
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
