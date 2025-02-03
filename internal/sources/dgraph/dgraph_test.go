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

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/sources/dgraph"
	"github.com/googleapis/genai-toolbox/internal/testutils"
)

func TestParseFromYamlDgraph(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		want server.SourceConfigs
	}{
		{
			desc: "basic example",
			in: `
			sources:
				my-dgraph-instance:
					kind: dgraph
					dgraphUrl: https://localhost:8080
					apiKey: abc123
					password: pass@123
					namespace: 0
					user: user123
			`,
			want: server.SourceConfigs{
				"my-dgraph-instance": dgraph.Config{
					Name:      "my-dgraph-instance",
					Kind:      dgraph.SourceKind,
					DgraphUrl: "https://localhost:8080",
					ApiKey:    "abc123",
					Password:  "pass@123",
					Namespace: 0,
					User:      "user123",
				},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := struct {
				Sources server.SourceConfigs `yaml:"sources"`
			}{}
			// Parse contents
			err := yaml.Unmarshal(testutils.FormatYaml(tc.in), &got)
			if err != nil {
				t.Fatalf("unable to unmarshal: %s", err)
			}

			if diff := cmp.Diff(tc.want, got.Sources); diff != "" {
				t.Fatalf("incorrect parse: diff %v", diff)
			}
		})
	}

}
