// Copyright 2024 Google LLC
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

package spannerexecutesql_test

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/mcp-toolbox/internal/server"
	"github.com/googleapis/mcp-toolbox/internal/testutils"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/tools/spanner/spannerexecutesql"
)

func TestParseFromYamlExecuteSql(t *testing.T) {
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
            type: spanner-execute-sql
            source: my-spanner-instance
            description: some description
			`,
			want: server.ToolConfigs{
				"example_tool": spannerexecutesql.Config{
					ConfigBase: tools.ConfigBase{
						Name:         "example_tool",
						Description:  "some description",
						AuthRequired: []string{},
					},
					Type:     "spanner-execute-sql",
					Source:   "my-spanner-instance",
					ReadOnly: false,
				},
			},
		},
		{
			desc: "read only set to true",
			in: `
            kind: tool
            name: example_tool
            type: spanner-execute-sql
            source: my-spanner-instance
            description: some description
            readOnly: true
			`,
			want: server.ToolConfigs{
				"example_tool": spannerexecutesql.Config{
					ConfigBase: tools.ConfigBase{
						Name:         "example_tool",
						Description:  "some description",
						AuthRequired: []string{},
					},
					Type:     "spanner-execute-sql",
					Source:   "my-spanner-instance",
					ReadOnly: true,
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			// Parse contents
			_, _, _, got, _, _, err := server.UnmarshalPrimitiveConfig(ctx, testutils.FormatYaml(tc.in))
			if err != nil {
				t.Fatalf("unable to unmarshal: %s", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("incorrect parse: diff %v", diff)
			}
		})
	}

}

func TestInitialize_ReadOnlyValidation(t *testing.T) {
	ctx := context.Background()

	ptr := func(b bool) *bool { return &b }

	tcs := []struct {
		desc        string
		cfg         spannerexecutesql.Config
		wantErr     bool
		errContains string
	}{
		{
			desc: "no conflict - both true",
			cfg: spannerexecutesql.Config{
				ConfigBase:  tools.ConfigBase{Name: "test-tool", Description: "desc"},
				ReadOnly:    true,
				Annotations: &tools.ToolAnnotations{ReadOnlyHint: ptr(true)},
			},
			wantErr: false,
		},
		{
			desc: "no conflict - both false",
			cfg: spannerexecutesql.Config{
				ConfigBase:  tools.ConfigBase{Name: "test-tool", Description: "desc"},
				ReadOnly:    false,
				Annotations: &tools.ToolAnnotations{ReadOnlyHint: ptr(false)},
			},
			wantErr: false,
		},
		{
			desc: "no conflict - readOnlyHint nil",
			cfg: spannerexecutesql.Config{
				ConfigBase:  tools.ConfigBase{Name: "test-tool", Description: "desc"},
				ReadOnly:    true,
				Annotations: &tools.ToolAnnotations{ReadOnlyHint: nil},
			},
			wantErr: false,
		},
		{
			desc: "conflict - readOnly false, readOnlyHint true",
			cfg: spannerexecutesql.Config{
				ConfigBase:  tools.ConfigBase{Name: "test-tool", Description: "desc"},
				ReadOnly:    false,
				Annotations: &tools.ToolAnnotations{ReadOnlyHint: ptr(true)},
			},
			wantErr:     true,
			errContains: "configuration conflict in tool \"test-tool\"",
		},
		{
			desc: "conflict - readOnly true, readOnlyHint false",
			cfg: spannerexecutesql.Config{
				ConfigBase:  tools.ConfigBase{Name: "test-tool", Description: "desc"},
				ReadOnly:    true,
				Annotations: &tools.ToolAnnotations{ReadOnlyHint: ptr(false)},
			},
			wantErr:     true,
			errContains: "configuration conflict in tool \"test-tool\"",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			_, err := tc.cfg.Initialize(ctx)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				// Use manual substring check since strings import might not be present
				errStr := err.Error()
				found := false
				for i := 0; i <= len(errStr)-len(tc.errContains); i++ {
					if errStr[i:i+len(tc.errContains)] == tc.errContains {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error to contain %q, got: %v", tc.errContains, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}
