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

package postgressql_test

import (
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/tools/postgres/postgressql"
)

func TestParseFromYamlPostgres(t *testing.T) {
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
					kind: postgres-sql
					source: my-pg-instance
					description: some description
					statement: |
						SELECT * FROM SQL_STATEMENT;
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
				"example_tool": postgressql.Config{
					Name:         "example_tool",
					Kind:         "postgres-sql",
					Source:       "my-pg-instance",
					Description:  "some description",
					Statement:    "SELECT * FROM SQL_STATEMENT;\n",
					AuthRequired: []string{"my-google-auth-service", "other-auth-service"},
					Parameters: []tools.Parameter{
						tools.NewStringParameterWithAuth("country", "some description",
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

func TestParseFromYamlWithTemplateParamsPostgres(t *testing.T) {
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
					kind: postgres-sql
					source: my-pg-instance
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
				"example_tool": postgressql.Config{
					Name:         "example_tool",
					Kind:         "postgres-sql",
					Source:       "my-pg-instance",
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

func TestParseFromYamlWithAuthFields(t *testing.T) {
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
			desc: "with authorization fields",
			in: `
			tools:
				example_tool:
					kind: postgres-sql
					source: my-pg-instance
					description: some description
					statement: |
						SELECT * FROM SQL_STATEMENT;
					authRequired:
						- my-azure-auth-service
					requiredRoles:
						- mcp.admin
						- mcp.support
					requiredPermissions:
						- read
						- write
					allowedOperations:
						- SELECT
						- UPDATE
					restrictedTables:
						- pg_user
						- system_config
					maxAffectedRows: 100
					parameters:
						- name: country
						  type: string
						  description: some description
			`,
			want: server.ToolConfigs{
				"example_tool": postgressql.Config{
					Name:               "example_tool",
					Kind:               "postgres-sql",
					Source:             "my-pg-instance",
					Description:        "some description",
					Statement:          "SELECT * FROM SQL_STATEMENT;\n",
					AuthRequired:       []string{"my-azure-auth-service"},
					RequiredRoles:      []string{"mcp.admin", "mcp.support"},
					RequiredPermissions: []string{"read", "write"},
					AllowedOperations:  []string{"SELECT", "UPDATE"},
					RestrictedTables:   []string{"pg_user", "system_config"},
					MaxAffectedRows:    100,
					Parameters: []tools.Parameter{
						tools.NewStringParameter("country", "some description"),
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
