// Copyright 2026 Google LLC
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

package vdraft

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
)

func TestCallToolResult_StructuredContent(t *testing.T) {
	tests := []struct {
		name     string
		input    CallToolResult
		wantJSON string
	}{
		{
			name: "structuredContent as object",
			input: CallToolResult{
				StructuredContent: map[string]any{
					"key": "value",
				},
			},
			wantJSON: `{"content":null,"structuredContent":{"key":"value"}}`,
		},
		{
			name: "structuredContent as array",
			input: CallToolResult{
				StructuredContent: []any{
					"item1",
					float64(42),
				},
			},
			wantJSON: `{"content":null,"structuredContent":["item1",42]}`,
		},
		{
			name: "structuredContent as primitive string",
			input: CallToolResult{
				StructuredContent: "just a string",
			},
			wantJSON: `{"content":null,"structuredContent":"just a string"}`,
		},
		{
			name: "structuredContent as primitive number",
			input: CallToolResult{
				StructuredContent: float64(123.45),
			},
			wantJSON: `{"content":null,"structuredContent":123.45}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Test Marshal
			gotBytes, err := json.Marshal(tc.input)
			if err != nil {
				t.Fatalf("failed to marshal CallToolResult: %s", err)
			}
			if string(gotBytes) != tc.wantJSON {
				t.Errorf("got JSON = %s, want %s", string(gotBytes), tc.wantJSON)
			}

			// Test Unmarshal
			var gotResult CallToolResult
			if err := json.Unmarshal(gotBytes, &gotResult); err != nil {
				t.Fatalf("failed to unmarshal CallToolResult: %s", err)
			}
			if !reflect.DeepEqual(gotResult.StructuredContent, tc.input.StructuredContent) {
				t.Errorf("got StructuredContent = %#v, want %#v", gotResult.StructuredContent, tc.input.StructuredContent)
			}
		})
	}
}

func TestInputSchema_AdvancedJSONSchema(t *testing.T) {
	input := InputSchema{
		Schema: "https://json-schema.org/draft/2020-12/schema",
		Type:   "object",
		Properties: map[string]parameters.ParameterMcpManifest{
			"any_of_param": {
				AnyOf: []*parameters.ParameterMcpManifest{
					{Type: "string"},
					{Type: "integer"},
				},
			},
			"one_of_param": {
				OneOf: []*parameters.ParameterMcpManifest{
					{Type: "string", Enum: []any{"red", "green", "blue"}},
					{Type: "null"},
				},
			},
			"all_of_param": {
				AllOf: []*parameters.ParameterMcpManifest{
					{Type: "object"},
					{
						Type: "object",
						Properties: map[string]*parameters.ParameterMcpManifest{
							"sub_key": {Type: "string", Description: "a nested sub property"},
						},
						Required: []string{"sub_key"},
					},
				},
			},
			"not_param": {
				Not: &parameters.ParameterMcpManifest{Type: "string"},
			},
		},
		Required:             []string{"any_of_param"},
		AdditionalProperties: false,
	}

	// Test Marshal
	gotBytes, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("failed to marshal InputSchema: %s", err)
	}

	// Test Unmarshal
	var gotResult InputSchema
	if err := json.Unmarshal(gotBytes, &gotResult); err != nil {
		t.Fatalf("failed to unmarshal InputSchema: %s", err)
	}

	// Verify that all properties and nested fields are unmarshaled and preserved
	if !reflect.DeepEqual(gotResult, input) {
		t.Errorf("unmarshaled schema mismatch\ngot:  %#v\nwant: %#v", gotResult, input)
	}
}
