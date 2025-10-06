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

package snowflake_test

import (
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/sources/snowflake"
	"github.com/googleapis/genai-toolbox/internal/testutils"
)

func TestParseFromYamlSnowflake(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		want server.SourceConfigs
	}{
		{
			desc: "basic example",
			in: `
				sources:
					my-snowflake-instance:
						kind: snowflake
						account: my-account
						user: my_user
						password: my_pass
						database: my_db
						schema: my_schema
			`,
			want: server.SourceConfigs{
				"my-snowflake-instance": snowflake.Config{
					Name:      "my-snowflake-instance",
					Kind:      snowflake.SourceKind,
					Account:   "my-account",
					User:      "my_user",
					Password:  "my_pass",
					Database:  "my_db",
					Schema:    "my_schema",
					Warehouse: "",
					Role:      "",
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
					my-snowflake-instance:
						kind: snowflake
						account: my-account
						user: my_user
						password: my_pass
						database: my_db
						schema: my_schema
						foo: bar
			`,
			err: "unable to parse source \"my-snowflake-instance\" as \"snowflake\": [3:1] unknown field \"foo\"\n   1 | account: my-account\n   2 | database: my_db\n>  3 | foo: bar\n       ^\n   4 | kind: snowflake\n   5 | password: my_pass\n   6 | schema: my_schema\n   7 | ",
		},
		{
			desc: "missing required field",
			in: `
				sources:
					my-snowflake-instance:
						kind: snowflake
						account: my-account
						user: my_user
						password: my_pass
						database: my_db
			`,
			err: "unable to parse source \"my-snowflake-instance\" as \"snowflake\": Key: 'Config.Schema' Error:Field validation for 'Schema' failed on the 'required' tag",
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
