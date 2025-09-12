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

package bigquery

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	bigqueryapi "cloud.google.com/go/bigquery"
	"github.com/google/uuid"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

var (
	BigquerySourceKind = "bigquery"
	BigqueryToolKind   = "bigquery-sql"
	BigqueryProject    = os.Getenv("BIGQUERY_PROJECT")
)

func getBigQueryVars(t *testing.T) map[string]any {
	switch "" {
	case BigqueryProject:
		t.Fatal("'BIGQUERY_PROJECT' not set")
	}

	return map[string]any{
		"kind":    BigquerySourceKind,
		"project": BigqueryProject,
	}
}

// Copied over from bigquery.go
func initBigQueryConnection(project string) (*bigqueryapi.Client, error) {
	ctx := context.Background()
	cred, err := google.FindDefaultCredentials(ctx, bigqueryapi.Scope)
	if err != nil {
		return nil, fmt.Errorf("failed to find default Google Cloud credentials with scope %q: %w", bigqueryapi.Scope, err)
	}

	client, err := bigqueryapi.NewClient(ctx, project, option.WithCredentials(cred))
	if err != nil {
		return nil, fmt.Errorf("failed to create BigQuery client for project %q: %w", project, err)
	}
	return client, nil
}

func TestBigQueryToolEndpoints(t *testing.T) {
	sourceConfig := getBigQueryVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	var args []string

	client, err := initBigQueryConnection(BigqueryProject)
	if err != nil {
		t.Fatalf("unable to create Cloud SQL connection pool: %s", err)
	}

	// create table name with UUID
	datasetName := fmt.Sprintf("temp_toolbox_test_%s", strings.ReplaceAll(uuid.New().String(), "-", ""))
	tableName := fmt.Sprintf("param_table_%s", strings.ReplaceAll(uuid.New().String(), "-", ""))
	tableNameParam := fmt.Sprintf("`%s.%s.%s`",
		BigqueryProject,
		datasetName,
		tableName,
	)
	tableNameAuth := fmt.Sprintf("`%s.%s.auth_table_%s`",
		BigqueryProject,
		datasetName,
		strings.ReplaceAll(uuid.New().String(), "-", ""),
	)
	tableNameTemplateParam := fmt.Sprintf("`%s.%s.template_param_table_%s`",
		BigqueryProject,
		datasetName,
		strings.ReplaceAll(uuid.New().String(), "-", ""),
	)
	tableNameDataType := fmt.Sprintf("`%s.%s.datatype_table_%s`",
		BigqueryProject,
		datasetName,
		strings.ReplaceAll(uuid.New().String(), "-", ""),
	)
	tableNameForecast := fmt.Sprintf("`%s.%s.forecast_table_%s`",
		BigqueryProject,
		datasetName,
		strings.ReplaceAll(uuid.New().String(), "-", ""),
	)

	tableNameAnalyzeContribution := fmt.Sprintf("`%s.%s.analyze_contribution_table_%s`",
		BigqueryProject,
		datasetName,
		strings.ReplaceAll(uuid.New().String(), "-", ""),
	)

	// set up data for param tool
	createParamTableStmt, insertParamTableStmt, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, paramTestParams := getBigQueryParamToolInfo(tableNameParam)
	teardownTable1 := setupBigQueryTable(t, ctx, client, createParamTableStmt, insertParamTableStmt, datasetName, tableNameParam, paramTestParams)
	defer teardownTable1(t)

	// set up data for auth tool
	createAuthTableStmt, insertAuthTableStmt, authToolStmt, authTestParams := getBigQueryAuthToolInfo(tableNameAuth)
	teardownTable2 := setupBigQueryTable(t, ctx, client, createAuthTableStmt, insertAuthTableStmt, datasetName, tableNameAuth, authTestParams)
	defer teardownTable2(t)

	// set up data for data type test tool
	createDataTypeTableStmt, insertDataTypeTableStmt, dataTypeToolStmt, arrayDataTypeToolStmt, dataTypeTestParams := getBigQueryDataTypeTestInfo(tableNameDataType)
	teardownTable3 := setupBigQueryTable(t, ctx, client, createDataTypeTableStmt, insertDataTypeTableStmt, datasetName, tableNameDataType, dataTypeTestParams)
	defer teardownTable3(t)

	// set up data for forecast tool
	createForecastTableStmt, insertForecastTableStmt, forecastTestParams := getBigQueryForecastToolInfo(tableNameForecast)
	teardownTable4 := setupBigQueryTable(t, ctx, client, createForecastTableStmt, insertForecastTableStmt, datasetName, tableNameForecast, forecastTestParams)
	defer teardownTable4(t)

	// set up data for analyze contribution tool
	createAnalyzeContributionTableStmt, insertAnalyzeContributionTableStmt, analyzeContributionTestParams := getBigQueryAnalyzeContributionToolInfo(tableNameAnalyzeContribution)
	teardownTable5 := setupBigQueryTable(t, ctx, client, createAnalyzeContributionTableStmt, insertAnalyzeContributionTableStmt, datasetName, tableNameAnalyzeContribution, analyzeContributionTestParams)
	defer teardownTable5(t)

	// Write config into a file and pass it to command
	toolsFile := tests.GetToolsConfig(sourceConfig, BigqueryToolKind, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, authToolStmt)
	toolsFile = addClientAuthSourceConfig(t, toolsFile)
	toolsFile = addBigQuerySqlToolConfig(t, toolsFile, dataTypeToolStmt, arrayDataTypeToolStmt)
	toolsFile = addBigQueryPrebuiltToolsConfig(t, toolsFile)
	tmplSelectCombined, tmplSelectFilterCombined := getBigQueryTmplToolStatement()
	toolsFile = tests.AddTemplateParamConfig(t, toolsFile, BigqueryToolKind, tmplSelectCombined, tmplSelectFilterCombined, "")

	cmd, cleanup, err := tests.StartCmd(ctx, toolsFile, args...)
	if err != nil {
		t.Fatalf("command initialization returned an error: %s", err)
	}
	defer cleanup()

	waitCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	out, err := testutils.WaitForString(waitCtx, regexp.MustCompile(`Server ready to serve`), cmd.Out)
	if err != nil {
		t.Logf("toolbox command logs: \n%s", out)
		t.Fatalf("toolbox didn't start successfully: %s", err)
	}

	// Get configs for tests
	select1Want := "[{\"f0_\":1}]"
	invokeParamWant := "[{\"id\":1,\"name\":\"Alice\"},{\"id\":3,\"name\":\"Sid\"}]"
	datasetInfoWant := "\"Location\":\"US\",\"DefaultTableExpiration\":0,\"Labels\":null,\"Access\":"
	tableInfoWant := "{\"Name\":\"\",\"Location\":\"US\",\"Description\":\"\",\"Schema\":[{\"Name\":\"id\""
	ddlWant := `"Query executed successfully and returned no content."`
	dataInsightsWant := `(?s)Schema Resolved.*Retrieval Query.*SQL Generated.*Answer`
	// Partial message; the full error message is too long.
	mcpMyFailToolWant := `{"jsonrpc":"2.0","id":"invoke-fail-tool","result":{"content":[{"type":"text","text":"final query validation failed: failed to insert dry run job: googleapi: Error 400: Syntax error: Unexpected identifier \"SELEC\" at [1:1]`
	mcpSelect1Want := `{"jsonrpc":"2.0","id":"invoke my-auth-required-tool","result":{"content":[{"type":"text","text":"{\"f0_\":1}"}]}}`
	createColArray := `["id INT64", "name STRING", "age INT64"]`
	selectEmptyWant := `"The query returned 0 rows."`

	// Run tests
	tests.RunToolGetTest(t)
	tests.RunToolInvokeTest(t, select1Want, tests.DisableOptionalNullParamTest(), tests.EnableClientAuthTest())
	tests.RunMCPToolCallMethod(t, mcpMyFailToolWant, mcpSelect1Want, tests.EnableMcpClientAuthTest())
	tests.RunToolInvokeWithTemplateParameters(t, tableNameTemplateParam,
		tests.WithCreateColArray(createColArray),
		tests.WithDdlWant(ddlWant),
		tests.WithSelectEmptyWant(selectEmptyWant),
		tests.WithInsert1Want(ddlWant),
	)

	runBigQueryExecuteSqlToolInvokeTest(t, select1Want, invokeParamWant, tableNameParam, ddlWant)
	runBigQueryExecuteSqlToolInvokeDryRunTest(t, datasetName)
	runBigQueryForecastToolInvokeTest(t, tableNameForecast)
	runBigQueryAnalyzeContributionToolInvokeTest(t, tableNameAnalyzeContribution)
	runBigQueryDataTypeTests(t)
	runBigQueryListDatasetToolInvokeTest(t, datasetName)
	runBigQueryGetDatasetInfoToolInvokeTest(t, datasetName, datasetInfoWant)
	runBigQueryListTableIdsToolInvokeTest(t, datasetName, tableName)
	runBigQueryGetTableInfoToolInvokeTest(t, datasetName, tableName, tableInfoWant)
	runBigQueryConversationalAnalyticsInvokeTest(t, datasetName, tableName, dataInsightsWant)
	runBigQuerySearchCatalogToolInvokeTest(t, datasetName, tableName)
}

