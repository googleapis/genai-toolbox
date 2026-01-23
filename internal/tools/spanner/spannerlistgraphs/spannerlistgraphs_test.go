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

package spannerlistgraphs_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools/spanner/spannerlistgraphs"
)

func TestParseFromYamlListGraphs(t *testing.T) {
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
            kind: tools
            name: example_tool
            type: spanner-list-graphs
            source: my-spanner-instance
            description: Lists graphs in the database
			`,
			want: server.ToolConfigs{
				"example_tool": spannerlistgraphs.Config{
					Name:         "example_tool",
					Type:         "spanner-list-graphs",
					Source:       "my-spanner-instance",
					Description:  "Lists graphs in the database",
					AuthRequired: []string{},
				},
			},
		},
		{
			desc: "with auth required",
			in: `
            kind: tools
            name: example_tool
            type: spanner-list-graphs
            source: my-spanner-instance
            description: Lists graphs in the database
            authRequired:
                - auth1
                - auth2
			`,
			want: server.ToolConfigs{
				"example_tool": spannerlistgraphs.Config{
					Name:         "example_tool",
					Type:         "spanner-list-graphs",
					Source:       "my-spanner-instance",
					Description:  "Lists graphs in the database",
					AuthRequired: []string{"auth1", "auth2"},
				},
			},
		},
		{
			desc: "minimal config",
			in: `
            kind: tools
            name: example_tool
            type: spanner-list-graphs
            source: my-spanner-instance
			`,
			want: server.ToolConfigs{
				"example_tool": spannerlistgraphs.Config{
					Name:         "example_tool",
					Type:         "spanner-list-graphs",
					Source:       "my-spanner-instance",
					Description:  "",
					AuthRequired: []string{},
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			// Parse contents
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
