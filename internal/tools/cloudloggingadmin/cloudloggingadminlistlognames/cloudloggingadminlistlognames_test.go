// Copyright 2026 Google LLC
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
package cloudloggingadminlistlognames_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools/cloudloggingadmin/cloudloggingadminlistlognames"
)

func TestParseFromYaml(t *testing.T) {
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
			type: cloud-logging-admin-list-log-names
			source: my-logging-admin-source
			description: list log names
			authRequired:
				- my-google-auth-service
			`,
			want: server.ToolConfigs{
				"example_tool": cloudloggingadminlistlognames.Config{
					Name:         "example_tool",
					Type:         "cloud-logging-admin-list-log-names",
					Source:       "my-logging-admin-source",
					Description:  "list log names",
					AuthRequired: []string{"my-google-auth-service"},
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			_, _, _, got, _, _, err := server.UnmarshalResourceConfig(context.Background(), testutils.FormatYaml(tc.in))
			if err != nil {
				t.Fatalf("unable to unmarshal: %s", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
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
			type: cloud-logging-admin-list-log-names
			description: some description
			`,
			err: `Key: 'Config.Source' Error:Field validation for 'Source' failed on the 'required' tag`,
		},
		{
			desc: "missing description",
			in: `
			kind: tools
			name: example_tool
			type: cloud-logging-admin-list-log-names
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
