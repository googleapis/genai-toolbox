// Copyright 2024 Google LLC
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

package spannerexecuteddl

import (
	"context"
	"errors"
	"os/exec"
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

func TestParseFromYamlExecuteDdl(t *testing.T) {
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
					kind: spanner-execute-ddl
					source: my-spanner-instance
					description: some description
			`,
			want: server.ToolConfigs{
				"example_tool": Config{
					Name:         "example_tool",
					Kind:         "spanner-execute-ddl",
					Source:       "my-spanner-instance",
					Description:  "some description",
					AuthRequired: []string{},
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

func TestInvoke(t *testing.T) {
	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	tcs := []struct {
		desc          string
		lookPath      func(string) (string, error)
		command       func(context.Context, string, ...string) *exec.Cmd
		params        tools.ParamValues
		expected      any
		expectedError string
	}{
		{
			desc: "gcloud not found",
			lookPath: func(s string) (string, error) {
				return "", errors.New("not found")
			},
			params:        tools.ParamValues{{Name: "ddl", Value: "ALTER TABLE Singers ADD COLUMN FirstName STRING(1024)"}},
			expectedError: "gcloud is not installed or not in your PATH. Please install the Google Cloud SDK to use this tool",
		},
		{
			desc: "successful execution",
			lookPath: func(s string) (string, error) {
				return "gcloud", nil
			},
			command: func(ctx context.Context, name string, args ...string) *exec.Cmd {
				return exec.Command("echo", "success")
			},
			params:   tools.ParamValues{{Name: "ddl", Value: "ALTER TABLE Singers ADD COLUMN FirstName STRING(1024)"}},
			expected: "success\n",
		},
		{
			desc: "failed execution",
			lookPath: func(s string) (string, error) {
				return "gcloud", nil
			},
			command: func(ctx context.Context, name string, args ...string) *exec.Cmd {
				return exec.Command("false") // "false" command always exits with a non-zero status
			},
			params:        tools.ParamValues{{Name: "ddl", Value: "ALTER TABLE Singers ADD COLUMN FirstName STRING(1024)"}},
			expectedError: "error executing gcloud command: exit status 1\n",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			execLookPath = tc.lookPath
			execCommandContext = tc.command

			tool := Tool{
				projectID:  "test-project",
				instanceID: "test-instance",
				databaseID: "test-database",
			}

			actual, err := tool.Invoke(ctx, tc.params, "test-token")
			if err != nil && tc.expectedError == "" {
				t.Fatalf("unexpected error: %s", err)
			}
			if err == nil && tc.expectedError != "" {
				t.Fatalf("expected error: %s, got none", tc.expectedError)
			}
			if err != nil && err.Error() != tc.expectedError {
				t.Fatalf("expected error: %s, got: %s", tc.expectedError, err.Error())
			}
			if err == nil && actual != tc.expected {
				t.Fatalf("expected: %v, got: %v", tc.expected, actual)
			}
		})
	}
}
