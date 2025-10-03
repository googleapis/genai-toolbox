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

package hana_test

import (
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/sources/hana"
	"github.com/googleapis/genai-toolbox/internal/testutils"
)

func TestParseFromYamlHana(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		want server.SourceConfigs
	}{
		{
			desc: "basic example",
			in: `
            sources:
                my-hana-instance:
                    kind: hana
                    host: hana-host
                    port: "39015"
                    database: HDB
                    user: my_user
                    password: my_pass
            `,
			want: server.SourceConfigs{
				"my-hana-instance": hana.Config{
					Name:     "my-hana-instance",
					Kind:     hana.SourceKind,
					Host:     "hana-host",
					Port:     "39015",
					Database: "HDB",
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
			if err := yaml.Unmarshal(testutils.FormatYaml(tc.in), &got); err != nil {
				t.Fatalf("unable to unmarshal: %s", err)
			}
			if diff := cmp.Diff(tc.want, got.Sources); diff != "" {
				t.Fatalf("incorrect parse: diff %v", diff)
			}
		})
	}
}

func TestFailParseFromYamlHana(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		err  string
	}{
		{
			desc: "extra field",
			in: `
            sources:
                my-hana-instance:
                    kind: hana
                    host: hana-host
                    port: "39015"
                    database: HDB
                    user: my_user
                    password: my_pass
                    foo: bar
            `,
			err: "unable to parse source \"my-hana-instance\" as \"hana\": [2:1] unknown field \"foo\"\n   1 | database: HDB\n>  2 | foo: bar\n       ^\n   3 | host: hana-host\n   4 | kind: hana\n   5 | password: my_pass\n   6 | ",
		},
		{
			desc: "missing required field",
			in: `
            sources:
                my-hana-instance:
                    kind: hana
                    host: hana-host
                    port: "39015"
                    database: HDB
                    user: my_user
            `,
			err: "unable to parse source \"my-hana-instance\" as \"hana\": Key: 'Config.Password' Error:Field validation for 'Password' failed on the 'required' tag",
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
			if err.Error() != tc.err {
				t.Fatalf("unexpected error: got %q, want %q", err.Error(), tc.err)
			}
		})
	}
}
