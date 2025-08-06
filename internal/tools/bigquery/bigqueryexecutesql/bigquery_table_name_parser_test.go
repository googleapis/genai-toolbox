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
	}{
		{
			name:             "single fully qualified table",
			sql:              "SELECT * FROM `my-project.my_dataset.my_table`",
			defaultProjectID: "default-proj",
			want:             []string{"my-project.my_dataset.my_table"},
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
			sql:              "SELECT * FROM `my_dataset.my_table`",
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
		},
		{
			name:             "execute immediate with multiple spaces",
			sql:              "EXECUTE  IMMEDIATE 'SELECT 1'",
			defaultProjectID: "default-proj",
			want:             nil,
			wantErr:          true,
		},
		{
			name:             "execute immediate with newline",
			sql:              "EXECUTE\nIMMEDIATE 'SELECT 1'",
			defaultProjectID: "default-proj",
			want:             nil,
			wantErr:          true,
		},
		{
			name:             "nested execute immediate",
			sql:              "EXECUTE IMMEDIATE \"EXECUTE IMMEDIATE '''SELECT * FROM `nested.exec.tbl`'''\"",
			defaultProjectID: "default-proj",
			want:             nil,
			wantErr:          true,
		},
		{
			name:             "begin execute immediate",
			sql:              "BEGIN EXECUTE IMMEDIATE 'SELECT * FROM `exec.proj.tbl`'; END;",
			defaultProjectID: "default-proj",
			want:             nil,
			wantErr:          true,
		},
		{
			name:             "table inside string literal should be ignored",
			sql:              "SELECT * FROM `real.table.one` WHERE name = 'select * from `fake.table.two`'",
			defaultProjectID: "default-proj",
			want:             []string{"real.table.one"},
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
			sql:              "CALL my-project.my_dataset.my_procedure(1, 'foo')",
			defaultProjectID: "default-proj",
			want:             []string{"my-project.my_dataset.my_procedure"},
			wantErr:          false,
		},
		{
			name:             "call partially qualified procedure",
			sql:              "CALL my_dataset.my_procedure()",
			defaultProjectID: "default-proj",
			want:             []string{"default-proj.my_dataset.my_procedure"},
			wantErr:          false,
		},
		{
			name:             "call procedure in begin...end block",
			sql:              "BEGIN CALL proj.data.proc1(); SELECT * FROM proj.data.tbl1; END;",
			defaultProjectID: "default-proj",
			want:             []string{"proj.data.proc1", "proj.data.tbl1"},
			wantErr:          false,
		},
		{
			name:             "call procedure without default project should fail",
			sql:              "CALL my_dataset.my_procedure()",
			defaultProjectID: "",
			want:             nil,
			wantErr:          true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := bigqueryexecutesql.TableParser(tc.sql, tc.defaultProjectID)
			if (err != nil) != tc.wantErr {
				t.Errorf("TableParser() error = %v, wantErr %v", err, tc.wantErr)
				return
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
