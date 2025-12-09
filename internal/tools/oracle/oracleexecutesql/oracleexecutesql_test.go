// Copyright © 2025, Oracle and/or its affiliates.

package oracleexecutesql_test

import (
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools/oracle/oracleexecutesql"
)

func TestParseFromYamlOracleExecuteSql(t *testing.T) {
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
			desc: "basic example with auth",
			in: `
            tools:
                run_adhoc_query:
                    kind: oracle-execute-sql
                    source: my-oracle-instance
                    description: Executes arbitrary SQL statements like INSERT or UPDATE.
                    authRequired:
                        - my-google-auth-service
            `,
			want: server.ToolConfigs{
				"run_adhoc_query": oracleexecutesql.Config{
					Name:         "run_adhoc_query",
					Kind:         "oracle-execute-sql",
					Source:       "my-oracle-instance",
					Description:  "Executes arbitrary SQL statements like INSERT or UPDATE.",
					AuthRequired: []string{"my-google-auth-service"},
				},
			},
		},
		{
			desc: "example without authRequired",
			in: `
            tools:
                run_simple_update:
                    kind: oracle-execute-sql
                    source: db-dev
                    description: Runs a simple update operation.
            `,
			want: server.ToolConfigs{
				"run_simple_update": oracleexecutesql.Config{
					Name:         "run_simple_update",
					Kind:         "oracle-execute-sql",
					Source:       "db-dev",
					Description:  "Runs a simple update operation.",
					AuthRequired: []string{},
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
