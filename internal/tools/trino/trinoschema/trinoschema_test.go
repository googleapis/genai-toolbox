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

package trinoschema_test

import (
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools/trino/trinoschema"
)

func TestParseFromYamlTrinoSchema(t *testing.T) {
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
				get_schema:
					kind: trino-schema
					source: my-trino-instance
					description: Gets comprehensive schema information
			`,
			want: server.ToolConfigs{
				"get_schema": trinoschema.Config{
					Name:         "get_schema",
					Kind:         "trino-schema",
					Source:       "my-trino-instance",
					Description:  "Gets comprehensive schema information",
					AuthRequired: []string{},
				},
			},
		},
		{
			desc: "with cache expiration",
			in: `
			tools:
				get_schema_cached:
					kind: trino-schema
					source: my-trino-instance
					description: Gets schema with custom cache
					cacheExpireMinutes: 30
			`,
			want: server.ToolConfigs{
				"get_schema_cached": trinoschema.Config{
					Name:               "get_schema_cached",
					Kind:               "trino-schema",
					Source:             "my-trino-instance",
					Description:        "Gets schema with custom cache",
					AuthRequired:       []string{},
					CacheExpireMinutes: intPtr(30),
				},
			},
		},
		{
			desc: "with auth",
			in: `
			tools:
				get_schema_auth:
					kind: trino-schema
					source: my-trino-instance
					description: Gets schema with authentication
					authRequired:
						- my-auth-service
					cacheExpireMinutes: 5
			`,
			want: server.ToolConfigs{
				"get_schema_auth": trinoschema.Config{
					Name:               "get_schema_auth",
					Kind:               "trino-schema",
					Source:             "my-trino-instance",
					Description:        "Gets schema with authentication",
					AuthRequired:       []string{"my-auth-service"},
					CacheExpireMinutes: intPtr(5),
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

func TestTrinoSchemaToolConfigKind(t *testing.T) {
	config := trinoschema.Config{
		Name:        "test",
		Kind:        "trino-schema",
		Source:      "test-source",
		Description: "test",
	}

	if kind := config.ToolConfigKind(); kind != "trino-schema" {
		t.Errorf("expected ToolConfigKind to return 'trino-schema', got %s", kind)
	}
}

func TestTrinoCacheExpiration(t *testing.T) {
	// Test default cache expiration
	config1 := trinoschema.Config{
		Name:        "test1",
		Kind:        "trino-schema",
		Source:      "test-source",
		Description: "test",
		// CacheExpireMinutes not set, should default to 10
	}

	// Test custom cache expiration
	customExpire := 30
	config2 := trinoschema.Config{
		Name:               "test2",
		Kind:               "trino-schema",
		Source:             "test-source",
		Description:        "test",
		CacheExpireMinutes: &customExpire,
	}

	// Verify configs are created correctly
	if config1.CacheExpireMinutes != nil {
		t.Errorf("expected CacheExpireMinutes to be nil for default, got %v", *config1.CacheExpireMinutes)
	}

	if config2.CacheExpireMinutes == nil || *config2.CacheExpireMinutes != 30 {
		t.Errorf("expected CacheExpireMinutes to be 30, got %v", config2.CacheExpireMinutes)
	}
}

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}
