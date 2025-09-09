// Copyright 2024 Google LLC
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

package postgres_test

import (
	"sort"
	"strings"
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/sources/postgres"
	"github.com/googleapis/genai-toolbox/internal/testutils"
)

func TestParseFromYamlPostgres(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		want server.SourceConfigs
	}{
		{
			desc: "basic example",
			in: `
			sources:
				my-pg-instance:
					type: postgres
					host: my-host
					port: my-port
					database: my_db
					user: my_user
					password: my_pass
			`,
			want: server.SourceConfigs{
				"my-pg-instance": postgres.Config{
					Name:     "my-pg-instance",
					Type:     postgres.SourceType,
					Host:     "my-host",
					Port:     "my-port",
					Database: "my_db",
					User:     "my_user",
					Password: "my_pass",
				},
			},
		},
		{
			desc: "example with query params",
			in: `
			sources:
				my-pg-instance:
					type: postgres
					host: my-host
					port: my-port
					database: my_db
					user: my_user
					password: my_pass
					queryParams:
						sslmode: verify-full
						sslrootcert: /tmp/ca.crt
			`,
			want: server.SourceConfigs{
				"my-pg-instance": postgres.Config{
					Name:     "my-pg-instance",
					Type:     postgres.SourceType,
					Host:     "my-host",
					Port:     "my-port",
					Database: "my_db",
					User:     "my_user",
					Password: "my_pass",
					QueryParams: map[string]string{
						"sslmode":     "verify-full",
						"sslrootcert": "/tmp/ca.crt",
					},
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
				my-pg-instance:
					type: postgres
					host: my-host
					port: my-port
					database: my_db
					user: my_user
					password: my_pass
					foo: bar
			`,
			err: "unable to parse source \"my-pg-instance\" as \"postgres\": [2:1] unknown field \"foo\"\n   1 | database: my_db\n>  2 | foo: bar\n       ^\n   3 | host: my-host\n   4 | type: postgres\n   5 | password: my_pass\n   6 | ",
		},
		{
			desc: "missing required field",
			in: `
			sources:
				my-pg-instance:
					type: postgres
					host: my-host
					port: my-port
					database: my_db
					user: my_user
			`,
			err: "unable to parse source \"my-pg-instance\" as \"postgres\": Key: 'Config.Password' Error:Field validation for 'Password' failed on the 'required' tag",
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

func TestConvertParamMapToRawQuery(t *testing.T) {
	tcs := []struct {
		desc string
		in   map[string]string
		want string
	}{
		{
			desc: "nil param",
			in:   nil,
			want: "",
		},
		{
			desc: "single query param",
			in: map[string]string{
				"foo": "bar",
			},
			want: "foo=bar",
		},
		{
			desc: "more than one query param",
			in: map[string]string{
				"foo":   "bar",
				"hello": "world",
			},
			want: "foo=bar&hello=world",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := postgres.ConvertParamMapToRawQuery(tc.in)
			if strings.Contains(got, "&") {
				splitGot := strings.Split(got, "&")
				sort.Strings(splitGot)
				got = strings.Join(splitGot, "&")
			}
			if got != tc.want {
				t.Fatalf("incorrect conversion: got %s want %s", got, tc.want)
			}
		})
	}
}