func TestBigQueryToolWithDatasetRestriction(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	client, err := initBigQueryConnection(BigqueryProject)
	if err != nil {
		t.Fatalf("unable to create BigQuery client: %s", err)
	}

	// Create two datasets, one allowed, one not.
	baseName := strings.ReplaceAll(uuid.New().String(), "-", "")
	allowedDatasetName1 := fmt.Sprintf("allowed_dataset_1_%s", baseName)
	allowedDatasetName2 := fmt.Sprintf("allowed_dataset_2_%s", baseName)
	disallowedDatasetName := fmt.Sprintf("disallowed_dataset_%s", baseName)
	allowedTableName1 := "allowed_table_1"
	allowedTableName2 := "allowed_table_2"
	disallowedTableName := "disallowed_table"
	allowedForecastTableName1 := "allowed_forecast_table_1"
	allowedForecastTableName2 := "allowed_forecast_table_2"
	disallowedForecastTableName := "disallowed_forecast_table"

	// Setup allowed table
	allowedTableNameParam1 := fmt.Sprintf("`%s.%s.%s`", BigqueryProject, allowedDatasetName1, allowedTableName1)
	createAllowedTableStmt1 := fmt.Sprintf("CREATE TABLE %s (id INT64)", allowedTableNameParam1)
	teardownAllowed1 := setupBigQueryTable(t, ctx, client, createAllowedTableStmt1, "", allowedDatasetName1, allowedTableNameParam1, nil)
	defer teardownAllowed1(t)

	allowedTableNameParam2 := fmt.Sprintf("`%s.%s.%s`", BigqueryProject, allowedDatasetName2, allowedTableName2)
	createAllowedTableStmt2 := fmt.Sprintf("CREATE TABLE %s (id INT64)", allowedTableNameParam2)
	teardownAllowed2 := setupBigQueryTable(t, ctx, client, createAllowedTableStmt2, "", allowedDatasetName2, allowedTableNameParam2, nil)
	defer teardownAllowed2(t)

	// Setup allowed forecast table
	allowedForecastTableFullName1 := fmt.Sprintf("`%s.%s.%s`", BigqueryProject, allowedDatasetName1, allowedForecastTableName1)
	createForecastStmt1, insertForecastStmt1, forecastParams1 := getBigQueryForecastToolInfo(allowedForecastTableFullName1)
	teardownAllowedForecast1 := setupBigQueryTable(t, ctx, client, createForecastStmt1, insertForecastStmt1, allowedDatasetName1, allowedForecastTableFullName1, forecastParams1)
	defer teardownAllowedForecast1(t)

	allowedForecastTableFullName2 := fmt.Sprintf("`%s.%s.%s`", BigqueryProject, allowedDatasetName2, allowedForecastTableName2)
	createForecastStmt2, insertForecastStmt2, forecastParams2 := getBigQueryForecastToolInfo(allowedForecastTableFullName2)
	teardownAllowedForecast2 := setupBigQueryTable(t, ctx, client, createForecastStmt2, insertForecastStmt2, allowedDatasetName2, allowedForecastTableFullName2, forecastParams2)
	defer teardownAllowedForecast2(t)

	// Setup disallowed table
	disallowedTableNameParam := fmt.Sprintf("`%s.%s.%s`", BigqueryProject, disallowedDatasetName, disallowedTableName)
	createDisallowedTableStmt := fmt.Sprintf("CREATE TABLE %s (id INT64)", disallowedTableNameParam)
	teardownDisallowed := setupBigQueryTable(t, ctx, client, createDisallowedTableStmt, "", disallowedDatasetName, disallowedTableNameParam, nil)
	defer teardownDisallowed(t)

	// Setup disallowed forecast table
	disallowedForecastTableFullName := fmt.Sprintf("`%s.%s.%s`", BigqueryProject, disallowedDatasetName, disallowedForecastTableName)
	createDisallowedForecastStmt, insertDisallowedForecastStmt, disallowedForecastParams := getBigQueryForecastToolInfo(disallowedForecastTableFullName)
	teardownDisallowedForecast := setupBigQueryTable(t, ctx, client, createDisallowedForecastStmt, insertDisallowedForecastStmt, disallowedDatasetName, disallowedForecastTableFullName, disallowedForecastParams)
	defer teardownDisallowedForecast(t)

	// Configure source with dataset restriction.
	sourceConfig := getBigQueryVars(t)
	sourceConfig["allowedDatasets"] = []string{allowedDatasetName1, allowedDatasetName2}

	// Configure tool
	toolsConfig := map[string]any{
		"list-table-ids-restricted": map[string]any{
			"kind":        "bigquery-list-table-ids",
			"source":      "my-instance",
			"description": "Tool to list table within a dataset",
		},
	}

	// Create config file
	config := map[string]any{
		"sources": map[string]any{
			"my-instance": sourceConfig,
		},
		"tools": toolsConfig,
	}

	// Start server
	cmd, cleanup, err := tests.StartCmd(ctx, config)
	if err != nil {
		t.Fatalf("command initialization returned an error: %s", err)
	}
	defer cleanup()

	waitCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	out, err := testutils.WaitForString(waitCtx, regexp.MustCompile(`Server ready to serve`), cmd.Out)
	if err != nil {
		t.Logf("toolbox command logs: \n%s", out)
		t.Fatalf("toolbox didn't start successfully: %s", err)
	}

	// Run tests
	runListTableIdsWithRestriction(t, allowedDatasetName1, disallowedDatasetName, allowedTableName1, allowedForecastTableName1)
	runListTableIdsWithRestriction(t, allowedDatasetName2, disallowedDatasetName, allowedTableName2, allowedForecastTableName2)
}

// getBigQueryParamToolInfo returns statements and param for my-tool for bigquery kind
func getBigQueryParamToolInfo(tableName string) (string, string, string, string, string, string, []bigqueryapi.QueryParameter) {
	createStatement := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (id INT64, name STRING);`, tableName)
	insertStatement := fmt.Sprintf(`
		INSERT INTO %s (id, name) VALUES (?, ?), (?, ?), (?, ?), (?, NULL);`, tableName)
	toolStatement := fmt.Sprintf(`SELECT * FROM %s WHERE id = ? OR name = ? ORDER BY id;`, tableName)
	idToolStatement := fmt.Sprintf(`SELECT * FROM %s WHERE id = ? ORDER BY id;`, tableName)
	nameToolStatement := fmt.Sprintf(`SELECT * FROM %s WHERE name = ? ORDER BY id;`, tableName)
	arrayToolStatememt := fmt.Sprintf(`SELECT * FROM %s WHERE id IN UNNEST(@idArray) AND name IN UNNEST(@nameArray) ORDER BY id;`, tableName)
	params := []bigqueryapi.QueryParameter{
		{Value: int64(1)}, {Value: "Alice"},
		{Value: int64(2)}, {Value: "Jane"},
		{Value: int64(3)}, {Value: "Sid"},
		{Value: int64(4)},
	}
	return createStatement, insertStatement, toolStatement, idToolStatement, nameToolStatement, arrayToolStatememt, params
}

// getBigQueryAuthToolInfo returns statements and param of my-auth-tool for bigquery kind
func getBigQueryAuthToolInfo(tableName string) (string, string, string, []bigqueryapi.QueryParameter) {
	createStatement := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (id INT64, name STRING, email STRING)`, tableName)
	insertStatement := fmt.Sprintf(`
		INSERT INTO %s (id, name, email) VALUES (?, ?, ?), (?, ?, ?)`, tableName)
	toolStatement := fmt.Sprintf(`
		SELECT name FROM %s WHERE email = ?`, tableName)
	params := []bigqueryapi.QueryParameter{
		{Value: int64(1)}, {Value: "Alice"}, {Value: tests.ServiceAccountEmail},
		{Value: int64(2)}, {Value: "Jane"}, {Value: "janedoe@gmail.com"},
	}
	return createStatement, insertStatement, toolStatement, params
}

// getBigQueryDataTypeTestInfo returns statements and params for data type tests.
func getBigQueryDataTypeTestInfo(tableName string) (string, string, string, string, []bigqueryapi.QueryParameter) {
	createStatement := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (id INT64, int_val INT64, string_val STRING, float_val FLOAT64, bool_val BOOL);`, tableName)
	insertStatement := fmt.Sprintf(`
		INSERT INTO %s (id, int_val, string_val, float_val, bool_val) VALUES (?, ?, ?, ?, ?), (?, ?, ?, ?, ?), (?, ?, ?, ?, ?);`, tableName)
	toolStatement := fmt.Sprintf(`SELECT * FROM %s WHERE int_val = ? AND string_val = ? AND float_val = ? AND bool_val = ?;`, tableName)
	arrayToolStatement := fmt.Sprintf(`SELECT * FROM %s WHERE int_val IN UNNEST(@int_array) AND string_val IN UNNEST(@string_array) AND float_val IN UNNEST(@float_array) AND bool_val IN UNNEST(@bool_array) ORDER BY id;`, tableName)
	params := []bigqueryapi.QueryParameter{
		{Value: int64(1)}, {Value: int64(123)}, {Value: "hello"}, {Value: 3.14}, {Value: true},
		{Value: int64(2)}, {Value: int64(-456)}, {Value: "world"}, {Value: -0.55}, {Value: false},
		{Value: int64(3)}, {Value: int64(789)}, {Value: "test"}, {Value: 100.1}, {Value: true},
	}
	return createStatement, insertStatement, toolStatement, arrayToolStatement, params
}

// getBigQueryForecastToolInfo returns statements and params for the forecast tool.
func getBigQueryForecastToolInfo(tableName string) (string, string, []bigqueryapi.QueryParameter) {
	createStatement := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (ts TIMESTAMP, data FLOAT64, id STRING);`, tableName)
	insertStatement := fmt.Sprintf(`
		INSERT INTO %s (ts, data, id) VALUES
		(?, ?, ?), (?, ?, ?), (?, ?, ?), 
		(?, ?, ?), (?, ?, ?), (?, ?, ?);`, tableName)
	params := []bigqueryapi.QueryParameter{
		{Value: "2025-01-01T00:00:00Z"}, {Value: 10.0}, {Value: "a"},
		{Value: "2025-01-01T01:00:00Z"}, {Value: 11.0}, {Value: "a"},
		{Value: "2025-01-01T02:00:00Z"}, {Value: 12.0}, {Value: "a"},
		{Value: "2025-01-01T00:00:00Z"}, {Value: 20.0}, {Value: "b"},
		{Value: "2025-01-01T01:00:00Z"}, {Value: 21.0}, {Value: "b"},
		{Value: "2025-01-01T02:00:00Z"}, {Value: 22.0}, {Value: "b"},
	}
	return createStatement, insertStatement, params
}

