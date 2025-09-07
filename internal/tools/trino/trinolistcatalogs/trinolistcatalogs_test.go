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

package trinolistcatalogs_test

import (
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools/trino/trinolistcatalogs"
)

func TestParseFromYamlTrinoListCatalogs(t *testing.T) {
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
				list_catalogs:
					kind: trino-list-catalogs
					source: my-trino-instance
					description: Lists all available catalogs
			`,
			want: server.ToolConfigs{
				"list_catalogs": trinolistcatalogs.Config{
					Name:         "list_catalogs",
					Kind:         "trino-list-catalogs",
					Source:       "my-trino-instance",
					Description:  "Lists all available catalogs",
					AuthRequired: []string{},
				},
			},
		},
		{
			desc: "with auth",
			in: `
			tools:
				list_catalogs_auth:
					kind: trino-list-catalogs
					source: my-trino-instance
					description: Lists catalogs with authentication
					authRequired:
						- my-auth-service
			`,
			want: server.ToolConfigs{
				"list_catalogs_auth": trinolistcatalogs.Config{
					Name:         "list_catalogs_auth",
					Kind:         "trino-list-catalogs",
					Source:       "my-trino-instance",
					Description:  "Lists catalogs with authentication",
					AuthRequired: []string{"my-auth-service"},
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
