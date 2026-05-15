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

package mysqlgetqueryplan_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/mcp-toolbox/internal/server"
	"github.com/googleapis/mcp-toolbox/internal/testutils"
	"github.com/googleapis/mcp-toolbox/internal/tools/mysql/mysqlgetqueryplan"
)

func TestValidateSQLStatement(t *testing.T) {
	tcs := []struct {
		desc    string
		input   string
		wantErr bool
	}{
		{desc: "select ok", input: "SELECT 1", wantErr: false},
		{desc: "select with where ok", input: "SELECT id FROM users WHERE id = 1", wantErr: false},
		{desc: "update ok", input: "UPDATE users SET x=1 WHERE id=2", wantErr: false},
		{desc: "insert ok", input: "INSERT INTO t VALUES (1)", wantErr: false},
		{desc: "delete ok", input: "DELETE FROM t WHERE id=1", wantErr: false},
		{desc: "with cte ok", input: "WITH cte AS (SELECT 1) SELECT * FROM cte", wantErr: false},
		// ANALYZE turns EXPLAIN into an execution primitive — must be rejected.
		{desc: "ANALYZE rejected", input: "ANALYZE SELECT * FROM mysql.user", wantErr: true},
		{desc: "analyze lowercase rejected", input: "analyze select 1", wantErr: true},
		// Semicolons enable multi-statement injection.
		{desc: "semicolon rejected", input: "SELECT 1; DROP TABLE users", wantErr: true},
		{desc: "trailing semicolon rejected", input: "SELECT 1;", wantErr: true},
		// Arbitrary DDL/DCL must not be explainable.
		{desc: "DROP rejected", input: "DROP TABLE users", wantErr: true},
		{desc: "CREATE rejected", input: "CREATE TABLE t (id INT)", wantErr: true},
		{desc: "CALL rejected", input: "CALL stored_proc()", wantErr: true},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			err := mysqlgetqueryplan.ValidateSQLStatement(tc.input)
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidateSQLStatement(%q) error = %v, wantErr %v", tc.input, err, tc.wantErr)
			}
		})
	}
}

func TestParseFromYamlGetQueryPlan(t *testing.T) {
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
            kind: tool
            name: example_tool
            type: mysql-get-query-plan
            source: my-instance
            description: some description
            authRequired:
                - my-google-auth-service
                - other-auth-service
			`,
			want: server.ToolConfigs{
				"example_tool": mysqlgetqueryplan.Config{
					Name:         "example_tool",
					Type:         "mysql-get-query-plan",
					Source:       "my-instance",
					Description:  "some description",
					AuthRequired: []string{"my-google-auth-service", "other-auth-service"},
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			// Parse contents
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
