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

package couchbase_test

import (
	"testing"

	"github.com/googleapis/genai-toolbox/internal/tools/couchbase"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

func TestParseFromYamlCouchbase(t *testing.T) {
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
					kind: couchbase-sql
					source: my-couchbase-instance
					description: some tool description
					statement: |
						select * from hotel WHERE name = $hotel;
					parameters:
						- name: hotel
						  type: string
						  description: hotel parameter description
			`,
			want: server.ToolConfigs{
				"example_tool": couchbase.Config{
					Name:         "example_tool",
					Kind:         couchbase.ToolKind,
					AuthRequired: []string{},
					Source:       "my-couchbase-instance",
					Description:  "some tool description",
					Statement:    "select * from hotel WHERE name = $hotel;\n",
					Parameters: []tools.Parameter{
						tools.NewStringParameter("hotel", "hotel parameter description"),
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

			// Create a context with a logger
			ctx, err := testutils.ContextWithNewLogger()
			if err != nil {
				t.Fatalf("unable to create context with logger: %s", err)
			}

			// Parse contents with context
			err = yaml.UnmarshalContext(ctx, testutils.FormatYaml(tc.in), &got)
			if err != nil {
				t.Fatalf("unable to unmarshal: %s", err)
			}
			if diff := cmp.Diff(tc.want, got.Tools); diff != "" {
				t.Fatalf("incorrect parse: diff %v", diff)
			}
		})
	}
}
