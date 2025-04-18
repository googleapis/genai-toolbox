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

package couchbase_test

import (
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/sources/couchbase"
	"github.com/googleapis/genai-toolbox/internal/testutils"
)

func TestParseFromYamlCouchbase(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		want server.SourceConfigs
	}{
		{
			desc: "basic example",
			in: `
			sources:
				my-couchbase-instance:
					kind: couchbase
					connectionString: localhost
					username: Administrator
					password: password
					bucket: travel-sample
					scope: inventory
			`,
			want: server.SourceConfigs{
				"my-couchbase-instance": couchbase.Config{
					Name:             "my-couchbase-instance",
					Kind:             couchbase.SourceKind,
					ConnectionString: "localhost",
					Username:         "Administrator",
					Password:         "password",
					Bucket:           "travel-sample",
					Scope:            "inventory",
				},
			},
		},
		{
			desc: "with TLS configuration",
			in: `
			sources:
				my-couchbase-instance:
					kind: couchbase
					connectionString: couchbases://localhost
					bucket: travel-sample
					scope: inventory
					client_cert: /path/to/cert.pem
					client_key: /path/to/key.pem
					client_cert_password: password
					client_key_password: password
					ca_cert: /path/to/ca.pem
					no_ssl_verify: false
			`,
			want: server.SourceConfigs{
				"my-couchbase-instance": couchbase.Config{
					Name:               "my-couchbase-instance",
					Kind:               couchbase.SourceKind,
					ConnectionString:   "couchbases://localhost",
					Bucket:             "travel-sample",
					Scope:              "inventory",
					ClientCert:         "/path/to/cert.pem",
					ClientKey:          "/path/to/key.pem",
					ClientCertPassword: "password",
					ClientKeyPassword:  "password",
					CACert:             "/path/to/ca.pem",
					NoSSLVerify:        false,
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
				my-couchbase-instance:
					kind: couchbase
					connectionString: localhost
					username: Administrator
					password: password
					bucket: travel-sample
					scope: inventory
					foo: bar
			`,
			err: "unable to parse as \"couchbase\": [3:1] unknown field \"foo\"\n   1 | bucket: travel-sample\n   2 | connectionString: localhost\n>  3 | foo: bar\n       ^\n   4 | kind: couchbase\n   5 | password: password\n   6 | scope: inventory\n   7 | ",
		},
		{
			desc: "missing required field",
			in: `
			sources:
				my-couchbase-instance:
					kind: couchbase
					username: Administrator
					password: password
					bucket: travel-sample
					scope: inventory
			`,
			err: "unable to parse as \"couchbase\": Key: 'Config.ConnectionString' Error:Field validation for 'ConnectionString' failed on the 'required' tag",
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
