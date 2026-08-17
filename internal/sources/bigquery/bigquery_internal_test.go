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

package bigquery

import (
	"reflect"
	"testing"
)

func TestMergeJobLabels(t *testing.T) {
	cases := []struct {
		desc      string
		explicit  map[string]string
		commenter map[string]string
		want      map[string]string
	}{
		{
			desc:      "both nil",
			explicit:  nil,
			commenter: nil,
			want:      nil,
		},
		{
			desc:      "no commenter labels returns explicit unchanged",
			explicit:  map[string]string{"mcp-toolbox-tool": "bigquery-execute-sql"},
			commenter: nil,
			want:      map[string]string{"mcp-toolbox-tool": "bigquery-execute-sql"},
		},
		{
			desc:      "commenter labels only",
			explicit:  nil,
			commenter: map[string]string{"tool_name": "execute_sql"},
			want:      map[string]string{"tool_name": "execute_sql"},
		},
		{
			desc:      "disjoint keys merged",
			explicit:  map[string]string{"mcp-toolbox-tool": "bigquery-execute-sql"},
			commenter: map[string]string{"tool_name": "execute_sql", "client": "test-client"},
			want: map[string]string{
				"mcp-toolbox-tool": "bigquery-execute-sql",
				"tool_name":        "execute_sql",
				"client":           "test-client",
			},
		},
		{
			desc:      "explicit wins on collision",
			explicit:  map[string]string{"tool_name": "explicit-value"},
			commenter: map[string]string{"tool_name": "commenter-value"},
			want:      map[string]string{"tool_name": "explicit-value"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got := mergeJobLabels(tc.explicit, tc.commenter)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("mergeJobLabels(%v, %v) = %v, want %v", tc.explicit, tc.commenter, got, tc.want)
			}
		})
	}
}
