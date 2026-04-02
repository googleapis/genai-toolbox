// Copyright 2026 Google LLC
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

package cloudsqlmysql

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"
)

func TestCloudSQLMySQLToolEndpointsMCP(t *testing.T) {
	sourceConfig := getCloudSQLMySQLVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	pool, err := initCloudSQLMySQLConnectionPool(CloudSQLMySQLProject, CloudSQLMySQLRegion, CloudSQLMySQLInstance, "public", CloudSQLMySQLUser, CloudSQLMySQLPass, CloudSQLMySQLDatabase)
	if err != nil {
		t.Fatalf("unable to create Cloud SQL connection pool: %s", err)
	}

	// cleanup test environment
	tests.CleanupMySQLTables(t, ctx, pool)

	// create table name with UUID
	tableNameParam := "param_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableNameAuth := "auth_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableNameTemplateParam := "template_param_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")

	// set up data for param tool
	createParamTableStmt, insertParamTableStmt, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, paramTestParams := tests.GetMySQLParamToolInfo(tableNameParam)
	teardownTable1 := tests.SetupMySQLTable(t, ctx, pool, createParamTableStmt, insertParamTableStmt, tableNameParam, paramTestParams)
	defer teardownTable1(t)

	// set up data for auth tool
	createAuthTableStmt, insertAuthTableStmt, authToolStmt, authTestParams := tests.GetMySQLAuthToolInfo(tableNameAuth)
	teardownTable2 := tests.SetupMySQLTable(t, ctx, pool, createAuthTableStmt, insertAuthTableStmt, tableNameAuth, authTestParams)
	defer teardownTable2(t)

	// Write config into a file and pass it to command
	toolsFile := tests.GetToolsConfig(sourceConfig, CloudSQLMySQLToolType, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, authToolStmt)
	toolsFile = tests.AddMySqlExecuteSqlConfig(t, toolsFile)
	tmplSelectCombined, tmplSelectFilterCombined := tests.GetMySQLTmplToolStatement()
	toolsFile = tests.AddTemplateParamConfig(t, toolsFile, CloudSQLMySQLToolType, tmplSelectCombined, tmplSelectFilterCombined, "")
	toolsFile = tests.AddMySQLPrebuiltToolConfig(t, toolsFile)

	cmd, cleanup, err := tests.StartCmd(ctx, toolsFile)
	if err != nil {
		t.Fatalf("command initialization returned an error: %s", err)
	}
	defer cleanup()

	waitCtx, waitCancel := context.WithTimeout(ctx, 10*time.Second)
	defer waitCancel()
	out, err := testutils.WaitForString(waitCtx, regexp.MustCompile(`Server ready to serve`), cmd.Out)
	if err != nil {
		t.Logf("toolbox command logs: \n%s", out)
		t.Fatalf("toolbox didn't start successfully: %s", err)
	}

	// Get configs for tests
	select1Want, mcpMyFailToolWant, createTableStatement, mcpSelect1Want := tests.GetMySQLWants()

	// Expected Manifest
	expectedTools := tests.GetBaseMCPExpectedTools()
	expectedTools = append(expectedTools, tests.GetExecuteSQLMCPExpectedTools()...)
	expectedTools = append(expectedTools, tests.GetTemplateParamMCPExpectedTools()...)
	expectedTools = append(expectedTools, []tests.MCPToolManifest{
		{
			Name:        "list_tables",
			Description: "Lists tables in the database.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"output_format": map[string]any{
						"default":     "detailed",
						"description": "Optional: Use 'simple' for names only or 'detailed' for full info.",
						"type":        "string",
					},
					"table_names": map[string]any{
						"default":     "",
						"description": "Optional: A comma-separated list of table names. If empty, details for all tables will be listed.",
						"type":        "string",
					},
				},
				"required": []any{},
			},
		},
		{
			Name:        "list_active_queries",
			Description: "Lists active queries in the database.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"limit": map[string]any{
						"default":     float64(100),
						"description": "Optional: The maximum number of rows to return.",
						"type":        "integer",
					},
					"min_duration_secs": map[string]any{
						"default":     float64(0),
						"description": "Optional: Only show queries running for at least this long in seconds",
						"type":        "integer",
					},
				},
				"required": []any{},
			},
		},
		{Name: "list_tables_missing_unique_indexes", Description: "Lists tables that do not have primary or unique indexes in the database.", InputSchema: map[string]any{"type": "object", "properties": map[string]any{"limit": map[string]any{"default": float64(0), "description": "Optional: Maximum number of tables to return. If 0 or negative, returns all matching tables. (Default: 0)", "type": "integer"}, "table_schema": map[string]any{"default": "", "description": "Optional: Database name to filter tables missing unique indexes. Defaults to the database specified in the connection string.", "type": "string"}}, "required": []any{}}},
		{Name: "list_table_fragmentation", Description: "Lists table fragmentation in the database.", InputSchema: map[string]any{"type": "object", "properties": map[string]any{"data_free_threshold_bytes": map[string]any{"default": float64(10485760), "description": "Optional: Minimum fragmented space in bytes required for a table to be included in the result. Defaults to 10MB (10485760 bytes).", "type": "integer"}, "limit": map[string]any{"default": float64(0), "description": "Optional: Maximum number of tables to return, ordered by highest fragmentation percentage first. If 0 or negative, returns all matching tables. (Default: 0)", "type": "integer"}, "table_name": map[string]any{"default": "", "description": "Optional: Specific table name to retrieve fragmentation details for. If empty, retrieves for all tables in the specified schema.", "type": "string"}, "table_schema": map[string]any{"default": "", "description": "Optional: Database name to filter table fragmentation. Defaults to the database specified in the connection string.", "type": "string"}}, "required": []any{}}},
		{Name: "get_query_plan", Description: "Gets the query plan for a SQL statement.", InputSchema: map[string]any{"type": "object", "properties": map[string]any{"sql_statement": map[string]any{"type": "string", "description": "The SQL query statement to analyze."}}, "required": []any{"sql_statement"}}},
	}...)

	t.Run("verify tools/list registry returns complete manifest", func(t *testing.T) {
		tests.RunMCPToolsListMethod(t, expectedTools)
	})

	// Run Shared Tool Executions
	t.Run("verify standard shared tool executions", func(t *testing.T) {
		tests.RunMCPToolInvokeTest(t, select1Want, tests.DisableArrayTest())
		tests.RunMCPToolCallMethod(t, mcpMyFailToolWant, mcpSelect1Want)

		statusCode, failResp, err := tests.InvokeMCPTool(t, "my-fail-tool", map[string]any{}, nil)
		if err == nil && statusCode == http.StatusOK {
			tests.AssertMCPError(t, failResp, "You have an error in your SQL syntax")
		}

		statusCode, mcpResp, err := tests.InvokeMCPTool(t, "my-exec-sql-tool", map[string]any{"sql": createTableStatement}, nil)
		if err == nil && statusCode == http.StatusOK && !mcpResp.Result.IsError {
			tests.RunMCPCustomToolCallMethod(t, "my-exec-sql-tool", map[string]any{"sql": "SELECT 1"}, select1Want)
		}

		tests.InvokeMCPTool(t, "create-table-templateParams-tool", map[string]any{"tableName": tableNameTemplateParam, "columns": []any{"id INT", "name VARCHAR(20)", "age INT"}}, nil)
		tests.InvokeMCPTool(t, "insert-table-templateParams-tool", map[string]any{"tableName": tableNameTemplateParam, "columns": []any{"id", "name", "age"}, "values": "1, 'Alex', 21"}, nil)
		tests.RunMCPCustomToolCallMethod(t, "select-filter-templateParams-combined-tool", map[string]any{"name": "Alex", "tableName": tableNameTemplateParam, "columnFilter": "name"}, `[{"id":1,"name":"Alex","age":21}]`)
	})

	// Run specific MySQL prebuilt tool tests via MCP
	const expectedOwner = "'toolbox-identity'@'%'"
	tests.RunMySQLListTablesTest(t, CloudSQLMySQLDatabase, tableNameParam, tableNameAuth, expectedOwner, tests.WithMCPExec())
	tests.RunMySQLListActiveQueriesTest(t, ctx, pool, tests.WithMCPExec())
	tests.RunMySQLGetQueryPlanTest(t, ctx, pool, CloudSQLMySQLDatabase, tableNameParam, tests.WithMCPExec())
	tests.RunMySQLListTablesMissingUniqueIndexes(t, ctx, pool, CloudSQLMySQLDatabase, tests.WithMCPExec())
	tests.RunMySQLListTableFragmentationTest(t, CloudSQLMySQLDatabase, tableNameParam, tableNameAuth, tests.WithMCPExec())

	// Verify parameter validation for prebuilt tools (MCP Specific)
	t.Run("verify parameter validation for prebuilt tools", func(t *testing.T) {
		statusCode, mcpResp, err := tests.InvokeMCPTool(t, "get_query_plan", map[string]any{}, nil)
		if err != nil && statusCode == http.StatusOK {
			t.Fatalf("native error executing get_query_plan: %v", err)
		}
		if statusCode != http.StatusOK {
			t.Fatalf("expected status 200, got %d", statusCode)
		}
		tests.AssertMCPError(t, mcpResp, `parameter "sql_statement" is required`)
	})
}