// getBigQueryAnalyzeContributionToolInfo returns statements and params for the analyze-contribution tool.
func getBigQueryAnalyzeContributionToolInfo(tableName string) (string, string, []bigqueryapi.QueryParameter) {
	createStatement := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (dim1 STRING, dim2 STRING, is_test BOOL, metric FLOAT64);`, tableName)
	insertStatement := fmt.Sprintf(`
		INSERT INTO %s (dim1, dim2, is_test, metric) VALUES 
		(?, ?, ?, ?), (?, ?, ?, ?), (?, ?, ?, ?), (?, ?, ?, ?);`, tableName)
	params := []bigqueryapi.QueryParameter{
		{Value: "a"}, {Value: "x"}, {Value: true}, {Value: 100.0},
		{Value: "a"}, {Value: "x"}, {Value: false}, {Value: 110.0},
		{Value: "a"}, {Value: "y"}, {Value: true}, {Value: 120.0},
		{Value: "a"}, {Value: "y"}, {Value: false}, {Value: 100.0},
		{Value: "b"}, {Value: "x"}, {Value: true}, {Value: 40.0},
		{Value: "b"}, {Value: "x"}, {Value: false}, {Value: 100.0},
		{Value: "b"}, {Value: "y"}, {Value: true}, {Value: 60.0},
		{Value: "b"}, {Value: "y"}, {Value: false}, {Value: 60.0},
	}
	return createStatement, insertStatement, params
}

// getBigQueryTmplToolStatement returns statements for template parameter test cases for bigquery kind
func getBigQueryTmplToolStatement() (string, string) {
	tmplSelectCombined := "SELECT * FROM {{.tableName}} WHERE id = ? ORDER BY id"
	tmplSelectFilterCombined := "SELECT * FROM {{.tableName}} WHERE {{.columnFilter}} = ? ORDER BY id"
	return tmplSelectCombined, tmplSelectFilterCombined
}

func setupBigQueryTable(t *testing.T, ctx context.Context, client *bigqueryapi.Client, createStatement, insertStatement, datasetName string, tableName string, params []bigqueryapi.QueryParameter) func(*testing.T) {
	// Create dataset
	dataset := client.Dataset(datasetName)
	_, err := dataset.Metadata(ctx)

	if err != nil {
		apiErr, ok := err.(*googleapi.Error)
		if !ok || apiErr.Code != 404 {
			t.Fatalf("Failed to check dataset %q existence: %v", datasetName, err)
		}
		metadataToCreate := &bigqueryapi.DatasetMetadata{Name: datasetName}
		if err := dataset.Create(ctx, metadataToCreate); err != nil {
			t.Fatalf("Failed to create dataset %q: %v", datasetName, err)
		}
	}

	// Create table
	createJob, err := client.Query(createStatement).Run(ctx)

	if err != nil {
		t.Fatalf("Failed to start create table job for %s: %v", tableName, err)
	}
	createStatus, err := createJob.Wait(ctx)
	if err != nil {
		t.Fatalf("Failed to wait for create table job for %s: %v", tableName, err)
	}
	if err := createStatus.Err(); err != nil {
		t.Fatalf("Create table job for %s failed: %v", tableName, err)
	}

	if len(params) > 0 {
		// Insert test data
		insertQuery := client.Query(insertStatement)
		insertQuery.Parameters = params
		insertJob, err := insertQuery.Run(ctx)
		if err != nil {
			t.Fatalf("Failed to start insert job for %s: %v", tableName, err)
		}
		insertStatus, err := insertJob.Wait(ctx)
		if err != nil {
			t.Fatalf("Failed to wait for insert job for %s: %v", tableName, err)
		}
		if err := insertStatus.Err(); err != nil {
			t.Fatalf("Insert job for %s failed: %v", tableName, err)
		}
	}

	return func(t *testing.T) {
		// tear down table
		dropSQL := fmt.Sprintf("drop table %s", tableName)
		dropJob, err := client.Query(dropSQL).Run(ctx)
		if err != nil {
			t.Errorf("Failed to start drop table job for %s: %v", tableName, err)
			return
		}
		dropStatus, err := dropJob.Wait(ctx)
		if err != nil {
			t.Errorf("Failed to wait for drop table job for %s: %v", tableName, err)
			return
		}
		if err := dropStatus.Err(); err != nil {
			t.Errorf("Error dropping table %s: %v", tableName, err)
		}

		// tear down dataset
		datasetToTeardown := client.Dataset(datasetName)
		tablesIterator := datasetToTeardown.Tables(ctx)
		_, err = tablesIterator.Next()

		if err == iterator.Done {
			if err := datasetToTeardown.Delete(ctx); err != nil {
				t.Errorf("Failed to delete dataset %s: %v", datasetName, err)
			}
		} else if err != nil {
			t.Errorf("Failed to list tables in dataset %s to check emptiness: %v.", datasetName, err)
		}
	}
}

func addBigQueryPrebuiltToolsConfig(t *testing.T, config map[string]any) map[string]any {
	tools, ok := config["tools"].(map[string]any)
	if !ok {
		t.Fatalf("unable to get tools from config")
	}
	tools["my-exec-sql-tool"] = map[string]any{
		"kind":        "bigquery-execute-sql",
		"source":      "my-instance",
		"description": "Tool to execute sql",
	}
	tools["my-auth-exec-sql-tool"] = map[string]any{
		"kind":        "bigquery-execute-sql",
		"source":      "my-instance",
		"description": "Tool to execute sql",
		"authRequired": []string{
			"my-google-auth",
		},
	}
	tools["my-client-auth-exec-sql-tool"] = map[string]any{
		"kind":        "bigquery-execute-sql",
		"source":      "my-client-auth-source",
		"description": "Tool to execute sql",
	}
	tools["my-forecast-tool"] = map[string]any{
		"kind":        "bigquery-forecast",
		"source":      "my-instance",
		"description": "Tool to forecast time series data.",
	}
	tools["my-auth-forecast-tool"] = map[string]any{
		"kind":        "bigquery-forecast",
		"source":      "my-instance",
		"description": "Tool to forecast time series data with auth.",
		"authRequired": []string{
			"my-google-auth",
		},
	}
	tools["my-client-auth-forecast-tool"] = map[string]any{
		"kind":        "bigquery-forecast",
		"source":      "my-client-auth-source",
		"description": "Tool to forecast time series data with auth.",
	}
	tools["my-analyze-contribution-tool"] = map[string]any{
		"kind":        "bigquery-analyze-contribution",
		"source":      "my-instance",
		"description": "Tool to analyze contribution.",
	}
	tools["my-auth-analyze-contribution-tool"] = map[string]any{
		"kind":        "bigquery-analyze-contribution",
		"source":      "my-instance",
		"description": "Tool to analyze contribution with auth.",
		"authRequired": []string{
			"my-google-auth",
		},
	}
	tools["my-client-auth-analyze-contribution-tool"] = map[string]any{
		"kind":        "bigquery-analyze-contribution",
		"source":      "my-client-auth-source",
		"description": "Tool to analyze contribution with auth.",
	}
	tools["my-list-dataset-ids-tool"] = map[string]any{
		"kind":        "bigquery-list-dataset-ids",
		"source":      "my-instance",
		"description": "Tool to list dataset",
	}
	tools["my-auth-list-dataset-ids-tool"] = map[string]any{
		"kind":        "bigquery-list-dataset-ids",
		"source":      "my-instance",
		"description": "Tool to list dataset",
		"authRequired": []string{
			"my-google-auth",
		},
	}
	tools["my-client-auth-list-dataset-ids-tool"] = map[string]any{
		"kind":        "bigquery-list-dataset-ids",
		"source":      "my-client-auth-source",
		"description": "Tool to list dataset",
	}
	tools["my-get-dataset-info-tool"] = map[string]any{
		"kind":        "bigquery-get-dataset-info",
		"source":      "my-instance",
		"description": "Tool to show dataset metadata",
	}
	tools["my-auth-get-dataset-info-tool"] = map[string]any{
		"kind":        "bigquery-get-dataset-info",
		"source":      "my-instance",
		"description": "Tool to show dataset metadata",
		"authRequired": []string{
			"my-google-auth",
		},
	}
	tools["my-client-auth-get-dataset-info-tool"] = map[string]any{
		"kind":        "bigquery-get-dataset-info",
		"source":      "my-client-auth-source",
		"description": "Tool to show dataset metadata",
	}
	tools["my-list-table-ids-tool"] = map[string]any{
		"kind":        "bigquery-list-table-ids",
		"source":      "my-instance",
		"description": "Tool to list table within a dataset",
	}
	tools["my-auth-list-table-ids-tool"] = map[string]any{
		"kind":        "bigquery-list-table-ids",
		"source":      "my-instance",
		"description": "Tool to list table within a dataset",
		"authRequired": []string{
			"my-google-auth",
		},
	}
	tools["my-client-auth-list-table-ids-tool"] = map[string]any{
		"kind":        "bigquery-list-table-ids",
		"source":      "my-client-auth-source",
		"description": "Tool to list table within a dataset",
	}
	tools["my-get-table-info-tool"] = map[string]any{
		"kind":        "bigquery-get-table-info",
		"source":      "my-instance",
		"description": "Tool to show dataset metadata",
	}
	tools["my-auth-get-table-info-tool"] = map[string]any{
		"kind":        "bigquery-get-table-info",
		"source":      "my-instance",
		"description": "Tool to show dataset metadata",
		"authRequired": []string{
			"my-google-auth",
		},
	}
	tools["my-client-auth-get-table-info-tool"] = map[string]any{
		"kind":        "bigquery-get-table-info",
		"source":      "my-client-auth-source",
		"description": "Tool to show dataset metadata",
	}
	tools["my-conversational-analytics-tool"] = map[string]any{
		"kind":        "bigquery-conversational-analytics",
		"source":      "my-instance",
		"description": "Tool to ask BigQuery conversational analytics",
	}
	tools["my-auth-conversational-analytics-tool"] = map[string]any{
		"kind":        "bigquery-conversational-analytics",
		"source":      "my-instance",
		"description": "Tool to ask BigQuery conversational analytics",
		"authRequired": []string{
			"my-google-auth",
		},
	}
	tools["my-client-auth-conversational-analytics-tool"] = map[string]any{
		"kind":        "bigquery-conversational-analytics",
		"source":      "my-client-auth-source",
		"description": "Tool to ask BigQuery conversational analytics",
	}
	tools["my-search-catalog-tool"] = map[string]any{
		"kind":        "bigquery-search-catalog",
		"source":      "my-instance",
		"description": "Tool to search the BiqQuery catalog",
	}
	tools["my-auth-search-catalog-tool"] = map[string]any{
		"kind":        "bigquery-search-catalog",
		"source":      "my-instance",
		"description": "Tool to search the BiqQuery catalog",
		"authRequired": []string{
			"my-google-auth",
		},
	}
	config["tools"] = tools
	return config
}

func addClientAuthSourceConfig(t *testing.T, config map[string]any) map[string]any {
	sources, ok := config["sources"].(map[string]any)
	if !ok {
		t.Fatalf("unable to get sources from config")
	}
	sources["my-client-auth-source"] = map[string]any{
		"kind":           BigquerySourceKind,
		"project":        BigqueryProject,
		"useClientOAuth": true,
	}
	config["sources"] = sources
	return config
}

func addBigQuerySqlToolConfig(t *testing.T, config map[string]any, toolStatement, arrayToolStatement string) map[string]any {
	tools, ok := config["tools"].(map[string]any)
	if !ok {
		t.Fatalf("unable to get tools from config")
	}
	tools["my-scalar-datatype-tool"] = map[string]any{
		"kind":        "bigquery-sql",
		"source":      "my-instance",
		"description": "Tool to test various scalar data types.",
		"statement":   toolStatement,
		"parameters": []any{
			map[string]any{"name": "int_val", "type": "integer", "description": "an integer value"},
			map[string]any{"name": "string_val", "type": "string", "description": "a string value"},
			map[string]any{"name": "float_val", "type": "float", "description": "a float value"},
			map[string]any{"name": "bool_val", "type": "boolean", "description": "a boolean value"},
		},
	}
	tools["my-array-datatype-tool"] = map[string]any{
		"kind":        "bigquery-sql",
		"source":      "my-instance",
		"description": "Tool to test various array data types.",
		"statement":   arrayToolStatement,
		"parameters": []any{
			map[string]any{"name": "int_array", "type": "array", "description": "an array of integer values", "items": map[string]any{"name": "item", "type": "integer", "description": "desc"}},
			map[string]any{"name": "string_array", "type": "array", "description": "an array of string values", "items": map[string]any{"name": "item", "type": "string", "description": "desc"}},
			map[string]any{"name": "float_array", "type": "array", "description": "an array of float values", "items": map[string]any{"name": "item", "type": "float", "description": "desc"}},
			map[string]any{"name": "bool_array", "type": "array", "description": "an array of boolean values", "items": map[string]any{"name": "item", "type": "boolean", "description": "desc"}},
		},
	}
	tools["my-client-auth-tool"] = map[string]any{
		"kind":        "bigquery-sql",
		"source":      "my-client-auth-source",
		"description": "Tool to test client authorization.",
		"statement":   "SELECT 1",
	}
	config["tools"] = tools
	return config
}

func runBigQueryExecuteSqlToolInvokeTest(t *testing.T, select1Want, invokeParamWant, tableNameParam, ddlWant string) {
	// Get ID token
	idToken, err := tests.GetGoogleIdToken(tests.ClientId)
	if err != nil {
		t.Fatalf("error getting Google ID token: %s", err)
	}

	// Get access token
	accessToken, err := sources.GetIAMAccessToken(t.Context())
	if err != nil {
		t.Fatalf("error getting access token from ADC: %s", err)
	}
	accessToken = "Bearer " + accessToken

	// Test tool invoke endpoint
	invokeTcs := []struct {
		name          string
		api           string
		requestHeader map[string]string
		requestBody   io.Reader
		want          string
		isErr         bool
	}{
		{
			name:          "invoke my-exec-sql-tool without body",
			api:           "http://127.0.0.1:5000/api/tool/my-exec-sql-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-exec-sql-tool",
			api:           "http://127.0.0.1:5000/api/tool/my-exec-sql-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"sql":"SELECT 1"}`)),
			want:          select1Want,
			isErr:         false,
		},
		{
			name:          "invoke my-exec-sql-tool create table",
			api:           "http://127.0.0.1:5000/api/tool/my-exec-sql-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"sql":"CREATE TABLE t (id SERIAL PRIMARY KEY, name TEXT)"}`)),
			want:          ddlWant,
			isErr:         true,
		},
		{
			name:          "invoke my-exec-sql-tool with data present in table",
			api:           "http://127.0.0.1:5000/api/tool/my-exec-sql-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf("{\"sql\":\"SELECT * FROM %s WHERE id = 3 OR name = 'Alice' ORDER BY id\"}", tableNameParam))),
			want:          invokeParamWant,
			isErr:         false,
		},
		{
			name:          "invoke my-exec-sql-tool with no matching rows",
			api:           "http://127.0.0.1:5000/api/tool/my-exec-sql-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf("{\"sql\":\"SELECT * FROM %s WHERE id = 999\"}", tableNameParam))),
			want:          `"The query returned 0 rows."`,
			isErr:         false,
		},
		{
			name:          "invoke my-exec-sql-tool drop table",
			api:           "http://127.0.0.1:5000/api/tool/my-exec-sql-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"sql":"DROP TABLE t"}`)),
			want:          ddlWant,
			isErr:         true,
		},
		{
			name:          "invoke my-exec-sql-tool insert entry",
			api:           "http://127.0.0.1:5000/api/tool/my-exec-sql-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf("{\"sql\":\"INSERT INTO %s (id, name) VALUES (4, 'test_name')\"}", tableNameParam))),
			want:          ddlWant,
			isErr:         false,
		},
		{
			name:          "invoke my-exec-sql-tool without body",
			api:           "http://127.0.0.1:5000/api/tool/my-exec-sql-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			isErr:         true,
		},
		{
			name:          "Invoke my-auth-exec-sql-tool with auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-exec-sql-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(`{"sql":"SELECT 1"}`)),
			isErr:         false,
			want:          select1Want,
		},
		{
			name:          "Invoke my-auth-exec-sql-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-exec-sql-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": "INVALID_TOKEN"},
			requestBody:   bytes.NewBuffer([]byte(`{"sql":"SELECT 1"}`)),
			isErr:         true,
		},
		{
			name:          "Invoke my-auth-exec-sql-tool without auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-exec-sql-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"sql":"SELECT 1"}`)),
			isErr:         true,
		},
		{
			name:          "Invoke my-client-auth-exec-sql-tool with auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-exec-sql-tool/invoke",
			requestHeader: map[string]string{"Authorization": accessToken},
			requestBody:   bytes.NewBuffer([]byte(`{"sql":"SELECT 1"}`)),
			want:          "[{\"f0_\":1}]",
			isErr:         false,
		},
		{
			name:          "Invoke my-client-auth-exec-sql-tool without auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-exec-sql-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"sql":"SELECT 1"}`)),
			isErr:         true,
		},
		{

			name:          "Invoke my-client-auth-exec-sql-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-exec-sql-tool/invoke",
			requestHeader: map[string]string{"Authorization": "Bearer invalid-token"},
			requestBody:   bytes.NewBuffer([]byte(`{"sql":"SELECT 1"}`)),
			isErr:         true,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			// Send Tool invocation request
			req, err := http.NewRequest(http.MethodPost, tc.api, tc.requestBody)
			if err != nil {
				t.Fatalf("unable to create request: %s", err)
			}
			req.Header.Add("Content-type", "application/json")
			for k, v := range tc.requestHeader {
				req.Header.Add(k, v)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("unable to send request: %s", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				if tc.isErr {
					return
				}
				bodyBytes, _ := io.ReadAll(resp.Body)
				t.Fatalf("response status code is not 200, got %d: %s", resp.StatusCode, string(bodyBytes))
			}

			// Check response body
			var body map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&body)
			if err != nil {
				t.Fatalf("error parsing response body")
			}

			got, ok := body["result"].(string)
			if !ok {
				t.Fatalf("unable to find result in response body")
			}

			if got != tc.want {
				t.Fatalf("unexpected value: got %q, want %q", got, tc.want)
			}
		})
	}
}

