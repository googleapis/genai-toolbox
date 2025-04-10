// Copyright 2024 Google LLC
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

package neo4j_test

import (
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/tools/neo4j"
)

func TestParseFromYamlNeo4j(t *testing.T) {
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
					kind: neo4j-cypher
					source: my-neo4j-instance
					description: some tool description
					authRequired:
						- my-google-auth-service
						- other-auth-service
					statement: |
						MATCH (c:Country) WHERE c.name = $country RETURN c.id as id;
					parameters:
						- name: country
						  type: string
						  description: country parameter description
			`,
			want: server.ToolConfigs{
				"example_tool": neo4j.Config{
					Name:         "example_tool",
					Kind:         neo4j.ToolKind,
					Source:       "my-neo4j-instance",
					Description:  "some tool description",
					AuthRequired: []string{"my-google-auth-service", "other-auth-service"},
					Statement:    "MATCH (c:Country) WHERE c.name = $country RETURN c.id as id;\n",
					Parameters: []tools.Parameter{
						tools.NewStringParameter("country", "country parameter description"),
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
