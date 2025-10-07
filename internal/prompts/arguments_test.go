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
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/prompts"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util"
)

// Test type aliases for convenience
type (
	Argument     = prompts.Argument
	McpPromptArg = prompts.McpPromptArg
	Arguments    = prompts.Arguments
)

// Ptr is a helper function to create a pointer to a value.
func Ptr[T any](v T) *T {
	return &v
}

// -- Test Setup Helpers to reduce boilerplate in test cases --

func makeStrArg(name, desc string) Argument {
	return Argument{Parameter: tools.NewStringParameter(name, desc)}
}

func makeIntArg(name, desc string) Argument {
	return Argument{Parameter: tools.NewIntParameter(name, desc)}
}

func makeBoolArg(name, desc string, required bool) Argument {
	return Argument{Parameter: tools.NewBooleanParameterWithRequired(name, desc, required)}
}

func makeArrayArg(name, desc string, items tools.Parameter) Argument {
	return Argument{Parameter: tools.NewArrayParameter(name, desc, items)}
}

func makeMapArg(name, desc, valueType string) Argument {
	return Argument{Parameter: tools.NewMapParameter(name, desc, valueType)}
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
		{
			name: "Explicitly required with default",
			arg: Argument{
				Parameter: &tools.StringParameter{
					CommonParameter: tools.CommonParameter{Name: "name4", Type: tools.TypeString, Desc: "desc4", Required: Ptr(true)},
					Default:         Ptr("defaultVal"),
				},
			},
			expected: McpPromptArg{
				Name: "name4", Description: "desc4", Required: false,
			},
		},
		{
			name: "Implicitly required with no default",
			arg:  Argument{Parameter: tools.NewStringParameter("name5", "desc5")},
			expected: McpPromptArg{
				Name: "name5", Description: "desc5", Required: true,
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

func TestArguments_UnmarshalYAML(t *testing.T) {
	t.Parallel()

	var transformFunc func(tools.Parameter) any
	transformFunc = func(p tools.Parameter) any {
		s := struct {
			Name, Type, Desc string
			Required         bool
			Items            any
			ValueType        string
		}{
			Name:     p.GetName(),
			Type:     p.GetType(),
			Desc:     p.Manifest().Description,
			Required: p.GetRequired(),
		}
		if arr, ok := p.(*tools.ArrayParameter); ok {
			// Correct recursive call
			s.Items = transformFunc(arr.GetItems())
		}
		if m, ok := p.(*tools.MapParameter); ok {
			s.ValueType = m.GetValueType()
		}
		return s
	}

	// Finally, create the comparer option for cmp.Diff by passing our recursive function to cmp.Transformer.
	paramComparer := cmp.Transformer("Parameter", transformFunc)

	testCases := []struct {
		name         string
		yaml         string
		expectedArgs Arguments
		wantErr      string
	}{
		{
			name: "Successful unmarshal with various types",
			yaml: `
- name: param1
  description: string param
- name: param2
  description: int param
  type: integer
- name: param3
  description: bool param
  type: boolean
  required: false
`,
			expectedArgs: Arguments{
				makeStrArg("param1", "string param"),
				makeIntArg("param2", "int param"),
				makeBoolArg("param3", "bool param", false),
			},
		},
		{
			name: "Type defaults to string",
			yaml: `
- name: param_default
  description: a param that defaults to string type
`,
			expectedArgs: Arguments{
				makeStrArg("param_default", "a param that defaults to string type"),
			},
		},
		{
			name: "Array and Map types",
			yaml: `
- name: param_array
  description: an array
  type: array
  items:
    name: an_item_name_to_pass_validation
    type: string
    description: an item
- name: param_map
  description: a map
  type: map
  valueType: integer
`,
			expectedArgs: Arguments{
				makeArrayArg("param_array", "an array", tools.NewStringParameter("an_item_name_to_pass_validation", "an item")),
				makeMapArg("param_map", "a map", "integer"),
			},
		},
		{
			name:    "Unmarshal error - not a list",
			yaml:    `name: param1`,
			wantErr: "mapping was used where sequence is expected",
		},
		{
			name:    "Parse error - bad item in list",
			yaml:    `- "just a string"`,
			wantErr: "string was used where mapping is expected",
		},
		{
			name: "Parse error - invalid type",
			yaml: `
- name: param1
  description: desc1
  type: unsupported
`,
			wantErr: `"unsupported" is not valid type for a parameter`,
		},
		{
			name: "Parse error - missing name",
			yaml: `
- type: string
  description: desc1
`,
			wantErr: "Field validation for 'Name' failed on the 'required' tag",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var rawList []util.DelayedUnmarshaler
			err := yaml.Unmarshal([]byte(tc.yaml), &rawList)

			if err != nil {
				if tc.wantErr == "" {
					t.Fatalf("Initial unmarshal failed unexpectedly: %v", err)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("Initial unmarshal error mismatch:\nwant: %q\ngot:  %q", tc.wantErr, err.Error())
				}
				return
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
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			ctx := util.WithLogger(context.Background(), logger)
			err = args.UnmarshalYAML(ctx, unmarshalFunc)

			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("UnmarshalYAML() expected error, got nil")
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("UnmarshalYAML() error mismatch:\nwant: %q\ngot:  %q", tc.wantErr, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("UnmarshalYAML() returned unexpected error: %v", err)
			}

			if diff := cmp.Diff(tc.expectedArgs, args, paramComparer); diff != "" {
				t.Errorf("UnmarshalYAML() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
