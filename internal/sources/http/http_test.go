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

package http_test

import (
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/http"
	"github.com/googleapis/genai-toolbox/internal/testutils"
)

func TestParseFromYamlHttp(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		want server.SourceConfigs
	}{
		{
			desc: "basic example",
			in: `
			sources:
				my-http-instance:
					type: http
					baseUrl: http://test_server/
			`,
			want: map[string]sources.SourceConfig{
				"my-http-instance": http.Config{
					Name:                   "my-http-instance",
					Type:                   http.SourceType,
					BaseURL:                "http://test_server/",
					Timeout:                "30s",
					DisableSslVerification: false,
				},
			},
		},
		{
			desc: "advanced example",
			in: `
			sources:
				my-http-instance:
					type: http
					baseUrl: http://test_server/
					timeout: 10s
					headers:
						Authorization: test_header
						Custom-Header: custom
					queryParams:
						api-key: test_api_key
						param: param-value
					disableSslVerification: true
			`,
			want: map[string]sources.SourceConfig{
				"my-http-instance": http.Config{
					Name:                   "my-http-instance",
					Type:                   http.SourceType,
					BaseURL:                "http://test_server/",
					Timeout:                "10s",
					DefaultHeaders:         map[string]string{"Authorization": "test_header", "Custom-Header": "custom"},
					QueryParams:            map[string]string{"api-key": "test_api_key", "param": "param-value"},
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
				my-http-instance:
					type: http
					baseUrl: http://test_server/
					timeout: 10s
					headers:
						Authorization: test_header
					queryParams:
						api-key: test_api_key
					project: test-project
			`,
			err: "unable to parse source \"my-http-instance\" as \"http\": [5:1] unknown field \"project\"\n   2 | headers:\n   3 |   Authorization: test_header\n   4 | type: http\n>  5 | project: test-project\n       ^\n   6 | queryParams:\n   7 |   api-key: test_api_key\n   8 | timeout: 10s",
		},
		{
			desc: "missing required field",
			in: `
			sources:
				my-http-instance:
					baseUrl: http://test_server/
			`,
			err: "missing 'type' field for source \"my-http-instance\"",
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
