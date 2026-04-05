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

package tidbgetqueryplan

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
)

func TestParseFromYaml(t *testing.T) {
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
            name: get_query_plan
            type: tidb-get-query-plan
            source: my-tidb-instance
            description: Get query execution plan
			`,
			want: server.ToolConfigs{
				"get_query_plan": Config{
					Name:         "get_query_plan",
					Type:         "tidb-get-query-plan",
					Source:       "my-tidb-instance",
					Description:  "Get query execution plan",
					AuthRequired: []string{},
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

func TestStripSQLComments(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		want string
	}{
		{
			desc: "no comments",
			in:   "SELECT * FROM users",
			want: "SELECT * FROM users",
		},
		{
			desc: "single line comment",
			in:   "-- this is a comment\nSELECT * FROM users",
			want: "SELECT * FROM users",
		},
		{
			desc: "multi-line comment",
			in:   "/* comment */ SELECT * FROM users",
			want: "SELECT * FROM users",
		},
		{
			desc: "comment at end",
			in:   "SELECT * FROM users -- trailing comment",
			want: "SELECT * FROM users",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := stripSQLComments(tc.in)
			if got != tc.want {
				t.Errorf("stripSQLComments(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestIsSelectOrWithStatement(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		want bool
	}{
		{
			desc: "simple select",
			in:   "SELECT * FROM users",
			want: true,
		},
		{
			desc: "select with comment prefix",
			in:   "/* hint */ SELECT * FROM users",
			want: true,
		},
		{
			desc: "WITH CTE",
			in:   "WITH cte AS (SELECT 1) SELECT * FROM cte",
			want: true,
		},
		{
			desc: "lowercase select",
			in:   "select * from users",
			want: true,
		},
		{
			desc: "DELETE statement",
			in:   "DELETE FROM users WHERE id = 1",
			want: false,
		},
		{
			desc: "INSERT statement",
			in:   "INSERT INTO users VALUES (1, 'test')",
			want: false,
		},
		{
			desc: "UPDATE statement",
			in:   "UPDATE users SET name = 'test'",
			want: false,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := isSelectOrWithStatement(tc.in)
			if got != tc.want {
				t.Errorf("isSelectOrWithStatement(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestContainsMultipleStatements(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		want bool
	}{
		{
			desc: "single statement",
			in:   "SELECT * FROM users",
			want: false,
		},
		{
			desc: "multiple statements",
			in:   "SELECT * FROM users; DELETE FROM users",
			want: true,
		},
		{
			desc: "semicolon in string literal",
			in:   "SELECT * FROM users WHERE name = 'test;value'",
			want: false,
		},
		{
			desc: "semicolon in double quoted string",
			in:   `SELECT * FROM users WHERE name = "test;value"`,
			want: false,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := containsMultipleStatements(tc.in)
			if got != tc.want {
				t.Errorf("containsMultipleStatements(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}
