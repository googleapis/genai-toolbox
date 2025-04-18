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
	"reflect"
	"testing"

	yaml "github.com/goccy/go-yaml"
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
			name: "string with authSources",
			in: []map[string]any{
				{
					"name":        "my_string",
					"type":        "string",
					"description": "this param is a string",
					"authSources": []map[string]string{
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
			name: "int with authSources",
			in: []map[string]any{
				{
					"name":        "my_integer",
					"type":        "integer",
					"description": "this param is an int",
					"authSources": []map[string]string{
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
			name: "float with authSources",
			in: []map[string]any{
				{
					"name":        "my_float",
					"type":        "float",
					"description": "my param is a float",
					"authSources": []map[string]string{
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
			name: "bool with authSources",
			in: []map[string]any{
				{
					"name":        "my_bool",
					"type":        "boolean",
					"description": "this param is a boolean",
					"authSources": []map[string]string{
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
			name: "string array with authSources",
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
					"authSources": []map[string]string{
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
			name: "optional parameter with no value",
			params: tools.Parameters{
				tools.NewStringParameterWithOptions("optional_param", "this param is optional", true),
			},
			in:   map[string]any{},
			want: tools.ParamValues{tools.ParamValue{Name: "optional_param", Value: nil}},
		},
		{
			name: "optional parameter with value",
			params: tools.Parameters{
				tools.NewStringParameterWithOptions("optional_param", "this param is optional", true),
			},
			in: map[string]any{
				"optional_param": "hello",
			},
			want: tools.ParamValues{tools.ParamValue{Name: "optional_param", Value: "hello"}},
		},
		{
			name: "mixed required and optional parameters",
			params: tools.Parameters{
				tools.NewStringParameter("required_param", "this param is required"),
				tools.NewStringParameterWithOptions("optional_param", "this param is optional", true),
			},
			in: map[string]any{
				"required_param": "hello",
			},
			want: tools.ParamValues{tools.ParamValue{Name: "required_param", Value: "hello"}, tools.ParamValue{Name: "optional_param", Value: nil}},
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
			for i, got := range gotAll {
				want := tc.want[i]
				if got != want {
					t.Fatalf("unexpected value: got %q, want %q", got, want)
				}
				gotType, wantType := reflect.TypeOf(got), reflect.TypeOf(want)
				if gotType != wantType {
					t.Fatalf("unexpected value: got %q, want %q", got, want)
				}
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
			for i, got := range gotAll {
				want := tc.want[i]
				if got != want {
					t.Fatalf("unexpected value: got %q, want %q", got, want)
				}
				gotType, wantType := reflect.TypeOf(got), reflect.TypeOf(want)
				if gotType != wantType {
					t.Fatalf("unexpected value: got %q, want %q", got, want)
				}
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
			want: tools.ParameterManifest{Name: "foo-string", Type: "string", Description: "bar", AuthServices: []string{}},
		},
		{
			name: "int",
			in:   tools.NewIntParameter("foo-int", "bar"),
			want: tools.ParameterManifest{Name: "foo-int", Type: "integer", Description: "bar", AuthServices: []string{}},
		},
		{
			name: "float",
			in:   tools.NewFloatParameter("foo-float", "bar"),
			want: tools.ParameterManifest{Name: "foo-float", Type: "float", Description: "bar", AuthServices: []string{}},
		},
		{
			name: "boolean",
			in:   tools.NewBooleanParameter("foo-bool", "bar"),
			want: tools.ParameterManifest{Name: "foo-bool", Type: "boolean", Description: "bar", AuthServices: []string{}},
		},
		{
			name: "array",
			in:   tools.NewArrayParameter("foo-array", "bar", tools.NewStringParameter("foo-string", "bar")),
			want: tools.ParameterManifest{
				Name:         "foo-array",
				Type:         "array",
				Description:  "bar",
				AuthServices: []string{},
				Items:        &tools.ParameterManifest{Name: "foo-string", Type: "string", Description: "bar", AuthServices: []string{}},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.in.Manifest()
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("unexpected manifest: got %+v, want %+v", got, tc.want)
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
			want: tools.ParameterMcpManifest{Type: "float", Description: "bar"},
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
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.in.McpManifest()
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("unexpected manifest: got %+v, want %+v", got, tc.want)
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
			if errStr != tc.err {
				t.Fatalf("unexpected error: got %q, want %q", errStr, tc.err)
			}
		})
	}
}

func TestOptionalParameter(t *testing.T) {
	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	tests := []struct {
		name     string
		yamlStr  string
		wantErr  bool
		validate func(*testing.T, tools.Parameters)
	}{
		{
			name: "required parameter (default)",
			yamlStr: `
- name: required_param
  type: string
  description: "A required parameter"
`,
			validate: func(t *testing.T, ps tools.Parameters) {
				if len(ps) != 1 {
					t.Errorf("got %d parameters, want 1", len(ps))
				}
				p := ps[0]
				if p.IsOptional() {
					t.Error("parameter should not be optional")
				}

				// Verify that the parameter is included in the required field of McpManifest
				manifest := ps.McpManifest()
				t.Logf("MCPManifest for required parameter: %+v", manifest)
				if len(manifest.Required) != 1 || manifest.Required[0] != "required_param" {
					t.Errorf("required parameter not properly reflected in McpManifest, got %v", manifest.Required)
				}
			},
		},
		{
			name: "optional parameter",
			yamlStr: `
- name: optional_param
  type: string
  description: "An optional parameter"
  optional: true
`,
			validate: func(t *testing.T, ps tools.Parameters) {
				if len(ps) != 1 {
					t.Errorf("got %d parameters, want 1", len(ps))
				}
				p := ps[0]
				if !p.IsOptional() {
					t.Error("parameter should be optional")
				}

				// Verify that the parameter is not included in the required field of McpManifest
				manifest := ps.McpManifest()
				t.Logf("MCPManifest for optional parameter: %+v", manifest)
				if len(manifest.Required) != 0 {
					t.Errorf("optional parameter should not be in required list, got %v", manifest.Required)
				}
			},
		},
		{
			name: "mixed parameters",
			yamlStr: `
- name: required_param
  type: string
  description: "A required parameter"
- name: optional_param
  type: string
  description: "An optional parameter"
  optional: true
`,
			validate: func(t *testing.T, ps tools.Parameters) {
				if len(ps) != 2 {
					t.Errorf("got %d parameters, want 2", len(ps))
				}

				// Verify McpManifest
				manifest := ps.McpManifest()
				t.Logf("MCPManifest for mixed parameters: %+v", manifest)
				if len(manifest.Required) != 1 || manifest.Required[0] != "required_param" {
					t.Errorf("incorrect required parameters in McpManifest, got %v", manifest.Required)
				}

				// Verify each parameter
				for _, p := range ps {
					switch p.GetName() {
					case "required_param":
						if p.IsOptional() {
							t.Error("required_param should not be optional")
						}
					case "optional_param":
						if !p.IsOptional() {
							t.Error("optional_param should be optional")
						}
					default:
						t.Errorf("unexpected parameter name: %s", p.GetName())
					}
				}
			},
		},
		{
			name: "invalid optional value",
			yamlStr: `
- name: invalid_param
  type: string
  description: "A parameter with invalid optional value"
  optional: "not_a_boolean"
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ps tools.Parameters
			err := yaml.UnmarshalContext(ctx, []byte(tt.yamlStr), &ps)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.validate != nil {
				tt.validate(t, ps)
			}
		})
	}
}

func TestNewParameterWithOptional(t *testing.T) {
	tests := []struct {
		name    string
		param   tools.Parameter
		wantOpt bool
	}{
		{
			name:    "new string parameter (default required)",
			param:   tools.NewStringParameter("test", "description"),
			wantOpt: false,
		},
		{
			name:    "new optional string parameter",
			param:   tools.NewStringParameterWithOptions("test", "description", true),
			wantOpt: true,
		},
		{
			name:    "new int parameter (default required)",
			param:   tools.NewIntParameter("test", "description"),
			wantOpt: false,
		},
		{
			name:    "new optional int parameter",
			param:   tools.NewIntParameterWithOptions("test", "description", true),
			wantOpt: true,
		},
		{
			name:    "new float parameter (default required)",
			param:   tools.NewFloatParameter("test", "description"),
			wantOpt: false,
		},
		{
			name:    "new optional float parameter",
			param:   tools.NewFloatParameterWithOptions("test", "description", true),
			wantOpt: true,
		},
		{
			name:    "new boolean parameter (default required)",
			param:   tools.NewBooleanParameter("test", "description"),
			wantOpt: false,
		},
		{
			name:    "new optional boolean parameter",
			param:   tools.NewBooleanParameterWithOptions("test", "description", true),
			wantOpt: true,
		},
		{
			name:    "new array parameter (default required)",
			param:   tools.NewArrayParameter("test", "description", tools.NewStringParameter("item", "item description")),
			wantOpt: false,
		},
		{
			name:    "new optional array parameter",
			param:   tools.NewArrayParameterWithOptions("test", "description", tools.NewStringParameter("item", "item description"), true),
			wantOpt: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.param.IsOptional(); got != tt.wantOpt {
				t.Errorf("IsOptional() = %v, want %v", got, tt.wantOpt)
			}
		})
	}
}
