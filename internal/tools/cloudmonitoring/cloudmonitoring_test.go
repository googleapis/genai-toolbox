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

package cloudmonitoring_test

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/sources"
	cloudmonitoringsrc "github.com/googleapis/genai-toolbox/internal/sources/cloudmonitoring"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/tools/cloudmonitoring"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
)

// mockIncompatibleSource is a source of a different type to test error paths.
type mockIncompatibleSource struct{ sources.Source }

func TestInitialize(t *testing.T) {
	t.Parallel()
	testSource := &cloudmonitoringsrc.Source{Config: cloudmonitoringsrc.Config{Type: "cloud-monitoring"}}
	srcs := map[string]sources.Source{
		"my-monitoring-source": testSource,
		"incompatible-source":  &mockIncompatibleSource{},
	}

	wantParams := parameters.Parameters{
		parameters.NewStringParameterWithRequired("projectId", "The Id of the Google Cloud project.", true),
		parameters.NewStringParameterWithRequired("query", "The promql query to execute.", true),
	}

	testCases := []struct {
		desc    string
		cfg     cloudmonitoring.Config
		want    *tools.Manifest
		wantErr string
	}{
		{
			desc: "Success case with nil authRequired",
			cfg: cloudmonitoring.Config{
				Name:         "test-tool",
				Type:         "cloud-monitoring-query-prometheus",
				Source:       "my-monitoring-source",
				Description:  "A test description.",
				AuthRequired: nil,
			},
			want: &tools.Manifest{
				Description:  "A test description.",
				Parameters:   wantParams.Manifest(),
				AuthRequired: nil,
			},
		},
		{
			desc: "Success case with specified authRequired",
			cfg: cloudmonitoring.Config{
				Name:         "test-tool-with-auth",
				Type:         "cloud-monitoring-query-prometheus",
				Source:       "my-monitoring-source",
				Description:  "Another test description.",
				AuthRequired: []string{"google-auth-service"},
			},
			want: &tools.Manifest{
				Description:  "Another test description.",
				Parameters:   wantParams.Manifest(),
				AuthRequired: []string{"google-auth-service"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			tool, err := tc.cfg.Initialize(srcs)

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

func TestParseFromYamlCloudMonitoring(t *testing.T) {
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
			type: cloud-monitoring-query-prometheus
			source: my-instance
			description: some description
			`,
			want: server.ToolConfigs{
				"example_tool": cloudmonitoring.Config{
					Name:         "example_tool",
					Type:         "cloud-monitoring-query-prometheus",
					Source:       "my-instance",
					Description:  "some description",
					AuthRequired: []string{},
				},
			},
		},
		{
			desc: "advanced example",
			in: `
			kind: tools
			name: example_tool
			type: cloud-monitoring-query-prometheus
			source: my-instance
			description: some description
			authRequired:
				- my-google-auth-service
				- other-auth-service
			`,
			want: server.ToolConfigs{
				"example_tool": cloudmonitoring.Config{
					Name:         "example_tool",
					Type:         "cloud-monitoring-query-prometheus",
					Source:       "my-instance",
					Description:  "some description",
					AuthRequired: []string{"my-google-auth-service", "other-auth-service"},
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
			if diff := cmp.Diff(tc.want, got, cmp.AllowUnexported(cloudmonitoring.Config{})); diff != "" {
				t.Fatalf("incorrect parse: diff %v", diff)
			}
		})
	}
}

func TestFailParseFromYamlCloudMonitoring(t *testing.T) {
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
			desc: "Invalid type",
			in: `
			kind: tools
			name: example_tool
			type: invalid-type
			source: my-instance
			description: some description
			`,
			err: `unknown tool type: "invalid-type"`,
		},
		{
			desc: "missing source",
			in: `
			kind: tools
			name: example_tool
			type: cloud-monitoring-query-prometheus
			description: some description
			`,
			err: `Key: 'Config.Source' Error:Field validation for 'Source' failed on the 'required' tag`,
		},
		{
			desc: "missing description",
			in: `
			kind: tools
			name: example_tool
			type: cloud-monitoring-query-prometheus
			source: my-instance
			`,
			err: `Key: 'Config.Description' Error:Field validation for 'Description' failed on the 'required' tag`,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			_, _, _, _, _, _, err := server.UnmarshalResourceConfig(ctx, testutils.FormatYaml(tc.in))
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