func runBigQueryExecuteSqlToolInvokeDryRunTest(t *testing.T, datasetName string) {
	// Get ID token
	idToken, err := tests.GetGoogleIdToken(tests.ClientId)
	if err != nil {
		t.Fatalf("error getting Google ID token: %s", err)
	}

	newTableName := fmt.Sprintf("%s.new_dry_run_table_%s", datasetName, strings.ReplaceAll(uuid.New().String(), "-", ""))

	// Test tool invoke endpoint
	invokeTcs := []struct {
		name          string
		api           string
		requestHeader map[string]string
		requestBody   io.Reader
		want          string
		isErr         bool
	}{
		{
			name:          "invoke my-exec-sql-tool with dryRun",
			api:           "http://127.0.0.1:5000/api/tool/my-exec-sql-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"sql":"SELECT 1", "dry_run": true}`)),
			want:          `\"statementType\": \"SELECT\"`,
			isErr:         false,
		},
		{
			name:          "invoke my-exec-sql-tool with dryRun create table",
			api:           "http://127.0.0.1:5000/api/tool/my-exec-sql-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf(`{"sql":"CREATE TABLE %s (id INT64, name STRING)", "dry_run": true}`, newTableName))),
			want:          `\"statementType\": \"CREATE_TABLE\"`,
			isErr:         false,
		},
		{
			name:          "invoke my-exec-sql-tool with dryRun execute immediate",
			api:           "http://127.0.0.1:5000/api/tool/my-exec-sql-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf(`{"sql":"EXECUTE IMMEDIATE \"CREATE TABLE %s (id INT64, name STRING)\"", "dry_run": true}`, newTableName))),
			want:          `\"statementType\": \"SCRIPT\"`,
			isErr:         false,
		},
		{
			name:          "Invoke my-auth-exec-sql-tool with dryRun and auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-exec-sql-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(`{"sql":"SELECT 1", "dry_run": true}`)),
			isErr:         false,
			want:          `\"statementType\": \"SELECT\"`,
		},
		{
			name:          "Invoke my-auth-exec-sql-tool with dryRun and invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-exec-sql-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": "INVALID_TOKEN"},
			requestBody:   bytes.NewBuffer([]byte(`{"sql":"SELECT 1","dry_run": true}`)),
			isErr:         true,
		},
		{
			name:          "Invoke my-auth-exec-sql-tool with dryRun and without auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-exec-sql-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"sql":"SELECT 1", "dry_run": true}`)),
			isErr:         true,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			// Send Tool invocation request
			req, err := http.NewRequest(http.MethodPost, tc.api, tc.requestBody)
			if err != nil {
				t.Fatalf("unable to create request: %s", err)
			}
			req.Header.Add("Content-type", "application/json")
			for k, v := range tc.requestHeader {
				req.Header.Add(k, v)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("unable to send request: %s", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				if tc.isErr {
					return
				}
				bodyBytes, _ := io.ReadAll(resp.Body)
				t.Fatalf("response status code is not 200, got %d: %s", resp.StatusCode, string(bodyBytes))
			}

			// Check response body
			var body map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&body)
			if err != nil {
				t.Fatalf("error parsing response body")
			}

			got, ok := body["result"].(string)
			if !ok {
				t.Fatalf("unable to find result in response body")
			}

			if !strings.Contains(got, tc.want) {
				t.Fatalf("expected %q to contain %q, but it did not", got, tc.want)
			}
		})
	}
}

