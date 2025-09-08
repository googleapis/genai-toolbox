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

package alloydbadmin_test

import (
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/alloydbadmin"
	"github.com/googleapis/genai-toolbox/internal/testutils"
)

func TestParseFromYamlAlloyDBAdmin(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		want server.SourceConfigs
	}{
		{
			desc: "basic example",
			in: `
			sources:
				my-alloydb-admin-instance:
					kind: alloydb-admin
			`,
			want: map[string]sources.SourceConfig{
				"my-alloydb-admin-instance": alloydbadmin.Config{
					Name:                   "my-alloydb-admin-instance",
					Kind:                   alloydbadmin.SourceKind,
					Timeout:                "30s",
					DisableSslVerification: false,
				},
			},
		},
		{
			desc: "advanced example",
			in: `
			sources:
				my-alloydb-admin-instance:
					kind: alloydb-admin
					timeout: 10s
					headers:
						Custom-Header: custom
					disableSslVerification: true
			`,
			want: map[string]sources.SourceConfig{
				"my-alloydb-admin-instance": alloydbadmin.Config{
					Name:                   "my-alloydb-admin-instance",
					Kind:                   alloydbadmin.SourceKind,
					Timeout:                "10s",
					DefaultHeaders:         map[string]string{"Custom-Header": "custom"},
					DisableSslVerification: true,
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
				my-alloydb-admin-instance:
					kind: alloydb-admin
					timeout: 10s
					headers:
						Custom-Header: custom
					project: test-project
			`,
			err: "unable to parse source \"my-alloydb-admin-instance\" as \"alloydb-admin\": [4:1] unknown field \"project\"\n   1 | headers:\n   2 |   Custom-Header: custom\n   3 | kind: alloydb-admin\n>  4 | project: test-project\n       ^\n   5 | timeout: 10s",
		},
		{
			desc: "missing required field",
			in: `
			sources:
				my-alloydb-admin-instance:
					timeout: 10s
			`,
			err: "missing 'kind' field for source \"my-alloydb-admin-instance\"",
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
