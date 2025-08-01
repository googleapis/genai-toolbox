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
	"strings"
	"testing"
	"time"

	bigqueryapi "cloud.google.com/go/bigquery"
	"github.com/google/uuid"
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
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
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

	// Write config into a file and pass it to command
	toolsFile := tests.GetToolsConfig(sourceConfig, BigqueryToolKind, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, authToolStmt)
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

	tests.RunToolGetTest(t)

	select1Want := "[{\"f0_\":1}]"
	// Partial message; the full error message is too long.
	failInvocationWant := `{"jsonrpc":"2.0","id":"invoke-fail-tool","result":{"content":[{"type":"text","text":"final query validation failed: failed to insert dry run job: googleapi: Error 400: Syntax error: Unexpected identifier \"SELEC\" at [1:1]`
	datasetInfoWant := "\"Location\":\"US\",\"DefaultTableExpiration\":0,\"Labels\":null,\"Access\":"
	tableInfoWant := "{\"Name\":\"\",\"Location\":\"US\",\"Description\":\"\",\"Schema\":[{\"Name\":\"id\""
	ddlWant := `"Query executed successfully and returned no content."`
	invokeParamWant, invokeIdNullWant, nullWant, mcpInvokeParamWant := tests.GetNonSpannerInvokeParamWant()
	tests.RunToolInvokeTest(t, select1Want, invokeParamWant, invokeIdNullWant, nullWant, false, true)
	tests.RunMCPToolCallMethod(t, mcpInvokeParamWant, failInvocationWant)
	templateParamTestConfig := tests.NewTemplateParameterTestConfig(
		tests.WithCreateColArray(`["id INT64", "name STRING", "age INT64"]`),
		tests.WithDdlWant(ddlWant),
		tests.WithSelectEmptyWant(`"The query returned 0 rows."`),
		tests.WithInsert1Want(ddlWant),
	)
	tests.RunToolInvokeWithTemplateParameters(t, tableNameTemplateParam, templateParamTestConfig)

	runBigQueryExecuteSqlToolInvokeTest(t, select1Want, invokeParamWant, tableNameParam)
	runBigQueryDataTypeTests(t)
	runBigQueryListDatasetToolInvokeTest(t, datasetName)
	runBigQueryGetDatasetInfoToolInvokeTest(t, datasetName, datasetInfoWant)
	runBigQueryListTableIdsToolInvokeTest(t, datasetName, tableName)
	runBigQueryGetTableInfoToolInvokeTest(t, datasetName, tableName, tableInfoWant)
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
	config["tools"] = tools
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

	config["tools"] = tools
	return config
}

func runBigQueryExecuteSqlToolInvokeTest(t *testing.T, select1Want, invokeParamWant, tableNameParam string) {
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
			want:          `"Operation completed successfully."`,
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
			want:          `"Operation completed successfully."`,
			isErr:         true,
		},
		{
			name:          "invoke my-exec-sql-tool insert entry",
			api:           "http://127.0.0.1:5000/api/tool/my-exec-sql-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf("{\"sql\":\"INSERT INTO %s (id, name) VALUES (4, 'test_name')\"}", tableNameParam))),
			want:          `"Operation completed successfully."`,
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