func runBigQueryForecastToolInvokeTest(t *testing.T, tableName string) {
	idToken, err := tests.GetGoogleIdToken(tests.ClientId)
	if err != nil {
		t.Fatalf("error getting Google ID token: %s", err)
	}

	// Get access token
	accessToken, err := sources.GetIAMAccessToken(t.Context())
	if err != nil {
		t.Fatalf("error getting access token from ADC: %s", err)
	}
	accessToken = "Bearer " + accessToken

	historyDataTable := strings.ReplaceAll(tableName, "`", "")
	historyDataQuery := fmt.Sprintf("SELECT ts, data, id FROM %s", tableName)

	invokeTcs := []struct {
		name          string
		api           string
		requestHeader map[string]string
		requestBody   io.Reader
		want          string
		isErr         bool
	}{
		{
			name:          "invoke my-forecast-tool without required params",
			api:           "http://127.0.0.1:5000/api/tool/my-forecast-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf(`{"history_data": "%s"}`, historyDataTable))),
			isErr:         true,
		},
		{
			name:          "invoke my-forecast-tool with table",
			api:           "http://127.0.0.1:5000/api/tool/my-forecast-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf(`{"history_data": "%s", "timestamp_col": "ts", "data_col": "data"}`, historyDataTable))),
			want:          `"forecast_timestamp"`,
			isErr:         false,
		},
		{
			name:          "invoke my-forecast-tool with query and horizon",
			api:           "http://127.0.0.1:5000/api/tool/my-forecast-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf(`{"history_data": "%s", "timestamp_col": "ts", "data_col": "data", "horizon": 5}`, historyDataQuery))),
			want:          `"forecast_timestamp"`,
			isErr:         false,
		},
		{
			name:          "invoke my-forecast-tool with id_cols",
			api:           "http://127.0.0.1:5000/api/tool/my-forecast-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf(`{"history_data": "%s", "timestamp_col": "ts", "data_col": "data", "id_cols": ["id"]}`, historyDataTable))),
			want:          `"id"`,
			isErr:         false,
		},
		{
			name:          "invoke my-auth-forecast-tool with auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-forecast-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf(`{"history_data": "%s", "timestamp_col": "ts", "data_col": "data"}`, historyDataTable))),
			want:          `"forecast_timestamp"`,
			isErr:         false,
		},
		{
			name:          "invoke my-auth-forecast-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-forecast-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": "INVALID_TOKEN"},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf(`{"history_data": "%s", "timestamp_col": "ts", "data_col": "data"}`, historyDataTable))),
			isErr:         true,
		},
		{
			name:          "Invoke my-client-auth-forecast-tool with auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-forecast-tool/invoke",
			requestHeader: map[string]string{"Authorization": accessToken},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf(`{"history_data": "%s", "timestamp_col": "ts", "data_col": "data"}`, historyDataTable))),
			want:          `"forecast_timestamp"`,
			isErr:         false,
		},
		{
			name:          "Invoke my-client-auth-forecast-tool without auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-forecast-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf(`{"history_data": "%s", "timestamp_col": "ts", "data_col": "data"}`, historyDataTable))),
			isErr:         true,
		},
		{

			name:          "Invoke my-client-auth-forecast-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-forecast-tool/invoke",
			requestHeader: map[string]string{"Authorization": "Bearer invalid-token"},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf(`{"history_data": "%s", "timestamp_col": "ts", "data_col": "data"}`, historyDataTable))),
			isErr:         true,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			// Send Tool invocation request
			req, err := http.NewRequest(http.MethodPost, tc.api, tc.requestBody)
			if err != nil {
				t.Fatalf("unable to create request: %s", err)
			}
			req.Header.Add("Content-type", "application/json")
			for k, v := range tc.requestHeader {
				req.Header.Add(k, v)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("unable to send request: %s", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				if tc.isErr {
					return
				}
				bodyBytes, _ := io.ReadAll(resp.Body)
				t.Fatalf("response status code is not 200, got %d: %s", resp.StatusCode, string(bodyBytes))
			}

			// Check response body
			var body map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&body)
			if err != nil {
				t.Fatalf("error parsing response body")
			}

			got, ok := body["result"].(string)
			if !ok {
				t.Fatalf("unable to find result in response body")
			}

			if !strings.Contains(got, tc.want) {
				t.Fatalf("expected %q to contain %q, but it did not", got, tc.want)
			}
		})
	}
}

func runBigQueryAnalyzeContributionToolInvokeTest(t *testing.T, tableName string) {
	idToken, err := tests.GetGoogleIdToken(tests.ClientId)
	if err != nil {
		t.Fatalf("error getting Google ID token: %s", err)
	}

	// Get access token
	accessToken, err := sources.GetIAMAccessToken(t.Context())
	if err != nil {
		t.Fatalf("error getting access token from ADC: %s", err)
	}
	accessToken = "Bearer " + accessToken

	dataTable := strings.ReplaceAll(tableName, "`", "")

	invokeTcs := []struct {
		name          string
		api           string
		requestHeader map[string]string
		requestBody   io.Reader
		want          string
		isErr         bool
	}{
		{
			name:          "invoke my-analyze-contribution-tool without required params",
			api:           "http://127.0.0.1:5000/api/tool/my-analyze-contribution-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf(`{"input_data": "%s"}`, dataTable))),
			isErr:         true,
		},
		{
			name:          "invoke my-analyze-contribution-tool with table",
			api:           "http://127.0.0.1:5000/api/tool/my-analyze-contribution-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf(`{"input_data": "%s", "contribution_metric": "SUM(metric)", "is_test_col": "is_test", "dimension_id_cols": ["dim1", "dim2"]}`, dataTable))),
			want:          `"relative_difference"`,
			isErr:         false,
		},
		{
			name:          "invoke my-auth-analyze-contribution-tool with auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-analyze-contribution-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf(`{"input_data": "%s", "contribution_metric": "SUM(metric)", "is_test_col": "is_test", "dimension_id_cols": ["dim1", "dim2"]}`, dataTable))),
			want:          `"relative_difference"`,
			isErr:         false,
		},
		{
			name:          "invoke my-auth-analyze-contribution-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-analyze-contribution-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": "INVALID_TOKEN"},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf(`{"input_data": "%s", "contribution_metric": "SUM(metric)", "is_test_col": "is_test", "dimension_id_cols": ["dim1", "dim2"]}`, dataTable))),
			isErr:         true,
		},
		{
			name:          "Invoke my-client-auth-analyze-contribution-tool with auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-analyze-contribution-tool/invoke",
			requestHeader: map[string]string{"Authorization": accessToken},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf(`{"input_data": "%s", "contribution_metric": "SUM(metric)", "is_test_col": "is_test", "dimension_id_cols": ["dim1", "dim2"]}`, dataTable))),
			want:          `"relative_difference"`,
			isErr:         false,
		},
		{
			name:          "Invoke my-client-auth-analyze-contribution-tool without auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-analyze-contribution-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf(`{"input_data": "%s", "contribution_metric": "SUM(metric)", "is_test_col": "is_test", "dimension_id_cols": ["dim1", "dim2"]}`, dataTable))),
			isErr:         true,
		},
		{

			name:          "Invoke my-client-auth-analyze-contribution-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-analyze-contribution-tool/invoke",
			requestHeader: map[string]string{"Authorization": "Bearer invalid-token"},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf(`{"input_data": "%s", "contribution_metric": "SUM(metric)", "is_test_col": "is_test", "dimension_id_cols": ["dim1", "dim2"]}`, dataTable))),
			isErr:         true,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			// Send Tool invocation request
			req, err := http.NewRequest(http.MethodPost, tc.api, tc.requestBody)
			if err != nil {
				t.Fatalf("unable to create request: %s", err)
			}
			req.Header.Add("Content-type", "application/json")
			for k, v := range tc.requestHeader {
				req.Header.Add(k, v)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("unable to send request: %s", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				if tc.isErr {
					return
				}
				bodyBytes, _ := io.ReadAll(resp.Body)
				t.Fatalf("response status code is not 200, got %d: %s", resp.StatusCode, string(bodyBytes))
			}

			// Check response body
			var body map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&body)
			if err != nil {
				t.Fatalf("error parsing response body")
			}

			got, ok := body["result"].(string)
			if !ok {
				t.Fatalf("unable to find result in response body")
			}

			if !strings.Contains(got, tc.want) {
				t.Fatalf("expected %q to contain %q, but it did not", got, tc.want)
			}
		})
	}
}

