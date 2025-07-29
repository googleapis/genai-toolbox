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

package tidb_test

import (
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/sources/tidb"
	"github.com/googleapis/genai-toolbox/internal/testutils"
)

func TestParseFromYamlTiDB(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		want server.SourceConfigs
	}{
		{
			desc: "basic example",
			in: `
			sources:
				my-tidb-instance:
					kind: tidb
					host: 0.0.0.0
					port: my-port
					database: my_db
					user: my_user
					password: my_pass
					ssl: false
			`,
			want: server.SourceConfigs{
				"my-tidb-instance": tidb.Config{
					Name:     "my-tidb-instance",
					Kind:     tidb.SourceKind,
					Host:     "0.0.0.0",
					Port:     "my-port",
					Database: "my_db",
					User:     "my_user",
					Password: "my_pass",
					UseSSL:   false,
				},
			},
		},
		{
			desc: "with SSL enabled",
			in: `
			sources:
				my-tidb-cloud:
					kind: tidb
					host: gateway01.us-west-2.prod.aws.tidbcloud.com
					port: 4000
					database: test_db
					user: cloud_user
					password: cloud_pass
					ssl: true
			`,
			want: server.SourceConfigs{
				"my-tidb-cloud": tidb.Config{
					Name:     "my-tidb-cloud",
					Kind:     tidb.SourceKind,
					Host:     "gateway01.us-west-2.prod.aws.tidbcloud.com",
					Port:     "4000",
					Database: "test_db",
					User:     "cloud_user",
					Password: "cloud_pass",
					UseSSL:   true,
				},
			},
		},
		{
			desc: "Change SSL enabled due to TiDB Cloud host",
			in: `
			sources:
				my-tidb-cloud:
					kind: tidb
					host: gateway01.us-west-2.prod.aws.tidbcloud.com
					port: 4000
					database: test_db
					user: cloud_user
					password: cloud_pass
			`,
			want: server.SourceConfigs{
				"my-tidb-cloud": tidb.Config{
					Name:     "my-tidb-cloud",
					Kind:     tidb.SourceKind,
					Host:     "gateway01.us-west-2.prod.aws.tidbcloud.com",
					Port:     "4000",
					Database: "test_db",
					User:     "cloud_user",
					Password: "cloud_pass",
					UseSSL:   true,
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
				my-tidb-instance:
					kind: tidb
					host: 0.0.0.0
					port: my-port
					database: my_db
					user: my_user
					password: my_pass
					ssl: false
					foo: bar
			`,
			err: "unable to parse source \"my-tidb-instance\" as \"tidb\": [2:1] unknown field \"foo\"\n   1 | database: my_db\n>  2 | foo: bar\n       ^\n   3 | host: 0.0.0.0\n   4 | kind: tidb\n   5 | password: my_pass\n   6 | ",
		},
		{
			desc: "missing required field",
			in: `
			sources:
				my-tidb-instance:
					kind: tidb
					port: my-port
					database: my_db
					user: my_user
					password: my_pass
					ssl: false
			`,
			err: "unable to parse source \"my-tidb-instance\" as \"tidb\": Key: 'Config.Host' Error:Field validation for 'Host' failed on the 'required' tag",
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
