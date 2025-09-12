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

package tools_test

import (
	"bytes"
	"encoding/json"
	"math"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

func TestParametersMarshal(t *testing.T) {
	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	tcs := []struct {
		name string
		in   []map[string]any
		want tools.Parameters
	}{
		{
			name: "string",
			in: []map[string]any{
				{
					"name":        "my_string",
					"type":        "string",
					"description": "this param is a string",
				},
			},
			want: tools.Parameters{
				tools.NewStringParameter("my_string", "this param is a string"),
			},
		},
		{
			name: "string not required",
			in: []map[string]any{
				{
					"name":        "my_string",
					"type":        "string",
					"description": "this param is a string",
					"required":    false,
				},
			},
			want: tools.Parameters{
				tools.NewStringParameterWithRequired("my_string", "this param is a string", false),
			},
		},
		{
			name: "int",
			in: []map[string]any{
				{
					"name":        "my_integer",
					"type":        "integer",
					"description": "this param is an int",
				},
			},
			want: tools.Parameters{
				tools.NewIntParameter("my_integer", "this param is an int"),
			},
		},
		{
			name: "int not required",
			in: []map[string]any{
				{
					"name":        "my_integer",
					"type":        "integer",
					"description": "this param is an int",
					"required":    false,
				},
			},
			want: tools.Parameters{
				tools.NewIntParameterWithRequired("my_integer", "this param is an int", false),
			},
		},
		{
			name: "float",
			in: []map[string]any{
				{
					"name":        "my_float",
					"type":        "float",
					"description": "my param is a float",
				},
			},
			want: tools.Parameters{
				tools.NewFloatParameter("my_float", "my param is a float"),
			},
		},
		{
			name: "float not required",
			in: []map[string]any{
				{
					"name":        "my_float",
					"type":        "float",
					"description": "my param is a float",
					"required":    false,
				},
			},
			want: tools.Parameters{
				tools.NewFloatParameterWithRequired("my_float", "my param is a float", false),
			},
		},
		{
			name: "bool",
			in: []map[string]any{
				{
					"name":        "my_bool",
					"type":        "boolean",
					"description": "this param is a boolean",
				},
			},
			want: tools.Parameters{
				tools.NewBooleanParameter("my_bool", "this param is a boolean"),
			},
		},
		{
			name: "bool not required",
			in: []map[string]any{
				{
					"name":        "my_bool",
					"type":        "boolean",
					"description": "this param is a boolean",
					"required":    false,
				},
			},
			want: tools.Parameters{
				tools.NewBooleanParameterWithRequired("my_bool", "this param is a boolean", false),
			},
		},
		{
			name: "string array",
			in: []map[string]any{
				{
					"name":        "my_array",
					"type":        "array",
					"description": "this param is an array of strings",
					"items": map[string]string{
						"name":        "my_string",
						"type":        "string",
						"description": "string item",
					},
				},
			},
			want: tools.Parameters{
				tools.NewArrayParameter("my_array", "this param is an array of strings", tools.NewStringParameter("my_string", "string item")),
			},
		},
		{
			name: "string array not required",
			in: []map[string]any{
				{
					"name":        "my_array",
					"type":        "array",
					"description": "this param is an array of strings",
					"required":    false,
					"items": map[string]string{
						"name":        "my_string",
						"type":        "string",
						"description": "string item",
					},
				},
			},
			want: tools.Parameters{
				tools.NewArrayParameterWithRequired("my_array", "this param is an array of strings", false, tools.NewStringParameter("my_string", "string item")),
			},
		},
		{
			name: "float array",
			in: []map[string]any{
				{
					"name":        "my_array",
					"type":        "array",
					"description": "this param is an array of floats",
					"items": map[string]string{
						"name":        "my_float",
						"type":        "float",
						"description": "float item",
					},
				},
			},
			want: tools.Parameters{
				tools.NewArrayParameter("my_array", "this param is an array of floats", tools.NewFloatParameter("my_float", "float item")),
			},
		},
		{
			name: "string default",
			in: []map[string]any{
				{
					"name":        "my_string",
					"type":        "string",
					"default":     "foo",
					"description": "this param is a string",
				},
			},
			want: tools.Parameters{
				tools.NewStringParameterWithDefault("my_string", "foo", "this param is a string"),
			},
		},
		{
			name: "int default",
			in: []map[string]any{
				{
					"name":        "my_integer",
					"type":        "integer",
					"default":     5,
					"description": "this param is an int",
				},
			},
			want: tools.Parameters{
				tools.NewIntParameterWithDefault("my_integer", 5, "this param is an int"),
			},
		},
		{
			name: "float default",
			in: []map[string]any{
				{
					"name":        "my_float",
					"type":        "float",
					"default":     1.1,
					"description": "my param is a float",
				},
			},
			want: tools.Parameters{
				tools.NewFloatParameterWithDefault("my_float", 1.1, "my param is a float"),
			},
		},
		{
			name: "bool default",
			in: []map[string]any{
				{
					"name":        "my_bool",
					"type":        "boolean",
					"default":     true,
					"description": "this param is a boolean",
				},
			},
			want: tools.Parameters{
				tools.NewBooleanParameterWithDefault("my_bool", true, "this param is a boolean"),
			},
		},
		{
			name: "string array default",
			in: []map[string]any{
				{
					"name":        "my_array",
					"type":        "array",
					"default":     []any{"foo", "bar"},
					"description": "this param is an array of strings",
					"items": map[string]string{
						"name":        "my_string",
						"type":        "string",
						"description": "string item",
					},
				},
			},
			want: tools.Parameters{
				tools.NewArrayParameterWithDefault("my_array", []any{"foo", "bar"}, "this param is an array of strings", tools.NewStringParameter("my_string", "string item")),
			},
		},
		{
			name: "float array default",
			in: []map[string]any{
				{
					"name":        "my_array",
					"type":        "array",
					"default":     []any{1.0, 1.1},
					"description": "this param is an array of floats",
					"items": map[string]string{
						"name":        "my_float",
						"type":        "float",
						"description": "float item",
					},
				},
			},
			want: tools.Parameters{
				tools.NewArrayParameterWithDefault("my_array", []any{1.0, 1.1}, "this param is an array of floats", tools.NewFloatParameter("my_float", "float item")),
			},
		},
		{
			name: "map with string values",
			in: []map[string]any{
				{
					"name":        "my_map",
					"type":        "map",
					"description": "this param is a map of strings",
					"valueType":   "string",
				},
			},
			want: tools.Parameters{
				tools.NewMapParameter("my_map", "this param is a map of strings", "string"),
			},
		},
		{
			name: "map not required",
			in: []map[string]any{
				{
					"name":        "my_map",
					"type":        "map",
					"description": "this param is a map of strings",
					"required":    false,
					"valueType":   "string",
				},
			},
			want: tools.Parameters{
				tools.NewMapParameterWithRequired("my_map", "this param is a map of strings", false, "string"),
			},
		},
		{
			name: "map with default",
			in: []map[string]any{
				{
					"name":        "my_map",
					"type":        "map",
					"description": "this param is a map of strings",
					"default":     map[string]any{"key1": "val1"},
					"valueType":   "string",
				},
			},
			want: tools.Parameters{
				tools.NewMapParameterWithDefault("my_map", map[string]any{"key1": "val1"}, "this param is a map of strings", "string"),
			},
		},
		{
			name: "generic map (no valueType)",
			in: []map[string]any{
				{
					"name":        "my_generic_map",
					"type":        "map",
					"description": "this param is a generic map",
				},
			},
			want: tools.Parameters{
				tools.NewMapParameter("my_generic_map", "this param is a generic map", ""),
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			var got tools.Parameters
			// parse map to bytes
			data, err := yaml.Marshal(tc.in)
			if err != nil {
				t.Fatalf("unable to marshal input to yaml: %s", err)
			}
			// parse bytes to object
			err = yaml.UnmarshalContext(ctx, data, &got)
			if err != nil {
				t.Fatalf("unable to unmarshal: %s", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("incorrect parse: diff %v", diff)
			}
		})
	}
}

func TestAuthParametersMarshal(t *testing.T) {
	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	authServices := []tools.ParamAuthService{{Name: "my-google-auth-service", Field: "user_id"}, {Name: "other-auth-service", Field: "user_id"}}
	tcs := []struct {
		name string
		in   []map[string]any
		want tools.Parameters
	}{
		{
			name: "string",
			in: []map[string]any{
				{
					"name":        "my_string",
					"type":        "string",
					"description": "this param is a string",
					"authServices": []map[string]string{
						{
							"name":  "my-google-auth-service",
							"field": "user_id",
						},
						{
							"name":  "other-auth-service",
							"field": "user_id",
						},
					},
				},
			},
			want: tools.Parameters{
				tools.NewStringParameterWithAuth("my_string", "this param is a string", authServices),
			},
		},
		{
			name: "string with authServices",
			in: []map[string]any{
				{
					"name":        "my_string",
					"type":        "string",
					"description": "this param is a string",
					"authServices": []map[string]string{
						{
							"name":  "my-google-auth-service",
							"field": "user_id",
						},
						{
							"name":  "other-auth-service",
							"field": "user_id",
						},
					},
				},
			},
			want: tools.Parameters{
				tools.NewStringParameterWithAuth("my_string", "this param is a string", authServices),
			},
		},
		{
			name: "int",
			in: []map[string]any{
				{
					"name":        "my_integer",
					"type":        "integer",
					"description": "this param is an int",
					"authServices": []map[string]string{
						{
							"name":  "my-google-auth-service",
							"field": "user_id",
						},
						{
							"name":  "other-auth-service",
							"field": "user_id",
						},
					},
				},
			},
			want: tools.Parameters{
				tools.NewIntParameterWithAuth("my_integer", "this param is an int", authServices),
			},
		},
		{
			name: "int with authServices",
			in: []map[string]any{
				{
					"name":        "my_integer",
					"type":        "integer",
					"description": "this param is an int",
					"authServices": []map[string]string{
						{
							"name":  "my-google-auth-service",
							"field": "user_id",
						},
						{
							"name":  "other-auth-service",
							"field": "user_id",
						},
					},
				},
			},
			want: tools.Parameters{
				tools.NewIntParameterWithAuth("my_integer", "this param is an int", authServices),
			},
		},
		{
			name: "float",
			in: []map[string]any{
				{
					"name":        "my_float",
					"type":        "float",
					"description": "my param is a float",
					"authServices": []map[string]string{
						{
							"name":  "my-google-auth-service",
							"field": "user_id",
						},
						{
							"name":  "other-auth-service",
							"field": "user_id",
						},
					},
				},
			},
			want: tools.Parameters{
				tools.NewFloatParameterWithAuth("my_float", "my param is a float", authServices),
			},
		},
		{
			name: "float with authServices",
			in: []map[string]any{
				{
					"name":        "my_float",
					"type":        "float",
					"description": "my param is a float",
					"authServices": []map[string]string{
						{
							"name":  "my-google-auth-service",
							"field": "user_id",
						},
						{
							"name":  "other-auth-service",
							"field": "user_id",
						},
					},
				},
			},
			want: tools.Parameters{
				tools.NewFloatParameterWithAuth("my_float", "my param is a float", authServices),
			},
		},
		{
			name: "bool",
			in: []map[string]any{
				{
					"name":        "my_bool",
					"type":        "boolean",
					"description": "this param is a boolean",
					"authServices": []map[string]string{
						{
							"name":  "my-google-auth-service",
							"field": "user_id",
						},
						{
							"name":  "other-auth-service",
							"field": "user_id",
						},
					},
				},
			},
			want: tools.Parameters{
				tools.NewBooleanParameterWithAuth("my_bool", "this param is a boolean", authServices),
			},
		},
		{
			name: "bool with authServices",
			in: []map[string]any{
				{
					"name":        "my_bool",
					"type":        "boolean",
					"description": "this param is a boolean",
					"authServices": []map[string]string{
						{
							"name":  "my-google-auth-service",
							"field": "user_id",
						},
						{
							"name":  "other-auth-service",
							"field": "user_id",
						},
					},
				},
			},
			want: tools.Parameters{
				tools.NewBooleanParameterWithAuth("my_bool", "this param is a boolean", authServices),
			},
		},
		{
			name: "string array",
			in: []map[string]any{
				{
					"name":        "my_array",
					"type":        "array",
					"description": "this param is an array of strings",
					"items": map[string]string{
						"name":        "my_string",
						"type":        "string",
						"description": "string item",
					},
					"authServices": []map[string]string{
						{
							"name":  "my-google-auth-service",
							"field": "user_id",
						},
						{
							"name":  "other-auth-service",
							"field": "user_id",
						},
					},
				},
			},
			want: tools.Parameters{
				tools.NewArrayParameterWithAuth("my_array", "this param is an array of strings", tools.NewStringParameter("my_string", "string item"), authServices),
			},
		},
		{
			name: "string array with authServices",
			in: []map[string]any{
				{
					"name":        "my_array",
					"type":        "array",
					"description": "this param is an array of strings",
					"items": map[string]string{
						"name":        "my_string",
						"type":        "string",
						"description": "string item",
					},
					"authServices": []map[string]string{
						{
							"name":  "my-google-auth-service",
							"field": "user_id",
						},
						{
							"name":  "other-auth-service",
							"field": "user_id",
						},
					},
				},
			},
			want: tools.Parameters{
				tools.NewArrayParameterWithAuth("my_array", "this param is an array of strings", tools.NewStringParameter("my_string", "string item"), authServices),
			},
		},
		{
			name: "float array",
			in: []map[string]any{
				{
					"name":        "my_array",
					"type":        "array",
					"description": "this param is an array of floats",
					"items": map[string]string{
						"name":        "my_float",
						"type":        "float",
						"description": "float item",
					},
					"authServices": []map[string]string{
						{
							"name":  "my-google-auth-service",
							"field": "user_id",
						},
						{
							"name":  "other-auth-service",
							"field": "user_id",
						},
					},
				},
			},
			want: tools.Parameters{
				tools.NewArrayParameterWithAuth("my_array", "this param is an array of floats", tools.NewFloatParameter("my_float", "float item"), authServices),
			},
		},
		{
			name: "map",
			in: []map[string]any{
				{
					"name":        "my_map",
					"type":        "map",
					"description": "this param is a map of strings",
					"valueType":   "string",
					"authServices": []map[string]string{
						{"name": "my-google-auth-service", "field": "user_id"},
						{"name": "other-auth-service", "field": "user_id"},
					},
				},
			},
			want: tools.Parameters{
				tools.NewMapParameterWithAuth("my_map", "this param is a map of strings", "string", authServices),
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			var got tools.Parameters
			// parse map to bytes
			data, err := yaml.Marshal(tc.in)
			if err != nil {
				t.Fatalf("unable to marshal input to yaml: %s", err)
			}
			// parse bytes to object
			err = yaml.UnmarshalContext(ctx, data, &got)
			if err != nil {
				t.Fatalf("unable to unmarshal: %s", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("incorrect parse: diff %v", diff)
			}
		})
	}
}

func TestParametersParse(t *testing.T) {
	tcs := []struct {
		name   string
		params tools.Parameters
		in     map[string]any
		want   tools.ParamValues
	}{
		// ... (primitive type tests are unchanged)
		{
			name: "string",
			params: tools.Parameters{
				tools.NewStringParameter("my_string", "this param is a string"),
			},
			in: map[string]any{
				"my_string": "hello world",
			},
			want: tools.ParamValues{tools.ParamValue{Name: "my_string", Value: "hello world"}},
		},
		{
			name: "not string",
			params: tools.Parameters{
				tools.NewStringParameter("my_string", "this param is a string"),
			},
			in: map[string]any{
				"my_string": 4,
			},
		},
		{
			name: "int",
			params: tools.Parameters{
				tools.NewIntParameter("my_int", "this param is an int"),
			},
			in: map[string]any{
				"my_int": 100,
			},
			want: tools.ParamValues{tools.ParamValue{Name: "my_int", Value: 100}},
		},
		{
			name: "not int",
			params: tools.Parameters{
				tools.NewIntParameter("my_int", "this param is an int"),
			},
			in: map[string]any{
				"my_int": 14.5,
			},
		},
		{
			name: "not int (big)",
			params: tools.Parameters{
				tools.NewIntParameter("my_int", "this param is an int"),
			},
			in: map[string]any{
				"my_int": math.MaxInt64,
			},
			want: tools.ParamValues{tools.ParamValue{Name: "my_int", Value: math.MaxInt64}},
		},
		{
			name: "float",
			params: tools.Parameters{
				tools.NewFloatParameter("my_float", "this param is a float"),
			},
			in: map[string]any{
				"my_float": 1.5,
			},
			want: tools.ParamValues{tools.ParamValue{Name: "my_float", Value: 1.5}},
		},
		{
			name: "not float",
			params: tools.Parameters{
				tools.NewFloatParameter("my_float", "this param is a float"),
			},
			in: map[string]any{
				"my_float": true,
			},
		},
		{
			name: "bool",
			params: tools.Parameters{
				tools.NewBooleanParameter("my_bool", "this param is a bool"),
			},
			in: map[string]any{
				"my_bool": true,
			},
			want: tools.ParamValues{tools.ParamValue{Name: "my_bool", Value: true}},
		},
		{
			name: "not bool",
			params: tools.Parameters{
				tools.NewBooleanParameter("my_bool", "this param is a bool"),
			},
			in: map[string]any{
				"my_bool": 1.5,
			},
		},
		{
			name: "string default",
			params: tools.Parameters{
				tools.NewStringParameterWithDefault("my_string", "foo", "this param is a string"),
			},
			in:   map[string]any{},
			want: tools.ParamValues{tools.ParamValue{Name: "my_string", Value: "foo"}},
		},
		{
			name: "int default",
			params: tools.Parameters{
				tools.NewIntParameterWithDefault("my_int", 100, "this param is an int"),
			},
			in:   map[string]any{},
			want: tools.ParamValues{tools.ParamValue{Name: "my_int", Value: 100}},
		},
		{
			name: "int (big)",
			params: tools.Parameters{
				tools.NewIntParameterWithDefault("my_big_int", math.MaxInt64, "this param is an int"),
			},
			in:   map[string]any{},
			want: tools.ParamValues{tools.ParamValue{Name: "my_big_int", Value: math.MaxInt64}},
		},
		{
			name: "float default",
			params: tools.Parameters{
				tools.NewFloatParameterWithDefault("my_float", 1.1, "this param is a float"),
			},
			in:   map[string]any{},
			want: tools.ParamValues{tools.ParamValue{Name: "my_float", Value: 1.1}},
		},
		{
			name: "bool default",
			params: tools.Parameters{
				tools.NewBooleanParameterWithDefault("my_bool", true, "this param is a bool"),
			},
			in:   map[string]any{},
			want: tools.ParamValues{tools.ParamValue{Name: "my_bool", Value: true}},
		},
		{
			name: "string not required",
			params: tools.Parameters{
				tools.NewStringParameterWithRequired("my_string", "this param is a string", false),
			},
			in:   map[string]any{},
			want: tools.ParamValues{tools.ParamValue{Name: "my_string", Value: nil}},
		},
		{
			name: "int not required",
			params: tools.Parameters{
				tools.NewIntParameterWithRequired("my_int", "this param is an int", false),
			},
			in:   map[string]any{},
			want: tools.ParamValues{tools.ParamValue{Name: "my_int", Value: nil}},
		},
		{
			name: "float not required",
			params: tools.Parameters{
				tools.NewFloatParameterWithRequired("my_float", "this param is a float", false),
			},
			in:   map[string]any{},
			want: tools.ParamValues{tools.ParamValue{Name: "my_float", Value: nil}},
		},
		{
			name: "bool not required",
			params: tools.Parameters{
				tools.NewBooleanParameterWithRequired("my_bool", "this param is a bool", false),
			},
			in:   map[string]any{},
			want: tools.ParamValues{tools.ParamValue{Name: "my_bool", Value: nil}},
		},
		{
			name: "map",
			params: tools.Parameters{
				tools.NewMapParameter("my_map", "a map", "string"),
			},
			in: map[string]any{
				"my_map": map[string]any{"key1": "val1", "key2": "val2"},
			},
			want: tools.ParamValues{tools.ParamValue{Name: "my_map", Value: map[string]any{"key1": "val1", "key2": "val2"}}},
		},
		{
			name: "generic map",
			params: tools.Parameters{
				tools.NewMapParameter("my_map_generic_type", "a generic map", ""),
			},
			in: map[string]any{
				"my_map_generic_type": map[string]any{"key1": "val1", "key2": 123, "key3": true},
			},
			want: tools.ParamValues{tools.ParamValue{Name: "my_map_generic_type", Value: map[string]any{"key1": "val1", "key2": int64(123), "key3": true}}},
		},
		{
			name: "not map (value type mismatch)",
			params: tools.Parameters{
				tools.NewMapParameter("my_map", "a map", "string"),
			},
			in: map[string]any{
				"my_map": map[string]any{"key1": 123},
			},
		},
		{
			name: "map default",
			params: tools.Parameters{
				tools.NewMapParameterWithDefault("my_map_default", map[string]any{"default_key": "default_val"}, "a map", "string"),
			},
			in:   map[string]any{},
			want: tools.ParamValues{tools.ParamValue{Name: "my_map_default", Value: map[string]any{"default_key": "default_val"}}},
		},
		{
			name: "map not required",
			params: tools.Parameters{
				tools.NewMapParameterWithRequired("my_map_not_required", "a map", false, "string"),
			},
			in:   map[string]any{},
			want: tools.ParamValues{tools.ParamValue{Name: "my_map_not_required", Value: nil}},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			// parse map to bytes
			data, err := json.Marshal(tc.in)
			if err != nil {
				t.Fatalf("unable to marshal input to yaml: %s", err)
			}
			// parse bytes to object
			var m map[string]any

			d := json.NewDecoder(bytes.NewReader(data))
			d.UseNumber()
			err = d.Decode(&m)
			if err != nil {
				t.Fatalf("unable to unmarshal: %s", err)
			}

			wantErr := len(tc.want) == 0 // error is expected if no items in want
			gotAll, err := tools.ParseParams(tc.params, m, make(map[string]map[string]any))
			if err != nil {
				if wantErr {
					return
				}
				t.Fatalf("unexpected error from ParseParams: %s", err)
			}
			if wantErr {
				t.Fatalf("expected error but Param parsed successfully: %s", gotAll)
			}

			// Use cmp.Diff for robust comparison
			if diff := cmp.Diff(tc.want, gotAll); diff != "" {
				t.Fatalf("ParseParams() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAuthParametersParse(t *testing.T) {
	authServices := []tools.ParamAuthService{
		{
			Name:  "my-google-auth-service",
			Field: "auth_field",
		},
		{
			Name:  "other-auth-service",
			Field: "other_auth_field",
		}}
	tcs := []struct {
		name      string
		params    tools.Parameters
		in        map[string]any
		claimsMap map[string]map[string]any
		want      tools.ParamValues
	}{
		{
			name: "string",
			params: tools.Parameters{
				tools.NewStringParameterWithAuth("my_string", "this param is a string", authServices),
			},
			in: map[string]any{
				"my_string": "hello world",
			},
			claimsMap: map[string]map[string]any{"my-google-auth-service": {"auth_field": "hello"}},
			want:      tools.ParamValues{tools.ParamValue{Name: "my_string", Value: "hello"}},
		},
		{
			name: "not string",
			params: tools.Parameters{
				tools.NewStringParameterWithAuth("my_string", "this param is a string", authServices),
			},
			in: map[string]any{
				"my_string": 4,
			},
			claimsMap: map[string]map[string]any{},
		},
		{
			name: "int",
			params: tools.Parameters{
				tools.NewIntParameterWithAuth("my_int", "this param is an int", authServices),
			},
			in: map[string]any{
				"my_int": 100,
			},
			claimsMap: map[string]map[string]any{"other-auth-service": {"other_auth_field": 120}},
			want:      tools.ParamValues{tools.ParamValue{Name: "my_int", Value: 120}},
		},
		{
			name: "not int",
			params: tools.Parameters{
				tools.NewIntParameterWithAuth("my_int", "this param is an int", authServices),
			},
			in: map[string]any{
				"my_int": 14.5,
			},
			claimsMap: map[string]map[string]any{},
		},
		{
			name: "float",
			params: tools.Parameters{
				tools.NewFloatParameterWithAuth("my_float", "this param is a float", authServices),
			},
			in: map[string]any{
				"my_float": 1.5,
			},
			claimsMap: map[string]map[string]any{"my-google-auth-service": {"auth_field": 2.1}},
			want:      tools.ParamValues{tools.ParamValue{Name: "my_float", Value: 2.1}},
		},
		{
			name: "not float",
			params: tools.Parameters{
				tools.NewFloatParameterWithAuth("my_float", "this param is a float", authServices),
			},
			in: map[string]any{
				"my_float": true,
			},
			claimsMap: map[string]map[string]any{},
		},
		{
			name: "bool",
			params: tools.Parameters{
				tools.NewBooleanParameterWithAuth("my_bool", "this param is a bool", authServices),
			},
			in: map[string]any{
				"my_bool": true,
			},
			claimsMap: map[string]map[string]any{"my-google-auth-service": {"auth_field": false}},
			want:      tools.ParamValues{tools.ParamValue{Name: "my_bool", Value: false}},
		},
		{
			name: "not bool",
			params: tools.Parameters{
				tools.NewBooleanParameterWithAuth("my_bool", "this param is a bool", authServices),
			},
			in: map[string]any{
				"my_bool": 1.5,
			},
			claimsMap: map[string]map[string]any{},
		},
		{
			name: "username",
			params: tools.Parameters{
				tools.NewStringParameterWithAuth("username", "username string", authServices),
			},
			in: map[string]any{
				"username": "Violet",
			},
			claimsMap: map[string]map[string]any{"my-google-auth-service": {"auth_field": "Alice"}},
			want:      tools.ParamValues{tools.ParamValue{Name: "username", Value: "Alice"}},
		},
		{
			name: "expect claim error",
			params: tools.Parameters{
				tools.NewStringParameterWithAuth("username", "username string", authServices),
			},
			in: map[string]any{
				"username": "Violet",
			},
			claimsMap: map[string]map[string]any{"my-google-auth-service": {"not_an_auth_field": "Alice"}},
		},
		{
			name: "map",
			params: tools.Parameters{
				tools.NewMapParameterWithAuth("my_map", "a map", "string", authServices),
			},
			in:        map[string]any{"my_map": map[string]any{"key1": "val1"}},
			claimsMap: map[string]map[string]any{"my-google-auth-service": {"auth_field": map[string]any{"authed_key": "authed_val"}}},
			want:      tools.ParamValues{tools.ParamValue{Name: "my_map", Value: map[string]any{"authed_key": "authed_val"}}},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			// parse map to bytes
			data, err := json.Marshal(tc.in)
			if err != nil {
				t.Fatalf("unable to marshal input to yaml: %s", err)
			}
			// parse bytes to object
			var m map[string]any
			d := json.NewDecoder(bytes.NewReader(data))
			d.UseNumber()
			err = d.Decode(&m)
			if err != nil {
				t.Fatalf("unable to unmarshal: %s", err)
			}

			gotAll, err := tools.ParseParams(tc.params, m, tc.claimsMap)
			if err != nil {
				if len(tc.want) == 0 {
					// error is expected if no items in want
					return
				}
				t.Fatalf("unexpected error from ParseParams: %s", err)
			}

			if diff := cmp.Diff(tc.want, gotAll); diff != "" {
				t.Fatalf("ParseParams() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParamValues(t *testing.T) {
	tcs := []struct {
		name              string
		in                tools.ParamValues
		wantSlice         []any
		wantMap           map[string]interface{}
		wantMapOrdered    map[string]interface{}
		wantMapWithDollar map[string]interface{}
	}{
		{
			name:           "string",
			in:             tools.ParamValues{tools.ParamValue{Name: "my_bool", Value: true}, tools.ParamValue{Name: "my_string", Value: "hello world"}},
			wantSlice:      []any{true, "hello world"},
			wantMap:        map[string]interface{}{"my_bool": true, "my_string": "hello world"},
			wantMapOrdered: map[string]interface{}{"p1": true, "p2": "hello world"},
			wantMapWithDollar: map[string]interface{}{
				"$my_bool":   true,
				"$my_string": "hello world",
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			gotSlice := tc.in.AsSlice()
			gotMap := tc.in.AsMap()
			gotMapOrdered := tc.in.AsMapByOrderedKeys()
			gotMapWithDollar := tc.in.AsMapWithDollarPrefix()

			for i, got := range gotSlice {
				want := tc.wantSlice[i]
				if got != want {
					t.Fatalf("unexpected value: got %q, want %q", got, want)
				}
			}
			for i, got := range gotMap {
				want := tc.wantMap[i]
				if got != want {
					t.Fatalf("unexpected value: got %q, want %q", got, want)
				}
			}
			for i, got := range gotMapOrdered {
				want := tc.wantMapOrdered[i]
				if got != want {
					t.Fatalf("unexpected value: got %q, want %q", got, want)
				}
			}
			for key, got := range gotMapWithDollar {
				want := tc.wantMapWithDollar[key]
				if got != want {
					t.Fatalf("unexpected value in AsMapWithDollarPrefix: got %q, want %q", got, want)
				}
			}
		})
	}
}

func TestParamManifest(t *testing.T) {
	tcs := []struct {
		name string
		in   tools.Parameter
		want tools.ParameterManifest
	}{
		{
			name: "string",
			in:   tools.NewStringParameter("foo-string", "bar"),
			want: tools.ParameterManifest{Name: "foo-string", Type: "string", Required: true, Description: "bar", AuthServices: []string{}},
		},
		{
			name: "int",
			in:   tools.NewIntParameter("foo-int", "bar"),
			want: tools.ParameterManifest{Name: "foo-int", Type: "integer", Required: true, Description: "bar", AuthServices: []string{}},
		},
		{
			name: "float",
			in:   tools.NewFloatParameter("foo-float", "bar"),
			want: tools.ParameterManifest{Name: "foo-float", Type: "float", Required: true, Description: "bar", AuthServices: []string{}},
		},
		{
			name: "boolean",
			in:   tools.NewBooleanParameter("foo-bool", "bar"),
			want: tools.ParameterManifest{Name: "foo-bool", Type: "boolean", Required: true, Description: "bar", AuthServices: []string{}},
		},
		{
			name: "array",
			in:   tools.NewArrayParameter("foo-array", "bar", tools.NewStringParameter("foo-string", "bar")),
			want: tools.ParameterManifest{
				Name:         "foo-array",
				Type:         "array",
				Required:     true,
				Description:  "bar",
				AuthServices: []string{},
				Items:        &tools.ParameterManifest{Name: "foo-string", Type: "string", Required: true, Description: "bar", AuthServices: []string{}},
			},
		},
		{
			name: "string default",
			in:   tools.NewStringParameterWithDefault("foo-string", "foo", "bar"),
			want: tools.ParameterManifest{Name: "foo-string", Type: "string", Required: false, Description: "bar", AuthServices: []string{}},
		},
		{
			name: "int default",
			in:   tools.NewIntParameterWithDefault("foo-int", 1, "bar"),
			want: tools.ParameterManifest{Name: "foo-int", Type: "integer", Required: false, Description: "bar", AuthServices: []string{}},
		},
		{
			name: "float default",
			in:   tools.NewFloatParameterWithDefault("foo-float", 1.1, "bar"),
			want: tools.ParameterManifest{Name: "foo-float", Type: "float", Required: false, Description: "bar", AuthServices: []string{}},
		},
		{
			name: "boolean default",
			in:   tools.NewBooleanParameterWithDefault("foo-bool", true, "bar"),
			want: tools.ParameterManifest{Name: "foo-bool", Type: "boolean", Required: false, Description: "bar", AuthServices: []string{}},
		},
		{
			name: "array default",
			in:   tools.NewArrayParameterWithDefault("foo-array", []any{"foo", "bar"}, "bar", tools.NewStringParameter("foo-string", "bar")),
			want: tools.ParameterManifest{
				Name:         "foo-array",
				Type:         "array",
				Required:     false,
				Description:  "bar",
				AuthServices: []string{},
				Items:        &tools.ParameterManifest{Name: "foo-string", Type: "string", Required: false, Description: "bar", AuthServices: []string{}},
			},
		},
		{
			name: "string not required",
			in:   tools.NewStringParameterWithRequired("foo-string", "bar", false),
			want: tools.ParameterManifest{Name: "foo-string", Type: "string", Required: false, Description: "bar", AuthServices: []string{}},
		},
		{
			name: "int not required",
			in:   tools.NewIntParameterWithRequired("foo-int", "bar", false),
			want: tools.ParameterManifest{Name: "foo-int", Type: "integer", Required: false, Description: "bar", AuthServices: []string{}},
		},
		{
			name: "float not required",
			in:   tools.NewFloatParameterWithRequired("foo-float", "bar", false),
			want: tools.ParameterManifest{Name: "foo-float", Type: "float", Required: false, Description: "bar", AuthServices: []string{}},
		},
		{
			name: "boolean not required",
			in:   tools.NewBooleanParameterWithRequired("foo-bool", "bar", false),
			want: tools.ParameterManifest{Name: "foo-bool", Type: "boolean", Required: false, Description: "bar", AuthServices: []string{}},
		},
		{
			name: "array not required",
			in:   tools.NewArrayParameterWithRequired("foo-array", "bar", false, tools.NewStringParameter("foo-string", "bar")),
			want: tools.ParameterManifest{
				Name:         "foo-array",
				Type:         "array",
				Required:     false,
				Description:  "bar",
				AuthServices: []string{},
				Items:        &tools.ParameterManifest{Name: "foo-string", Type: "string", Required: false, Description: "bar", AuthServices: []string{}},
			},
		},
		{
			name: "map with string values",
			in:   tools.NewMapParameter("foo-map", "bar", "string"),
			want: tools.ParameterManifest{
				Name:                 "foo-map",
				Type:                 "object",
				Required:             true,
				Description:          "bar",
				AuthServices:         []string{},
				AdditionalProperties: &tools.ParameterManifest{Name: "", Type: "string", Required: true, Description: "", AuthServices: []string{}},
			},
		},
		{
			name: "map not required",
			in:   tools.NewMapParameterWithRequired("foo-map", "bar", false, "string"),
			want: tools.ParameterManifest{
				Name:                 "foo-map",
				Type:                 "object",
				Required:             false,
				Description:          "bar",
				AuthServices:         []string{},
				AdditionalProperties: &tools.ParameterManifest{Name: "", Type: "string", Required: true, Description: "", AuthServices: []string{}},
			},
		},
		{
			name: "generic map (additionalProperties true)",
			in:   tools.NewMapParameter("foo-map", "bar", ""),
			want: tools.ParameterManifest{
				Name:                 "foo-map",
				Type:                 "object",
				Required:             true,
				Description:          "bar",
				AuthServices:         []string{},
				AdditionalProperties: true,
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.in.Manifest()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("unexpected manifest (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParamMcpManifest(t *testing.T) {
	tcs := []struct {
		name string
		in   tools.Parameter
		want tools.ParameterMcpManifest
	}{
		{
			name: "string",
			in:   tools.NewStringParameter("foo-string", "bar"),
			want: tools.ParameterMcpManifest{Type: "string", Description: "bar"},
		},
		{
			name: "int",
			in:   tools.NewIntParameter("foo-int", "bar"),
			want: tools.ParameterMcpManifest{Type: "integer", Description: "bar"},
		},
		{
			name: "float",
			in:   tools.NewFloatParameter("foo-float", "bar"),
			want: tools.ParameterMcpManifest{Type: "number", Description: "bar"},
		},
		{
			name: "boolean",
			in:   tools.NewBooleanParameter("foo-bool", "bar"),
			want: tools.ParameterMcpManifest{Type: "boolean", Description: "bar"},
		},
		{
			name: "array",
			in:   tools.NewArrayParameter("foo-array", "bar", tools.NewStringParameter("foo-string", "bar")),
			want: tools.ParameterMcpManifest{
				Type:        "array",
				Description: "bar",
				Items:       &tools.ParameterMcpManifest{Type: "string", Description: "bar"},
			},
		},

		{
			name: "map with string values",
			in:   tools.NewMapParameter("foo-map", "bar", "string"),
			want: tools.ParameterMcpManifest{
				Type:                 "object",
				Description:          "bar",
				AdditionalProperties: &tools.ParameterMcpManifest{Type: "string", Description: ""},
			},
		},
		{
			name: "generic map (additionalProperties true)",
			in:   tools.NewMapParameter("foo-map", "bar", ""),
			want: tools.ParameterMcpManifest{
				Type:                 "object",
				Description:          "bar",
				AdditionalProperties: true,
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.in.McpManifest()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("unexpected manifest (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMcpManifest(t *testing.T) {
	tcs := []struct {
		name string
		in   tools.Parameters
		want tools.McpToolsSchema
	}{
		{
			name: "all types",
			in: tools.Parameters{
				tools.NewStringParameterWithDefault("foo-string", "foo", "bar"),
				tools.NewStringParameter("foo-string2", "bar"),
				tools.NewIntParameter("foo-int2", "bar"),
				tools.NewFloatParameter("foo-float", "bar"),
				tools.NewArrayParameter("foo-array2", "bar", tools.NewStringParameter("foo-string", "bar")),
				tools.NewMapParameter("foo-map-int", "a map of ints", "integer"),
				tools.NewMapParameter("foo-map-any", "a map of any", ""),
			},
			want: tools.McpToolsSchema{
				Type: "object",
				Properties: map[string]tools.ParameterMcpManifest{
					"foo-string":  {Type: "string", Description: "bar"},
					"foo-string2": {Type: "string", Description: "bar"},
					"foo-int2":    {Type: "integer", Description: "bar"},
					"foo-float":   {Type: "number", Description: "bar"},
					"foo-array2": {
						Type:        "array",
						Description: "bar",
						Items:       &tools.ParameterMcpManifest{Type: "string", Description: "bar"},
					},
					"foo-map-int": {
						Type:                 "object",
						Description:          "a map of ints",
						AdditionalProperties: &tools.ParameterMcpManifest{Type: "integer", Description: ""},
					},
					"foo-map-any": {
						Type:                 "object",
						Description:          "a map of any",
						AdditionalProperties: true,
					},
				},
				Required: []string{"foo-string2", "foo-int2", "foo-float", "foo-array2", "foo-map-int", "foo-map-any"},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.in.McpManifest()
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("unexpected manifest (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFailParametersUnmarshal(t *testing.T) {
	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	tcs := []struct {
		name string
		in   []map[string]any
		err  string
	}{
		{
			name: "common parameter missing name",
			in: []map[string]any{
				{
					"type":        "string",
					"description": "this is a param for string",
				},
			},
			err: "unable to parse as \"string\": Key: 'CommonParameter.Name' Error:Field validation for 'Name' failed on the 'required' tag",
		},
		{
			name: "common parameter missing type",
			in: []map[string]any{
				{
					"name":        "string",
					"description": "this is a param for string",
				},
			},
			err: "parameter is missing 'type' field: %!w(<nil>)",
		},
		{
			name: "common parameter missing description",
			in: []map[string]any{
				{
					"name": "my_string",
					"type": "string",
				},
			},
			err: "unable to parse as \"string\": Key: 'CommonParameter.Desc' Error:Field validation for 'Desc' failed on the 'required' tag",
		},
		{
			name: "array parameter missing items",
			in: []map[string]any{
				{
					"name":        "my_array",
					"type":        "array",
					"description": "this param is an array of strings",
				},
			},
			err: "unable to parse as \"array\": unable to parse 'items' field: error parsing parameters: nothing to unmarshal",
		},
		{
			name: "array parameter missing items' name",
			in: []map[string]any{
				{
					"name":        "my_array",
					"type":        "array",
					"description": "this param is an array of strings",
					"items": map[string]string{
						"type":        "string",
						"description": "string item",
					},
				},
			},
			err: "unable to parse as \"array\": unable to parse 'items' field: unable to parse as \"string\": Key: 'CommonParameter.Name' Error:Field validation for 'Name' failed on the 'required' tag",
		},
		// --- MODIFIED MAP PARAMETER TEST ---
		{
			name: "map with invalid valueType",
			in: []map[string]any{
				{
					"name":        "my_map",
					"type":        "map",
					"description": "this param is a map",
					"valueType":   "not-a-real-type",
				},
			},
			err: "unsupported valueType \"not-a-real-type\" for map parameter",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			var got tools.Parameters
			// parse map to bytes
			data, err := yaml.Marshal(tc.in)
			if err != nil {
				t.Fatalf("unable to marshal input to yaml: %s", err)
			}
			// parse bytes to object
			err = yaml.UnmarshalContext(ctx, data, &got)
			if err == nil {
				t.Fatalf("expect parsing to fail")
			}
			errStr := err.Error()

			if !strings.Contains(errStr, tc.err) {
				t.Fatalf("unexpected error: got %q, want to contain %q", errStr, tc.err)
			}
		})
	}
}

// ... (Remaining test functions do not involve parameter definitions and need no changes)

func TestConvertArrayParamToString(t *testing.T) {

	tcs := []struct {
		name string
		in   []any
		want string
	}{
		{
			in: []any{
				"id",
				"name",
				"location",
			},
			want: "id, name, location",
		},
		{
			in: []any{
				"id",
			},
			want: "id",
		},
		{
			in: []any{
				"id",
				"5",
				"false",
			},
			want: "id, 5, false",
		},
		{
			in:   []any{},
			want: "",
		},
		{
			in:   []any{},
			want: "",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got, _ := tools.ConvertArrayParamToString(tc.in)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("incorrect array param conversion: diff %v", diff)
			}
		})
	}
}

func TestFailConvertArrayParamToString(t *testing.T) {
	tcs := []struct {
		name string
		in   []any
		err  string
	}{
		{
			in:  []any{5, 10, 15},
			err: "templateParameter only supports string arrays",
		},
		{
			in:  []any{"id", "name", 15},
			err: "templateParameter only supports string arrays",
		},
		{
			in:  []any{false},
			err: "templateParameter only supports string arrays",
		},
		{
			in:  []any{10, true},
			err: "templateParameter only supports string arrays",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tools.ConvertArrayParamToString(tc.in)
			errStr := err.Error()
			if errStr != tc.err {
				t.Fatalf("unexpected error: got %q, want %q", errStr, tc.err)
			}
		})
	}
}

func TestGetParams(t *testing.T) {
	tcs := []struct {
		name   string
		in     map[string]any
		params tools.Parameters
		want   tools.ParamValues
	}{
		{
			name: "parameters to include and exclude",
			params: tools.Parameters{
				tools.NewStringParameter("my_string_inc", "this should be included"),
				tools.NewStringParameter("my_string_inc2", "this should be included"),
			},
			in: map[string]any{
				"my_string_inc":  "hello world A",
				"my_string_inc2": "hello world B",
				"my_string_exc":  "hello world C",
			},
			want: tools.ParamValues{
				tools.ParamValue{Name: "my_string_inc", Value: "hello world A"},
				tools.ParamValue{Name: "my_string_inc2", Value: "hello world B"},
			},
		},
		{
			name: "include all",
			params: tools.Parameters{
				tools.NewStringParameter("my_string_inc", "this should be included"),
			},
			in: map[string]any{
				"my_string_inc": "hello world A",
			},
			want: tools.ParamValues{
				tools.ParamValue{Name: "my_string_inc", Value: "hello world A"},
			},
		},
		{
			name:   "exclude all",
			params: tools.Parameters{},
			in: map[string]any{
				"my_string_exc":  "hello world A",
				"my_string_exc2": "hello world B",
			},
			want: tools.ParamValues{},
		},
		{
			name:   "empty",
			params: tools.Parameters{},
			in:     map[string]any{},
			want:   tools.ParamValues{},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got, _ := tools.GetParams(tc.params, tc.in)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("incorrect get params: diff %v", diff)
			}
		})
	}
}

func TestFailGetParams(t *testing.T) {

	tcs := []struct {
		name   string
		params tools.Parameters
		in     map[string]any
		err    string
	}{
		{
			name:   "missing the only parameter",
			params: tools.Parameters{tools.NewStringParameter("my_string", "this was missing")},
			in:     map[string]any{},
			err:    "missing parameter my_string",
		},
		{
			name: "missing one parameter of multiple",
			params: tools.Parameters{
				tools.NewStringParameter("my_string_inc", "this should be included"),
				tools.NewStringParameter("my_string_exc", "this was missing"),
			},
			in: map[string]any{
				"my_string_inc": "hello world A",
			},
			err: "missing parameter my_string_exc",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tools.GetParams(tc.params, tc.in)
			errStr := err.Error()
			if errStr != tc.err {
				t.Fatalf("unexpected error: got %q, want %q", errStr, tc.err)
			}
		})
	}
}

func TestResolveTemplateParameters(t *testing.T) {
	tcs := []struct {
		name           string
		templateParams tools.Parameters
		statement      string
		in             map[string]any
		want           string
	}{
		{
			name: "single template parameter",
			templateParams: tools.Parameters{
				tools.NewStringParameter("tableName", "this is a string template parameter"),
			},
			statement: "SELECT * FROM {{.tableName}}",
			in: map[string]any{
				"tableName": "hotels",
			},
			want: "SELECT * FROM hotels",
		},
		{
			name: "multiple template parameters",
			templateParams: tools.Parameters{
				tools.NewStringParameter("tableName", "this is a string template parameter"),
				tools.NewStringParameter("columnName", "this is a string template parameter"),
			},
			statement: "SELECT * FROM {{.tableName}} WHERE {{.columnName}} = 'Hilton'",
			in: map[string]any{
				"tableName":  "hotels",
				"columnName": "name",
			},
			want: "SELECT * FROM hotels WHERE name = 'Hilton'",
		},
		{
			name: "standard and template parameter",
			templateParams: tools.Parameters{
				tools.NewStringParameter("tableName", "this is a string template parameter"),
				tools.NewStringParameter("hotelName", "this is a string parameter"),
			},
			statement: "SELECT * FROM {{.tableName}} WHERE name = $1",
			in: map[string]any{
				"tableName": "hotels",
				"hotelName": "name",
			},
			want: "SELECT * FROM hotels WHERE name = $1",
		},
		{
			name: "standard parameter",
			templateParams: tools.Parameters{
				tools.NewStringParameter("hotelName", "this is a string parameter"),
			},
			statement: "SELECT * FROM hotels WHERE name = $1",
			in: map[string]any{
				"hotelName": "hotels",
			},
			want: "SELECT * FROM hotels WHERE name = $1",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got, _ := tools.ResolveTemplateParams(tc.templateParams, tc.statement, tc.in)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("incorrect resolved template params: diff %v", diff)
			}
		})
	}
}

func TestFailResolveTemplateParameters(t *testing.T) {
	tcs := []struct {
		name           string
		templateParams tools.Parameters
		statement      string
		in             map[string]any
		err            string
	}{
		{
			name: "wrong param name",
			templateParams: tools.Parameters{
				tools.NewStringParameter("tableName", "this is a string template parameter"),
			},
			statement: "SELECT * FROM {{.missingParam}}",
			in:        map[string]any{},
			err:       "error getting template params missing parameter tableName",
		},
		{
			name: "incomplete param template",
			templateParams: tools.Parameters{
				tools.NewStringParameter("tableName", "this is a string template parameter"),
			},
			statement: "SELECT * FROM {{.tableName",
			in: map[string]any{
				"tableName": "hotels",
			},
			err: "error creating go template template: statement:1: unclosed action",
		},
		{
			name: "undefined function",
			templateParams: tools.Parameters{
				tools.NewStringParameter("tableName", "this is a string template parameter"),
			},
			statement: "SELECT * FROM {{json .tableName}}",
			in: map[string]any{
				"tableName": "hotels",
			},
			err: "error creating go template template: statement:1: function \"json\" not defined",
		},
		{
			name: "undefined method",
			templateParams: tools.Parameters{
				tools.NewStringParameter("tableName", "this is a string template parameter"),
			},
			statement: "SELECT * FROM {{.tableName .wrong}}",
			in: map[string]any{
				"tableName": "hotels",
			},
			err: "error executing go template template: statement:1:16: executing \"statement\" at <.tableName>: tableName is not a method but has arguments",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tools.ResolveTemplateParams(tc.templateParams, tc.statement, tc.in)
			errStr := err.Error()
			if errStr != tc.err {
				t.Fatalf("unexpected error: got %q, want %q", errStr, tc.err)
			}
		})
	}
}

func TestCheckParamRequired(t *testing.T) {
	tcs := []struct {
		name     string
		required bool
		defaultV any
		want     bool
	}{
		{
			name:     "required and no default",
			required: true,
			defaultV: nil,
			want:     true,
		},
		{
			name:     "required and default",
			required: true,
			defaultV: "foo",
			want:     false,
		},
		{
			name:     "not required and no default",
			required: false,
			defaultV: nil,
			want:     false,
		},
		{
			name:     "not required and default",
			required: false,
			defaultV: "foo",
			want:     false,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got := tools.CheckParamRequired(tc.required, tc.defaultV)
			if got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}