func runBigQueryDataTypeTests(t *testing.T) {
	// Test tool invoke endpoint
	invokeTcs := []struct {
		name          string
		api           string
		requestHeader map[string]string
		requestBody   io.Reader
		want          string
		isErr         bool
	}{
		{
			name:          "invoke my-scalar-datatype-tool with values",
			api:           "http://127.0.0.1:5000/api/tool/my-scalar-datatype-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"int_val": 123, "string_val": "hello", "float_val": 3.14, "bool_val": true}`)),
			want:          `[{"bool_val":true,"float_val":3.14,"id":1,"int_val":123,"string_val":"hello"}]`,
			isErr:         false,
		},
		{
			name:          "invoke my-scalar-datatype-tool with missing params",
			api:           "http://127.0.0.1:5000/api/tool/my-scalar-datatype-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"int_val": 123}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-array-datatype-tool",
			api:           "http://127.0.0.1:5000/api/tool/my-array-datatype-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"int_array": [123, 789], "string_array": ["hello", "test"], "float_array": [3.14, 100.1], "bool_array": [true]}`)),
			want:          `[{"bool_val":true,"float_val":3.14,"id":1,"int_val":123,"string_val":"hello"},{"bool_val":true,"float_val":100.1,"id":3,"int_val":789,"string_val":"test"}]`,
			isErr:         false,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			// Send Tool invocation request
			req, err := http.NewRequest(http.MethodPost, tc.api, tc.requestBody)
			if err != nil {
				t.Fatalf("unable to create request: %s", err)
			}
			req.Header.Add("Content-type", "application/json")
			for k, v := range tc.requestHeader {
				req.Header.Add(k, v)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("unable to send request: %s", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				if tc.isErr {
					return
				}
				bodyBytes, _ := io.ReadAll(resp.Body)
				t.Fatalf("response status code is not 200, got %d: %s", resp.StatusCode, string(bodyBytes))
			}

			// Check response body
			var body map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&body)
			if err != nil {
				t.Fatalf("error parsing response body")
			}

			got, ok := body["result"].(string)
			if !ok {
				t.Fatalf("unable to find result in response body")
			}

			if got != tc.want {
				t.Fatalf("unexpected value: got %q, want %q", got, tc.want)
			}
		})
	}
}

func runBigQueryListDatasetToolInvokeTest(t *testing.T, datasetWant string) {
	// Get ID token
	idToken, err := tests.GetGoogleIdToken(tests.ClientId)
	if err != nil {
		t.Fatalf("error getting Google ID token: %s", err)
	}

	// Get access token
	accessToken, err := sources.GetIAMAccessToken(t.Context())
	if err != nil {
		t.Fatalf("error getting access token from ADC: %s", err)
	}
	accessToken = "Bearer " + accessToken

	// Test tool invoke endpoint
	invokeTcs := []struct {
		name          string
		api           string
		requestHeader map[string]string
		requestBody   io.Reader
		want          string
		isErr         bool
	}{
		{
			name:          "invoke my-list-dataset-ids-tool",
			api:           "http://127.0.0.1:5000/api/tool/my-list-dataset-ids-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			isErr:         false,
			want:          datasetWant,
		},
		{
			name:          "invoke my-list-dataset-ids-tool with project",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-list-dataset-ids-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf("{\"project\":\"%s\"}", BigqueryProject))),
			isErr:         false,
			want:          datasetWant,
		},
		{
			name:          "invoke my-list-dataset-ids-tool with non-existent project",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-list-dataset-ids-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf("{\"project\":\"%s-%s\"}", BigqueryProject, uuid.NewString()))),
			isErr:         true,
		},
		{
			name:          "invoke my-auth-list-dataset-ids-tool",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-list-dataset-ids-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			isErr:         false,
			want:          datasetWant,
		},
		{
			name:          "Invoke my-client-auth-list-dataset-ids-tool with auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-list-dataset-ids-tool/invoke",
			requestHeader: map[string]string{"Authorization": accessToken},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			isErr:         false,
			want:          datasetWant,
		},
		{
			name:          "Invoke my-client-auth-list-dataset-ids-tool without auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-list-dataset-ids-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			isErr:         true,
		},
		{

			name:          "Invoke my-client-auth-list-dataset-ids-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-list-dataset-ids-tool/invoke",
			requestHeader: map[string]string{"Authorization": "Bearer invalid-token"},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			isErr:         true,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			// Send Tool invocation request
			req, err := http.NewRequest(http.MethodPost, tc.api, tc.requestBody)
			if err != nil {
				t.Fatalf("unable to create request: %s", err)
			}
			req.Header.Add("Content-type", "application/json")
			for k, v := range tc.requestHeader {
				req.Header.Add(k, v)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("unable to send request: %s", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				if tc.isErr {
					return
				}
				bodyBytes, _ := io.ReadAll(resp.Body)
				t.Fatalf("response status code is not 200, got %d: %s", resp.StatusCode, string(bodyBytes))
			}

			// Check response body
			var body map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&body)
			if err != nil {
				t.Fatalf("error parsing response body")
			}

			got, ok := body["result"].(string)
			if !ok {
				t.Fatalf("unable to find result in response body")
			}

			if !strings.Contains(got, tc.want) {
				t.Fatalf("expected %q to contain %q, but it did not", got, tc.want)
			}
		})
	}
}

func runBigQueryGetDatasetInfoToolInvokeTest(t *testing.T, datasetName, datasetInfoWant string) {
	// Get ID token
	idToken, err := tests.GetGoogleIdToken(tests.ClientId)
	if err != nil {
		t.Fatalf("error getting Google ID token: %s", err)
	}

	// Get access token
	accessToken, err := sources.GetIAMAccessToken(t.Context())
	if err != nil {
		t.Fatalf("error getting access token from ADC: %s", err)
	}
	accessToken = "Bearer " + accessToken

	// Test tool invoke endpoint
	invokeTcs := []struct {
		name          string
		api           string
		requestHeader map[string]string
		requestBody   io.Reader
		want          string
		isErr         bool
	}{
		{
			name:          "invoke my-get-dataset-info-tool without body",
			api:           "http://127.0.0.1:5000/api/tool/my-get-dataset-info-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-get-dataset-info-tool",
			api:           "http://127.0.0.1:5000/api/tool/my-get-dataset-info-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf("{\"dataset\":\"%s\"}", datasetName))),
			want:          datasetInfoWant,
			isErr:         false,
		},
		{
			name:          "Invoke my-auth-get-dataset-info-tool with correct project",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-get-dataset-info-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf("{\"project\":\"%s\", \"dataset\":\"%s\"}", BigqueryProject, datasetName))),
			want:          datasetInfoWant,
			isErr:         false,
		},
		{
			name:          "Invoke my-auth-get-dataset-info-tool with non-existent project",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-get-dataset-info-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf("{\"project\":\"%s-%s\", \"dataset\":\"%s\"}", BigqueryProject, uuid.NewString(), datasetName))),
			isErr:         true,
		},
		{
			name:          "invoke my-auth-get-dataset-info-tool without body",
			api:           "http://127.0.0.1:5000/api/tool/my-get-dataset-info-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			isErr:         true,
		},
		{
			name:          "Invoke my-auth-get-dataset-info-tool with auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-get-dataset-info-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf("{\"dataset\":\"%s\"}", datasetName))),
			want:          datasetInfoWant,
			isErr:         false,
		},
		{
			name:          "Invoke my-auth-get-dataset-info-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-get-dataset-info-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": "INVALID_TOKEN"},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf("{\"dataset\":\"%s\"}", datasetName))),
			isErr:         true,
		},
		{
			name:          "Invoke my-auth-get-dataset-info-tool without auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-get-dataset-info-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf("{\"dataset\":\"%s\"}", datasetName))),
			isErr:         true,
		},
		{
			name:          "Invoke my-client-auth-get-dataset-info-tool with auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-get-dataset-info-tool/invoke",
			requestHeader: map[string]string{"Authorization": accessToken},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf("{\"dataset\":\"%s\"}", datasetName))),
			want:          datasetInfoWant,
			isErr:         false,
		},
		{
			name:          "Invoke my-client-auth-get-dataset-info-tool without auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-get-dataset-info-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf("{\"dataset\":\"%s\"}", datasetName))),
			isErr:         true,
		},
		{

			name:          "Invoke my-client-auth-get-dataset-info-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-get-dataset-info-tool/invoke",
			requestHeader: map[string]string{"Authorization": "Bearer invalid-token"},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf("{\"dataset\":\"%s\"}", datasetName))),
			isErr:         true,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			// Send Tool invocation request
			req, err := http.NewRequest(http.MethodPost, tc.api, tc.requestBody)
			if err != nil {
				t.Fatalf("unable to create request: %s", err)
			}
			req.Header.Add("Content-type", "application/json")
			for k, v := range tc.requestHeader {
				req.Header.Add(k, v)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("unable to send request: %s", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				if tc.isErr {
					return
				}
				bodyBytes, _ := io.ReadAll(resp.Body)
				t.Fatalf("response status code is not 200, got %d: %s", resp.StatusCode, string(bodyBytes))
			}

			// Check response body
			var body map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&body)
			if err != nil {
				t.Fatalf("error parsing response body")
			}

			got, ok := body["result"].(string)
			if !ok {
				t.Fatalf("unable to find result in response body")
			}

			if !strings.Contains(got, tc.want) {
				t.Fatalf("expected %q to contain %q, but it did not", got, tc.want)
			}
		})
	}
}

