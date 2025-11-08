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

package mariadb_test

import (
	"strings"
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/sources/mariadb"
	"github.com/googleapis/genai-toolbox/internal/testutils"
)

func TestParseFromYamlMariaDB(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		want server.SourceConfigs
	}{
		{
			desc: "basic example",
			in: `
			sources:
				my-mariadb-instance:
					kind: mariadb
					host: 127.0.0.1
					port: 3306
					database: my_db
					user: my_user
					password: my_pass
			`,
			want: server.SourceConfigs{
				"my-mariadb-instance": mariadb.Config{
					Name:     "my-mariadb-instance",
					Kind:     mariadb.SourceKind,
					Host:     "127.0.0.1",
					Port:     "3306",
					Database: "my_db",
					User:     "my_user",
					Password: "my_pass",
				},
			},
		},
		{
			desc: "queryParams override",
			in: `
			sources:
				my-mariadb-instance:
					kind: mariadb
					host: 127.0.0.1
					port: 3306
					database: my_db
					user: my_user
					password: my_pass
					queryParams:
						interpolateParams: "true"
						tls: skip-verify
			`,
			want: server.SourceConfigs{
				"my-mariadb-instance": mariadb.Config{
					Name:     "my-mariadb-instance",
					Kind:     mariadb.SourceKind,
					Host:     "127.0.0.1",
					Port:     "3306",
					Database: "my_db",
					User:     "my_user",
					Password: "my_pass",
					QueryParams: map[string]string{
						"interpolateParams": "true",
						"tls":               "skip-verify",
					},
				},
			},
		},
		{
			desc: "quoted port",
			in: `
			sources:
				my-mariadb-instance:
					kind: mariadb
					host: 127.0.0.1
					port: "3306"
					database: my_db
					user: my_user
					password: my_pass
			`,
			want: server.SourceConfigs{
				"my-mariadb-instance": mariadb.Config{
					Name:     "my-mariadb-instance",
					Kind:     mariadb.SourceKind,
					Host:     "127.0.0.1",
					Port:     "3306",
					Database: "my_db",
					User:     "my_user",
					Password: "my_pass",
				},
			},
		},
		{
			desc: "empty queryParams",
			in: `
			sources:
				my-mariadb-instance:
					kind: mariadb
					host: 127.0.0.1
					port: 3306
					database: my_db
					user: my_user
					password: my_pass
					queryParams: {}
			`,
			want: server.SourceConfigs{
				"my-mariadb-instance": mariadb.Config{
					Name:        "my-mariadb-instance",
					Kind:        mariadb.SourceKind,
					Host:        "127.0.0.1",
					Port:        "3306",
					Database:    "my_db",
					User:        "my_user",
					Password:    "my_pass",
					QueryParams: map[string]string{},
				},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := struct {
				Sources server.SourceConfigs `yaml:"sources"`
			}{}

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

func TestFailParseFromYamlMariaDB(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		err  string
	}{
		{
			desc: "extra field",
			in: `
			sources:
				my-mariadb-instance:
					kind: mariadb
					host: 127.0.0.1
					port: 3306
					database: my_db
					user: my_user
					password: my_pass
					foo: bar
			`,
			err: "unable to parse source \"my-mariadb-instance\" as \"mariadb\": [2:1] unknown field \"foo\"",
		},
		{
			desc: "missing required field",
			in: `
			sources:
				my-mariadb-instance:
					kind: mariadb
					host: 127.0.0.1
					port: 3306
					database: my_db
					user: my_user
			`,
			err: "unable to parse source \"my-mariadb-instance\" as \"mariadb\": Key: 'Config.Password' Error:Field validation for 'Password' failed on the 'required' tag",
		},
		{
			desc: "queryParams wrong type",
			in: `
			sources:
				my-mariadb-instance:
					kind: mariadb
					host: 127.0.0.1
					port: 3306
					database: my_db
					user: my_user
					password: my_pass
					queryParams: 123
			`,
			err: "unable to parse source \"my-mariadb-instance\" as \"mariadb\":",
		},
		{
			desc: "kind mismatch",
			in: `
			sources:
				my-mariadb-instance:
					kind: not-mariadb
					host: 127.0.0.1
					port: 3306
					database: my_db
					user: my_user
					password: my_pass
			`,
			err: "unknown source kind",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := struct {
				Sources server.SourceConfigs `yaml:"sources"`
			}{}

			err := yaml.Unmarshal(testutils.FormatYaml(tc.in), &got)
			if err == nil {
				t.Fatalf("expect parsing to fail")
			}

			if !strings.HasPrefix(err.Error(), tc.err) {
				t.Fatalf("unexpected error: got %q, want prefix %q", err.Error(), tc.err)
			}
		})
	}
}
