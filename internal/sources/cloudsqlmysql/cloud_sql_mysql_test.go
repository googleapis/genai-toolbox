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

package cloudsqlmysql_test

import (
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/sources/cloudsqlmysql"
	"github.com/googleapis/genai-toolbox/internal/testutils"
)

func TestParseFromYamlCloudSQLMySQL(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		want server.SourceConfigs
	}{
		{
			desc: "basic example",
			in: `
			sources:
				my-mysql-instance:
					kind: cloud-sql-mysql
					project: my-project
					region: my-region
					instance: my-instance
					database: my_db
					user: my_user
					password: my_pass
			`,
			want: server.SourceConfigs{
				"my-mysql-instance": cloudsqlmysql.Config{
					Name:     "my-mysql-instance",
					Type:     cloudsqlmysql.SourceType,
					Project:  "my-project",
					Region:   "my-region",
					Instance: "my-instance",
					IPType:   "public",
					Database: "my_db",
					User:     "my_user",
					Password: "my_pass",
				},
			},
		},
		{
			desc: "public ipType",
			in: `
			sources:
				my-mysql-instance:
					kind: cloud-sql-mysql
					project: my-project
					region: my-region
					instance: my-instance
					ipType: Public
					database: my_db
					user: my_user
					password: my_pass
			`,
			want: server.SourceConfigs{
				"my-mysql-instance": cloudsqlmysql.Config{
					Name:     "my-mysql-instance",
					Type:     cloudsqlmysql.SourceType,
					Project:  "my-project",
					Region:   "my-region",
					Instance: "my-instance",
					IPType:   "public",
					Database: "my_db",
					User:     "my_user",
					Password: "my_pass",
				},
			},
		},
		{
			desc: "private ipType",
			in: `
			sources:
				my-mysql-instance:
					kind: cloud-sql-mysql
					project: my-project
					region: my-region
					instance: my-instance
					ipType: private 
					database: my_db
					user: my_user
					password: my_pass
			`,
			want: server.SourceConfigs{
				"my-mysql-instance": cloudsqlmysql.Config{
					Name:     "my-mysql-instance",
					Type:     cloudsqlmysql.SourceType,
					Project:  "my-project",
					Region:   "my-region",
					Instance: "my-instance",
					IPType:   "private",
					Database: "my_db",
					User:     "my_user",
					Password: "my_pass",
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
			desc: "invalid ipType",
			in: `
			sources:
				my-mysql-instance:
					kind: cloud-sql-mysql
					project: my-project
					region: my-region
					instance: my-instance
					ipType: fail 
					database: my_db
					user: my_user
					password: my_pass
			`,
			err: "unable to parse source \"my-mysql-instance\" as \"cloud-sql-mysql\": ipType invalid: must be one of \"public\", or \"private\"",
		},
		{
			desc: "extra field",
			in: `
			sources:
				my-mysql-instance:
					kind: cloud-sql-mysql
					project: my-project
					region: my-region
					instance: my-instance
					database: my_db
					user: my_user
					password: my_pass
					foo: bar
			`,
			err: "unable to parse source \"my-mysql-instance\" as \"cloud-sql-mysql\": [2:1] unknown field \"foo\"\n   1 | database: my_db\n>  2 | foo: bar\n       ^\n   3 | instance: my-instance\n   4 | kind: cloud-sql-mysql\n   5 | password: my_pass\n   6 | ",
		},
		{
			desc: "missing required field",
			in: `
			sources:
				my-mysql-instance:
					kind: cloud-sql-mysql
					region: my-region
					instance: my-instance
					database: my_db
					user: my_user
					password: my_pass
			`,
			err: "unable to parse source \"my-mysql-instance\" as \"cloud-sql-mysql\": Key: 'Config.Project' Error:Field validation for 'Project' failed on the 'required' tag",
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
