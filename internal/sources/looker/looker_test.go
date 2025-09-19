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

package looker_test

import (
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/looker"
	"github.com/googleapis/genai-toolbox/internal/testutils"
)

func TestParseFromYamlLooker(t *testing.T) {
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
					kind: looker
					base_url: http://example.looker.com/
					client_id: jasdl;k;tjl
					client_secret: sdakl;jgflkasdfkfg
			`,
			want: map[string]sources.SourceConfig{
				"my-looker-instance": looker.Config{
					Name:               "my-looker-instance",
					Kind:               looker.SourceKind,
					BaseURL:            "http://example.looker.com/",
					ClientId:           "jasdl;k;tjl",
					ClientSecret:       "sdakl;jgflkasdfkfg",
					Timeout:            "600s",
					SslVerification:    true,
					UseClientOAuth:     false,
					ShowHiddenModels:   true,
					ShowHiddenExplores: true,
					ShowHiddenFields:   true,
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

func TestFailParseFromYamlLooker(t *testing.T) {
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
					kind: looker
					base_url: http://example.looker.com/
					client_id: jasdl;k;tjl
					client_secret: sdakl;jgflkasdfkfg
					project: test-project
			`,
			err: "unable to parse source \"my-looker-instance\" as \"looker\": [5:1] unknown field \"project\"\n   2 | client_id: jasdl;k;tjl\n   3 | client_secret: sdakl;jgflkasdfkfg\n   4 | kind: looker\n>  5 | project: test-project\n       ^\n",
		},
		{
			desc: "missing required field",
			in: `
			sources:
				my-looker-instance:
					kind: looker
			`,
			err: "unable to parse source \"my-looker-instance\" as \"looker\": Key: 'Config.BaseURL' Error:Field validation for 'BaseURL' failed on the 'required' tag",
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
