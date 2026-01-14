// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package neo4jschema

import (
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
)

func TestParseFromYamlNeo4j(t *testing.T) {
	ctx, err := testutils.ContextWithNewLogger()
	exp := 30

	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	tcs := []struct {
		desc string
		in   string
		want server.ToolConfigs
	}{
		{
			desc: "basic example with default cache expiration",
			in: `
			tools:
				example_tool:
					kind: neo4j-schema
					source: my-neo4j-instance
					description: some tool description
					authRequired:
						- my-google-auth-service
						- other-auth-service
			`,
			want: server.ToolConfigs{
				"example_tool": Config{
					Name:               "example_tool",
					Type:               "neo4j-schema",
					Source:             "my-neo4j-instance",
					Description:        "some tool description",
					AuthRequired:       []string{"my-google-auth-service", "other-auth-service"},
					CacheExpireMinutes: nil,
				},
			},
		},
		{
			desc: "cache expire minutes set explicitly",
			in: `
			tools:
				example_tool:
					kind: neo4j-schema
					source: my-neo4j-instance
					description: some tool description
					cacheExpireMinutes: 30
			`,
			want: server.ToolConfigs{
				"example_tool": Config{
					Name:               "example_tool",
					Type:               "neo4j-schema",
					Source:             "my-neo4j-instance",
					Description:        "some tool description",
					AuthRequired:       []string{}, // Expect an empty slice, not nil.
					CacheExpireMinutes: &exp,
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
			err = yaml.UnmarshalContext(ctx, testutils.FormatYaml(tc.in), &got)
			if err != nil {
				t.Fatalf("unable to unmarshal: %s", err)
			}
			if diff := cmp.Diff(tc.want, got.Tools); diff != "" {
				t.Fatalf("incorrect parse: diff %v", diff)
			}
		})
	}
}
