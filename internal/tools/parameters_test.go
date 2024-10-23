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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"gopkg.in/yaml.v3"
)

func TestParameters(t *testing.T) {
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
