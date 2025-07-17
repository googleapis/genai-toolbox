package kuzudb

import (
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

func TestParseFromYamlKuzuDB(t *testing.T) {
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
					kind: kuzudb-cypher
					source: my-kuzudb-instance
					description: some description
					statement: |
						match (a:user {name:$name}) return a.*;
					authRequired:
						- my-google-auth-service
						- other-auth-service
					parameters:
						- name: name
						  type: string
						  description: some description
						  authServices:
							- name: my-google-auth-service
							  field: user_id
							- name: other-auth-service
							  field: user_id
			`,
			want: server.ToolConfigs{
				"example_tool": KuzuDBToolConfig{
					Name:         "example_tool",
					Kind:         "kuzudb-cypher",
					Source:       "my-kuzudb-instance",
					Description:  "some description",
					Statement:    "match (a:user {name:$name}) return a.*;\n",
					AuthRequired: []string{"my-google-auth-service", "other-auth-service"},
					Parameters: []tools.Parameter{
						tools.NewStringParameterWithAuth("name", "some description",
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

func TestParseFromYamlWithTemplateKuzuDB(t *testing.T) {
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
			desc: "kuzudb template parameter example",
			in: `
			tools:
				example_tool:
					kind: kuzudb-cypher
					source: my-kuzudb-db
					description: some description
					statement: |
						match (a:{{.tableName}} {name:$name}) return a.*;
					authRequired:
						- my-google-auth-service
						- other-auth-service
					parameters:
						- name: name
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
						  description: The table to select hotels from.
			`,
			want: server.ToolConfigs{
				"example_tool": KuzuDBToolConfig{
					Name:         "example_tool",
					Kind:         "kuzudb-cypher",
					Source:       "my-kuzudb-db",
					Description:  "some description",
					Statement:    "match (a:{{.tableName}} {name:$name}) return a.*;\n",
					AuthRequired: []string{"my-google-auth-service", "other-auth-service"},
					Parameters: []tools.Parameter{
						tools.NewStringParameterWithAuth("name", "some description",
							[]tools.ParamAuthService{{Name: "my-google-auth-service", Field: "user_id"},
								{Name: "other-auth-service", Field: "user_id"}}),
					},
					TemplateParameters: []tools.Parameter{
						tools.NewStringParameter("tableName", "The table to select hotels from."),
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
