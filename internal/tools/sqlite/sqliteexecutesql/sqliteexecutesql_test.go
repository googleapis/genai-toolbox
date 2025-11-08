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

package sqliteexecutesql_test

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
	"github.com/googleapis/genai-toolbox/internal/tools/sqlite/sqliteexecutesql"
	"github.com/googleapis/genai-toolbox/internal/util/orderedmap"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
	_ "modernc.org/sqlite"
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
					kind: sqlite-execute-sql
					source: my-instance
					description: some description
					authRequired:
						- my-google-auth-service
						- other-auth-service
			`,
			want: server.ToolConfigs{
				"example_tool": sqliteexecutesql.Config{
					Name:         "example_tool",
					Kind:         "sqlite-execute-sql",
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

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open in-memory database: %v", err)
	}
	return db
}

func TestTool_Invoke(t *testing.T) {
	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	type fields struct {
		Name         string
		Kind         string
		AuthRequired []string
		Parameters   parameters.Parameters
		DB           *sql.DB
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
			name: "create table",
			fields: fields{
				DB: setupTestDB(t),
			},
			args: args{
				ctx: ctx,
				params: []parameters.ParamValue{
					{Name: "sql", Value: "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, age INTEGER)"},
				},
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "insert data",
			fields: fields{
				DB: setupTestDB(t),
			},
			args: args{
				ctx: ctx,
				params: []parameters.ParamValue{
					{Name: "sql", Value: "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, age INTEGER); INSERT INTO users (id, name, age) VALUES (1, 'Alice', 30), (2, 'Bob', 25)"},
				},
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "select data",
			fields: fields{
				DB: func() *sql.DB {
					db := setupTestDB(t)
					if _, err := db.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, age INTEGER); INSERT INTO users (id, name, age) VALUES (1, 'Alice', 30), (2, 'Bob', 25)"); err != nil {
						t.Fatalf("Failed to set up database for select: %v", err)
					}
					return db
				}(),
			},
			args: args{
				ctx: ctx,
				params: []parameters.ParamValue{
					{Name: "sql", Value: "SELECT * FROM users"},
				},
			},
			want: []any{
				orderedmap.Row{
					Columns: []orderedmap.Column{
						{Name: "id", Value: int64(1)},
						{Name: "name", Value: "Alice"},
						{Name: "age", Value: int64(30)},
					},
				},
				orderedmap.Row{
					Columns: []orderedmap.Column{
						{Name: "id", Value: int64(2)},
						{Name: "name", Value: "Bob"},
						{Name: "age", Value: int64(25)},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "drop table",
			fields: fields{
				DB: func() *sql.DB {
					db := setupTestDB(t)
					if _, err := db.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, age INTEGER)"); err != nil {
						t.Fatalf("Failed to set up database for drop: %v", err)
					}
					return db
				}(),
			},
			args: args{
				ctx: ctx,
				params: []parameters.ParamValue{
					{Name: "sql", Value: "DROP TABLE users"},
				},
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "invalid sql",
			fields: fields{
				DB: setupTestDB(t),
			},
			args: args{
				ctx: ctx,
				params: []parameters.ParamValue{
					{Name: "sql", Value: "SELECT * FROM non_existent_table"},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "empty sql",
			fields: fields{
				DB: setupTestDB(t),
			},
			args: args{
				ctx: ctx,
				params: []parameters.ParamValue{
					{Name: "sql", Value: ""},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "data types",
			fields: fields{
				DB: func() *sql.DB {
					db := setupTestDB(t)
					if _, err := db.Exec("CREATE TABLE data_types (id INTEGER PRIMARY KEY, null_col TEXT, blob_col BLOB)"); err != nil {
						t.Fatalf("Failed to set up database for data types: %v", err)
					}
					if _, err := db.Exec("INSERT INTO data_types (id, null_col, blob_col) VALUES (1, NULL, ?)", []byte{1, 2, 3}); err != nil {
						t.Fatalf("Failed to insert data for data types: %v", err)
					}
					return db
				}(),
			},
			args: args{
				ctx: ctx,
				params: []parameters.ParamValue{
					{Name: "sql", Value: "SELECT * FROM data_types"},
				},
			},
			want: []any{
				orderedmap.Row{
					Columns: []orderedmap.Column{
						{Name: "id", Value: int64(1)},
						{Name: "null_col", Value: nil},
						{Name: "blob_col", Value: []byte{1, 2, 3}},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "join operation",
			fields: fields{
				DB: func() *sql.DB {
					db := setupTestDB(t)
					if _, err := db.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, age INTEGER)"); err != nil {
						t.Fatalf("Failed to set up database for join: %v", err)
					}
					if _, err := db.Exec("INSERT INTO users (id, name, age) VALUES (1, 'Alice', 30), (2, 'Bob', 25)"); err != nil {
						t.Fatalf("Failed to insert data for join: %v", err)
					}
					if _, err := db.Exec("CREATE TABLE orders (id INTEGER PRIMARY KEY, user_id INTEGER, item TEXT)"); err != nil {
						t.Fatalf("Failed to set up database for join: %v", err)
					}
					if _, err := db.Exec("INSERT INTO orders (id, user_id, item) VALUES (1, 1, 'Laptop'), (2, 2, 'Keyboard')"); err != nil {
						t.Fatalf("Failed to insert data for join: %v", err)
					}
					return db
				}(),
			},
			args: args{
				ctx: ctx,
				params: []parameters.ParamValue{
					{Name: "sql", Value: "SELECT u.name, o.item FROM users u JOIN orders o ON u.id = o.user_id"},
				},
			},
			want: []any{
				orderedmap.Row{
					Columns: []orderedmap.Column{
						{Name: "name", Value: "Alice"},
						{Name: "item", Value: "Laptop"},
					},
				},
				orderedmap.Row{
					Columns: []orderedmap.Column{
						{Name: "name", Value: "Bob"},
						{Name: "item", Value: "Keyboard"},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &sqliteexecutesql.Tool{
				Name:         tt.fields.Name,
				Kind:         tt.fields.Kind,
				AuthRequired: tt.fields.AuthRequired,
				Parameters:   tt.fields.Parameters,
				DB:           tt.fields.DB,
			}
			got, err := tr.Invoke(tt.args.ctx, tt.args.params, tt.args.accessToken)
			if (err != nil) != tt.wantErr {
				t.Errorf("Tool.Invoke() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			isEqual := false
			if got != nil && len(got.([]any)) == 0 && len(tt.want.([]any)) == 0 {
				isEqual = true // Special case for empty slices, since DeepEqual returns false
			} else {
				isEqual = reflect.DeepEqual(got, tt.want)
			}

			if !isEqual {
				t.Errorf("Tool.Invoke() = %+v, want %v", got, tt.want)
			}
		})
	}
}
