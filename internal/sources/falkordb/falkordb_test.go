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
	"github.com/googleapis/mcp-toolbox/internal/server"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/sources/falkordb"
	"github.com/googleapis/mcp-toolbox/internal/testutils"
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
			kind: source
			name: my-falkordb-instance
			type: falkordb
			host: my-host
			port: "6379"
			graph: my_graph
			`,
			want: map[string]sources.SourceConfig{
				"my-falkordb-instance": falkordb.Config{
					Name:  "my-falkordb-instance",
					Type:  falkordb.SourceType,
					Host:  "my-host",
					Port:  "6379",
					Graph: "my_graph",
				},
			},
		},
		{
			desc: "with auth, timeout and TLS",
			in: `
			kind: source
			name: my-falkordb-instance
			type: falkordb
			host: my-host
			port: "6380"
			username: my_user
			password: my_pass
			graph: my_graph
			queryTimeoutMs: 5000
			tls:
			    enabled: true
			    insecureSkipVerify: true
			`,
			want: map[string]sources.SourceConfig{
				"my-falkordb-instance": falkordb.Config{
					Name:           "my-falkordb-instance",
					Type:           falkordb.SourceType,
					Host:           "my-host",
					Port:           "6380",
					Username:       "my_user",
					Password:       "my_pass",
					Graph:          "my_graph",
					QueryTimeoutMs: 5000,
					TLS: falkordb.TLSConfig{
						Enabled:            true,
						InsecureSkipVerify: true,
					},
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got, _, _, _, _, _, err := server.UnmarshalPrimitiveConfig(context.Background(), testutils.FormatYaml(tc.in))
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
			kind: source
			name: my-falkordb-instance
			type: falkordb
			host: my-host
			port: "6379"
			graph: my_graph
			foo: bar
			`,
			err: "error unmarshaling source: unable to parse source \"my-falkordb-instance\" as \"falkordb\": [1:1] unknown field \"foo\"\n>  1 | foo: bar\n       ^\n   2 | graph: my_graph\n   3 | host: my-host\n   4 | name: my-falkordb-instance\n   5 | ",
		},
		{
			desc: "missing required field",
			in: `
			kind: source
			name: my-falkordb-instance
			type: falkordb
			host: my-host
			port: "6379"
			`,
			err: "error unmarshaling source: unable to parse source \"my-falkordb-instance\" as \"falkordb\": Key: 'Config.Graph' Error:Field validation for 'Graph' failed on the 'required' tag",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			_, _, _, _, _, _, err := server.UnmarshalPrimitiveConfig(context.Background(), testutils.FormatYaml(tc.in))
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
