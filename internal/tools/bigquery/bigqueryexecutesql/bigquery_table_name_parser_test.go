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

package bigqueryexecutesql_test

import (
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/tools/bigquery/bigqueryexecutesql"
)

func TestTableParser(t *testing.T) {
	testCases := []struct {
		name             string
		sql              string
		defaultProjectID string
		want             []string
		wantErr          bool
		wantErrMsg       string
	}{
		{
			name:             "single fully qualified table",
			sql:              "SELECT * FROM `my-project.my_dataset.my_table`",
			defaultProjectID: "default-proj",
			want:             []string{"my-project.my_dataset.my_table"},
			wantErr:          false,
		},
		{
			name:             "multiple statements with same table",
			sql:              "select * from proj1.data1.tbl1 limit 1; select A.b from proj1.data1.tbl1 as A limit 1;",
			defaultProjectID: "default-proj",
			want:             []string{"proj1.data1.tbl1"},
			wantErr:          false,
		},
		{
			name:             "multiple fully qualified tables",
			sql:              "SELECT * FROM `proj1.data1`.`tbl1` JOIN proj2.`data2.tbl2` ON id",
			defaultProjectID: "default-proj",
			want:             []string{"proj1.data1.tbl1", "proj2.data2.tbl2"},
			wantErr:          false,
		},
		{
			name:             "duplicate tables",
			sql:              "SELECT * FROM `proj1.data1.tbl1` JOIN proj1.data1.tbl1 ON id",
			defaultProjectID: "default-proj",
			want:             []string{"proj1.data1.tbl1"},
			wantErr:          false,
		},
		{
			name:             "partial table with default project",
			sql:              "SELECT * FROM `my_dataset`.my_table",
			defaultProjectID: "default-proj",
			want:             []string{"default-proj.my_dataset.my_table"},
			wantErr:          false,
		},
		{
			name:             "partial table without default project",
			sql:              "SELECT * FROM `my_dataset.my_table`",
			defaultProjectID: "",
			want:             nil,
			wantErr:          true,
		},
		{
			name:             "mixed fully qualified and partial tables",
			sql:              "SELECT t1.*, t2.* FROM `proj1.data1.tbl1` AS t1 JOIN `data2.tbl2` AS t2 ON t1.id = t2.id",
			defaultProjectID: "default-proj",
			want:             []string{"proj1.data1.tbl1", "default-proj.data2.tbl2"},
			wantErr:          false,
		},
		{
			name:             "no tables",
			sql:              "SELECT 1+1",
			defaultProjectID: "default-proj",
			want:             []string{},
			wantErr:          false,
		},
		{
			name:             "ignore single part identifiers (like CTEs)",
			sql:              "WITH my_cte AS (SELECT 1) SELECT * FROM `my_cte`",
			defaultProjectID: "default-proj",
			want:             []string{},
			wantErr:          false,
		},
		{
			name:             "complex CTE",
			sql:              "WITH cte1 AS (SELECT * FROM `real.table.one`), cte2 AS (SELECT * FROM cte1) SELECT * FROM cte2 JOIN `real.table.two` ON true",
			defaultProjectID: "default-proj",
			want:             []string{"real.table.one", "real.table.two"},
			wantErr:          false,
		},
		{
			name:             "nested subquery should be parsed",
			sql:              "SELECT * FROM (SELECT a FROM (SELECT A.b FROM `real.table.nested` AS A))",
			defaultProjectID: "default-proj",
			want:             []string{"real.table.nested"},
			wantErr:          false,
		},
		{
			name:             "from clause with unnest",
			sql:              "SELECT event.name FROM `my-project.my_dataset.my_table` AS A, UNNEST(A.events) AS event",
			defaultProjectID: "default-proj",
			want:             []string{"my-project.my_dataset.my_table"},
			wantErr:          false,
		},
		{
			name:             "ignore more than 3 parts",
			sql:              "SELECT * FROM `proj.data.tbl.col`",
			defaultProjectID: "default-proj",
			want:             []string{},
			wantErr:          false,
		},
		{
			name:             "complex query",
			sql:              "SELECT name FROM (SELECT name FROM `proj1.data1.tbl1`) UNION ALL SELECT name FROM `data2.tbl2`",
			defaultProjectID: "default-proj",
			want:             []string{"proj1.data1.tbl1", "default-proj.data2.tbl2"},
			wantErr:          false,
		},
		{
			name:             "empty sql",
			sql:              "",
			defaultProjectID: "default-proj",
			want:             []string{},
			wantErr:          false,
		},
		{
			name:             "with comments",
			sql:              "SELECT * FROM `proj1.data1.tbl1`; -- comment `fake.table.one` \n SELECT * FROM `proj2.data2.tbl2`; # comment `fake.table.two`",
			defaultProjectID: "default-proj",
			want:             []string{"proj1.data1.tbl1", "proj2.data2.tbl2"},
			wantErr:          false,
		},
		{
			name:             "multi-statement with semicolon",
			sql:              "SELECT * FROM `proj1.data1.tbl1`; SELECT * FROM `proj2.data2.tbl2`",
			defaultProjectID: "default-proj",
			want:             []string{"proj1.data1.tbl1", "proj2.data2.tbl2"},
			wantErr:          false,
		},
		{
			name:             "simple execute immediate",
			sql:              "EXECUTE IMMEDIATE 'SELECT * FROM `exec.proj.tbl`'",
			defaultProjectID: "default-proj",
			want:             nil,
			wantErr:          true,
			wantErrMsg:       "EXECUTE IMMEDIATE is not allowed when dataset restrictions are in place",
		},
		{
			name:             "execute immediate with multiple spaces",
			sql:              "EXECUTE  IMMEDIATE 'SELECT 1'",
			defaultProjectID: "default-proj",
			want:             nil,
			wantErr:          true,
			wantErrMsg:       "EXECUTE IMMEDIATE is not allowed when dataset restrictions are in place",
		},
		{
			name:             "execute immediate with newline",
			sql:              "EXECUTE\nIMMEDIATE 'SELECT 1'",
			defaultProjectID: "default-proj",
			want:             nil,
			wantErr:          true,
			wantErrMsg:       "EXECUTE IMMEDIATE is not allowed when dataset restrictions are in place",
		},
		{
			name:             "execute immediate with comment",
			sql:              "EXECUTE -- some comment\n IMMEDIATE 'SELECT * FROM `exec.proj.tbl`'",
			defaultProjectID: "default-proj",
			want:             nil,
			wantErr:          true,
			wantErrMsg:       "EXECUTE IMMEDIATE is not allowed when dataset restrictions are in place",
		},
		{
			name:             "nested execute immediate",
			sql:              "EXECUTE IMMEDIATE \"EXECUTE IMMEDIATE '''SELECT * FROM `nested.exec.tbl`'''\"",
			defaultProjectID: "default-proj",
			want:             nil,
			wantErr:          true,
			wantErrMsg:       "EXECUTE IMMEDIATE is not allowed when dataset restrictions are in place",
		},
		{
			name:             "begin execute immediate",
			sql:              "BEGIN EXECUTE IMMEDIATE 'SELECT * FROM `exec.proj.tbl`'; END;",
			defaultProjectID: "default-proj",
			want:             nil,
			wantErr:          true,
			wantErrMsg:       "EXECUTE IMMEDIATE is not allowed when dataset restrictions are in place",
		},
		{
			name:             "table inside string literal should be ignored",
			sql:              "SELECT * FROM `real.table.one` WHERE name = 'select * from `fake.table.two`'",
			defaultProjectID: "default-proj",
			want:             []string{"real.table.one"},
			wantErr:          false,
		},
		{
			name:             "string with escaped single quote",
			sql:              "SELECT 'this is a string with an escaped quote \\' and a fake table `fake.table.one`' FROM `real.table.two`",
			defaultProjectID: "default-proj",
			want:             []string{"real.table.two"},
			wantErr:          false,
		},
		{
			name:             "string with escaped double quote",
			sql:              `SELECT "this is a string with an escaped quote \" and a fake table ` + "`fake.table.one`" + `" FROM ` + "`real.table.two`",
			defaultProjectID: "default-proj",
			want:             []string{"real.table.two"},
			wantErr:          false,
		},
		{
			name:             "multi-line comment",
			sql:              "/* `fake.table.1` */ SELECT * FROM `real.table.2`",
			defaultProjectID: "default-proj",
			want:             []string{"real.table.2"},
			wantErr:          false,
		},
		{
			name:             "raw string with backslash should be ignored",
			sql:              "SELECT * FROM `real.table.one` WHERE name = r'a raw string with a \\ and a fake table `fake.table.two`'",
			defaultProjectID: "default-proj",
			want:             []string{"real.table.one"},
			wantErr:          false,
		},
		{
			name:             "capital R raw string with quotes inside should be ignored",
			sql:              `SELECT * FROM ` + "`real.table.one`" + ` WHERE name = R"""a raw string with a ' and a " and a \ and a fake table ` + "`fake.table.two`" + `"""`,
			defaultProjectID: "default-proj",
			want:             []string{"real.table.one"},
			wantErr:          false,
		},
		{
			name:             "triple quoted raw string should be ignored",
			sql:              "SELECT * FROM `real.table.one` WHERE name = r'''a raw string with a ' and a \" and a \\ and a fake table `fake.table.two`'''",
			defaultProjectID: "default-proj",
			want:             []string{"real.table.one"},
			wantErr:          false,
		},
		{
			name:             "triple quoted capital R raw string should be ignored",
			sql:              `SELECT * FROM ` + "`real.table.one`" + ` WHERE name = R"""a raw string with a ' and a " and a \ and a fake table ` + "`fake.table.two`" + `"""`,
			defaultProjectID: "default-proj",
			want:             []string{"real.table.one"},
			wantErr:          false,
		},
		{
			name:             "unquoted fully qualified table",
			sql:              "SELECT * FROM my-project.my_dataset.my_table",
			defaultProjectID: "default-proj",
			want:             []string{"my-project.my_dataset.my_table"},
			wantErr:          false,
		},
		{
			name:             "unquoted partial table with default project",
			sql:              "SELECT * FROM my_dataset.my_table",
			defaultProjectID: "default-proj",
			want:             []string{"default-proj.my_dataset.my_table"},
			wantErr:          false,
		},
		{
			name:             "unquoted partial table without default project",
			sql:              "SELECT * FROM my_dataset.my_table",
			defaultProjectID: "",
			want:             nil,
			wantErr:          true,
		},
		{
			name:             "mixed quoting style 1",
			sql:              "SELECT * FROM `my-project`.my_dataset.my_table",
			defaultProjectID: "default-proj",
			want:             []string{"my-project.my_dataset.my_table"},
			wantErr:          false,
		},
		{
			name:             "mixed quoting style 2",
			sql:              "SELECT * FROM `my-project`.`my_dataset`.my_table",
			defaultProjectID: "default-proj",
			want:             []string{"my-project.my_dataset.my_table"},
			wantErr:          false,
		},
		{
			name:             "mixed quoting style 3",
			sql:              "SELECT * FROM `my-project`.`my_dataset`.`my_table`",
			defaultProjectID: "default-proj",
			want:             []string{"my-project.my_dataset.my_table"},
			wantErr:          false,
		},
		{
			name:             "mixed quoted and unquoted tables",
			sql:              "SELECT * FROM `proj1.data1.tbl1` JOIN proj2.data2.tbl2 ON id",
			defaultProjectID: "default-proj",
			want:             []string{"proj1.data1.tbl1", "proj2.data2.tbl2"},
			wantErr:          false,
		},
		{
			name:             "create table statement",
			sql:              "CREATE TABLE `my-project.my_dataset.my_table` (x INT64)",
			defaultProjectID: "default-proj",
			want:             []string{"my-project.my_dataset.my_table"},
			wantErr:          false,
		},
		{
			name:             "insert into statement",
			sql:              "INSERT INTO `my-project.my_dataset.my_table` (x) VALUES (1)",
			defaultProjectID: "default-proj",
			want:             []string{"my-project.my_dataset.my_table"},
			wantErr:          false,
		},
		{
			name:             "update statement",
			sql:              "UPDATE `my-project.my_dataset.my_table` SET x = 2 WHERE true",
			defaultProjectID: "default-proj",
			want:             []string{"my-project.my_dataset.my_table"},
			wantErr:          false,
		},
		{
			name:             "delete from statement",
			sql:              "DELETE FROM `my-project.my_dataset.my_table` WHERE true",
			defaultProjectID: "default-proj",
			want:             []string{"my-project.my_dataset.my_table"},
			wantErr:          false,
		},
		{
			name:             "merge into statement",
			sql:              "MERGE `proj.data.target` T USING `proj.data.source` S ON T.id = S.id WHEN NOT MATCHED THEN INSERT ROW",
			defaultProjectID: "default-proj",
			want:             []string{"proj.data.source", "proj.data.target"},
			wantErr:          false,
		},
		{
			name:             "create schema statement",
			sql:              "CREATE SCHEMA `my-project.my_dataset`",
			defaultProjectID: "default-proj",
			want:             nil,
			wantErr:          true,
			wantErrMsg:       "dataset-level operations like 'CREATE SCHEMA' are not allowed",
		},
		{
			name:             "create dataset statement",
			sql:              "CREATE DATASET `my-project.my_dataset`",
			defaultProjectID: "default-proj",
			want:             nil,
			wantErr:          true,
			wantErrMsg:       "dataset-level operations like 'CREATE DATASET' are not allowed",
		},
		{
			name:             "drop schema statement",
			sql:              "DROP SCHEMA `my-project.my_dataset`",
			defaultProjectID: "default-proj",
			want:             nil,
			wantErr:          true,
			wantErrMsg:       "dataset-level operations like 'DROP SCHEMA' are not allowed",
		},
		{
			name:             "drop dataset statement",
			sql:              "DROP DATASET `my-project.my_dataset`",
			defaultProjectID: "default-proj",
			want:             nil,
			wantErr:          true,
			wantErrMsg:       "dataset-level operations like 'DROP DATASET' are not allowed",
		},
		{
			name:             "alter schema statement",
			sql:              "ALTER SCHEMA my_dataset SET OPTIONS(description='new description')",
			defaultProjectID: "default-proj",
			want:             nil,
			wantErr:          true,
			wantErrMsg:       "dataset-level operations like 'ALTER SCHEMA' are not allowed",
		},
		{
			name:             "alter dataset statement",
			sql:              "ALTER DATASET my_dataset SET OPTIONS(description='new description')",
			defaultProjectID: "default-proj",
			want:             nil,
			wantErr:          true,
			wantErrMsg:       "dataset-level operations like 'ALTER DATASET' are not allowed",
		},
		{
			name:             "begin...end block",
			sql:              "BEGIN CREATE TABLE `proj.data.tbl1` (x INT64); INSERT `proj.data.tbl2` (y) VALUES (1); END;",
			defaultProjectID: "default-proj",
			want:             []string{"proj.data.tbl1", "proj.data.tbl2"},
			wantErr:          false,
		},
		{
			name: "complex begin...end block with comments and different quoting",
			sql: `
				BEGIN
					-- Create a new table
					CREATE TABLE proj.data.tbl1 (x INT64);
					/* Insert some data from another table */
					INSERT INTO ` + "`proj.data.tbl2`" + ` (y) SELECT y FROM proj.data.source;
				END;`,
			defaultProjectID: "default-proj",
			want:             []string{"proj.data.source", "proj.data.tbl1", "proj.data.tbl2"},
			wantErr:          false,
		},
		{
			name:             "call fully qualified procedure",
			sql:              "CALL my-project.my_dataset.my_procedure()",
			defaultProjectID: "default-proj",
			want:             nil,
			wantErr:          true,
			wantErrMsg:       "CALL is not allowed when dataset restrictions are in place",
		},
		{
			name:             "call partially qualified procedure",
			sql:              "CALL my_dataset.my_procedure()",
			defaultProjectID: "default-proj",
			want:             nil,
			wantErr:          true,
			wantErrMsg:       "CALL is not allowed when dataset restrictions are in place",
		},
		{
			name:             "call procedure in begin...end block",
			sql:              "BEGIN CALL proj.data.proc1(); SELECT * FROM proj.data.tbl1; END;",
			defaultProjectID: "default-proj",
			want:             nil,
			wantErr:          true,
			wantErrMsg:       "CALL is not allowed when dataset restrictions are in place",
		},
		{
			name:             "call procedure with newline",
			sql:              "CALL\nmy_dataset.my_procedure()",
			defaultProjectID: "default-proj",
			want:             nil,
			wantErr:          true,
			wantErrMsg:       "CALL is not allowed when dataset restrictions are in place",
		},
		{
			name:             "call procedure without default project should fail",
			sql:              "CALL my_dataset.my_procedure()",
			defaultProjectID: "",
			want:             nil,
			wantErr:          true,
			wantErrMsg:       "CALL is not allowed when dataset restrictions are in place",
		},
		{
			name:             "create procedure statement",
			sql:              "CREATE PROCEDURE my_dataset.my_procedure() BEGIN SELECT 1; END;",
			defaultProjectID: "default-proj",
			want:             nil,
			wantErr:          true,
			wantErrMsg:       "unanalyzable statements like 'CREATE PROCEDURE' are not allowed",
		},
		{
			name:             "create or replace procedure statement",
			sql:              "CREATE\n OR \nREPLACE \nPROCEDURE my_dataset.my_procedure() BEGIN SELECT 1; END;",
			defaultProjectID: "default-proj",
			want:             nil,
			wantErr:          true,
			wantErrMsg:       "unanalyzable statements like 'CREATE OR REPLACE PROCEDURE' are not allowed",
		},
		{
			name:             "create function statement",
			sql:              "CREATE FUNCTION my_dataset.my_function() RETURNS INT64 AS (1);",
			defaultProjectID: "default-proj",
			want:             nil,
			wantErr:          true,
			wantErrMsg:       "unanalyzable statements like 'CREATE FUNCTION' are not allowed",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := bigqueryexecutesql.TableParser(tc.sql, tc.defaultProjectID)
			if (err != nil) != tc.wantErr {
				t.Errorf("TableParser() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if tc.wantErr && tc.wantErrMsg != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErrMsg) {
					t.Errorf("TableParser() error = %v, want err containing %q", err, tc.wantErrMsg)
				}
			}
			// Sort slices to ensure comparison is order-independent.
			sort.Strings(got)
			sort.Strings(tc.want)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("TableParser() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
