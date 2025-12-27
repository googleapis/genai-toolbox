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
package cloudloggingadmin_test

import (
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/sources/cloudloggingadmin"
	"github.com/googleapis/genai-toolbox/internal/testutils"
)

func TestParseFromYamlCloudLoggingAdmin(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		want server.SourceConfigs
	}{
		{
			desc: "basic example",
			in: `
			sources:
				my-instance:
					kind: cloud-logging-admin
					project: my-project
			`,
			want: server.SourceConfigs{
				"my-instance": cloudloggingadmin.Config{
					Name:    "my-instance",
					Kind:    cloudloggingadmin.SourceKind,
					Project: "my-project",
				},
			},
		},
		{
			desc: "with client oauth",
			in: `
			sources:
				my-instance:
					kind: cloud-logging-admin
					project: my-project
					useClientOAuth: true
			`,
			want: server.SourceConfigs{
				"my-instance": cloudloggingadmin.Config{
					Name:           "my-instance",
					Kind:           cloudloggingadmin.SourceKind,
					Project:        "my-project",
					UseClientOAuth: true,
				},
			},
		},
		{
			desc: "with service account impersonation",
			in: `
			sources:
				my-instance:
					kind: cloud-logging-admin
					project: my-project
					impersonateServiceAccount: service-account@my-project.iam.gserviceaccount.com
			`,
			want: server.SourceConfigs{
				"my-instance": cloudloggingadmin.Config{
					Name:                      "my-instance",
					Kind:                      cloudloggingadmin.SourceKind,
					Project:                   "my-project",
					ImpersonateServiceAccount: "service-account@my-project.iam.gserviceaccount.com",
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := struct {
				Sources server.SourceConfigs `yaml:"sources"`
			}{}
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
				my-instance:
					kind: cloud-logging-admin
					project: my-project
					foo: bar
			`,
			err: "unable to parse source \"my-instance\" as \"cloud-logging-admin\": [1:1] unknown field \"foo\"\n>  1 | foo: bar\n       ^\n   2 | kind: cloud-logging-admin\n   3 | project: my-project",
		},
		{
			desc: "missing required field",
			in: `
			sources:
				my-instance:
					kind: cloud-logging-admin
			`,
			err: "unable to parse source \"my-instance\" as \"cloud-logging-admin\": Key: 'Config.Project' Error:Field validation for 'Project' failed on the 'required' tag",
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
			errStr := err.Error()
			if errStr != tc.err {
				t.Fatalf("unexpected error: got %q, want %q", errStr, tc.err)
			}
		})
	}
}
