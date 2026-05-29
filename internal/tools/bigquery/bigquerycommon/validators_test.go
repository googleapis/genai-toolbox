// Copyright 2026 Google LLC
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

package bigquerycommon_test

import (
	"testing"

	"github.com/googleapis/mcp-toolbox/internal/tools/bigquery/bigquerycommon"
)

func TestValidTableID(t *testing.T) {
	tcs := []struct {
		in    string
		valid bool
	}{
		// Allowed: dataset.table
		{"my_dataset.my_table", true},
		{"ds.t", true},
		{"ds_1.tbl_2", true},

		// Allowed: project.dataset.table
		{"proj.ds.tbl", true},
		{"PROJ.DS.TBL", true},
		{"my_project.my_dataset.my_table", true},

		// Rejected: hyphens (not valid in dataset/table IDs).
		{"my-project.my-dataset.my_table", false},
		{"my-project.my_dataset.my-table", false},
		{"my-project-123.dataset.table", true},

		// Rejected: only one component (no dot)
		{"my_dataset", false},
		{"", false},

		// Rejected: too many dots (4+ parts)
		{"a.b.c.d", false},

		// Rejected: injection characters
		{"dataset.table`", false},
		{"dataset.table` UNION ALL SELECT 1 --", false},
		{"dataset.table'; DROP TABLE x --", false},
		{"dataset.table\n", false},
		{"dataset.table ", false},
		{" dataset.table", false},
		{"dataset.table\t", false},

		// Rejected: backtick (closes identifier in SQL)
		{"ds.`table`", false},

		// Rejected: SQL metacharacters
		{"ds.table--", false},
		{"ds.table/*", false},
		{"ds.table;", false},
	}
	for _, tc := range tcs {
		if got := bigquerycommon.ValidTableID(tc.in); got != tc.valid {
			t.Errorf("ValidTableID(%q) = %v, want %v", tc.in, got, tc.valid)
		}
	}
}

func TestValidColumnName(t *testing.T) {
	tcs := []struct {
		in    string
		valid bool
	}{
		// Allowed: simple identifiers.
		{"sales", true},
		{"sales_col", true},
		{"_internal", true},
		{"Col1", true},
		{"A", true},
		{"is_test", true},
		{"timestamp_col", true},

		// Rejected: empty string.
		{"", false},

		// Rejected: leading digit.
		{"1col", false},

		// Rejected: SQL injection characters.
		{"col'", false},
		{"col`", false},
		{"col; DROP TABLE x", false},
		{"col UNION SELECT", false},
		{"col--", false},
		{"col/*", false},
		{"col(", false},
		{"col)", false},
		{"col/col2", false},

		// Rejected: whitespace.
		{"col name", false},
		{" col", false},
		{"col ", false},

		// Rejected: dots (not valid in unquoted column names).
		{"ds.col", false},
	}
	for _, tc := range tcs {
		if got := bigquerycommon.ValidColumnName(tc.in); got != tc.valid {
			t.Errorf("ValidColumnName(%q) = %v, want %v", tc.in, got, tc.valid)
		}
	}
}
