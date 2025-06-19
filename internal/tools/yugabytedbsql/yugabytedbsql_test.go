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

package yugabytedbsql_test

import (
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/tools/yugabytedbsql"
)

func TestParseFromYamlYugabyteDBSQL(t *testing.T) {
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
			desc: "basic valid config",
			in: `
			tools:
			  hotel_search:
			    kind: yugabytedb-sql
			    source: yb-source
			    description: search hotels by city
			    statement: |
			      SELECT * FROM hotels WHERE city = $1;
			    authRequired:
			      - auth-service-a
			      - auth-service-b
			    parameters:
			      - name: city
			        type: string
			        description: city name
			        authServices:
			          - name: auth-service-a
			            field: user_id
			          - name: auth-service-b
			            field: user_id
			`,
			want: server.ToolConfigs{
				"hotel_search": yugabytedbsql.Config{
					Name:         "hotel_search",
					Kind:         "yugabytedb-sql",
					Source:       "yb-source",
					Description:  "search hotels by city",
					Statement:    "SELECT * FROM hotels WHERE city = $1;\n",
					AuthRequired: []string{"auth-service-a", "auth-service-b"},
					Parameters: []tools.Parameter{
						tools.NewStringParameterWithAuth("city", "city name",
							[]tools.ParamAuthService{
								{Name: "auth-service-a", Field: "user_id"},
								{Name: "auth-service-b", Field: "user_id"},
							},
						),
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

func TestFailParseFromYamlYugabyteDBSQL(t *testing.T) {
	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	cases := []struct {
		desc string
		in   string
	}{
		{
			desc: "missing required field (statement)",
			in: `
			tools:
			  tool1:
			    kind: yugabytedb-sql
			    source: yb-source
			    description: incomplete config
			`,
		},
		{
			desc: "unknown field (foo)",
			in: `
			tools:
			  tool2:
			    kind: yugabytedb-sql
			    source: yb-source
			    description: test
			    statement: SELECT 1;
			    foo: bar
			`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			cfg := struct {
				Tools server.ToolConfigs `yaml:"tools"`
			}{}
			err := yaml.UnmarshalContext(ctx, testutils.FormatYaml(tc.in), &cfg)
			if err == nil {
				t.Fatalf("expected error but got none")
			}
		})
	}
}

func TestParseFromYamlWithTemplateParamsYugabyteDB(t *testing.T) {
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
					kind: yugabytedb-sql
					source: my-yb-instance
					description: some description
					statement: |
						SELECT * FROM SQL_STATEMENT;
					parameters:
						- name: name
						  type: string
						  description: some description
					templateParameters:
						- name: tableName
						  type: string
						  description: The table to select hotels from.
						- name: fieldArray
						  type: array
						  description: The columns to return for the query.
						  items: 
								name: column
								type: string
								description: A column name that will be returned from the query.
			`,
			want: server.ToolConfigs{
				"example_tool": yugabytedbsql.Config{
					Name:         "example_tool",
					Kind:         "yugabytedb-sql",
					Source:       "my-yb-instance",
					Description:  "some description",
					Statement:    "SELECT * FROM SQL_STATEMENT;\n",
					AuthRequired: []string{},
					Parameters: []tools.Parameter{
						tools.NewStringParameter("name", "some description"),
					},
					TemplateParameters: []tools.Parameter{
						tools.NewStringParameter("tableName", "The table to select hotels from."),
						tools.NewArrayParameter("fieldArray", "The columns to return for the query.", tools.NewStringParameter("column", "A column name that will be returned from the query.")),
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
