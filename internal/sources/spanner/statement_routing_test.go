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

package spanner

import "testing"

func TestShouldUseReadOnlyTransaction(t *testing.T) {
	tcs := []struct {
		name string
		sql  string
		want bool
	}{
		{name: "select statement", sql: "SELECT 1", want: true},
		{name: "with statement", sql: "WITH cte AS (SELECT 1) SELECT * FROM cte", want: true},
		{name: "update statement", sql: "UPDATE my_table SET value = 1 WHERE id = 1", want: false},
		{name: "insert statement", sql: "INSERT INTO my_table (id) VALUES (1)", want: false},
		{name: "leading whitespace", sql: "\n  WITH cte AS (SELECT 1) SELECT * FROM cte", want: true},
		{name: "leading single-line comment", sql: "-- comment\nSELECT 1", want: true},
		{name: "leading multi-line comment", sql: "/* comment */ SELECT 1", want: true},
		{name: "leading parentheses", sql: "(SELECT 1)", want: true},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldUseReadOnlyTransaction(tc.sql); got != tc.want {
				t.Fatalf("shouldUseReadOnlyTransaction(%q) = %v, want %v", tc.sql, got, tc.want)
			}
		})
	}
}
