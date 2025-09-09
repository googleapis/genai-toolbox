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

package http_test

import (
	"strings"
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools"
	http "github.com/googleapis/genai-toolbox/internal/tools/http"
)

func TestParseFromYamlHTTP(t *testing.T) {
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
			tools:
				example_tool:
					type: http
					source: my-instance
					method: GET
					description: some description
					path: search
				`,
			want: server.ToolConfigs{
				"example_tool": http.Config{
					Name:         "example_tool",
					Type:         "http",
					Source:       "my-instance",
					Method:       "GET",
					Path:         "search",
					Description:  "some description",
					AuthRequired: []string{},
				},
			},
		},
		{
			desc: "advanced example",
			in: `
			tools:
				example_tool:
					type: http
					source: my-instance
					method: GET
					path: "{{.pathParam}}?name=alice&pet=cat"
					description: some description
					authRequired:
						- my-google-auth-service
						- other-auth-service
					queryParams:
						- name: country
						  type: string
						  description: some description
						  authServices:
							- name: my-google-auth-service
							  field: user_id
							- name: other-auth-service
							  field: user_id
					pathParams:
					    - name: pathParam
					      type: string
					      description: path param
					requestBody: |
							{
								"age": {{.age}},
								"city": "{{.city}}",
								"food": {{.food}}
							}
					bodyParams:
						- name: age
						  type: integer
						  description: age num
						- name: city
						  type: string
						  description: city string
					headers:
						Authorization: API_KEY
						Content-Type: application/json
					headerParams:
						- name: Language
						  type: string
						  description: language string
			`,
			want: server.ToolConfigs{
				"example_tool": http.Config{
					Name:         "example_tool",
					Type:         "http",
					Source:       "my-instance",
					Method:       "GET",
					Path:         "{{.pathParam}}?name=alice&pet=cat",
					Description:  "some description",
					AuthRequired: []string{"my-google-auth-service", "other-auth-service"},
					QueryParams: []tools.Parameter{
						tools.NewStringParameterWithAuth("country", "some description",
							[]tools.ParamAuthService{{Name: "my-google-auth-service", Field: "user_id"},
								{Name: "other-auth-service", Field: "user_id"}}),
					},
					PathParams: tools.Parameters{
						&tools.StringParameter{
							CommonParameter: tools.CommonParameter{Name: "pathParam", Type: "string", Desc: "path param"},
						},
					},
					RequestBody: `{
  "age": {{.age}},
  "city": "{{.city}}",
  "food": {{.food}}
}
`,
					BodyParams:   []tools.Parameter{tools.NewIntParameter("age", "age num"), tools.NewStringParameter("city", "city string")},
					Headers:      map[string]string{"Authorization": "API_KEY", "Content-Type": "application/json"},
					HeaderParams: []tools.Parameter{tools.NewStringParameter("Language", "language string")},
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := struct {
				Tools server.ToolConfigs `yaml:"tools"`
			}{}
			// Parse contents
			err := yaml.UnmarshalContext(ctx, testutils.FormatYaml(tc.in), &got)
			if err != nil {
				t.Fatalf("unable to unmarshal: %s", err)
			}
			if diff := cmp.Diff(tc.want, got.Tools); diff != "" {
				t.Fatalf("incorrect parse: diff %v", diff)
			}
		})
	}

}

func TestFailParseFromYamlHTTP(t *testing.T) {
	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	tcs := []struct {
		desc string
		in   string
		err  string
	}{
		{
			desc: "Invalid method",
			in: `
			tools:
				example_tool:
					type: http
					source: my-instance
					method: GOT
					path: "search?name=alice&pet=cat"
					description: some description
					authRequired:
						- my-google-auth-service
						- other-auth-service
					queryParams:
						- name: country
						  type: string
						  description: some description
						  authServices:
							- name: my-google-auth-service
							  field: user_id
							- name: other-auth-service
							  field: user_id
					requestBody: |
							{
								"age": {{.age}},
								"city": "{{.city}}"
							}
					bodyParams:
						- name: age
						  type: integer
						  description: age num
						- name: city
						  type: string
						  description: city string
					headers:
						Authorization: API_KEY
						Content-Type: application/json
					headerParams:
						- name: Language
						  type: string
						  description: language string
			`,
			err: `GOT is not a valid http method`,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := struct {
				Tools server.ToolConfigs `yaml:"tools"`
			}{}
			// Parse contents
			err := yaml.UnmarshalContext(ctx, testutils.FormatYaml(tc.in), &got)
			if err == nil {
				t.Fatalf("expect parsing to fail")
			}
			errStr := err.Error()
			if !strings.Contains(errStr, tc.err) {
				t.Fatalf("unexpected error string: got %q, want substring %q", errStr, tc.err)
			}
		})
	}

}
