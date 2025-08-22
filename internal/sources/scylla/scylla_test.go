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

package scylla

import (
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
)

func TestParseFromYamlScylla(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		want server.SourceConfigs
	}{
		{
			desc: "basic example",
			in: `
			sources:
				my-scylla-instance:
					kind: scylla
					hosts:
						- localhost
					port: "9042"
					keyspace: mykeyspace
			`,
			want: server.SourceConfigs{
				"my-scylla-instance": Config{
					Name:     "my-scylla-instance",
					Kind:     SourceKind,
					Hosts:    []string{"localhost"},
					Port:     "9042",
					Keyspace: "mykeyspace",
				},
			},
		},
		{
			desc: "example with multiple hosts and authentication",
			in: `
			sources:
				my-scylla-cluster:
					kind: scylla
					hosts:
						- host1.example.com
						- host2.example.com
						- host3.example.com
					port: "9042"
					keyspace: mykeyspace
					username: cassandra
					password: cassandra
					consistency: QUORUM
			`,
			want: server.SourceConfigs{
				"my-scylla-cluster": Config{
					Name:        "my-scylla-cluster",
					Kind:        SourceKind,
					Hosts:       []string{"host1.example.com", "host2.example.com", "host3.example.com"},
					Port:        "9042",
					Keyspace:    "mykeyspace",
					Username:    "cassandra",
					Password:    "cassandra",
					Consistency: "QUORUM",
				},
			},
		},
		{
			desc: "example with all optional fields",
			in: `
			sources:
				my-scylla-advanced:
					kind: scylla
					hosts:
						- localhost
					port: "9042"
					keyspace: mykeyspace
					username: admin
					password: admin123
					consistency: LOCAL_QUORUM
					connectTimeout: "10s"
					timeout: "30s"
					disableInitialHostLookup: true
					numConnections: 10
					protoVersion: 4
					sslEnabled: true
			`,
			want: server.SourceConfigs{
				"my-scylla-advanced": Config{
					Name:                     "my-scylla-advanced",
					Kind:                     SourceKind,
					Hosts:                    []string{"localhost"},
					Port:                     "9042",
					Keyspace:                 "mykeyspace",
					Username:                 "admin",
					Password:                 "admin123",
					Consistency:              "LOCAL_QUORUM",
					ConnectTimeout:           "10s",
					Timeout:                  "30s",
					DisableInitialHostLookup: true,
					NumConnections:           10,
					ProtoVersion:             4,
					SSLEnabled:               true,
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
