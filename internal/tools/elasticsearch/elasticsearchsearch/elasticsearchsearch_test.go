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

package elasticsearchsearch

import (
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

func TestParseFromYamlElasticsearch(t *testing.T) {
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
				kind: elasticsearch-search
				source: my-elasticsearch-instance
				description: Elasticsearch search tool
				query: |
				  {
					  "query": {
						"match_all": {}
					  }
				  }
		`,
			want: server.ToolConfigs{
				"example_tool": Config{
					Name:         "example_tool",
					Kind:         "elasticsearch-search",
					Source:       "my-elasticsearch-instance",
					Description:  "Elasticsearch search tool",
					AuthRequired: []string{},
					Query:        "{\n  \"query\": {\n  \"match_all\": {}\n  }\n}\n",
				},
			},
		},
		{
			desc: "search with customizable sort parameter",
			in: `
tools:
	example_tool:
		kind: elasticsearch-search
		source: my-elasticsearch-instance
		description: Elasticsearch search tool with customizable sort
		parameters:
			- name: sort
			  type: string
			  description: Sort order for the query
		query: |
		  {
		    "query": {
		      "match_all": {}
		    },
		    "sort": [
		      { "$sort" }
		    ]
		  }
`,
			want: server.ToolConfigs{
				"example_tool": Config{
					Name:         "example_tool",
					Kind:         "elasticsearch-search",
					Source:       "my-elasticsearch-instance",
					Description:  "Elasticsearch search tool with customizable sort",
					AuthRequired: []string{},
					Parameters: tools.Parameters{
						tools.NewStringParameter("sort", "Sort order for the query"),
					},
					Query: "{\n  \"query\": {\n    \"match_all\": {}\n  },\n  \"sort\": [\n    { \"$sort\" }\n  ]\n}\n",
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
