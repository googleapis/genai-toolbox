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

package alloydblistusers_test

import (
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	alloydblistusers "github.com/googleapis/genai-toolbox/internal/tools/alloydb/alloydblistusers"
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
			desc: "basic example",
			in: `
			tools:
				list-my-users:
					kind: alloydb-list-users
					source: my-alloydb-admin-source
					description: some description
			`,
			want: server.ToolConfigs{
				"list-my-users": alloydblistusers.Config{
					Name:         "list-my-users",
					Kind:         "alloydb-list-users",
					Source:       "my-alloydb-admin-source",
					Description:  "some description",
					AuthRequired: []string{},
				},
			},
		},
		{
			desc: "with auth required",
			in: `
			tools:
				list-my-users-auth:
					kind: alloydb-list-users
					source: my-alloydb-admin-source
					description: some description
					authRequired:
						- my-google-auth-service
						- other-auth-service
			`,
			want: server.ToolConfigs{
				"list-my-users-auth": alloydblistusers.Config{
					Name:         "list-my-users-auth",
					Kind:         "alloydb-list-users",
					Source:       "my-alloydb-admin-source",
					Description:  "some description",
					AuthRequired: []string{"my-google-auth-service", "other-auth-service"},
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
				t.Fatalf("incorrect parse: diff %v", diff)
			}
		})
	}
}
