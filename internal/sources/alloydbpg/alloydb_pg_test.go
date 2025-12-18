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

package alloydbpg_test

import (
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/alloydbpg"
	"github.com/googleapis/genai-toolbox/internal/testutils"
)

func TestParseFromYamlAlloyDBPg(t *testing.T) {
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
					kind: alloydb-postgres
					project: my-project
					region: my-region
					cluster: my-cluster
					instance: my-instance
					database: my_db
					user: my_user
					password: my_pass
			`,
			want: map[string]sources.SourceConfig{
				"my-pg-instance": alloydbpg.Config{
					Name:     "my-pg-instance",
					Kind:     alloydbpg.SourceKind,
					Project:  "my-project",
					Region:   "my-region",
					Cluster:  "my-cluster",
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
				my-pg-instance:
					kind: alloydb-postgres
					project: my-project
					region: my-region
					cluster: my-cluster
					instance: my-instance
					ipType: Public
					database: my_db
					user: my_user
					password: my_pass
			`,
			want: map[string]sources.SourceConfig{
				"my-pg-instance": alloydbpg.Config{
					Name:     "my-pg-instance",
					Kind:     alloydbpg.SourceKind,
					Project:  "my-project",
					Region:   "my-region",
					Cluster:  "my-cluster",
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
				my-pg-instance:
					kind: alloydb-postgres
					project: my-project
					region: my-region
					cluster: my-cluster
					instance: my-instance
					ipType: private
					database: my_db
					user: my_user
					password: my_pass
			`,
			want: map[string]sources.SourceConfig{
				"my-pg-instance": alloydbpg.Config{
					Name:     "my-pg-instance",
					Kind:     alloydbpg.SourceKind,
					Project:  "my-project",
					Region:   "my-region",
					Cluster:  "my-cluster",
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
				my-pg-instance:
					kind: alloydb-postgres
					project: my-project
					region: my-region
					cluster: my-cluster
					instance: my-instance
					ipType: fail 
					database: my_db
					user: my_user
					password: my_pass
			`,
			err: "unable to parse source \"my-pg-instance\" as \"alloydb-postgres\": ipType invalid: must be one of \"public\", \"private\", or \"psc\"",
		},
		{
			desc: "extra field",
			in: `
			sources:
				my-pg-instance:
					kind: alloydb-postgres
					project: my-project
					region: my-region
					cluster: my-cluster
					instance: my-instance
					database: my_db
					user: my_user
					password: my_pass
					foo: bar
			`,
			err: "unable to parse source \"my-pg-instance\" as \"alloydb-postgres\": [3:1] unknown field \"foo\"\n   1 | cluster: my-cluster\n   2 | database: my_db\n>  3 | foo: bar\n       ^\n   4 | instance: my-instance\n   5 | kind: alloydb-postgres\n   6 | password: my_pass\n   7 | ",
		},
		{
			desc: "missing required field",
			in: `
			sources:
				my-pg-instance:
					kind: alloydb-postgres
					region: my-region
					cluster: my-cluster
					instance: my-instance
					database: my_db
					user: my_user
					password: my_pass
			`,
			err: "unable to parse source \"my-pg-instance\" as \"alloydb-postgres\": Key: 'Config.Project' Error:Field validation for 'Project' failed on the 'required' tag",
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
