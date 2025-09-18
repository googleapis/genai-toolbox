// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package opengemini_test

import (
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/sources/opengemini"
	"github.com/googleapis/genai-toolbox/internal/testutils"
)

func TestParseFromYamlOpenGemini(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		want server.SourceConfigs
	}{
		{
			desc: "basic example",
			in: `
			sources:
				my-opengemini-instance:
					kind: opengemini
					host: my-host
					port: 8086
					database: my_db
					retentionpolicy: autogen
			`,
			want: server.SourceConfigs{
				"my-opengemini-instance": opengemini.Config{
					Name:            "my-opengemini-instance",
					Kind:            opengemini.SourceKind,
					Host:            "my-host",
					Port:            8086,
					Database:        "my_db",
					RetentionPolicy: "autogen",
				},
			},
		},
		{
			desc: "auth with username and password",
			in: `
			sources:
				my-opengemini-instance:
					kind: opengemini
					host: my-host
					port: 8086
					authtype: 1
					user: my_user
					password: my_pass
					database: my_db
					retentionpolicy: autogen
			`,
			want: server.SourceConfigs{
				"my-opengemini-instance": opengemini.Config{
					Name:            "my-opengemini-instance",
					Kind:            opengemini.SourceKind,
					Host:            "my-host",
					Port:            8086,
					AuthType:        1,
					User:            "my_user",
					Password:        "my_pass",
					Database:        "my_db",
					RetentionPolicy: "autogen",
				},
			},
		},
		{
			desc: "auth with token",
			in: `
			sources:
				my-opengemini-instance:
					kind: opengemini
					host: my-host
					port: 8086
					authtype: 2
					token: my_token
					database: my_db
					retentionpolicy: autogen
			`,
			want: server.SourceConfigs{
				"my-opengemini-instance": opengemini.Config{
					Name:            "my-opengemini-instance",
					Kind:            opengemini.SourceKind,
					Host:            "my-host",
					Port:            8086,
					AuthType:        2,
					Token:           "my_token",
					Database:        "my_db",
					RetentionPolicy: "autogen",
				},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()
			got := struct {
				Sources server.SourceConfigs `yaml:"sources"`
			}{}
			// Parse contents
			err := yaml.Unmarshal(testutils.FormatYaml(tc.in), &got)
			if err != nil {
				t.Fatalf("unable to unmarshal: %s", err)
			}
			if diff := cmp.Diff(tc.want, got.Sources, cmpopts.EquateEmpty()); diff != "" {
				t.Fatalf("mismatch (-want +got):\n%s", diff)
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
				my-opengemini-instance:
					kind: opengemini
					host: my-host
					port: 8086
					foo: bar
					database: my_db
					retentionpolicy: autogen
			`,
			err: "unknown field \"foo\"",
		},
		{
			desc: "missing required field",
			in: `
			sources:
				my-opengemini-instance:
					kind: opengemini
					port: 8086
					database: my_db
					retentionpolicy: autogen
			`,
			err: "Field validation for 'Host' failed",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()
			got := struct {
				Sources server.SourceConfigs `yaml:"sources"`
			}{}
			// Parse contents
			err := yaml.Unmarshal(testutils.FormatYaml(tc.in), &got)
			if err == nil {
				t.Fatalf("expect parsing to fail")
			}
			errStr := err.Error()
			if !strings.Contains(errStr, tc.err) {
				t.Fatalf("unexpected error: got %q, want substring %q", errStr, tc.err)
			}
		})
	}
}
