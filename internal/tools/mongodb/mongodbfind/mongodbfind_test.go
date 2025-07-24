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

package mongodbfind_test

import (
	"strings"
	"testing"

	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/tools/mongodb/mongodbfind"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
)

func TestParseFromYamlMongoQuery(t *testing.T) {
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
					kind: mongodb-find
					source: my-instance
					description: some description
					database: test_db
					collection: test_coll
					filterPayload: |
					    { name: {{json .name}} }
					filterParams:
                        - name: name 
                          type: string
                          description: small description
					projectPayload: |
					  { name: 1, age: 1 }
					projectParams: []
					sortPayload: |
					  { timestamp: -1 }
					sortParams: []
			`,
			want: server.ToolConfigs{
				"example_tool": mongodbfind.Config{
					Name:          "example_tool",
					Kind:          "mongodb-find",
					Source:        "my-instance",
					AuthRequired:  []string{},
					Database:      "test_db",
					Collection:    "test_coll",
					Description:   "some description",
					FilterPayload: "{ name: {{json .name}} }\n",
					FilterParams: tools.Parameters{
						&tools.StringParameter{
							CommonParameter: tools.CommonParameter{
								Name: "name",
								Type: "string",
								Desc: "small description",
							},
						},
					},
					ProjectPayload: "{ name: 1, age: 1 }\n",
					ProjectParams:  tools.Parameters{},
					SortPayload:    "{ timestamp: -1 }\n",
					SortParams:     tools.Parameters{},
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

func TestFailParseFromYamlMongoQuery(t *testing.T) {
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
					kind: mongodb-find
					source: my-instance
					description: some description
					collection: test_coll
					filterPayload: |
					  { name : {{json .name}} }
			`,
			err: `unable to parse tool "example_tool" as kind "mongodb-find"`,
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
