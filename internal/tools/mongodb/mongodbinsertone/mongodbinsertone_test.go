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

package mongodbinsertone_test

import (
	"strings"
	"testing"

	"github.com/googleapis/genai-toolbox/internal/tools/mongodb/mongodbinsertone"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
)

func TestParseFromYamlMongoQuery(t *testing.T) {
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
				example_tool:
					kind: mongodb-insert-one
					source: my-instance
					description: some description
					database: test_db
					collection: test_coll
			`,
			want: server.ToolConfigs{
				"example_tool": mongodbinsertone.Config{
					Name:         "example_tool",
					Kind:         "mongodb-insert-one",
					Source:       "my-instance",
					AuthRequired: []string{},
					Database:     "test_db",
					Collection:   "test_coll",
					Canonical:    false,
					Description:  "some description",
				},
			},
		},
		{
			desc: "true canonical",
			in: `
			tools:
				example_tool:
					kind: mongodb-insert-one
					source: my-instance
					description: some description
					database: test_db
					collection: test_coll
					canonical: true
			`,
			want: server.ToolConfigs{
				"example_tool": mongodbinsertone.Config{
					Name:         "example_tool",
					Kind:         "mongodb-insert-one",
					Source:       "my-instance",
					AuthRequired: []string{},
					Database:     "test_db",
					Collection:   "test_coll",
					Canonical:    true,
					Description:  "some description",
				},
			},
		},
		{
			desc: "false canonical",
			in: `
			tools:
				example_tool:
					kind: mongodb-insert-one
					source: my-instance
					description: some description
					database: test_db
					collection: test_coll
					canonical: false
			`,
			want: server.ToolConfigs{
				"example_tool": mongodbinsertone.Config{
					Name:         "example_tool",
					Kind:         "mongodb-insert-one",
					Source:       "my-instance",
					AuthRequired: []string{},
					Database:     "test_db",
					Collection:   "test_coll",
					Canonical:    false,
					Description:  "some description",
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

func TestAnnotations(t *testing.T) {
	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	t.Run("default annotations", func(t *testing.T) {
		in := `
		tools:
			test_tool:
				kind: mongodb-insert-one
				source: my-instance
				description: test description
				database: test_db
				collection: test_coll
		`
		got := struct {
			Tools server.ToolConfigs `yaml:"tools"`
		}{}
		err := yaml.UnmarshalContext(ctx, testutils.FormatYaml(in), &got)
		if err != nil {
			t.Fatalf("unable to unmarshal: %s", err)
		}

		tool, err := got.Tools["test_tool"].Initialize(nil)
		if err != nil {
			t.Fatalf("unable to initialize: %s", err)
		}

		mcpManifest := tool.McpManifest()
		if mcpManifest.Annotations == nil {
			t.Fatal("expected annotations to be set")
		}
		if mcpManifest.Annotations.DestructiveHint == nil || !*mcpManifest.Annotations.DestructiveHint {
			t.Error("expected destructiveHint to be true for destructive tool")
		}
		if mcpManifest.Annotations.ReadOnlyHint == nil || *mcpManifest.Annotations.ReadOnlyHint {
			t.Error("expected readOnlyHint to be false for destructive tool")
		}
	})

	t.Run("custom annotations from YAML", func(t *testing.T) {
		in := `
		tools:
			test_tool:
				kind: mongodb-insert-one
				source: my-instance
				description: test description
				database: test_db
				collection: test_coll
				annotations:
					destructiveHint: true
					idempotentHint: true
		`
		got := struct {
			Tools server.ToolConfigs `yaml:"tools"`
		}{}
		err := yaml.UnmarshalContext(ctx, testutils.FormatYaml(in), &got)
		if err != nil {
			t.Fatalf("unable to unmarshal: %s", err)
		}

		tool, err := got.Tools["test_tool"].Initialize(nil)
		if err != nil {
			t.Fatalf("unable to initialize: %s", err)
		}

		mcpManifest := tool.McpManifest()
		if mcpManifest.Annotations == nil {
			t.Fatal("expected annotations to be set")
		}
		if mcpManifest.Annotations.IdempotentHint == nil || !*mcpManifest.Annotations.IdempotentHint {
			t.Error("expected idempotentHint from YAML to be applied")
		}
	})
}

func TestFailParseFromYamlMongoQuery(t *testing.T) {
	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	tcs := []struct {
		desc string
		in   string
		err  string
	}{
		{
			desc: "Invalid method",
			in: `
			tools:
				example_tool:
					kind: mongodb-insert-one
					source: my-instance
					description: some description
					collection: test_coll
					canonical: true
			`,
			err: `unable to parse tool "example_tool" as kind "mongodb-insert-one"`,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := struct {
				Tools server.ToolConfigs `yaml:"tools"`
			}{}
			// Parse contents
			err := yaml.UnmarshalContext(ctx, testutils.FormatYaml(tc.in), &got)
			if err == nil {
				t.Fatalf("expect parsing to fail")
			}
			errStr := err.Error()
			if !strings.Contains(errStr, tc.err) {
				t.Fatalf("unexpected error string: got %q, want substring %q", errStr, tc.err)
			}
		})
	}

}
