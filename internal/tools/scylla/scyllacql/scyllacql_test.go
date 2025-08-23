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

package scyllacql_test

import (
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/tools/scylla/scyllacql"
)

func TestParseFromYamlScylla(t *testing.T) {
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
					kind: scylla-cql
					source: my-scylla-instance
					description: Query users by ID
					statement: |
						SELECT * FROM users WHERE user_id = ?
					authRequired:
						- my-google-auth-service
						- other-auth-service
					parameters:
						- name: user_id
						  type: string
						  description: User ID to filter by
						  authServices:
							- name: my-google-auth-service
							  field: user_id
							- name: other-auth-service
							  field: user_id
			`,
			want: server.ToolConfigs{
				"example_tool": scyllacql.Config{
					Name:         "example_tool",
					Kind:         "scylla-cql",
					Source:       "my-scylla-instance",
					Description:  "Query users by ID",
					Statement:    "SELECT * FROM users WHERE user_id = ?\n",
					AuthRequired: []string{"my-google-auth-service", "other-auth-service"},
					Parameters: []tools.Parameter{
						tools.NewStringParameterWithAuth("user_id", "User ID to filter by",
							[]tools.ParamAuthService{{Name: "my-google-auth-service", Field: "user_id"},
								{Name: "other-auth-service", Field: "user_id"}}),
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

func TestParseFromYamlWithTemplateParamsScylla(t *testing.T) {
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
			desc: "example with template parameters",
			in: `
			tools:
				example_tool:
					kind: scylla-cql
					source: my-scylla-instance
					description: Query from dynamic table
					statement: |
						SELECT * FROM {{ .keyspace }}.{{ .tableName }} WHERE status = ?
					authRequired:
						- my-google-auth-service
					parameters:
						- name: status
						  type: string
						  description: Status to filter by
					templateParameters:
						- name: keyspace
						  type: string
						  description: The keyspace to query from
						- name: tableName
						  type: string
						  description: The table to select data from
						- name: fieldArray
						  type: array
						  description: The columns to return for the query
						  items: 
								name: column
								type: string
								description: A column name that will be returned from the query
			`,
			want: server.ToolConfigs{
				"example_tool": scyllacql.Config{
					Name:         "example_tool",
					Kind:         "scylla-cql",
					Source:       "my-scylla-instance",
					Description:  "Query from dynamic table",
					Statement:    "SELECT * FROM {{ .keyspace }}.{{ .tableName }} WHERE status = ?\n",
					AuthRequired: []string{"my-google-auth-service"},
					Parameters: []tools.Parameter{
						tools.NewStringParameter("status", "Status to filter by"),
					},
					TemplateParameters: []tools.Parameter{
						tools.NewStringParameter("keyspace", "The keyspace to query from"),
						tools.NewStringParameter("tableName", "The table to select data from"),
						tools.NewArrayParameter("fieldArray", "The columns to return for the query", tools.NewStringParameter("column", "A column name that will be returned from the query")),
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
