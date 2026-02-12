// Copyright 2026 Google LLC
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

package falkordbcypher

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
)

func TestParseFromYamlFalkorDB(t *testing.T) {
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
            kind: tools
            name: example_tool
            type: falkordb-cypher
            source: my-falkordb-instance
            description: some tool description
            authRequired:
                - my-google-auth-service
                - other-auth-service
            statement: |
                MATCH (p:Person) WHERE p.name = $name RETURN p.id as id;
            parameters:
                - name: name
                  type: string
                  description: person name parameter
			`,
			want: server.ToolConfigs{
				"example_tool": Config{
					Name:         "example_tool",
					Type:         "falkordb-cypher",
					Source:       "my-falkordb-instance",
					Description:  "some tool description",
					AuthRequired: []string{"my-google-auth-service", "other-auth-service"},
					Statement:    "MATCH (p:Person) WHERE p.name = $name RETURN p.id as id;\n",
					Parameters: []parameters.Parameter{
						parameters.NewStringParameter("name", "person name parameter"),
					},
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			// Parse contents
			_, _, _, got, _, _, err := server.UnmarshalResourceConfig(ctx, testutils.FormatYaml(tc.in))
			if err != nil {
				t.Fatalf("unable to unmarshal: %s", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("incorrect parse: diff %v", diff)
			}
		})
	}
}
