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

func TestInputSchema_TopLevelComposition(t *testing.T) {
	input := InputSchema{
		Schema: "https://json-schema.org/draft/2020-12/schema",
		OneOf: []*parameters.ParameterMcpManifest{
			{
				Type: "object",
				Properties: map[string]*parameters.ParameterMcpManifest{
					"action": {Type: "string", Enum: []any{"create"}},
					"data":   {Type: "string"},
				},
				Required: []string{"action", "data"},
			},
			{
				Type: "object",
				Properties: map[string]*parameters.ParameterMcpManifest{
					"action": {Type: "string", Enum: []any{"delete"}},
					"id":     {Type: "integer"},
				},
				Required: []string{"action", "id"},
			},
		},
	}

	// Test Marshal
	gotBytes, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("failed to marshal InputSchema: %s", err)
	}

	// Verify that standard flat object fields (like type, properties, required) are omitted from the JSON because of omitempty tags
	var rawMap map[string]any
	if err := json.Unmarshal(gotBytes, &rawMap); err != nil {
		t.Fatalf("failed to unmarshal JSON into map: %s", err)
	}
	if _, ok := rawMap["type"]; ok {
		t.Error("expected 'type' field to be omitted from top-level union InputSchema JSON")
	}
	if _, ok := rawMap["properties"]; ok {
		t.Error("expected 'properties' field to be omitted from top-level union InputSchema JSON")
	}
	if _, ok := rawMap["required"]; ok {
		t.Error("expected 'required' field to be omitted from top-level union InputSchema JSON")
	}

	// Test Unmarshal
	var gotResult InputSchema
	if err := json.Unmarshal(gotBytes, &gotResult); err != nil {
		t.Fatalf("failed to unmarshal InputSchema: %s", err)
	}

	// Verify that the original structure is perfectly preserved
	if !reflect.DeepEqual(gotResult, input) {
		t.Errorf("unmarshaled schema mismatch\ngot:  %#v\nwant: %#v", gotResult, input)
	}
}

