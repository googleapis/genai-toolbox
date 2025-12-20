// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package cloudloggingadminlistresourcetypes_test

import (
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/sources"
	cla "github.com/googleapis/genai-toolbox/internal/sources/cloudloggingadmin"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools"
	cloudloggingadminlistresourcetypes "github.com/googleapis/genai-toolbox/internal/tools/cloudloggingadmin/cloudloggingadminlistresourcetypes"
)

type mockIncompatibleSource struct{ sources.Source }

func TestInitialize(t *testing.T) {
	t.Parallel()
	testSource := &cla.Source{Config: cla.Config{Kind: "cloud-logging-admin"}}
	sourcesMap := map[string]sources.Source{
		"my-logging-admin-source": testSource,
		"incompatible-source":     &mockIncompatibleSource{},
	}

	testCases := []struct {
		desc    string
		cfg     cloudloggingadminlistresourcetypes.Config
		want    *tools.Manifest
		wantErr string
	}{
		{
			desc: "Success case with specified authRequired",
			cfg: cloudloggingadminlistresourcetypes.Config{
				Name:         "test-tool",
				Kind:         "cloud-logging-admin-list-resource-types",
				Source:       "my-logging-admin-source",
				Description:  "list resource types",
				AuthRequired: []string{"my-google-auth-service"},
			},
			want: &tools.Manifest{
				Description:  "list resource types",
				Parameters:   nil,
				AuthRequired: []string{"my-google-auth-service"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			tool, err := tc.cfg.Initialize(sourcesMap)

			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("Initialize() succeeded, want error containing %q", tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("Initialize() error = %q, want error containing %q", err, tc.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("Initialize() failed: %v", err)
			}

			got := tool.Manifest()
			if diff := cmp.Diff(tc.want, &got); diff != "" {
				t.Errorf("Initialize() manifest mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParseFromYaml(t *testing.T) {
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
					kind: cloud-logging-admin-list-resource-types
					source: my-logging-admin-source
					description: list resource types
					authRequired:
						- my-google-auth-service
			`,
			want: server.ToolConfigs{
				"example_tool": cloudloggingadminlistresourcetypes.Config{
					Name:         "example_tool",
					Kind:         "cloud-logging-admin-list-resource-types",
					Source:       "my-logging-admin-source",
					Description:  "list resource types",
					AuthRequired: []string{"my-google-auth-service"},
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := struct {
				Tools server.ToolConfigs `yaml:"tools"`
			}{}
			err := yaml.Unmarshal(testutils.FormatYaml(tc.in), &got)
			if err != nil {
				t.Fatalf("unable to unmarshal: %s", err)
			}
			if diff := cmp.Diff(tc.want, got.Tools); diff != "" {
				t.Fatalf("incorrect parse: diff %v", diff)
			}
		})
	}
}

func TestFailParseFromYaml(t *testing.T) {
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
			desc: "Invalid kind",
			in: `
			tools:
				example_tool:
					kind: invalid-kind
					source: my-instance
					description: some description
			`,
			err: `unknown tool kind: "invalid-kind"`,
		},
		{
			desc: "missing source",
			in: `
			tools:
				example_tool:
					kind: cloud-logging-admin-list-resource-types
					description: some description
			`,
			err: `Key: 'Config.Source' Error:Field validation for 'Source' failed on the 'required' tag`,
		},
		{
			desc: "missing description",
			in: `
			tools:
				example_tool:
					kind: cloud-logging-admin-list-resource-types
					source: my-instance
			`,
			err: `Key: 'Config.Description' Error:Field validation for 'Description' failed on the 'required' tag`,
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
