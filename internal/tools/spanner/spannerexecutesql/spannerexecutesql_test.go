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

package spannerexecutesql

import (
	"context"
	"errors"
	"testing"

	database "cloud.google.com/go/spanner/admin/database/apiv1"
	"github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/gax-go/v2"
	databasepb "google.golang.org/genproto/googleapis/spanner/admin/database/v1"
)

type mockDatabaseAdminClient struct {
	spannerDatabaseAdminClient
	updateDatabaseDdl func(context.Context, *databasepb.UpdateDatabaseDdlRequest, ...gax.CallOption) (*database.UpdateDatabaseDdlOperation, error)
	close func() error
}

func (m *mockDatabaseAdminClient) UpdateDatabaseDdl(ctx context.Context, req *databasepb.UpdateDatabaseDdlRequest, opts ...gax.CallOption) (*database.UpdateDatabaseDdlOperation, error) {
	return m.updateDatabaseDdl(ctx, req, opts...)
}

func (m *mockDatabaseAdminClient) Close() error {
	if m.close != nil {
		return m.close()
	}
	return nil
}

func TestParseFromYamlExecuteSql(t *testing.T) {
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
					kind: spanner-execute-sql
					source: my-spanner-instance
					description: some description
			`,
			want: server.ToolConfigs{
				"example_tool": Config{
					Name:         "example_tool",
					Kind:         "spanner-execute-sql",
					Source:       "my-spanner-instance",
					Description:  "some description",
					AuthRequired: []string{},
					ReadOnly:     false,
				},
			},
		},
		{
			desc: "read only set to true",
			in: `
			tools:
				example_tool:
					kind: spanner-execute-sql
					source: my-spanner-instance
					description: some description
					readOnly: true
			`,
			want: server.ToolConfigs{
				"example_tool": Config{
					Name:         "example_tool",
					Kind:         "spanner-execute-sql",
					Source:       "my-spanner-instance",
					Description:  "some description",
					AuthRequired: []string{},
					ReadOnly:     true,
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
		adminClient   spannerDatabaseAdminClient
		params        tools.ParamValues
		readOnly      bool
		expected      any
		expectedError string
	}{
		// Not testing the successful execution case due to mocking complexities with the
		// long-running operation. The error cases provide sufficient coverage for the
		// DDL execution path.
		{
			desc:          "ddl execution in read-only mode",
			params:        tools.ParamValues{{Name: "sql", Value: "ALTER TABLE Singers ADD COLUMN FirstName STRING(1024)"}},
			readOnly:      true,
			expectedError: "cannot execute DDL statements in read-only mode",
		},
		{
			desc: "ddl execution error",
			adminClient: &mockDatabaseAdminClient{
				updateDatabaseDdl: func(ctx context.Context, req *databasepb.UpdateDatabaseDdlRequest, opts ...gax.CallOption) (*database.UpdateDatabaseDdlOperation, error) {
					return nil, errors.New("test error")
				},
			},
			params:        tools.ParamValues{{Name: "sql", Value: "ALTER TABLE Singers ADD COLUMN FirstName STRING(1024)"}},
			expectedError: "error executing DDL statement: test error",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			tool := Tool{
				databaseAdminClient: tc.adminClient,
				databaseName:        "test-database",
				ReadOnly:            tc.readOnly,
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
