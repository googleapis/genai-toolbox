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
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/alloydbpg"
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
            type: sources
            name: my-pg-instance
            type: alloydb-postgres
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
					Type:     alloydbpg.SourceType,
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
            type: sources
            name: my-pg-instance
            type: alloydb-postgres
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
					Type:     alloydbpg.SourceType,
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
            type: sources
            name: my-pg-instance
            type: alloydb-postgres
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
					Type:     alloydbpg.SourceType,
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
			sources, _, _, _, err := server.UnmarshalResourceConfig(context.Background(), []byte(tc.in))
			if err != nil {
				t.Fatalf("unable to unmarshal: %s", err)
			}
			if !cmp.Equal(tc.want, sources) {
				t.Fatalf("incorrect parse: want %v, got %v", tc.want, sources)
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
            type: sources
            name: my-pg-instance
            type: alloydb-postgres
            project: my-project
            region: my-region
            cluster: my-cluster
            instance: my-instance
            ipType: fail 
            database: my_db
            user: my_user
            password: my_pass
            `,
			err: "error unmarshaling sources: unable to parse source \"my-pg-instance\" as \"alloydb-postgres\": ipType invalid: must be one of \"public\", or \"private\"",
		},
		{
			desc: "extra field",
			in: `
            type: sources
            name: my-pg-instance
            type: alloydb-postgres
            project: my-project
            region: my-region
            cluster: my-cluster
            instance: my-instance
            database: my_db
            user: my_user
            password: my_pass
            foo: bar
            `,
			err: "error unmarshaling sources: unable to parse source \"my-pg-instance\" as \"alloydb-postgres\": [3:1] unknown field \"foo\"\n   1 | cluster: my-cluster\n   2 | database: my_db\n>  3 | foo: bar\n       ^\n   4 | instance: my-instance\n   5 | name: my-pg-instance\n   6 | password: my_pass\n   7 | ",
		},
		{
			desc: "missing required field",
			in: `
            type: sources
            name: my-pg-instance
            type: alloydb-postgres
            region: my-region
            cluster: my-cluster
            instance: my-instance
            database: my_db
            user: my_user
            password: my_pass
            `,
			err: "error unmarshaling sources: unable to parse source \"my-pg-instance\" as \"alloydb-postgres\": Key: 'Config.Project' Error:Field validation for 'Project' failed on the 'required' tag",
		},
		{
			desc: "old tools file format",
			in: `
            sources:
                my-pg-instance:
                    type: alloydb-postgres
                    region: my-region
                    cluster: my-cluster
                    instance: my-instance
                    database: my_db
                    user: my_user
                    password: my_pass
            `,
			err: "missing 'type' field or it is not a string",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			_, _, _, _, err := server.UnmarshalResourceConfig(context.Background(), []byte(tc.in))
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
