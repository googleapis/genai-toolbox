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

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	alloydbcreatecluster "github.com/googleapis/genai-toolbox/internal/tools/alloydb/alloydbcreatecluster"
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
			tools:
				create-my-cluster:
					kind: alloydb-create-cluster
					description: some description
			`,
			want: server.ToolConfigs{
				"create-my-cluster": alloydbcreatecluster.Config{
					Name:         "create-my-cluster",
					Kind:         "alloydb-create-cluster",
					Description:  "some description",
					AuthRequired: []string{},
				},
			},
		},
		{
			desc: "with auth required",
			in: `
			tools:
				create-my-cluster-auth:
					kind: alloydb-create-cluster
					description: some description
					authRequired: 
						- my-google-auth-service
						- other-auth-service
			`,
			want: server.ToolConfigs{
				"create-my-cluster-auth": alloydbcreatecluster.Config{
					Name:         "create-my-cluster-auth",
					Kind:         "alloydb-create-cluster",
					Description:  "some description",
					AuthRequired: []string{"my-google-auth-service", "other-auth-service"},
				},
			},
		},
		{
			desc: "with base url",
			in: `
			tools:
				create-my-cluster-baseurl:
					kind: alloydb-create-cluster
					description: some description
					baseURL: "https://example.com"
			`,
			want: server.ToolConfigs{
				"create-my-cluster-baseurl": alloydbcreatecluster.Config{
					Name:         "create-my-cluster-baseurl",
					Kind:         "alloydb-create-cluster",
					Description:  "some description",
					BaseURL:      "https://example.com",
					AuthRequired: []string{},
				},
			},
		},
		{
			desc: "with auth and base url",
			in: `
			tools:
				create-my-cluster-all:
					kind: alloydb-create-cluster
					description: some description
					authRequired: 
						- my-google-auth-service
						- other-auth-service
					baseURL: "https://example.com"
			`,
			want: server.ToolConfigs{
				"create-my-cluster-all": alloydbcreatecluster.Config{
					Name:         "create-my-cluster-all",
					Kind:         "alloydb-create-cluster",
					Description:  "some description",
					AuthRequired: []string{"my-google-auth-service", "other-auth-service"},
					BaseURL:      "https://example.com",
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
