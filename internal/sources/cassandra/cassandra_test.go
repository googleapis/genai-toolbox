// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cassandra_test

import (
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/sources/cassandra"
	"github.com/googleapis/genai-toolbox/internal/testutils"
)

func TestParseFromYamlCassandra(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		want server.SourceConfigs
	}{
		{
			desc: "basic example (without optional fields)",
			in: `
			sources:
				my-cassandra-instance:
					kind: cassandra
					hosts:
						- "my-host1"
						- "my-host2"
			`,
			want: server.SourceConfigs{
				"my-cassandra-instance": cassandra.Config{
					Name:                   "my-cassandra-instance",
					Kind:                   cassandra.SourceKind,
					Hosts:                  []string{"my-host1", "my-host2"},
					Username:               "",
					Password:               "",
					ProtoVersion:           0,
					CAPath:                 "",
					CertPath:               "",
					KeyPath:                "",
					Keyspace:               "",
					EnableHostVerification: false,
				},
			},
		},
		{
			desc: "with optional fields",
			in: `
			sources:
				my-cassandra-instance:
					kind: cassandra
					hosts:
						- "my-host1"
						- "my-host2"
					username: "user"
					password: "pass"
					keyspace: "example_keyspace"
					protoVersion: 4
					caPath: "path/to/ca.crt"
					certPath: "path/to/cert"
					keyPath: "path/to/key"
					enableHostVerification: true
			`,
			want: server.SourceConfigs{
				"my-cassandra-instance": cassandra.Config{
					Name:                   "my-cassandra-instance",
					Kind:                   cassandra.SourceKind,
					Hosts:                  []string{"my-host1", "my-host2"},
					Username:               "user",
					Password:               "pass",
					Keyspace:               "example_keyspace",
					ProtoVersion:           4,
					CAPath:                 "path/to/ca.crt",
					CertPath:               "path/to/cert",
					KeyPath:                "path/to/key",
					EnableHostVerification: true,
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := struct {
				Sources server.SourceConfigs `yaml:"sources"`
			}{}
			// Parse contents
			err := yaml.Unmarshal(testutils.FormatYaml(tc.in), &got)
			if err != nil {
				t.Fatalf("unable to unmarshal: %s", err)
			}
			if !cmp.Equal(tc.want, got.Sources) {
				t.Fatalf("incorrect parse: want %v, got %v", tc.want, got.Sources)
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
			sources:
				my-cassandra-instance:
					kind: cassandra
					hosts:
						- "my-host"
					foo: bar
			`,
			err: "unable to parse source \"my-cassandra-instance\" as \"cassandra\": [1:1] unknown field \"foo\"\n>  1 | foo: bar\n       ^\n   2 | hosts:\n   3 | - my-host\n   4 | kind: cassandra",
		},
		{
			desc: "missing required field",
			in: `
			sources:
				my-cassandra-instance:
					kind: cassandra
			`,
			err: "unable to parse source \"my-cassandra-instance\" as \"cassandra\": Key: 'Config.Hosts' Error:Field validation for 'Hosts' failed on the 'required' tag",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := struct {
				Sources server.SourceConfigs `yaml:"sources"`
			}{}
			// Parse contents
			err := yaml.Unmarshal(testutils.FormatYaml(tc.in), &got)
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
