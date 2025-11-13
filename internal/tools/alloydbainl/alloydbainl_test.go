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

package alloydbainl_test

import (
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools/alloydbainl"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
)

func TestParseFromYamlAlloyDBNLA(t *testing.T) {
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
					kind: alloydb-ai-nl
					source: my-alloydb-instance
					description: AlloyDB natural language query tool
					nlConfig: 'my_nl_config'
					authRequired:
						- my-google-auth-service
					nlConfigParameters:
						- name: user_id
						  type: string
						  description: user_id to use
						  authServices:
							- name: my-google-auth-service
							  field: sub
			`,
			want: server.ToolConfigs{
				"example_tool": alloydbainl.Config{
					Name:         "example_tool",
					Kind:         "alloydb-ai-nl",
					Source:       "my-alloydb-instance",
					Description:  "AlloyDB natural language query tool",
					NLConfig:     "my_nl_config",
					AuthRequired: []string{"my-google-auth-service"},
					NLConfigParameters: []parameters.Parameter{
						parameters.NewStringParameterWithAuth("user_id", "user_id to use",
							[]parameters.ParamAuthService{{Name: "my-google-auth-service", Field: "sub"}}),
					},
				},
			},
		},
		{
			desc: "with multiple parameters",
			in: `
			tools:
				complex_tool:
					kind: alloydb-ai-nl
					source: my-alloydb-instance
					description: AlloyDB natural language query tool with multiple parameters
					nlConfig: 'complex_nl_config'
					authRequired:
						- my-google-auth-service
						- other-auth-service
					nlConfigParameters:
						- name: user_id
						  type: string
						  description: user_id to use
						  authServices:
							- name: my-google-auth-service
							  field: sub
						- name: user_email
						  type: string
						  description: user_email to use
						  authServices:
							- name: my-google-auth-service
							  field: user_email
			`,
			want: server.ToolConfigs{
				"complex_tool": alloydbainl.Config{
					Name:         "complex_tool",
					Kind:         "alloydb-ai-nl",
					Source:       "my-alloydb-instance",
					Description:  "AlloyDB natural language query tool with multiple parameters",
					NLConfig:     "complex_nl_config",
					AuthRequired: []string{"my-google-auth-service", "other-auth-service"},
					NLConfigParameters: []parameters.Parameter{
						parameters.NewStringParameterWithAuth("user_id", "user_id to use",
							[]parameters.ParamAuthService{{Name: "my-google-auth-service", Field: "sub"}}),
						parameters.NewStringParameterWithAuth("user_email", "user_email to use",
							[]parameters.ParamAuthService{{Name: "my-google-auth-service", Field: "user_email"}}),
					},
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
