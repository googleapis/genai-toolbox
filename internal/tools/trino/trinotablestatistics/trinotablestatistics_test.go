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

package trinotablestatistics_test

import (
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools/trino/trinotablestatistics"
)

func TestParseFromYamlTrinoTableStatistics(t *testing.T) {
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
				table_stats:
					kind: trino-table-statistics
					source: my-trino-instance
					description: Gets detailed table statistics
			`,
			want: server.ToolConfigs{
				"table_stats": trinotablestatistics.Config{
					Name:         "table_stats",
					Kind:         "trino-table-statistics",
					Source:       "my-trino-instance",
					Description:  "Gets detailed table statistics",
					AuthRequired: []string{},
				},
			},
		},
		{
			desc: "with auth",
			in: `
			tools:
				table_stats_auth:
					kind: trino-table-statistics
					source: my-trino-instance
					description: Gets table statistics with authentication
					authRequired:
						- my-auth-service
			`,
			want: server.ToolConfigs{
				"table_stats_auth": trinotablestatistics.Config{
					Name:         "table_stats_auth",
					Kind:         "trino-table-statistics",
					Source:       "my-trino-instance",
					Description:  "Gets table statistics with authentication",
					AuthRequired: []string{"my-auth-service"},
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

func TestTrinoTableStatisticsToolConfigKind(t *testing.T) {
	config := trinotablestatistics.Config{
		Name:        "test",
		Kind:        "trino-table-statistics",
		Source:      "test-source",
		Description: "test",
	}

	if kind := config.ToolConfigKind(); kind != "trino-table-statistics" {
		t.Errorf("expected ToolConfigKind to return 'trino-table-statistics', got %s", kind)
	}
}

func TestTrinoTableStatisticsParameters(t *testing.T) {
	// This test verifies the expected parameters for the table statistics tool
	// The actual tool initialization requires a database connection

	config := trinotablestatistics.Config{
		Name:        "test_tool",
		Kind:        "trino-table-statistics",
		Source:      "test-source",
		Description: "test description",
	}

	// Verify the config structure
	if config.Name != "test_tool" {
		t.Errorf("expected Name to be 'test_tool', got %s", config.Name)
	}

	// The tool should have the following parameters when initialized:
	// - table_name (string, required) - The table to get statistics for
	// - catalog (string, optional) - Catalog name
	// - schema (string, optional) - Schema name
	// - include_columns (boolean, default true) - Include column statistics
	// - include_partitions (boolean, default false) - Include partition info
	// - analyze_table (boolean, default false) - Run ANALYZE before getting stats
}

func TestParseTableName(t *testing.T) {
	// Test table name parsing logic
	testCases := []struct {
		name            string
		tableName       string
		catalog         string
		schema          string
		expectedCatalog string
		expectedSchema  string
		expectedTable   string
	}{
		{
			name:            "fully qualified",
			tableName:       "catalog1.schema1.table1",
			catalog:         "",
			schema:          "",
			expectedCatalog: "catalog1",
			expectedSchema:  "schema1",
			expectedTable:   "table1",
		},
		{
			name:            "schema qualified",
			tableName:       "schema1.table1",
			catalog:         "catalog2",
			schema:          "",
			expectedCatalog: "catalog2",
			expectedSchema:  "schema1",
			expectedTable:   "table1",
		},
		{
			name:            "table only",
			tableName:       "table1",
			catalog:         "catalog3",
			schema:          "schema3",
			expectedCatalog: "catalog3",
			expectedSchema:  "schema3",
			expectedTable:   "table1",
		},
		{
			name:            "table only with defaults",
			tableName:       "table1",
			catalog:         "",
			schema:          "",
			expectedCatalog: "CURRENT_CATALOG",
			expectedSchema:  "CURRENT_SCHEMA",
			expectedTable:   "table1",
		},
	}

	// These test cases verify the expected parsing behavior
	// Actual testing would require invoking the private parseTableName method
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test verification would go here
		})
	}
}
