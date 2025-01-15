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
	"github.com/googleapis/genai-toolbox/internal/tools"
)

func TestParametersMarshal(t *testing.T) {
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
						"type": "string",
					},
				},
			},
			want: tools.Parameters{
				tools.NewArrayParameter("my_array", "this param is an array of strings", tools.NewStringParameter("", "")),
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
						"type": "float",
					},
				},
			},
			want: tools.Parameters{
				tools.NewArrayParameter("my_array", "this param is an array of floats", tools.NewFloatParameter("", "")),
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
			err = yaml.Unmarshal(data, &got)
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
	authSources := []tools.ParamAuthSource{{Name: "my-google-auth-service", Field: "user_id"}, {Name: "other-auth-service", Field: "user_id"}}
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
				tools.NewStringParameterWithAuth("my_string", "this param is a string", authSources),
			},
		},
		{
			name: "int",
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
				tools.NewIntParameterWithAuth("my_integer", "this param is an int", authSources),
			},
		},
		{
			name: "float",
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
				tools.NewFloatParameterWithAuth("my_float", "my param is a float", authSources),
			},
		},
		{
			name: "bool",
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
				tools.NewBooleanParameterWithAuth("my_bool", "this param is a boolean", authSources),
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
						"type": "string",
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
				tools.NewArrayParameterWithAuth("my_array", "this param is an array of strings", tools.NewStringParameter("", ""), authSources),
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
						"type": "float",
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
				tools.NewArrayParameterWithAuth("my_array", "this param is an array of floats", tools.NewFloatParameter("", ""), authSources),
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
			err = yaml.Unmarshal(data, &got)
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
	authSources := []tools.ParamAuthSource{
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
				tools.NewStringParameterWithAuth("my_string", "this param is a string", authSources),
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
				tools.NewStringParameterWithAuth("my_string", "this param is a string", authSources),
			},
			in: map[string]any{
				"my_string": 4,
			},
			claimsMap: map[string]map[string]any{},
		},
		{
			name: "int",
			params: tools.Parameters{
				tools.NewIntParameterWithAuth("my_int", "this param is an int", authSources),
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
				tools.NewIntParameterWithAuth("my_int", "this param is an int", authSources),
			},
			in: map[string]any{
				"my_int": 14.5,
			},
			claimsMap: map[string]map[string]any{},
		},
		{
			name: "float",
			params: tools.Parameters{
				tools.NewFloatParameterWithAuth("my_float", "this param is a float", authSources),
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
				tools.NewFloatParameterWithAuth("my_float", "this param is a float", authSources),
			},
			in: map[string]any{
				"my_float": true,
			},
			claimsMap: map[string]map[string]any{},
		},
		{
			name: "bool",
			params: tools.Parameters{
				tools.NewBooleanParameterWithAuth("my_bool", "this param is a bool", authSources),
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
				tools.NewBooleanParameterWithAuth("my_bool", "this param is a bool", authSources),
			},
			in: map[string]any{
				"my_bool": 1.5,
			},
			claimsMap: map[string]map[string]any{},
		},
		{
			name: "username",
			params: tools.Parameters{
				tools.NewStringParameterWithAuth("username", "username string", authSources),
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
				tools.NewStringParameterWithAuth("username", "username string", authSources),
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
		name           string
		in             tools.ParamValues
		wantSlice      []any
		wantMap        map[string]interface{}
		wantMapOrdered map[string]interface{}
	}{
		{
			name:           "string",
			in:             tools.ParamValues{tools.ParamValue{Name: "my_bool", Value: true}, tools.ParamValue{Name: "my_string", Value: "hello world"}},
			wantSlice:      []any{true, "hello world"},
			wantMap:        map[string]interface{}{"my_bool": true, "my_string": "hello world"},
			wantMapOrdered: map[string]interface{}{"p1": true, "p2": "hello world"},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			gotSlice := tc.in.AsSlice()
			gotMap := tc.in.AsMap()
			gotMapOrdered := tc.in.AsMapByOrderedKeys()

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
		})
	}
}
