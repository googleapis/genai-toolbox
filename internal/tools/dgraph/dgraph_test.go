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

package dgraph_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools/dgraph"
)

func TestParseFromYamlDgraph(t *testing.T) {
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
			desc: "basic query example",
			in: `
			kind: tools
			name: example_tool
			type: dgraph-dql
			source: my-dgraph-instance
			description: some tool description
			isQuery: true
			timeout: 20s
			statement: |
				    query {q(func: eq(email, "example@email.com")) {email}}
			`,
			want: server.ToolConfigs{
				"example_tool": dgraph.Config{
					Name:         "example_tool",
					Type:         "dgraph-dql",
					Source:       "my-dgraph-instance",
					AuthRequired: []string{},
					Description:  "some tool description",
					IsQuery:      true,
					Timeout:      "20s",
					Statement:    "query {q(func: eq(email, \"example@email.com\")) {email}}\n",
				},
			},
		},
		{
			desc: "basic mutation example",
			in: `
			kind: tools
			name: example_tool
			type: dgraph-dql
			source: my-dgraph-instance
			description: some tool description
			statement: |
			  mutation {set { _:a <name> "a@email.com" . _:b <email> "b@email.com" .}}
			`,
			want: server.ToolConfigs{
				"example_tool": dgraph.Config{
					Name:         "example_tool",
					Type:         "dgraph-dql",
					Source:       "my-dgraph-instance",
					Description:  "some tool description",
					AuthRequired: []string{},
					Statement:    "mutation {set { _:a <name> \"a@email.com\" . _:b <email> \"b@email.com\" .}}\n",
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			_, _, _, got, _, _, err := server.UnmarshalResourceConfig(ctx, testutils.FormatYaml(tc.in))
			if err != nil {
				t.Fatalf("unable to unmarshal: %s", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("incorrect parse: diff %v", diff)
			}
		})
	}

}
