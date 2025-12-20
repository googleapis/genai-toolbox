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
package cloudloggingadminquerylogs_test

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
	"github.com/googleapis/genai-toolbox/internal/tools/cloudloggingadmin/cloudloggingadminquerylogs"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
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
		cfg     cloudloggingadminquerylogs.Config
		want    *tools.Manifest
		wantErr string
	}{
		{
			desc: "Success case with specified authRequired",
			cfg: cloudloggingadminquerylogs.Config{
				Name:         "test-tool",
				Kind:         "cloud-logging-admin-query-logs",
				Source:       "my-logging-admin-source",
				Description:  "query logs",
				AuthRequired: []string{"my-google-auth-service"},
			},
			want: &tools.Manifest{
				Description: "query logs",
				Parameters: []parameters.ParameterManifest{
					{Name: "filter", Description: "Cloud Logging filter query. Common fields: resource.type, resource.labels.*, logName, severity, textPayload, jsonPayload.*, protoPayload.*, labels.*, httpRequest.*. Operators: =, !=, <, <=, >, >=, :, =~, AND, OR, NOT.", Type: "string", Required: false, AuthServices: []string{}},
					{Name: "newestFirst", Description: "Set to true for newest logs first. Defaults to oldest first.", Type: "boolean", Required: false, AuthServices: []string{}},
					{Name: "startTime", Description: "Start time in RFC3339 format (e.g., 2025-12-09T00:00:00Z). Defaults to 30 days ago.", Type: "string", Required: false, AuthServices: []string{}},
					{Name: "endTime", Description: "End time in RFC3339 format (e.g., 2025-12-09T23:59:59Z). Defaults to now.", Type: "string", Required: false, AuthServices: []string{}},
					{Name: "verbose", Description: "Include additional fields (insertId, trace, spanId, httpRequest, labels, operation, sourceLocation). Defaults to false.", Type: "boolean", Required: false, AuthServices: []string{}},
					{Name: "limit", Description: "Maximum number of log entries to return (default: 200).", Type: "integer", Required: false, AuthServices: []string{}},
				},
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
					kind: cloud-logging-admin-query-logs
					source: my-logging-admin-source
					description: query logs
					authRequired:
						- my-google-auth-service
			`,
			want: server.ToolConfigs{
				"example_tool": cloudloggingadminquerylogs.Config{
					Name:         "example_tool",
					Kind:         "cloud-logging-admin-query-logs",
					Source:       "my-logging-admin-source",
					Description:  "query logs",
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
					kind: cloud-logging-admin-query-logs
					description: some description
			`,
			err: `Key: 'Config.Source' Error:Field validation for 'Source' failed on the 'required' tag`,
		},
		{
			desc: "missing description",
			in: `
			tools:
				example_tool:
					kind: cloud-logging-admin-query-logs
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
