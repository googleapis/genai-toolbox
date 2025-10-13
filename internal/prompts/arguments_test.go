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

package prompts_test

import (
	"fmt"
	"strings"
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/prompts"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util"
)

// Test type aliases for convenience.
type (
	Argument     = prompts.Argument
	McpPromptArg = prompts.McpPromptArg
	Arguments    = prompts.Arguments
)

// Ptr is a helper function to create a pointer to a value.
func Ptr[T any](v T) *T {
	return &v
}

func makeArrayArg(name, desc string, items tools.Parameter) Argument {
	return Argument{Parameter: tools.NewArrayParameter(name, desc, items)}
}

func TestArgument_McpPromptManifest(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		arg      Argument
		expected McpPromptArg
	}{
		{
			name: "Required with no default",
			arg:  Argument{Parameter: tools.NewStringParameterWithRequired("name1", "desc1", true)},
			expected: McpPromptArg{
				Name: "name1", Description: "desc1", Required: true,
			},
		},
		{
			name: "Not required with no default",
			arg:  Argument{Parameter: tools.NewStringParameterWithRequired("name2", "desc2", false)},
			expected: McpPromptArg{
				Name: "name2", Description: "desc2", Required: false,
			},
		},
		{
			name: "Implicitly required with default",
			arg:  Argument{Parameter: tools.NewStringParameterWithDefault("name3", "defaultVal", "desc3")},
			expected: McpPromptArg{
				Name: "name3", Description: "desc3", Required: false,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.arg.McpPromptManifest()
			if diff := cmp.Diff(tc.expected, got); diff != "" {
				t.Errorf("McpPromptManifest() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestArguments_UnmarshalYAML tests all unmarshaling logic for the Arguments type.
func TestArguments_UnmarshalYAML(t *testing.T) {
	t.Parallel()
	// paramComparer allows cmp.Diff to intelligently compare the parsed results.
	var transformFunc func(tools.Parameter) any
	transformFunc = func(p tools.Parameter) any {
		s := struct{ Name, Type, Desc string }{
			Name: p.GetName(),
			Type: p.GetType(),
			Desc: p.Manifest().Description,
		}
		if arr, ok := p.(*tools.ArrayParameter); ok {
			s.Desc = fmt.Sprintf("%s items:%v", s.Desc, transformFunc(arr.GetItems()))
		}
		return s
	}
	paramComparer := cmp.Transformer("Parameter", transformFunc)

	testCases := []struct {
		name         string
		yamlInput    any
		expectedArgs Arguments
		wantErr      string
	}{
		{
			name: "Defaults type to string when omitted",
			yamlInput: []map[string]any{
				{"name": "p1", "description": "d1"},
			},
			expectedArgs: Arguments{
				{Parameter: tools.NewStringParameter("p1", "d1")},
			},
		},
		{
			name: "Respects type when present",
			yamlInput: []map[string]any{
				{"name": "p1", "description": "d1", "type": "integer"},
			},
			expectedArgs: Arguments{
				{Parameter: tools.NewIntParameter("p1", "d1")},
			},
		},
		{
			name: "Parses complex types like arrays correctly",
			yamlInput: []map[string]any{
				{
					"name":        "param_array",
					"description": "an array",
					"type":        "array",
					"items": map[string]any{
						"name":        "item_name",
						"type":        "string",
						"description": "an item",
					},
				},
			},
			expectedArgs: Arguments{
				makeArrayArg("param_array", "an array", tools.NewStringParameter("item_name", "an item")),
			},
		},
		{
			name: "Propagates parsing error for unsupported type",
			yamlInput: []map[string]any{
				{"name": "p1", "description": "d1", "type": "unsupported"},
			},
			wantErr: `"unsupported" is not valid type for a parameter`,
		},
		{
			name:      "Returns error when input is not a list",
			yamlInput: map[string]any{"name": "param1"}, // This is a map, not a slice.
			wantErr:   "mapping was used where sequence is expected",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			yamlBytes, err := yaml.Marshal(tc.yamlInput)
			var rawList []util.DelayedUnmarshaler
			err = yaml.Unmarshal(yamlBytes, &rawList)

			if tc.name == "Returns error when input is not a list" {
				if err == nil {
					t.Fatalf("Expected a structural parsing error, but got nil")
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("Structural error mismatch:\nwant to contain: %q\ngot: %q", tc.wantErr, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected structural parsing error: %v", err)
			}

			unmarshalFunc := func(v interface{}) error {
				dest, ok := v.(*[]util.DelayedUnmarshaler)
				if !ok {
					return fmt.Errorf("unexpected type for unmarshal: %T", v)
				}
				*dest = rawList
				return nil
			}

			var args Arguments
			ctx, err := testutils.ContextWithNewLogger()
			err = args.UnmarshalYAML(ctx, unmarshalFunc)

			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("UnmarshalYAML() expected error but got nil")
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("UnmarshalYAML() error mismatch:\nwant to contain: %q\ngot: %q", tc.wantErr, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("UnmarshalYAML() returned unexpected error: %v", err)
				}
				if diff := cmp.Diff(tc.expectedArgs, args, paramComparer); diff != "" {
					t.Errorf("UnmarshalYAML() result mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
