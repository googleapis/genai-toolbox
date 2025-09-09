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

package alloydbcreateuser_test

import (
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	alloydbcreateuser "github.com/googleapis/genai-toolbox/internal/tools/alloydb/alloydbcreateuser"
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
				create-my-user:
					kind: alloydb-create-user
					source: my-alloydb-admin-source
					description: some description
			`,
			want: server.ToolConfigs{
				"create-my-user": alloydbcreateuser.Config{
					Name:         "create-my-user",
					Kind:         "alloydb-create-user",
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
				create-my-user-auth:
					kind: alloydb-create-user
					source: my-alloydb-admin-source
					description: some description
					authRequired: 
						- my-google-auth-service
						- other-auth-service
			`,
			want: server.ToolConfigs{
				"create-my-user-auth": alloydbcreateuser.Config{
					Name:         "create-my-user-auth",
					Kind:         "alloydb-create-user",
					Source:       "my-alloydb-admin-source",
					Description:  "some description",
					AuthRequired: []string{"my-google-auth-service", "other-auth-service"},
				},
			},
		},
		{
			desc: "with base url",
			in: `
			tools:
				create-my-user-baseurl:
					kind: alloydb-create-user
					source: my-alloydb-admin-source
					description: some description
					baseURL: "https://example.com"
			`,
			want: server.ToolConfigs{
				"create-my-user-baseurl": alloydbcreateuser.Config{
					Name:         "create-my-user-baseurl",
					Kind:         "alloydb-create-user",
					Source:       "my-alloydb-admin-source",
					Description:  "some description",
					BaseURL:      "https://example.com",
					AuthRequired: []string{},
				},
			},
		},
		{
			desc: "with auth and base url",
			in: `
			tools:
				create-my-user-all:
					kind: alloydb-create-user
					source: my-alloydb-admin-source
					description: some description
					authRequired: 
						- my-google-auth-service
						- other-auth-service
					baseURL: "https://example.com"
			`,
			want: server.ToolConfigs{
				"create-my-user-all": alloydbcreateuser.Config{
					Name:         "create-my-user-all",
					Kind:         "alloydb-create-user",
					Source:       "my-alloydb-admin-source",
					Description:  "some description",
					AuthRequired: []string{"my-google-auth-service", "other-auth-service"},
					BaseURL:      "https://example.com",
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
