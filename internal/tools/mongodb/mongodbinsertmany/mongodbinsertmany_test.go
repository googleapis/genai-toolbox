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

package mongodbinsertmany_test

import (
	"strings"
	"testing"

	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/tools/mongodb/mongodbinsertmany"

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
            kind: tool
            name: example_tool
            type: mongodb-insert-many
            source: my-instance
            description: some description
            database: test_db
            collection: test_coll
			`,
			want: server.ToolConfigs{
				"example_tool": mongodbinsertmany.Config{
					Name:         "example_tool",
					Type:         "mongodb-insert-many",
					Source:       "my-instance",
					AuthRequired: []string{},
					Database:     "test_db",
					Collection:   "test_coll",
					Description:  "some description",
					Canonical:    false,
				},
			},
		},
		{
			desc: "true canonical",
			in: `
            kind: tool
            name: example_tool
            type: mongodb-insert-many
            source: my-instance
            description: some description
            database: test_db
            collection: test_coll
            canonical: true
			`,
			want: server.ToolConfigs{
				"example_tool": mongodbinsertmany.Config{
					Name:         "example_tool",
					Type:         "mongodb-insert-many",
					Source:       "my-instance",
					AuthRequired: []string{},
					Database:     "test_db",
					Collection:   "test_coll",
					Description:  "some description",
					Canonical:    true,
				},
			},
		},
		{
			desc: "false canonical",
			in: `
            kind: tool
            name: example_tool
            type: mongodb-insert-many
            source: my-instance
            description: some description
            database: test_db
            collection: test_coll
            canonical: false
			`,
			want: server.ToolConfigs{
				"example_tool": mongodbinsertmany.Config{
					Name:         "example_tool",
					Type:         "mongodb-insert-many",
					Source:       "my-instance",
					AuthRequired: []string{},
					Database:     "test_db",
					Collection:   "test_coll",
					Description:  "some description",
					Canonical:    false,
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			_, _, _, got, _, _, err := server.UnmarshalResourceConfig(ctx, testutils.FormatYaml(tc.in))
			if err != nil {
				t.Fatalf("unable to unmarshal: %s", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("incorrect parse: diff %v", diff)
			}
		})
	}

}

func TestAnnotations(t *testing.T) {
	// Test default annotations for destructive tool
	t.Run("default annotations", func(t *testing.T) {
		annotations := tools.GetAnnotationsOrDefault(nil, tools.NewDestructiveAnnotations)
		if annotations == nil {
			t.Fatal("expected non-nil annotations")
		}
		if annotations.DestructiveHint == nil || *annotations.DestructiveHint != true {
			t.Error("expected destructiveHint to be true")
		}
		if annotations.ReadOnlyHint == nil || *annotations.ReadOnlyHint != false {
			t.Error("expected readOnlyHint to be false")
		}
	})

	// Test custom annotations override default
	t.Run("custom annotations", func(t *testing.T) {
		customDestructive := false
		custom := &tools.ToolAnnotations{DestructiveHint: &customDestructive}
		annotations := tools.GetAnnotationsOrDefault(custom, tools.NewDestructiveAnnotations)
		if annotations.DestructiveHint == nil || *annotations.DestructiveHint != false {
			t.Error("expected custom destructiveHint to be false")
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
            kind: tool
            name: example_tool
            type: mongodb-insert-many
            source: my-instance
            description: some description
            collection: test_coll
			`,
			err: `unable to parse tool "example_tool" as type "mongodb-insert-many"`,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			_, _, _, _, _, _, err := server.UnmarshalResourceConfig(ctx, testutils.FormatYaml(tc.in))
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
