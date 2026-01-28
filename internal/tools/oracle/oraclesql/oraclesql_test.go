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

	valTrue := true
	valFalse := false

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
					ReadOnly:     nil,
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
					ReadOnly:     nil,
					AuthRequired: []string{},
				},
			},
		},
		{
			desc: "explicit: readOnly set to true",
			in: `
            tools:
                safe_query:
                    kind: oracle-sql
                    source: db-prod
                    description: Safe read operation.
                    readOnly: true
                    statement: "SELECT * FROM orders"
            `,
			want: server.ToolConfigs{
				"safe_query": oraclesql.Config{
					Name:         "safe_query",
					Kind:         "oracle-sql",
					Source:       "db-prod",
					Description:  "Safe read operation.",
					Statement:    "SELECT * FROM orders",
					ReadOnly:     &valTrue,
					AuthRequired: []string{},
				},
			},
		},
		{
			desc: "example with readonly flag set to false (DML)",
			in: `
            tools:
                update_user:
                    kind: oracle-sql
                    source: db-prod
                    description: Updates user email.
                    readOnly: false   # <--- We are testing this specific line
                    statement: "UPDATE users SET email = :1 WHERE id = :2"
            `,
			want: server.ToolConfigs{
				"update_user": oraclesql.Config{
					Name:         "update_user",
					Kind:         "oracle-sql",
					Source:       "db-prod",
					Description:  "Updates user email.",
					Statement:    "UPDATE users SET email = :1 WHERE id = :2",
					ReadOnly:     &valFalse,
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
