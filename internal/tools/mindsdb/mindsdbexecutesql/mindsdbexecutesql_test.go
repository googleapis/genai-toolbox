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

package mindsdbexecutesql_test

import (
	"strings"
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	mindsdbexecutesql "github.com/googleapis/genai-toolbox/internal/tools/mindsdb/mindsdbexecutesql"
)

func TestParseFromYamlMindsDBExecuteSQL(t *testing.T) {
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
				example_tool:
					kind: mindsdb-execute-sql
					source: my-instance
					description: Execute SQL queries directly on MindsDB database
				`,
			want: server.ToolConfigs{
				"example_tool": mindsdbexecutesql.Config{
					Name:         "example_tool",
					Kind:         "mindsdb-execute-sql",
					Source:       "my-instance",
					Description:  "Execute SQL queries directly on MindsDB database",
					AuthRequired: []string{},
				},
			},
		},
		{
			desc: "with auth required",
			in: `
			tools:
				example_tool:
					kind: mindsdb-execute-sql
					source: my-instance
					description: Execute SQL queries directly on MindsDB database
					authRequired:
						- my-google-auth-service
						- other-auth-service
				`,
			want: server.ToolConfigs{
				"example_tool": mindsdbexecutesql.Config{
					Name:         "example_tool",
					Kind:         "mindsdb-execute-sql",
					Source:       "my-instance",
					Description:  "Execute SQL queries directly on MindsDB database",
					AuthRequired: []string{"my-google-auth-service", "other-auth-service"},
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

func TestFailParseFromYamlMindsDBExecuteSQL(t *testing.T) {
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
			desc: "missing required fields",
			in: `
			tools:
				example_tool:
					kind: mindsdb-execute-sql
					description: Execute SQL queries directly on MindsDB database
				`,
			err: "Source' failed on the 'required' tag",
		},
		{
			desc: "invalid kind",
			in: `
			tools:
				example_tool:
					kind: invalid-kind
					source: my-instance
					description: Execute SQL queries directly on MindsDB database
				`,
			err: "unknown tool kind",
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