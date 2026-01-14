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

package spanner_test

import (
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/spanner"
	"github.com/googleapis/genai-toolbox/internal/testutils"
)

func TestParseFromYamlSpannerDb(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		want server.SourceConfigs
	}{
		{
			desc: "basic example",
			in: `
			sources:
				my-spanner-instance:
					kind: spanner
					project: my-project
					instance: my-instance
					database: my_db
			`,
			want: map[string]sources.SourceConfig{
				"my-spanner-instance": spanner.Config{
					Name:     "my-spanner-instance",
					Type:     spanner.SourceType,
					Project:  "my-project",
					Instance: "my-instance",
					Dialect:  "googlesql",
					Database: "my_db",
				},
			},
		},
		{
			desc: "gsql dialect",
			in: `
			sources:
				my-spanner-instance:
					kind: spanner
					project: my-project
					instance: my-instance
					dialect: Googlesql 
					database: my_db
			`,
			want: map[string]sources.SourceConfig{
				"my-spanner-instance": spanner.Config{
					Name:     "my-spanner-instance",
					Type:     spanner.SourceType,
					Project:  "my-project",
					Instance: "my-instance",
					Dialect:  "googlesql",
					Database: "my_db",
				},
			},
		},
		{
			desc: "postgresql dialect",
			in: `
			sources:
				my-spanner-instance:
					kind: spanner
					project: my-project
					instance: my-instance
					dialect: postgresql
					database: my_db
			`,
			want: map[string]sources.SourceConfig{
				"my-spanner-instance": spanner.Config{
					Name:     "my-spanner-instance",
					Type:     spanner.SourceType,
					Project:  "my-project",
					Instance: "my-instance",
					Dialect:  "postgresql",
					Database: "my_db",
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
			desc: "invalid dialect",
			in: `
			sources:
				my-spanner-instance:
					kind: spanner
					project: my-project
					instance: my-instance
					dialect: fail
					database: my_db
			`,
			err: "unable to parse source \"my-spanner-instance\" as \"spanner\": dialect invalid: must be one of \"googlesql\", or \"postgresql\"",
		},
		{
			desc: "extra field",
			in: `
			sources:
				my-spanner-instance:
					kind: spanner
					project: my-project
					instance: my-instance
					database: my_db
					foo: bar
			`,
			err: "unable to parse source \"my-spanner-instance\" as \"spanner\": [2:1] unknown field \"foo\"\n   1 | database: my_db\n>  2 | foo: bar\n       ^\n   3 | instance: my-instance\n   4 | kind: spanner\n   5 | project: my-project",
		},
		{
			desc: "missing required field",
			in: `
			sources:
				my-spanner-instance:
					kind: spanner
					project: my-project
					instance: my-instance
			`,
			err: "unable to parse source \"my-spanner-instance\" as \"spanner\": Key: 'Config.Database' Error:Field validation for 'Database' failed on the 'required' tag",
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