func TestInputSchema_EdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input InputSchema
	}{
		{
			name: "simultaneous anyOf and allOf composition",
			input: InputSchema{
				Schema: "https://json-schema.org/draft/2020-12/schema",
				Type:   "object",
				AllOf: []*parameters.ParameterMcpManifest{
					{
						Properties: map[string]*parameters.ParameterMcpManifest{
							"base_id": {Type: "string"},
						},
						Required: []string{"base_id"},
					},
				},
				AnyOf: []*parameters.ParameterMcpManifest{
					{
						Properties: map[string]*parameters.ParameterMcpManifest{
							"token": {Type: "string"},
						},
						Required: []string{"token"},
					},
					{
						Properties: map[string]*parameters.ParameterMcpManifest{
							"api_key": {Type: "string"},
						},
						Required: []string{"api_key"},
					},
				},
			},
		},
		{
			name: "nested conditional validation inside properties",
			input: InputSchema{
				Schema: "https://json-schema.org/draft/2020-12/schema",
				Type:   "object",
				Properties: map[string]parameters.ParameterMcpManifest{
					"connection_config": {
						Type: "object",
						Properties: map[string]*parameters.ParameterMcpManifest{
							"ssl":      {Type: "boolean"},
							"ssl_cert": {Type: "string"},
						},
						If: &parameters.ParameterMcpManifest{
							Properties: map[string]*parameters.ParameterMcpManifest{
								"ssl": {Enum: []any{true}},
							},
						},
						Then: &parameters.ParameterMcpManifest{
							Required: []string{"ssl_cert"},
						},
					},
				},
			},
		},
		{
			name: "conditional validation with if and then only",
			input: InputSchema{
				Schema: "https://json-schema.org/draft/2020-12/schema",
				Type:   "object",
				Properties: map[string]parameters.ParameterMcpManifest{
					"ssl_enabled": {Type: "boolean"},
					"cert_path":   {Type: "string"},
				},
				If: &parameters.ParameterMcpManifest{
					Properties: map[string]*parameters.ParameterMcpManifest{
						"ssl_enabled": {Enum: []any{true}},
					},
				},
				Then: &parameters.ParameterMcpManifest{
					Required: []string{"cert_path"},
				},
			},
		},
		{
			name: "chained if-else-if conditional validation",
			input: InputSchema{
				Schema: "https://json-schema.org/draft/2020-12/schema",
				Type:   "object",
				Properties: map[string]parameters.ParameterMcpManifest{
					"auth_type": {Type: "string", Enum: []any{"basic", "oauth", "none"}},
					"username":  {Type: "string"},
					"token":     {Type: "string"},
				},
				If: &parameters.ParameterMcpManifest{
					Properties: map[string]*parameters.ParameterMcpManifest{
						"auth_type": {Enum: []any{"basic"}},
					},
				},
				Then: &parameters.ParameterMcpManifest{
					Required: []string{"username"},
				},
				Else: &parameters.ParameterMcpManifest{
					If: &parameters.ParameterMcpManifest{
						Properties: map[string]*parameters.ParameterMcpManifest{
							"auth_type": {Enum: []any{"oauth"}},
						},
					},
					Then: &parameters.ParameterMcpManifest{
						Required: []string{"token"},
					},
				},
			},
		},
		{
			name: "top-level conditional validation (if-then-else)",
			input: InputSchema{
				Schema: "https://json-schema.org/draft/2020-12/schema",
				Type:   "object",
				Properties: map[string]parameters.ParameterMcpManifest{
					"auth_method": {Type: "string", Enum: []any{"password", "token"}},
					"password":    {Type: "string"},
					"token":       {Type: "string"},
				},
				If: &parameters.ParameterMcpManifest{
					Properties: map[string]*parameters.ParameterMcpManifest{
						"auth_method": {Enum: []any{"password"}},
					},
				},
				Then: &parameters.ParameterMcpManifest{
					Required: []string{"password"},
				},
				Else: &parameters.ParameterMcpManifest{
					Required: []string{"token"},
				},
			},
		},
		{
			name: "top-level not constraint",
			input: InputSchema{
				Schema: "https://json-schema.org/draft/2020-12/schema",
				Not: &parameters.ParameterMcpManifest{
					Type: "array",
				},
			},
		},
		{
			name: "top-level enum list of constant objects",
			input: InputSchema{
				Schema: "https://json-schema.org/draft/2020-12/schema",
				Enum: []any{
					map[string]any{"action": "ping"},
					map[string]any{"action": "status"},
				},
			},
		},
		{
			name: "hybrid schema - base properties with anyOf composition",
			input: InputSchema{
				Schema: "https://json-schema.org/draft/2020-12/schema",
				Type:   "object",
				Properties: map[string]parameters.ParameterMcpManifest{
					"base_param": {Type: "string"},
				},
				Required: []string{"base_param"},
				AnyOf: []*parameters.ParameterMcpManifest{
					{
						Properties: map[string]*parameters.ParameterMcpManifest{
							"extra_string": {Type: "string"},
						},
						Required: []string{"extra_string"},
					},
					{
						Properties: map[string]*parameters.ParameterMcpManifest{
							"extra_number": {Type: "integer"},
						},
						Required: []string{"extra_number"},
					},
				},
			},
		},
		{
			name: "deeply nested composition",
			input: InputSchema{
				Schema: "https://json-schema.org/draft/2020-12/schema",
				AllOf: []*parameters.ParameterMcpManifest{
					{
						Type: "object",
						Properties: map[string]*parameters.ParameterMcpManifest{
							"level1": {
								Type: "object",
								Properties: map[string]*parameters.ParameterMcpManifest{
									"level2": {
										AnyOf: []*parameters.ParameterMcpManifest{
											{Type: "string"},
											{
												Type: "array",
												Items: &parameters.ParameterMcpManifest{
													Type: "integer",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Test Marshal
			gotBytes, err := json.Marshal(tc.input)
			if err != nil {
				t.Fatalf("failed to marshal InputSchema: %s", err)
			}

			// Test Unmarshal
			var gotResult InputSchema
			if err := json.Unmarshal(gotBytes, &gotResult); err != nil {
				t.Fatalf("failed to unmarshal InputSchema: %s", err)
			}

			// Verify structural preservation
			if !reflect.DeepEqual(gotResult, tc.input) {
				t.Errorf("unmarshaled schema mismatch\ngot:  %#v\nwant: %#v", gotResult, tc.input)
			}
		})
	}
}

func TestCallToolResult_EdgeCases(t *testing.T) {
	type CustomStruct struct {
		FieldA string `json:"field_a"`
		FieldB int    `json:"field_b"`
	}

	tests := []struct {
		name          string
		input         CallToolResult
		wantJSON      string
		wantUnmarshal any
	}{
		{
			name: "structuredContent is nil",
			input: CallToolResult{
				StructuredContent: nil,
			},
			wantJSON:      `{"content":null}`,
			wantUnmarshal: nil,
		},
		{
			name: "structuredContent is empty string",
			input: CallToolResult{
				StructuredContent: "",
			},
			wantJSON:      `{"content":null,"structuredContent":""}`,
			wantUnmarshal: "",
		},
		{
			name: "structuredContent is boolean",
			input: CallToolResult{
				StructuredContent: true,
			},
			wantJSON:      `{"content":null,"structuredContent":true}`,
			wantUnmarshal: true,
		},
		{
			name: "structuredContent is complex nested mixed types",
			input: CallToolResult{
				StructuredContent: []any{
					map[string]any{"nested_map": []any{float64(1), "two"}},
					"flat_string",
				},
			},
			wantJSON: `{"content":null,"structuredContent":[{"nested_map":[1,"two"]},"flat_string"]}`,
			wantUnmarshal: []any{
				map[string]any{"nested_map": []any{float64(1), "two"}},
				"flat_string",
			},
		},
		{
			name: "structuredContent is custom Go struct",
			input: CallToolResult{
				StructuredContent: CustomStruct{
					FieldA: "hello",
					FieldB: 42,
				},
			},
			wantJSON: `{"content":null,"structuredContent":{"field_a":"hello","field_b":42}}`,
			wantUnmarshal: map[string]any{
				"field_a": "hello",
				"field_b": float64(42),
			},
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
			if !reflect.DeepEqual(gotResult.StructuredContent, tc.wantUnmarshal) {
				t.Errorf("got StructuredContent = %#v, want %#v", gotResult.StructuredContent, tc.wantUnmarshal)
			}
		})
	}
}
