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

func makeStrParam(name, desc string) tools.Parameter {
	return tools.NewStringParameter(name, desc)
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

// TestArguments_UnmarshalYAML is a UNIT TEST focused on the specific logic in arguments.go:
// defaulting the 'type' field to 'string' if it is omitted in the YAML.
func TestArguments_UnmarshalYAML(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name         string
		yamlInput    any
		expectedType string
	}{
		{
			name: "Type field is missing",
			yamlInput: []map[string]any{
				{"name": "p1", "description": "d1"},
			},
			expectedType: "string",
		},
		{
			name: "Type field is present",
			yamlInput: []map[string]any{
				{"name": "p1", "description": "d1", "type": "integer"},
			},
			expectedType: "integer",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			args := runUnmarshalTest(t, tc.yamlInput, "") // We expect no error here.

			if len(args) != 1 {
				t.Fatalf("expected 1 argument to be parsed, got %d", len(args))
			}
			gotType := args[0].GetType()
			if gotType != tc.expectedType {
				t.Errorf("expected parameter type to be %q, got %q", tc.expectedType, gotType)
			}
		})
	}
}

// TestArguments_UnmarshalYAML_Integration verifies that UnmarshalYAML correctly interacts with the tools package.
func TestArguments_UnmarshalYAML_Integration(t *testing.T) {
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
			// To compare arrays, we also need to compare their items.
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
			name: "Complex types (array/map) are passed correctly",
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
				makeArrayArg("param_array", "an array", makeStrParam("item_name", "an item")),
			},
		},
		{
			name: "Error from tools.ParseParameter is propagated correctly",
			yamlInput: []map[string]any{
				{"name": "p1", "description": "d1", "type": "unsupported"},
			},
			wantErr: `"unsupported" is not valid type for a parameter`,
		},
		{
			name:      "Unmarshal error - not a list",
			yamlInput: map[string]any{"name": "param1"}, // This is a map, not a slice.
			wantErr:   "mapping was used where sequence is expected",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			args := runUnmarshalTest(t, tc.yamlInput, tc.wantErr)

			// If an error was expected, the helper already verified it.
			if tc.wantErr != "" {
				return
			}

			// If no error was expected, compare the parsed result.
			if diff := cmp.Diff(tc.expectedArgs, args, paramComparer); diff != "" {
				t.Errorf("UnmarshalYAML() result mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// runUnmarshalTest is a test helper that marshals Go test data into YAML before testing.
func runUnmarshalTest(t *testing.T, yamlInput any, wantErr string) Arguments {
	t.Helper()

	yamlBytes, err := yaml.Marshal(yamlInput)
	if err != nil {
		t.Fatalf("Test setup failure: could not marshal test input to YAML: %v", err)
	}

	var rawList []util.DelayedUnmarshaler
	err = yaml.Unmarshal(yamlBytes, &rawList)

	// This block handles cases where the initial parsing into a list is expected to fail.
	if err != nil {
		if wantErr == "" {
			t.Fatalf("Initial YAML parsing failed unexpectedly: %v", err)
		}
		if !strings.Contains(err.Error(), wantErr) {
			t.Errorf("Initial unmarshal error mismatch:\nwant to contain: %q\ngot: %q", wantErr, err.Error())
		}
		return nil // Test is complete.
	}

	// This block handles the main test path.
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

	if wantErr != "" {
		if err == nil {
			t.Fatalf("UnmarshalYAML() expected error but got nil")
		}
		if !strings.Contains(err.Error(), wantErr) {
			t.Errorf("UnmarshalYAML() error mismatch:\nwant: %q\ngot:  %q", wantErr, err.Error())
		}
	} else if err != nil {
		t.Fatalf("UnmarshalYAML() returned unexpected error: %v", err)
	}

	return args
}
