package hanaexecutesql_test

import (
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools/hana/hanaexecutesql"
)

func TestParseFromYamlHanaExecuteSQL(t *testing.T) {
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
                exec_tool:
                    kind: hana-execute-sql
                    source: my-hana-instance
                    description: execute any sql
                    authRequired:
                        - corp-auth-service
            `,
			want: server.ToolConfigs{
				"exec_tool": hanaexecutesql.Config{
					Name:         "exec_tool",
					Kind:         "hana-execute-sql",
					Source:       "my-hana-instance",
					Description:  "execute any sql",
					AuthRequired: []string{"corp-auth-service"},
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
