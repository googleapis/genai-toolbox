package cassandracql_test

import (
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/tools/cassandra/cassandracql"
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
