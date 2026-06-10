// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package alloydbcreatecluster_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/mcp-toolbox/internal/server"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/testutils"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	alloydbcreatecluster "github.com/googleapis/mcp-toolbox/internal/tools/alloydb/alloydbcreatecluster"
)

func TestParseFromYaml(t *testing.T) {
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
            kind: tool
            name: create-my-cluster
            type: alloydb-create-cluster
            source: my-alloydb-admin-source
            description: some description
            `,
			want: server.ToolConfigs{
				"create-my-cluster": alloydbcreatecluster.Config{
					ConfigBase: tools.ConfigBase{
						Name:         "create-my-cluster",
						Description:  "some description",
						AuthRequired: []string{},
					},
					Type:   "alloydb-create-cluster",
					Source: "my-alloydb-admin-source",
				},
			},
		},
		{
			desc: "with auth required",
			in: `
            kind: tool
            name: create-my-cluster-auth
            type: alloydb-create-cluster
            source: my-alloydb-admin-source
            description: some description
            authRequired: 
            - my-google-auth-service
            - other-auth-service
            `,
			want: server.ToolConfigs{
				"create-my-cluster-auth": alloydbcreatecluster.Config{
					ConfigBase: tools.ConfigBase{
						Name:         "create-my-cluster-auth",
						Description:  "some description",
						AuthRequired: []string{"my-google-auth-service", "other-auth-service"},
					},
					Type:   "alloydb-create-cluster",
					Source: "my-alloydb-admin-source",
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

func TestResolveParamsWithoutSource(t *testing.T) {
	cfg := alloydbcreatecluster.Config{
		ConfigBase: tools.ConfigBase{Name: "create-my-cluster", Description: "some description"},
		Type:       "alloydb-create-cluster",
		Source:     "my-alloydb-admin-source",
	}
	tool, err := cfg.Initialize()
	if err != nil {
		t.Fatalf("unable to initialize tool: %s", err)
	}

	// A nil sources map (e.g. offline manifest generation) degrades to the
	// static skeleton baked at Initialize rather than erroring.
	params, err := tool.GetParameters(nil)
	if err != nil {
		t.Fatalf("GetParameters(nil) returned error: %s", err)
	}
	if len(params) == 0 {
		t.Fatal("GetParameters(nil) returned no parameters")
	}
	if _, err := tool.Manifest(nil); err != nil {
		t.Fatalf("Manifest(nil) returned error: %s", err)
	}

	// A non-nil map missing the configured source still fails fast.
	if _, err := tool.GetParameters(map[string]sources.Source{}); err == nil {
		t.Fatal("GetParameters with a missing source: expected error, got nil")
	}
}