func runBigQueryListTableIdsToolInvokeTest(t *testing.T, datasetName, tablename_want string) {
	// Get ID token
	idToken, err := tests.GetGoogleIdToken(tests.ClientId)
	if err != nil {
		t.Fatalf("error getting Google ID token: %s", err)
	}

	// Get access token
	accessToken, err := sources.GetIAMAccessToken(t.Context())
	if err != nil {
		t.Fatalf("error getting access token from ADC: %s", err)
	}
	accessToken = "Bearer " + accessToken

	// Test tool invoke endpoint
	invokeTcs := []struct {
		name          string
		api           string
		requestHeader map[string]string
		requestBody   io.Reader
		want          string
		isErr         bool
	}{
		{
			name:          "invoke my-list-table-ids-tool without body",
			api:           "http://127.0.0.1:5000/api/tool/my-list-table-ids-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-list-table-ids-tool",
			api:           "http://127.0.0.1:5000/api/tool/my-list-table-ids-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf("{\"dataset\":\"%s\"}", datasetName))),
			want:          tablename_want,
			isErr:         false,
		},
		{
			name:          "invoke my-list-table-ids-tool without body",
			api:           "http://127.0.0.1:5000/api/tool/my-list-table-ids-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			isErr:         true,
		},
		{
			name:          "Invoke my-auth-list-table-ids-tool with auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-list-table-ids-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf("{\"dataset\":\"%s\"}", datasetName))),
			want:          tablename_want,
			isErr:         false,
		},
		{
			name:          "Invoke my-auth-list-table-ids-tool with correct project",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-list-table-ids-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf("{\"project\":\"%s\", \"dataset\":\"%s\"}", BigqueryProject, datasetName))),
			want:          tablename_want,
			isErr:         false,
		},
		{
			name:          "Invoke my-auth-list-table-ids-tool with non-existent project",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-list-table-ids-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf("{\"project\":\"%s-%s\", \"dataset\":\"%s\"}", BigqueryProject, uuid.NewString(), datasetName))),
			isErr:         true,
		},
		{
			name:          "Invoke my-auth-list-table-ids-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-list-table-ids-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": "INVALID_TOKEN"},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf("{\"dataset\":\"%s\"}", datasetName))),
			isErr:         true,
		},
		{
			name:          "Invoke my-auth-list-table-ids-tool without auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-list-table-ids-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf("{\"dataset\":\"%s\"}", datasetName))),
			isErr:         true,
		},
		{
			name:          "Invoke my-client-auth-list-table-ids-tool with auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-list-table-ids-tool/invoke",
			requestHeader: map[string]string{"Authorization": accessToken},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf("{\"dataset\":\"%s\"}", datasetName))),
			want:          tablename_want,
			isErr:         false,
		},
		{
			name:          "Invoke my-client-auth-list-table-ids-tool without auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-list-table-ids-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf("{\"dataset\":\"%s\"}", datasetName))),
			isErr:         true,
		},
		{

			name:          "Invoke my-client-auth-list-table-ids-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-list-table-ids-tool/invoke",
			requestHeader: map[string]string{"Authorization": "Bearer invalid-token"},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf("{\"dataset\":\"%s\"}", datasetName))),
			isErr:         true,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			// Send Tool invocation request
			req, err := http.NewRequest(http.MethodPost, tc.api, tc.requestBody)
			if err != nil {
				t.Fatalf("unable to create request: %s", err)
			}
			req.Header.Add("Content-type", "application/json")
			for k, v := range tc.requestHeader {
				req.Header.Add(k, v)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("unable to send request: %s", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				if tc.isErr {
					return
				}
				bodyBytes, _ := io.ReadAll(resp.Body)
				t.Fatalf("response status code is not 200, got %d: %s", resp.StatusCode, string(bodyBytes))
			}

			// Check response body
			var body map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&body)
			if err != nil {
				t.Fatalf("error parsing response body")
			}

			got, ok := body["result"].(string)
			if !ok {
				t.Fatalf("unable to find result in response body")
			}

			if !strings.Contains(got, tc.want) {
				t.Fatalf("expected %q to contain %q, but it did not", got, tc.want)
			}
		})
	}
}

func runBigQueryGetTableInfoToolInvokeTest(t *testing.T, datasetName, tableName, tableInfoWant string) {
	// Get ID token
	idToken, err := tests.GetGoogleIdToken(tests.ClientId)
	if err != nil {
		t.Fatalf("error getting Google ID token: %s", err)
	}

	// Get access token
	accessToken, err := sources.GetIAMAccessToken(t.Context())
	if err != nil {
		t.Fatalf("error getting access token from ADC: %s", err)
	}
	accessToken = "Bearer " + accessToken

	// Test tool invoke endpoint
	invokeTcs := []struct {
		name          string
		api           string
		requestHeader map[string]string
		requestBody   io.Reader
		want          string
		isErr         bool
	}{
		{
			name:          "invoke my-get-table-info-tool without body",
			api:           "http://127.0.0.1:5000/api/tool/my-get-table-info-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-get-table-info-tool",
			api:           "http://127.0.0.1:5000/api/tool/my-get-table-info-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf("{\"dataset\":\"%s\", \"table\":\"%s\"}", datasetName, tableName))),
			want:          tableInfoWant,
			isErr:         false,
		},
		{
			name:          "invoke my-auth-get-table-info-tool without body",
			api:           "http://127.0.0.1:5000/api/tool/my-get-table-info-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			isErr:         true,
		},
		{
			name:          "Invoke my-auth-get-table-info-tool with auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-get-table-info-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf("{\"dataset\":\"%s\", \"table\":\"%s\"}", datasetName, tableName))),
			want:          tableInfoWant,
			isErr:         false,
		},
		{
			name:          "Invoke my-auth-get-table-info-tool with correct project",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-get-table-info-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf("{\"project\":\"%s\", \"dataset\":\"%s\", \"table\":\"%s\"}", BigqueryProject, datasetName, tableName))),
			want:          tableInfoWant,
			isErr:         false,
		},
		{
			name:          "Invoke my-auth-get-table-info-tool with non-existent project",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-get-table-info-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf("{\"project\":\"%s-%s\", \"dataset\":\"%s\", \"table\":\"%s\"}", BigqueryProject, uuid.NewString(), datasetName, tableName))),
			isErr:         true,
		},
		{
			name:          "Invoke my-auth-get-table-info-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-get-table-info-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": "INVALID_TOKEN"},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf("{\"dataset\":\"%s\", \"table\":\"%s\"}", datasetName, tableName))),
			isErr:         true,
		},
		{
			name:          "Invoke my-auth-get-table-info-tool without auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-get-table-info-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf("{\"dataset\":\"%s\", \"table\":\"%s\"}", datasetName, tableName))),
			isErr:         true,
		},
		{
			name:          "Invoke my-client-auth-get-table-info-tool with auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-get-table-info-tool/invoke",
			requestHeader: map[string]string{"Authorization": accessToken},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf("{\"dataset\":\"%s\", \"table\":\"%s\"}", datasetName, tableName))),
			want:          tableInfoWant,
			isErr:         false,
		},
		{
			name:          "Invoke my-client-auth-get-table-info-tool without auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-get-table-info-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf("{\"dataset\":\"%s\", \"table\":\"%s\"}", datasetName, tableName))),
			isErr:         true,
		},
		{

			name:          "Invoke my-client-auth-get-table-info-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-get-table-info-tool/invoke",
			requestHeader: map[string]string{"Authorization": "Bearer invalid-token"},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf("{\"dataset\":\"%s\", \"table\":\"%s\"}", datasetName, tableName))),
			isErr:         true,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			// Send Tool invocation request
			req, err := http.NewRequest(http.MethodPost, tc.api, tc.requestBody)
			if err != nil {
				t.Fatalf("unable to create request: %s", err)
			}
			req.Header.Add("Content-type", "application/json")
			for k, v := range tc.requestHeader {
				req.Header.Add(k, v)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("unable to send request: %s", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				if tc.isErr {
					return
				}
				bodyBytes, _ := io.ReadAll(resp.Body)
				t.Fatalf("response status code is not 200, got %d: %s", resp.StatusCode, string(bodyBytes))
			}

			// Check response body
			var body map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&body)
			if err != nil {
				t.Fatalf("error parsing response body")
			}

			got, ok := body["result"].(string)
			if !ok {
				t.Fatalf("unable to find result in response body")
			}

			if !strings.Contains(got, tc.want) {
				t.Fatalf("expected %q to contain %q, but it did not", got, tc.want)
			}
		})
	}
}

