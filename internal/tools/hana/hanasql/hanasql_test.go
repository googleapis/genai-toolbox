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

package hanasql_test

import (
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/tools/hana/hanasql"
)

func TestParseFromYamlHanaSQL(t *testing.T) {
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
                    kind: hana-sql
                    source: my-hana-instance
                    description: some description
                    statement: |
                        SELECT * FROM TABLES;
                    authRequired:
                        - corp-auth-service
                    parameters:
                        - name: schema
                          type: string
                          description: schema name
            `,
			want: server.ToolConfigs{
				"example_tool": hanasql.Config{
					Name:         "example_tool",
					Kind:         "hana-sql",
					Source:       "my-hana-instance",
					Description:  "some description",
					Statement:    "SELECT * FROM TABLES;\n",
					AuthRequired: []string{"corp-auth-service"},
					Parameters: []tools.Parameter{
						tools.NewStringParameter("schema", "schema name"),
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
			if err := yaml.UnmarshalContext(ctx, testutils.FormatYaml(tc.in), &got); err != nil {
				t.Fatalf("unable to unmarshal: %s", err)
			}
			if diff := cmp.Diff(tc.want, got.Tools); diff != "" {
				t.Fatalf("incorrect parse: diff %v", diff)
			}
		})
	}
}
