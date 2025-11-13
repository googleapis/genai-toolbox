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

package sqlitesql_test

import (
	"context"
	"database/sql"
	"reflect"
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/tools/sqlite/sqlitesql"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
	_ "modernc.org/sqlite"
)

func TestParseFromYamlSQLite(t *testing.T) {
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
					kind: sqlite-sql
					source: my-sqlite-instance
					description: some description
					statement: |
						SELECT * FROM SQL_STATEMENT;
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
				"example_tool": sqlitesql.Config{
					Name:         "example_tool",
					Kind:         "sqlite-sql",
					Source:       "my-sqlite-instance",
					Description:  "some description",
					Statement:    "SELECT * FROM SQL_STATEMENT;\n",
					AuthRequired: []string{"my-google-auth-service", "other-auth-service"},
					Parameters: []parameters.Parameter{
						parameters.NewStringParameterWithAuth("country", "some description",
							[]parameters.ParamAuthService{{Name: "my-google-auth-service", Field: "user_id"},
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

func TestParseFromYamlWithTemplateSqlite(t *testing.T) {
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
					kind: sqlite-sql
					source: my-sqlite-db
					description: some description
					statement: |
						SELECT * FROM SQL_STATEMENT;
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
				"example_tool": sqlitesql.Config{
					Name:         "example_tool",
					Kind:         "sqlite-sql",
					Source:       "my-sqlite-db",
					Description:  "some description",
					Statement:    "SELECT * FROM SQL_STATEMENT;\n",
					AuthRequired: []string{"my-google-auth-service", "other-auth-service"},
					Parameters: []parameters.Parameter{
						parameters.NewStringParameterWithAuth("country", "some description",
							[]parameters.ParamAuthService{{Name: "my-google-auth-service", Field: "user_id"},
								{Name: "other-auth-service", Field: "user_id"}}),
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

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory database: %v", err)
	}

	createTable := `
	CREATE TABLE users (
		id INTEGER PRIMARY KEY,
		name TEXT,
		age INTEGER
	);`
	if _, err := db.Exec(createTable); err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	insertData := `
	INSERT INTO users (id, name, age) VALUES
	(1, 'Alice', 30),
	(2, 'Bob', 25);`
	if _, err := db.Exec(insertData); err != nil {
		t.Fatalf("Failed to insert data: %v", err)
	}

	return db
}

func TestTool_Invoke(t *testing.T) {
	type fields struct {
		Name               string
		Kind               string
		AuthRequired       []string
		Parameters         parameters.Parameters
		TemplateParameters parameters.Parameters
		AllParams          parameters.Parameters
		Db                 *sql.DB
		Statement          string
	}
	type args struct {
		ctx         context.Context
		params      parameters.ParamValues
		accessToken tools.AccessToken
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    any
		wantErr bool
	}{
		{
			name: "simple select",
			fields: fields{
				Db:        setupTestDB(t),
				Statement: "SELECT * FROM users",
			},
			args: args{
				ctx: context.Background(),
			},
			want: []any{
				map[string]any{"id": int64(1), "name": "Alice", "age": int64(30)},
				map[string]any{"id": int64(2), "name": "Bob", "age": int64(25)},
			},
			wantErr: false,
		},
		{
			name: "select with parameter",
			fields: fields{
				Db:        setupTestDB(t),
				Statement: "SELECT * FROM users WHERE name = ?",
				Parameters: []parameters.Parameter{
					parameters.NewStringParameter("name", "user name"),
				},
			},
			args: args{
				ctx: context.Background(),
				params: []parameters.ParamValue{
					{Name: "name", Value: "Alice"},
				},
			},
			want: []any{
				map[string]any{"id": int64(1), "name": "Alice", "age": int64(30)},
			},
			wantErr: false,
		},
		{
			name: "select with template parameter",
			fields: fields{
				Db:        setupTestDB(t),
				Statement: "SELECT * FROM {{.tableName}}",
				TemplateParameters: []parameters.Parameter{
					parameters.NewStringParameter("tableName", "table name"),
				},
			},
			args: args{
				ctx: context.Background(),
				params: []parameters.ParamValue{
					{Name: "tableName", Value: "users"},
				},
			},
			want: []any{
				map[string]any{"id": int64(1), "name": "Alice", "age": int64(30)},
				map[string]any{"id": int64(2), "name": "Bob", "age": int64(25)},
			},
			wantErr: false,
		},
		{
			name: "invalid sql",
			fields: fields{
				Db:        setupTestDB(t),
				Statement: "SELECT * FROM non_existent_table",
			},
			args: args{
				ctx: context.Background(),
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := sqlitesql.Tool{
				Name:               tt.fields.Name,
				Kind:               tt.fields.Kind,
				AuthRequired:       tt.fields.AuthRequired,
				Parameters:         tt.fields.Parameters,
				TemplateParameters: tt.fields.TemplateParameters,
				AllParams:          tt.fields.AllParams,
				Db:                 tt.fields.Db,
				Statement:          tt.fields.Statement,
			}
			got, err := tr.Invoke(tt.args.ctx, tt.args.params, tt.args.accessToken)
			if (err != nil) != tt.wantErr {
				t.Errorf("Tool.Invoke() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Tool.Invoke() = %v, want %v", got, tt.want)
			}
		})
	}
}