func runBigQueryConversationalAnalyticsInvokeTest(t *testing.T, datasetName, tableName, dataInsightsWant string) {
	// Get ID token
	idToken, err := tests.GetGoogleIdToken(tests.ClientId)
	if err != nil {
		t.Fatalf("error getting Google ID token: %s", err)
	}

	// Get access token
	accessToken, err := sources.GetIAMAccessToken(t.Context())
	if err != nil {
		t.Fatalf("error getting access token from ADC: %s", err)
	}
	accessToken = "Bearer " + accessToken

	tableRefsJSON := fmt.Sprintf(`[{"projectId":"%s","datasetId":"%s","tableId":"%s"}]`, BigqueryProject, datasetName, tableName)

	invokeTcs := []struct {
		name          string
		api           string
		requestHeader map[string]string
		requestBody   io.Reader
		want          string
		isErr         bool
	}{
		{
			name:          "invoke my-conversational-analytics-tool successfully",
			api:           "http://127.0.0.1:5000/api/tool/my-conversational-analytics-tool/invoke",
			requestHeader: map[string]string{},
			requestBody: bytes.NewBuffer([]byte(fmt.Sprintf(
				`{"user_query_with_context": "What are the names in the table?", "table_references": %q}`,
				tableRefsJSON,
			))),
			want:  dataInsightsWant,
			isErr: false,
		},
		{
			name:          "invoke my-auth-conversational-analytics-tool with auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-conversational-analytics-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody: bytes.NewBuffer([]byte(fmt.Sprintf(
				`{"user_query_with_context": "What are the names in the table?", "table_references": %q}`,
				tableRefsJSON,
			))),
			want:  dataInsightsWant,
			isErr: false,
		},
		{
			name:          "invoke my-auth-conversational-analytics-tool without auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-conversational-analytics-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"user_query_with_context": "What are the names in the table?"}`)),
			isErr:         true,
		},
		{
			name:          "Invoke my-client-auth-conversational-analytics-tool with auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-conversational-analytics-tool/invoke",
			requestHeader: map[string]string{"Authorization": accessToken},
			requestBody: bytes.NewBuffer([]byte(fmt.Sprintf(
				`{"user_query_with_context": "What are the names in the table?", "table_references": %q}`,
				tableRefsJSON,
			))),
			want:  "[{\"f0_\":1}]",
			isErr: false,
		},
		{
			name:          "Invoke my-client-auth-conversational-analytics-tool without auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-conversational-analytics-tool/invoke",
			requestHeader: map[string]string{},
			requestBody: bytes.NewBuffer([]byte(fmt.Sprintf(
				`{"user_query_with_context": "What are the names in the table?", "table_references": %q}`,
				tableRefsJSON,
			))),
			isErr: true,
		},
		{

			name:          "Invoke my-client-auth-conversational-analytics-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-client-auth-conversational-analytics-tool/invoke",
			requestHeader: map[string]string{"Authorization": "Bearer invalid-token"},
			requestBody: bytes.NewBuffer([]byte(fmt.Sprintf(
				`{"user_query_with_context": "What are the names in the table?", "table_references": %q}`,
				tableRefsJSON,
			))),
			isErr: true,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			// Send Tool invocation request
			req, err := http.NewRequest(http.MethodPost, tc.api, tc.requestBody)
			if err != nil {
				t.Fatalf("unable to create request: %s", err)
			}
			req.Header.Add("Content-type", "application/json")
			for k, v := range tc.requestHeader {
				req.Header.Add(k, v)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("unable to send request: %s", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				if tc.isErr {
					return
				}
				bodyBytes, _ := io.ReadAll(resp.Body)
				t.Fatalf("response status code is not 200, got %d: %s", resp.StatusCode, string(bodyBytes))
			}

			var body map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&body)
			if err != nil {
				t.Fatalf("error parsing response body: %v", err)
			}

			got, ok := body["result"].(string)
			if !ok {
				t.Fatalf("unable to find result in response body")
			}

			wantPattern := regexp.MustCompile(tc.want)
			if !wantPattern.MatchString(got) {
				t.Fatalf("response did not match the expected pattern.\nFull response:\n%s", got)
			}
		})
	}
}

func runListTableIdsWithRestriction(t *testing.T, allowedDatasetName, disallowedDatasetName string, allowedTableNames ...string) {
	sort.Strings(allowedTableNames)
	var quotedNames []string
	for _, name := range allowedTableNames {
		quotedNames = append(quotedNames, fmt.Sprintf(`"%s"`, name))
	}
	wantResult := fmt.Sprintf(`[%s]`, strings.Join(quotedNames, ","))

	testCases := []struct {
		name           string
		dataset        string
		wantStatusCode int
		wantInResult   string
		wantInError    string
	}{
		{
			name:           "invoke on allowed dataset",
			dataset:        allowedDatasetName,
			wantStatusCode: http.StatusOK,
			wantInResult:   wantResult,
		},
		{
			name:           "invoke on disallowed dataset",
			dataset:        disallowedDatasetName,
			wantStatusCode: http.StatusBadRequest, // Or the specific error code returned
			wantInError:    fmt.Sprintf("access denied to dataset '%s'", disallowedDatasetName),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			body := bytes.NewBuffer([]byte(fmt.Sprintf(`{"dataset":"%s"}`, tc.dataset)))
			req, err := http.NewRequest(http.MethodPost, "http://127.0.0.1:5000/api/tool/list-table-ids-restricted/invoke", body)
			if err != nil {
				t.Fatalf("unable to create request: %s", err)
			}
			req.Header.Add("Content-type", "application/json")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("unable to send request: %s", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.wantStatusCode {
				bodyBytes, _ := io.ReadAll(resp.Body)
				t.Fatalf("unexpected status code: got %d, want %d. Body: %s", resp.StatusCode, tc.wantStatusCode, string(bodyBytes))
			}

			if tc.wantInResult != "" {
				var respBody map[string]interface{}
				if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
					t.Fatalf("error parsing response body: %v", err)
				}
				got, ok := respBody["result"].(string)
				if !ok {
					t.Fatalf("unable to find result in response body")
				}

				var gotSlice []string
				if err := json.Unmarshal([]byte(got), &gotSlice); err != nil {
					t.Fatalf("error unmarshalling result: %v", err)
				}
				sort.Strings(gotSlice)
				sortedGotBytes, err := json.Marshal(gotSlice)
				if err != nil {
					t.Fatalf("error marshalling sorted result: %v", err)
				}

				if string(sortedGotBytes) != tc.wantInResult {
					t.Errorf("unexpected result: got %q, want %q", string(sortedGotBytes), tc.wantInResult)
				}
			}

			if tc.wantInError != "" {
				bodyBytes, _ := io.ReadAll(resp.Body)
				if !strings.Contains(string(bodyBytes), tc.wantInError) {
					t.Errorf("unexpected error message: got %q, want to contain %q", string(bodyBytes), tc.wantInError)
				}
			}
		})
	}
}

func runBigQuerySearchCatalogToolInvokeTest(t *testing.T, datasetName string, tableName string) {
	// Get ID token
	idToken, err := tests.GetGoogleIdToken(tests.ClientId)
	if err != nil {
		t.Fatalf("error getting Google ID token: %s", err)
	}

	// Test tool invoke endpoint
	invokeTcs := []struct {
		name          string
		api           string
		requestHeader map[string]string
		requestBody   io.Reader
		wantKey       string
		isErr         bool
	}{
		{
			name:          "invoke my-search-catalog-tool without body",
			api:           "http://127.0.0.1:5000/api/tool/my-search-catalog-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-search-catalog-tool",
			api:           "http://127.0.0.1:5000/api/tool/my-search-catalog-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf("{\"prompt\":\"%s\", \"types\":[\"TABLE\"], \"datasetIds\":[\"%s\"]}", tableName, datasetName))),
			wantKey:       "DisplayName",
			isErr:         false,
		},
		{
			name:          "Invoke my-auth-search-catalog-tool with auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-search-catalog-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf("{\"prompt\":\"%s\", \"types\":[\"TABLE\"], \"datasetIds\":[\"%s\"]}", tableName, datasetName))),
			wantKey:       "DisplayName",
			isErr:         false,
		},
		{
			name:          "Invoke my-auth-search-catalog-tool with correct project",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-search-catalog-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf("{\"prompt\":\"%s\", \"types\":[\"TABLE\"], \"projectIds\":[\"%s\"], \"datasetIds\":[\"%s\"]}", tableName, BigqueryProject, datasetName))),
			wantKey:       "DisplayName",
			isErr:         false,
		},
		{
			name:          "Invoke my-auth-search-catalog-tool with non-existent project",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-search-catalog-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf("{\"prompt\":\"%s\", \"types\":[\"TABLE\"], \"projectIds\":[\"%s-%s\"], \"datasetIds\":[\"%s\"]}", tableName, BigqueryProject, uuid.NewString(), datasetName))),
			isErr:         true,
		},
		{
			name:          "Invoke my-auth-search-catalog-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-search-catalog-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": "INVALID_TOKEN"},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf("{\"prompt\":\"%s\", \"types\":[\"TABLE\"], \"datasetIds\":[\"%s\"]}", tableName, datasetName))),
			isErr:         true,
		},
		{
			name:          "Invoke my-auth-search-catalog-tool without auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-search-catalog-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf("{\"prompt\":\"%s\", \"types\":[\"TABLE\"], \"datasetIds\":[\"%s\"]}", tableName, datasetName))),
			isErr:         true,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			// Send Tool invocation request
			req, err := http.NewRequest(http.MethodPost, tc.api, tc.requestBody)
			if err != nil {
				t.Fatalf("unable to create request: %s", err)
			}
			req.Header.Add("Content-type", "application/json")
			for k, v := range tc.requestHeader {
				req.Header.Add(k, v)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("unable to send request: %s", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				if tc.isErr {
					return
				}
				bodyBytes, _ := io.ReadAll(resp.Body)
				t.Fatalf("response status code is not 200, got %d: %s", resp.StatusCode, string(bodyBytes))
			}

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				t.Fatalf("error parsing response body: %s", err)
			}
			resultStr, ok := result["result"].(string)
			if !ok {
				if result["result"] == nil && tc.isErr {
					return
				}
				t.Fatalf("expected 'result' field to be a string, got %T", result["result"])
			}
			if tc.isErr && (resultStr == "" || resultStr == "[]") {
				return
			}
			var entries []interface{}
			if err := json.Unmarshal([]byte(resultStr), &entries); err != nil {
				t.Fatalf("error unmarshalling result string: %v", err)
			}

			if !tc.isErr {
				if len(entries) != 1 {
					t.Fatalf("expected exactly one entry, but got %d", len(entries))
				}
				entry, ok := entries[0].(map[string]interface{})
				if !ok {
					t.Fatalf("expected first entry to be a map, got %T", entries[0])
				}
				respTable, ok := entry[tc.wantKey]
				if !ok {
					t.Fatalf("expected entry to have key '%s', but it was not found in %v", tc.wantKey, entry)
				}
				if respTable != tableName {
					t.Fatalf("expected key '%s' to have value '%s', but got %s", tc.wantKey, tableName, respTable)
				}
			} else {
				if len(entries) != 0 {
					t.Fatalf("expected 0 entries, but got %d", len(entries))
				}
			}
		})
	}
}
