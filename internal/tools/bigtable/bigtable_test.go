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

package bigtable_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools/bigtable"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
)

func TestParseFromYamlBigtable(t *testing.T) {
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
			type: bigtable-sql
			source: my-pg-instance
			description: some description
			statement: |
				SELECT * FROM SQL_STATEMENT;
			parameters:
				- name: country
				  type: string
				  description: some description
			`,
			want: server.ToolConfigs{
				"example_tool": bigtable.Config{
					Name:         "example_tool",
					Type:         "bigtable-sql",
					Source:       "my-pg-instance",
					Description:  "some description",
					Statement:    "SELECT * FROM SQL_STATEMENT;\n",
					AuthRequired: []string{},
					Parameters: []parameters.Parameter{
						parameters.NewStringParameter("country", "some description"),
					},
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
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

func TestParseFromYamlWithTemplateBigtable(t *testing.T) {
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
			type: bigtable-sql
			source: my-pg-instance
			description: some description
			statement: |
				SELECT * FROM SQL_STATEMENT;
			parameters:
				- name: country
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
				"example_tool": bigtable.Config{
					Name:         "example_tool",
					Type:         "bigtable-sql",
					Source:       "my-pg-instance",
					Description:  "some description",
					Statement:    "SELECT * FROM SQL_STATEMENT;\n",
					AuthRequired: []string{},
					Parameters: []parameters.Parameter{
						parameters.NewStringParameter("country", "some description"),
					},
					TemplateParameters: []parameters.Parameter{
						parameters.NewStringParameter("tableName", "The table to select hotels from."),
						parameters.NewArrayParameter("fieldArray", "The columns to return for the query.", parameters.NewStringParameter("column", "A column name that will be returned from the query.")),
					},
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
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
