// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cassandracql_test

import (
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools/cassandra/cassandracql"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
)

func TestParseFromYamlCassandra(t *testing.T) {
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
					kind: cassandra-cql
					source: my-cassandra-instance
					description: some description
					statement: |
						SELECT * FROM CQL_STATEMENT;
					authRequired:
						- my-google-auth-service
						- other-auth-service
					parameters:
						- name: country
						  type: string
						  description: some description
						  authServices:
							- name: my-google-auth-service
							  field: user_id
							- name: other-auth-service
							  field: user_id
			`,
			want: server.ToolConfigs{
				"example_tool": cassandracql.Config{
					Name:         "example_tool",
					Kind:         "cassandra-cql",
					Source:       "my-cassandra-instance",
					Description:  "some description",
					Statement:    "SELECT * FROM CQL_STATEMENT;\n",
					AuthRequired: []string{"my-google-auth-service", "other-auth-service"},
					Parameters: []parameters.Parameter{
						parameters.NewStringParameterWithAuth("country", "some description",
							[]parameters.ParamAuthService{{Name: "my-google-auth-service", Field: "user_id"},
								{Name: "other-auth-service", Field: "user_id"}}),
					},
				},
			},
		},
		{
			desc: "with template parameters",
			in: `
			tools:
				example_tool:
					kind: cassandra-cql
					source: my-cassandra-instance
					description: some description
					statement: |
						SELECT * FROM CQL_STATEMENT;
					authRequired:
						- my-google-auth-service
						- other-auth-service
					parameters:
						- name: country
						  type: string
						  description: some description
						  authServices:
							- name: my-google-auth-service
							  field: user_id
							- name: other-auth-service
							  field: user_id
					templateParameters:
						- name: tableName
						  type: string
						  description: some description.
						- name: fieldArray
						  type: array
						  description: The columns to return for the query.
						  items: 
								name: column
								type: string
								description: A column name that will be returned from the query.
			`,
			want: server.ToolConfigs{
				"example_tool": cassandracql.Config{
					Name:         "example_tool",
					Kind:         "cassandra-cql",
					Source:       "my-cassandra-instance",
					Description:  "some description",
					Statement:    "SELECT * FROM CQL_STATEMENT;\n",
					AuthRequired: []string{"my-google-auth-service", "other-auth-service"},
					Parameters: []parameters.Parameter{
						parameters.NewStringParameterWithAuth("country", "some description",
							[]parameters.ParamAuthService{{Name: "my-google-auth-service", Field: "user_id"},
								{Name: "other-auth-service", Field: "user_id"}}),
					},
					TemplateParameters: []parameters.Parameter{
						parameters.NewStringParameter("tableName", "some description."),
						parameters.NewArrayParameter("fieldArray", "The columns to return for the query.", parameters.NewStringParameter("column", "A column name that will be returned from the query.")),
					},
				},
			},
		},
		{
			desc: "without optional fields",
			in: `
			tools:
				example_tool:
					kind: cassandra-cql
					source: my-cassandra-instance
					description: some description
					statement: |
						SELECT * FROM CQL_STATEMENT;
			`,
			want: server.ToolConfigs{
				"example_tool": cassandracql.Config{
					Name:               "example_tool",
					Kind:               "cassandra-cql",
					Source:             "my-cassandra-instance",
					Description:        "some description",
					Statement:          "SELECT * FROM CQL_STATEMENT;\n",
					AuthRequired:       []string{},
					Parameters:         nil,
					TemplateParameters: nil,
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
