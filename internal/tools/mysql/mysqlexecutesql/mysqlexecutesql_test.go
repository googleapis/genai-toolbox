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

package mysqlexecutesql_test

import (
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/orderedmap"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/tools/mysql/mysqlexecutesql"
)

func TestParseFromYamlExecuteSql(t *testing.T) {
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
					kind: mysql-execute-sql
					source: my-instance
					description: some description
					authRequired:
						- my-google-auth-service
						- other-auth-service
			`,
			want: server.ToolConfigs{
				"example_tool": mysqlexecutesql.Config{
					Name:         "example_tool",
					Kind:         "mysql-execute-sql",
					Source:       "my-instance",
					Description:  "some description",
					AuthRequired: []string{"my-google-auth-service", "other-auth-service"},
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

func TestTool_Invoke(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	tool := newMockTool(t, db)
	params, err := tool.ParseParams(map[string]any{"sql": "SELECT C, A, B FROM users"}, nil)
	if err != nil {
		t.Fatalf("failed to parse params: %s", err)
	}

	rows := sqlmock.NewRows([]string{"C", "A", "B"}).
		AddRow("c1", "a1", "b1").
		AddRow("c2", "a2", "b2")

	mock.ExpectQuery("SELECT C, A, B FROM users").WillReturnRows(rows)

	var accessToken tools.AccessToken
	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	result, err := tool.Invoke(ctx, params, accessToken)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	want := []orderedmap.Row{
		{Columns: []orderedmap.Column{{Name: "C", Value: "c1"}, {Name: "A", Value: "a1"}, {Name: "B", Value: "b1"}}},
		{Columns: []orderedmap.Column{{Name: "C", Value: "c2"}, {Name: "A", Value: "a2"}, {Name: "B", Value: "b2"}}},
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal result to JSON: %s", err)
	}

	wantJSON, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("failed to marshal want to JSON: %s", err)
	}

	if string(resultJSON) != string(wantJSON) {
		t.Errorf("unexpected result: got %s, want %s", string(resultJSON), string(wantJSON))
	}
}

func newMockTool(t *testing.T, db *sql.DB) tools.Tool {
	t.Helper()

	cfg := mysqlexecutesql.Config{
		Name:        "test-tool",
		Kind:        "mysql-execute-sql",
		Source:      "test-source",
		Description: "test description",
	}

	mockSource := &mockSource{pool: db}
	srcs := map[string]sources.Source{"test-source": mockSource}

	tool, err := cfg.Initialize(srcs)
	if err != nil {
		t.Fatalf("failed to initialize tool: %s", err)
	}

	return tool
}

type mockSource struct {
	pool *sql.DB
}

func (s *mockSource) MySQLPool() *sql.DB {
	return s.pool
}

func (s *mockSource) SourceKind() string {
	return "mock-mysql"
}