// Test connection with different IP type over MCP
func TestCloudSQLMySQLMCPIpConnection(t *testing.T) {
	sourceConfig := getCloudSQLMySQLVars(t)
	tcs := []struct {
		name   string
		ipType string
	}{
		{name: "verify connection using public ip over MCP", ipType: "public"},
		{name: "verify connection using private ip over MCP", ipType: "private"},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			sourceConfig["ipType"] = tc.ipType
			err := tests.RunSourceConnectionTest(t, sourceConfig, CloudSQLMySQLToolType, tests.WithMCP())
			if err != nil {
				t.Fatalf("Connection test failure via MCP: %s", err)
			}
		})
	}
}

func TestCloudSQLMySQLMCPIAMConnection(t *testing.T) {
	getCloudSQLMySQLVars(t)
	serviceAccountEmail, _, _ := strings.Cut(tests.ServiceAccountEmail, "@")

	noPassSourceConfig := map[string]any{
		"type":     CloudSQLMySQLSourceType,
		"project":  CloudSQLMySQLProject,
		"instance": CloudSQLMySQLInstance,
		"region":   CloudSQLMySQLRegion,
		"database": CloudSQLMySQLDatabase,
		"user":     serviceAccountEmail,
	}
	noUserSourceConfig := map[string]any{
		"type":     CloudSQLMySQLSourceType,
		"project":  CloudSQLMySQLProject,
		"instance": CloudSQLMySQLInstance,
		"region":   CloudSQLMySQLRegion,
		"database": CloudSQLMySQLDatabase,
		"password": "random",
	}
	noUserNoPassSourceConfig := map[string]any{
		"type":     CloudSQLMySQLSourceType,
		"project":  CloudSQLMySQLProject,
		"instance": CloudSQLMySQLInstance,
		"region":   CloudSQLMySQLRegion,
		"database": CloudSQLMySQLDatabase,
	}
	tcs := []struct {
		name         string
		sourceConfig map[string]any
		isErr        bool
	}{
		{name: "verify successful IAM connection omitting user and password", sourceConfig: noUserNoPassSourceConfig, isErr: false},
		{name: "verify successful IAM connection with valid user and omitted password", sourceConfig: noPassSourceConfig, isErr: false},
		{name: "verify failing IAM connection with omitted user and arbitrary password", sourceConfig: noUserSourceConfig, isErr: true},
	}
	for i, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()

			uniqueSourceName := fmt.Sprintf("iam-test-mcp-%d", i)

			toolsFile := map[string]any{
				"sources": map[string]any{
					uniqueSourceName: tc.sourceConfig,
				},
				"tools": map[string]any{
					"my-simple-tool": map[string]any{
						"type":        CloudSQLMySQLToolType,
						"source":      uniqueSourceName,
						"description": "Simple tool to test end to end functionality.",
						"statement":   "SELECT 1;",
					},
				},
			}

			cmd, cleanup, err := tests.StartCmd(ctx, toolsFile)
			if err != nil {
				t.Fatalf("command initialization returned an error: %s", err)
			}
			defer cleanup()

			waitCtx, waitCancel := context.WithTimeout(ctx, 10*time.Second)
			defer waitCancel()

			out, err := testutils.WaitForString(waitCtx, regexp.MustCompile(`Server ready to serve`), cmd.Out)
			if err != nil {
				if tc.isErr {
					outLower := strings.ToLower(out)
					if !strings.Contains(outLower, "error") && !strings.Contains(outLower, "fail") {
						t.Fatalf("Expected an authentication/connection error, but server failed for another reason. Output: %s", out)
					}
					return
				}
				t.Logf("toolbox command logs: \n%s", out)
				t.Fatalf("Connection test failure: toolbox didn't start successfully: %s", err)
			}

			if tc.isErr {
				t.Fatalf("Expected error but test passed.")
			}
		})
	}
}
