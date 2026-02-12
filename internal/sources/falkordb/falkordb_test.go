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

package falkordb_test

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/falkordb"
	"github.com/googleapis/genai-toolbox/internal/testutils"
)

func TestParseFromYamlFalkorDB(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		want server.SourceConfigs
	}{
		{
			desc: "basic example",
			in: `
			kind: sources
			name: my-falkordb-instance
			type: falkordb
			addr: localhost:6379
			graph: social
			`,
			want: map[string]sources.SourceConfig{
				"my-falkordb-instance": falkordb.Config{
					Name:  "my-falkordb-instance",
					Type:  falkordb.SourceType,
					Addr:  "localhost:6379",
					Graph: "social",
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got, _, _, _, _, _, err := server.UnmarshalResourceConfig(context.Background(), testutils.FormatYaml(tc.in))
			if err != nil {
				t.Fatalf("unable to unmarshal: %s", err)
			}
			if !cmp.Equal(tc.want, got) {
				t.Fatalf("incorrect parse: want %v, got %v", tc.want, got)
			}
		})
	}
}

func TestFailParseFromYaml(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		err  string
	}{
		{
			desc: "extra field",
			in: `
			kind: sources
			name: my-falkordb-instance
			type: falkordb
			addr: localhost:6379
			graph: social
			foo: bar
			`,
			err: "error unmarshaling sources: unable to parse source \"my-falkordb-instance\" as \"falkordb\": [2:1] unknown field \"foo\"\n   1 | addr: localhost:6379\n>  2 | foo: bar\n       ^\n   3 | graph: social\n   4 | name: my-falkordb-instance\n   5 | type: falkordb",
		},
		{
			desc: "missing required field",
			in: `
			kind: sources
			name: my-falkordb-instance
			type: falkordb
			addr: localhost:6379
			`,
			err: "error unmarshaling sources: unable to parse source \"my-falkordb-instance\" as \"falkordb\": Key: 'Config.Graph' Error:Field validation for 'Graph' failed on the 'required' tag",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			_, _, _, _, _, _, err := server.UnmarshalResourceConfig(context.Background(), testutils.FormatYaml(tc.in))
			if err == nil {
				t.Fatalf("expect parsing to fail")
			}
			errStr := err.Error()
			if errStr != tc.err {
				t.Fatalf("unexpected error: got %q, want %q", errStr, tc.err)
			}
		})
	}
}
