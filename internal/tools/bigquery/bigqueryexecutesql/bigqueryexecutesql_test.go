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

package bigqueryexecutesql_test

import (
	"fmt"
	"testing"

	bigqueryapi "cloud.google.com/go/bigquery"
	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/tools/bigquery/bigqueryexecutesql"
)

func TestParseFromYamlBigQueryExecuteSql(t *testing.T) {
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
					kind: bigquery-execute-sql
					source: my-instance
					description: some description
			`,
			want: server.ToolConfigs{
				"example_tool": bigqueryexecutesql.Config{
					Name:         "example_tool",
					Kind:         "bigquery-execute-sql",
					Source:       "my-instance",
					Description:  "some description",
					AuthRequired: []string{},
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			var cfg server.ToolConfigs
			err := yaml.Unmarshal([]byte(tc.in), &cfg)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if diff := cmp.Diff(tc.want, cfg); diff != "" {
				t.Errorf("unexpected diff (-want +got):\n%s", diff)
			}
		})
	}
}

// mockSource implements the compatibleSource interface for testing
type mockSource struct {
	allowedDataset string
}

func (m *mockSource) ValidateDatasetAccess(dataset string) error {
	if m.allowedDataset == "" {
		return fmt.Errorf("no dataset access configured - all queries are denied")
	}
	if m.allowedDataset == "*" {
		return nil
	}
	if dataset != m.allowedDataset {
		return fmt.Errorf("access denied to dataset '%s', only '%s' is allowed", dataset, m.allowedDataset)
	}
	return nil
}

func (m *mockSource) BigQueryClient() *bigqueryapi.Client {
	return &bigqueryapi.Client{}
}

func TestValidateSQLDatasetAccess(t *testing.T) {
	tests := []struct {
		name           string
		sql            string
		allowedDataset string
		wantError      bool
		errorContains  string
	}{
		// Backtick-quoted table references
		{
			name:           "valid backtick dataset.table",
			sql:            "SELECT * FROM `BillingReport_test.unifiedTable` WHERE startTime >= '2025-01-01' LIMIT 1000",
			allowedDataset: "BillingReport_test",
			wantError:      false,
		},
		{
			name:           "valid backtick project.dataset.table",
			sql:            "SELECT * FROM `project123.BillingReport_test.unifiedTable` WHERE startTime >= '2025-01-01' LIMIT 1000",
			allowedDataset: "BillingReport_test",
			wantError:      false,
		},
		{
			name:           "invalid backtick dataset access",
			sql:            "SELECT * FROM `BillingReport_restricted.unifiedTable` WHERE startTime >= '2025-01-01' LIMIT 1000",
			allowedDataset: "BillingReport_allowed",
			wantError:      true,
			errorContains:  "access denied to dataset 'BillingReport_restricted', only 'BillingReport_allowed' is allowed",
		},
		// Non-backtick table references
		{
			name:           "valid non-backtick dataset.table",
			sql:            "SELECT * FROM BillingReport_test.unifiedTable WHERE startTime >= '2025-01-01' LIMIT 1000",
			allowedDataset: "BillingReport_test",
			wantError:      false,
		},
		{
			name:           "valid non-backtick project.dataset.table",
			sql:            "SELECT * FROM project123.BillingReport_test.unifiedTable WHERE startTime >= '2025-01-01' LIMIT 1000",
			allowedDataset: "BillingReport_test",
			wantError:      false,
		},
		// Wildcard access
		{
			name:           "wildcard allows any dataset",
			sql:            "SELECT * FROM `BillingReport_anything.unifiedTable` WHERE startTime >= '2025-01-01' LIMIT 1000",
			allowedDataset: "*",
			wantError:      false,
		},
		// Multiple datasets in query
		{
			name:           "multiple valid datasets",
			sql:            "SELECT * FROM `BillingReport_test.table1` t1 JOIN `BillingReport_test.table2` t2 ON t1.id = t2.id WHERE t1.startTime >= '2025-01-01' LIMIT 1000",
			allowedDataset: "BillingReport_test",
			wantError:      false,
		},
		{
			name:           "mixed valid and invalid datasets",
			sql:            "SELECT * FROM `BillingReport_allowed.table1` t1 JOIN `BillingReport_restricted.table2` t2 ON t1.id = t2.id WHERE t1.startTime >= '2025-01-01' LIMIT 1000",
			allowedDataset: "BillingReport_allowed",
			wantError:      true,
			errorContains:  "access denied to dataset 'BillingReport_restricted'",
		},
		// No dataset access configured
		{
			name:           "no dataset access configured",
			sql:            "SELECT * FROM `BillingReport_test.unifiedTable` WHERE startTime >= '2025-01-01' LIMIT 1000",
			allowedDataset: "",
			wantError:      true,
			errorContains:  "no dataset access configured - all queries are denied",
		},
		// Missing LIMIT clause
		{
			name:           "missing LIMIT clause",
			sql:            "SELECT * FROM `BillingReport_test.unifiedTable` WHERE startTime >= '2025-01-01'",
			allowedDataset: "BillingReport_test",
			wantError:      true,
			errorContains:  "query must include LIMIT clause",
		},
		// No table references found
		{
			name:           "no table references",
			sql:            "SELECT 1 as test LIMIT 1000",
			allowedDataset: "BillingReport_test",
			wantError:      true,
			errorContains:  "no valid table references found in SQL query",
		},
		// Case sensitivity tests
		{
			name:           "case insensitive FROM keyword",
			sql:            "select * from `BillingReport_test.unifiedTable` where startTime >= '2025-01-01' limit 1000",
			allowedDataset: "BillingReport_test",
			wantError:      false,
		},
		{
			name:           "case insensitive LIMIT keyword",
			sql:            "SELECT * FROM `BillingReport_test.unifiedTable` WHERE startTime >= '2025-01-01' limit 1000",
			allowedDataset: "BillingReport_test",
			wantError:      false,
		},
		// Complex SQL patterns
		{
			name:           "subquery with backticks",
			sql:            "SELECT * FROM (SELECT * FROM `BillingReport_test.unifiedTable` WHERE cost > 0) WHERE startTime >= '2025-01-01' LIMIT 1000",
			allowedDataset: "BillingReport_test",
			wantError:      false,
		},
		{
			name:           "CTE with backticks",
			sql:            "WITH costs AS (SELECT * FROM `BillingReport_test.unifiedTable` WHERE cost > 0) SELECT * FROM costs WHERE startTime >= '2025-01-01' LIMIT 1000",
			allowedDataset: "BillingReport_test",
			wantError:      false,
		},
		// Edge cases
		{
			name:           "table name with underscores and numbers",
			sql:            "SELECT * FROM `BillingReport_test_123.unified_table_v2` WHERE startTime >= '2025-01-01' LIMIT 1000",
			allowedDataset: "BillingReport_test_123",
			wantError:      false,
		},
		{
			name:           "mixed backtick and non-backtick references",
			sql:            "SELECT * FROM `BillingReport_test.table1` t1 JOIN BillingReport_test.table2 t2 ON t1.id = t2.id WHERE t1.startTime >= '2025-01-01' LIMIT 1000",
			allowedDataset: "BillingReport_test",
			wantError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock tool with the test configuration
			tool := bigqueryexecutesql.Tool{
				Name:         "test-tool",
				Kind:         "bigquery-execute-sql",
				AuthRequired: []string{},
				Parameters:   tools.Parameters{},
				Client:       &bigqueryapi.Client{}, // Mock client
				Source:       &mockSource{allowedDataset: tt.allowedDataset},
			}

			// Use reflection to call the private method
			// Note: In a real implementation, you might want to make this method public for testing
			// or use a different testing approach
			err := tool.ValidateSQLDatasetAccess(tt.sql)

			if tt.wantError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorContains != "" && !contains(err.Error(), tt.errorContains) {
					t.Errorf("expected error to contain '%s', but got: %s", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got: %s", err.Error())
				}
			}
		})
	}
}

// Helper function to check if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > len(substr) && containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
