// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cloudsqlpgprecheckmajorversionupgrade_test

import (
	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools/cloudsqlpg/cloudsqlpgprecheckmajorversionupgrade"
	"testing"
)

func TestParseFromYaml(t *testing.T) {
	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	tcs := []struct {
		desc string
		in   string
		want server.ToolConfigs
	}{
		{
			desc: "basic precheck example",
			in: `
			tools:
				precheck-upgrade-tool:
					kind: cloud-sql-postgres-pre-check-major-version-upgrade
					description: a precheck test description
					source: some-admin-source
					authRequired:
						- https://www.googleapis.com/auth/cloud-platform
			`,
			want: server.ToolConfigs{
				"precheck-upgrade-tool": cloudsqlpgprecheckmajorversionupgrade.Config{
					Name:         "precheck-upgrade-tool",
					Kind:         "cloud-sql-postgres-pre-check-major-version-upgrade",
					Description:  "a precheck test description",
					Source:       "some-admin-source",
					AuthRequired: []string{"https://www.googleapis.com/auth/cloud-platform"},
				},
			},
		},
		{
			desc: "precheck example with no auth",
			in: `
			tools:
				precheck-upgrade-tool-no-auth:
					kind: cloud-sql-postgres-pre-check-major-version-upgrade
					description: a precheck test description no auth
					source: other-admin-source
			`,
			want: server.ToolConfigs{
				"precheck-upgrade-tool-no-auth": cloudsqlpgprecheckmajorversionupgrade.Config{
					Name:         "precheck-upgrade-tool-no-auth",
					Kind:         "cloud-sql-postgres-pre-check-major-version-upgrade",
					Description:  "a precheck test description no auth",
					Source:       "other-admin-source",
					AuthRequired: []string{},
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := struct {
				Tools server.ToolConfigs `yaml:"tools"`
			}{}
			// Parse contents
			err := yaml.UnmarshalContext(ctx, testutils.FormatYaml(tc.in), &got)
			if err != nil {
				t.Fatalf("unable to unmarshal: %s", err)
			}
			if diff := cmp.Diff(tc.want, got.Tools); diff != "" {
				t.Fatalf("incorrect parse: diff (-want +got):\n%s", diff)
			}
		})
	}
}
