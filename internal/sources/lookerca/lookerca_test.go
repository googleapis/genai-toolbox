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

package lookerca_test

import (
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/lookerca"
	"github.com/googleapis/genai-toolbox/internal/testutils"
)

func TestParseFromYamlLookerCA(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		want server.SourceConfigs
	}{
		{
			desc: "basic example",
			in: `
			sources:
				my-looker-instance:
					kind: lookerca
					base_url: http://example.looker.com/
					client_id: jasdl;k;tjl
					client_secret: sdakl;jgflkasdfkfg
			`,
			want: map[string]sources.SourceConfig{
				"my-looker-instance": lookerca.Config{
					Name:            "my-looker-instance",
					Kind:            lookerca.SourceKind,
					BaseURL:         "http://example.looker.com/",
					ClientId:        "jasdl;k;tjl",
					ClientSecret:    "sdakl;jgflkasdfkfg",
					Timeout:         "600s",
					SslVerification: true,
					UseClientOAuth:  false,
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

func TestFailParseFromYamlLookerCA(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		err  string
	}{
		{
			desc: "extra field",
			in: `
			sources:
				my-looker-instance:
					kind: lookerca
					base_url: http://example.looker.com/
					client_id: jasdl;k;tjl
					client_secret: sdakl;jgflkasdfkfg
					foo: test-foo
			`,
			err: "unable to parse source \"my-looker-instance\" as \"lookerca\": [4:1] unknown field \"foo\"\n   1 | base_url: http://example.looker.com/\n   2 | client_id: jasdl;k;tjl\n   3 | client_secret: sdakl;jgflkasdfkfg\n>  4 | foo: test-foo\n       ^\n   5 | kind: lookerca",
		},
		{
			desc: "missing required field",
			in: `
			sources:
				my-looker-instance:
					kind: lookerca
					client_id: jasdl;k;tjl
			`,
			err: "unable to parse source \"my-looker-instance\" as \"lookerca\": Key: 'Config.BaseURL' Error:Field validation for 'BaseURL' failed on the 'required' tag",
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
