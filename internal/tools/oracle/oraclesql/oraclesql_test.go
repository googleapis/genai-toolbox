// Copyright © 2025, Oracle and/or its affiliates.
package oraclesql_test

import (
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools/oracle/oraclesql"
)

func TestParseFromYamlOracleSql(t *testing.T) {
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
			desc: "basic example with statement and auth",
			in: `
            tools:
                get_user_by_id:
                    kind: oracle-sql
                    source: my-oracle-instance
                    description: Retrieves user details by ID.
                    statement: "SELECT id, name, email FROM users WHERE id = :1"
                    authRequired:
                        - my-google-auth-service
            `,
			want: server.ToolConfigs{
				"get_user_by_id": oraclesql.Config{
					Name:         "get_user_by_id",
					Kind:         "oracle-sql",
					Source:       "my-oracle-instance",
					Description:  "Retrieves user details by ID.",
					Statement:    "SELECT id, name, email FROM users WHERE id = :1",
					AuthRequired: []string{"my-google-auth-service"},
				},
			},
		},
		{
			desc: "example with parameters and template parameters",
			in: `
            tools:
                get_orders:
                    kind: oracle-sql
                    source: db-prod
                    description: Gets orders for a customer with optional filtering.
                    statement: "SELECT * FROM ${SCHEMA}.ORDERS WHERE customer_id = :customer_id AND status = :status"
            `,
			want: server.ToolConfigs{
				"get_orders": oraclesql.Config{
					Name:         "get_orders",
					Kind:         "oracle-sql",
					Source:       "db-prod",
					Description:  "Gets orders for a customer with optional filtering.",
					Statement:    "SELECT * FROM ${SCHEMA}.ORDERS WHERE customer_id = :customer_id AND status = :status",
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
