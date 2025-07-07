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

package spanner

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

	"cloud.google.com/go/spanner"
	database "cloud.google.com/go/spanner/admin/database/apiv1"
	"cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	"github.com/google/uuid"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/tests"
)

var (
	SpannerSourceKind = "spanner"
	SpannerToolKind   = "spanner-sql"
	SpannerProject    = os.Getenv("SPANNER_PROJECT")
	SpannerDatabase   = os.Getenv("SPANNER_DATABASE")
	SpannerInstance   = os.Getenv("SPANNER_INSTANCE")
)

func getSpannerVars(t *testing.T) map[string]any {
	switch "" {
	case SpannerProject:
		t.Fatal("'SPANNER_PROJECT' not set")
	case SpannerDatabase:
		t.Fatal("'SPANNER_DATABASE' not set")
	case SpannerInstance:
		t.Fatal("'SPANNER_INSTANCE' not set")
	}

	return map[string]any{
		"kind":     SpannerSourceKind,
		"project":  SpannerProject,
		"instance": SpannerInstance,
		"database": SpannerDatabase,
	}
}

func initSpannerClients(ctx context.Context, project, instance, dbname string) (*spanner.Client, *database.DatabaseAdminClient, error) {
	// Configure the connection to the database
	db := fmt.Sprintf("projects/%s/instances/%s/databases/%s", project, instance, dbname)

	// Configure session pool to automatically clean inactive transactions
	sessionPoolConfig := spanner.SessionPoolConfig{
		TrackSessionHandles: true,
		InactiveTransactionRemovalOptions: spanner.InactiveTransactionRemovalOptions{
			ActionOnInactiveTransaction: spanner.WarnAndClose,
		},
	}

	// Create Spanner client (for queries)
	dataClient, err := spanner.NewClientWithConfig(context.Background(), db, spanner.ClientConfig{SessionPoolConfig: sessionPoolConfig})
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create new Spanner client: %w", err)
	}

	// Create Spanner admin client (for creating databases)
	adminClient, err := database.NewDatabaseAdminClient(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create new Spanner admin client: %w", err)
	}

	return dataClient, adminClient, nil
}

func TestSpannerToolEndpoints(t *testing.T) {
	sourceConfig := getSpannerVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	var args []string

	// Create Spanner client
	dataClient, adminClient, err := initSpannerClients(ctx, SpannerProject, SpannerInstance, SpannerDatabase)
	if err != nil {
		t.Fatalf("unable to create Spanner client: %s", err)
	}

	// create table name with UUID
	tableNameParam := "param_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableNameAuth := "auth_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableNameTemplateParam := "template_param_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")

	// set up data for param tool
	createStatement1, insertStatement1, paramToolStatement1, paramToolStatement2, params1 := getSpannerParamToolInfo(tableNameParam)
	dbString := fmt.Sprintf(
		"projects/%s/instances/%s/databases/%s",
		SpannerProject,
		SpannerInstance,
		SpannerDatabase,
	)
	teardownTable1 := setupSpannerTable(t, ctx, adminClient, dataClient, createStatement1, insertStatement1, tableNameParam, dbString, params1)
	defer teardownTable1(t)

	// set up data for auth tool
	createStatement2, insertStatement2, authToolStatement, params2 := getSpannerAuthToolInfo(tableNameAuth)
	teardownTable2 := setupSpannerTable(t, ctx, adminClient, dataClient, createStatement2, insertStatement2, tableNameAuth, dbString, params2)
	defer teardownTable2(t)

	// set up data for template param tool
	createStatementTmpl := fmt.Sprintf("CREATE TABLE %s (id INT64, name STRING(MAX), age INT64) PRIMARY KEY (id)", tableNameTemplateParam)
	teardownTableTmpl := setupSpannerTable(t, ctx, adminClient, dataClient, createStatementTmpl, "", tableNameTemplateParam, dbString, nil)
	defer teardownTableTmpl(t)

	// Write config into a file and pass it to command
	toolsFile := tests.GetToolsConfig(sourceConfig, SpannerToolKind, paramToolStatement1, paramToolStatement2, authToolStatement)
	toolsFile = addSpannerExecuteSqlConfig(t, toolsFile)
	toolsFile = addSpannerReadOnlyConfig(t, toolsFile)
	toolsFile = addTemplateParamConfig(t, toolsFile)

	cmd, cleanup, err := tests.StartCmd(ctx, toolsFile, args...)
	if err != nil {
		t.Fatalf("command initialization returned an error: %s", err)
	}
	defer cleanup()

	waitCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	out, err := cmd.WaitForString(waitCtx, regexp.MustCompile(`Server ready to serve`))
	if err != nil {
		t.Logf("toolbox command logs: \n%s", out)
		t.Fatalf("toolbox didn't start successfully: %s", err)
	}

	tests.RunToolGetTest(t)

	select1Want := "[{\"\":\"1\"}]"
	accessSchemaWant := "[{\"schema_name\":\"INFORMATION_SCHEMA\"}]"
	invokeParamWant := "[{\"id\":\"1\",\"name\":\"Alice\"},{\"id\":\"3\",\"name\":\"Sid\"}]"
	invokeParamWantNull := `[{"id":"4","name":null}]`
	mcpInvokeParamWant := `{"jsonrpc":"2.0","id":"my-param-tool","result":{"content":[{"type":"text","text":"{\"id\":\"1\",\"name\":\"Alice\"}"},{"type":"text","text":"{\"id\":\"3\",\"name\":\"Sid\"}"}]}}`
	failInvocationWant := `"jsonrpc":"2.0","id":"invoke-fail-tool","result":{"content":[{"type":"text","text":"unable to execute client: unable to parse row: spanner: code = \"InvalidArgument\", desc = \"Syntax error: Unexpected identifier \\\\\\\"SELEC\\\\\\\" [at 1:1]\\\\nSELEC 1;\\\\n^\"`

	tests.RunToolInvokeTest(t, select1Want, invokeParamWant, invokeParamWantNull)
	tests.RunMCPToolCallMethod(t, mcpInvokeParamWant, failInvocationWant)
	runSpannerSchemaToolInvokeTest(t, accessSchemaWant)
	runSpannerExecuteSqlToolInvokeTest(t, select1Want, invokeParamWant, tableNameParam, tableNameAuth)

	templateParamTestConfig := tests.NewTemplateParameterTestConfig(
		tests.WithIgnoreDdl(),
		tests.WithSelectAllWant("[{\"age\":\"21\",\"id\":\"1\",\"name\":\"Alex\"},{\"age\":\"100\",\"id\":\"2\",\"name\":\"Alice\"}]"),
		tests.WithSelect1Want("[{\"age\":\"21\",\"id\":\"1\",\"name\":\"Alex\"}]"),
	)
	tests.RunToolInvokeWithTemplateParameters(t, tableNameTemplateParam, templateParamTestConfig)
}

// getSpannerToolInfo returns statements and param for my-param-tool for spanner-sql kind
func getSpannerParamToolInfo(tableName string) (string, string, string, string, map[string]any) {
	createStatement := fmt.Sprintf("CREATE TABLE %s (id INT64, name STRING(MAX)) PRIMARY KEY (id)", tableName)
	insertStatement := fmt.Sprintf("INSERT INTO %s (id, name) VALUES (1, @name1), (2, @name2), (3, @name3), (4, @name4)", tableName)
	toolStatement := fmt.Sprintf("SELECT * FROM %s WHERE id = @id OR name = @name", tableName)
	toolStatement2 := fmt.Sprintf("SELECT * FROM %s WHERE id = @id", tableName)
	params := map[string]any{"name1": "Alice", "name2": "Jane", "name3": "Sid", "name4": nil}
	return createStatement, insertStatement, toolStatement, toolStatement2, params
}

// getSpannerAuthToolInfo returns statements and param of my-auth-tool for spanner-sql kind
func getSpannerAuthToolInfo(tableName string) (string, string, string, map[string]any) {
	createStatement := fmt.Sprintf("CREATE TABLE %s (id INT64, name STRING(MAX), email STRING(MAX)) PRIMARY KEY (id)", tableName)
	insertStatement := fmt.Sprintf("INSERT INTO %s (id, name, email) VALUES (1, @name1, @email1), (2, @name2, @email2)", tableName)
	toolStatement := fmt.Sprintf("SELECT name FROM %s WHERE email = @email", tableName)
	params := map[string]any{
		"name1":  "Alice",
		"email1": tests.ServiceAccountEmail,
		"name2":  "Jane",
		"email2": "janedoe@gmail.com",
	}
	return createStatement, insertStatement, toolStatement, params
}

// setupSpannerTable creates and inserts data into a table of tool
// compatible with spanner-sql tool
func setupSpannerTable(t *testing.T, ctx context.Context, adminClient *database.DatabaseAdminClient, dataClient *spanner.Client, createStatement, insertStatement, tableName, dbString string, params map[string]any) func(*testing.T) {

	// Create table
	op, err := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
		Database:   dbString,
		Statements: []string{createStatement},
	})
	if err != nil {
		t.Fatalf("unable to start create table operation %s: %s", tableName, err)
	}
	err = op.Wait(ctx)
	if err != nil {
		t.Fatalf("unable to create test table %s: %s", tableName, err)
	}

	// Insert test data
	if insertStatement != "" {
		_, err = dataClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			stmt := spanner.Statement{
				SQL:    insertStatement,
				Params: params,
			}
			_, err := txn.Update(ctx, stmt)
			return err
		})
		if err != nil {
			t.Fatalf("unable to insert test data: %s", err)
		}
	}

	return func(t *testing.T) {
		// tear down test
		op, err = adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
			Database:   dbString,
			Statements: []string{fmt.Sprintf("DROP TABLE %s", tableName)},
		})
		if err != nil {
			t.Errorf("unable to start drop table operation: %s", err)
			return
		}

		err = op.Wait(ctx)
		if err != nil {
			t.Errorf("Teardown failed: %s", err)
		}
	}
}

// addSpannerExecuteSqlConfig gets the tools config for `spanner-execute-sql`
func addSpannerExecuteSqlConfig(t *testing.T, config map[string]any) map[string]any {
	tools, ok := config["tools"].(map[string]any)
	if !ok {
		t.Fatalf("unable to get tools from config")
	}
	tools["my-exec-sql-tool-read-only"] = map[string]any{
		"kind":        "spanner-execute-sql",
		"source":      "my-instance",
		"description": "Tool to execute sql",
		"readOnly":    true,
	}
	tools["my-exec-sql-tool"] = map[string]any{
		"kind":        "spanner-execute-sql",
		"source":      "my-instance",
		"description": "Tool to execute sql",
	}
	tools["my-auth-exec-sql-tool"] = map[string]any{
		"kind":        "spanner-execute-sql",
		"source":      "my-instance",
		"description": "Tool to execute sql",
		"authRequired": []string{
			"my-google-auth",
		},
	}
	config["tools"] = tools
	return config
}

func addSpannerReadOnlyConfig(t *testing.T, config map[string]any) map[string]any {
	tools, ok := config["tools"].(map[string]any)
	if !ok {
		t.Fatalf("unable to get tools from config")
	}
	tools["access-schema-read-only"] = map[string]any{
		"kind":        "spanner-sql",
		"source":      "my-instance",
		"description": "Tool to access information schema in read-only mode.",
		"statement":   "SELECT schema_name FROM `INFORMATION_SCHEMA`.SCHEMATA WHERE schema_name='INFORMATION_SCHEMA';",
		"readOnly":    true,
	}
	tools["access-schema"] = map[string]any{
		"kind":        "spanner-sql",
		"source":      "my-instance",
		"description": "Tool to access information schema.",
		"statement":   "SELECT schema_name FROM `INFORMATION_SCHEMA`.SCHEMATA WHERE schema_name='INFORMATION_SCHEMA';",
	}
	config["tools"] = tools
	return config
}

func addTemplateParamConfig(t *testing.T, config map[string]any) map[string]any {
	toolsMap, ok := config["tools"].(map[string]any)
	if !ok {
		t.Fatalf("unable to get tools from config")
	}
	toolsMap["insert-table-templateParams-tool"] = map[string]any{
		"kind":        "spanner-sql",
		"source":      "my-instance",
		"description": "Insert tool with template parameters",
		"statement":   "INSERT INTO {{.tableName}} ({{array .columns}}) VALUES ({{.values}})",
		"templateParameters": []tools.Parameter{
			tools.NewStringParameter("tableName", "some description"),
			tools.NewArrayParameter("columns", "The columns to insert into", tools.NewStringParameter("column", "A column name that will be returned from the query.")),
			tools.NewStringParameter("values", "The values to insert as a comma separated string"),
		},
	}
	toolsMap["select-templateParams-tool"] = map[string]any{
		"kind":        "spanner-sql",
		"source":      "my-instance",
		"description": "Create table tool with template parameters",
		"statement":   "SELECT * FROM {{.tableName}}",
		"templateParameters": []tools.Parameter{
			tools.NewStringParameter("tableName", "some description"),
		},
	}
	toolsMap["select-templateParams-combined-tool"] = map[string]any{
		"kind":        "spanner-sql",
		"source":      "my-instance",
		"description": "Create table tool with template parameters",
		"statement":   "SELECT * FROM {{.tableName}} WHERE id = @id",
		"parameters":  []tools.Parameter{tools.NewIntParameter("id", "the id of the user")},
		"templateParameters": []tools.Parameter{
			tools.NewStringParameter("tableName", "some description"),
		},
	}
	toolsMap["select-fields-templateParams-tool"] = map[string]any{
		"kind":        "spanner-sql",
		"source":      "my-instance",
		"description": "Create table tool with template parameters",
		"statement":   "SELECT {{array .fields}} FROM {{.tableName}}",
		"templateParameters": []tools.Parameter{
			tools.NewStringParameter("tableName", "some description"),
			tools.NewArrayParameter("fields", "The fields to select from", tools.NewStringParameter("field", "A field that will be returned from the query.")),
		},
	}
	toolsMap["select-filter-templateParams-combined-tool"] = map[string]any{
		"kind":        "spanner-sql",
		"source":      "my-instance",
		"description": "Create table tool with template parameters",
		"statement":   "SELECT * FROM {{.tableName}} WHERE {{.columnFilter}} = @name",
		"parameters":  []tools.Parameter{tools.NewStringParameter("name", "the name of the user")},
		"templateParameters": []tools.Parameter{
			tools.NewStringParameter("tableName", "some description"),
			tools.NewStringParameter("columnFilter", "some description"),
		},
	}
	config["tools"] = toolsMap
	return config
}

func runSpannerExecuteSqlToolInvokeTest(t *testing.T, select1Want, invokeParamWant, tableNameParam, tableNameAuth string) {
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
			name:          "invoke my-exec-sql-tool-read-only",
			api:           "http://127.0.0.1:5000/api/tool/my-exec-sql-tool-read-only/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"sql":"SELECT 1"}`)),
			want:          select1Want,
			isErr:         false,
		},
		{
			name:          "invoke my-exec-sql-tool-read-only with data present in table",
			api:           "http://127.0.0.1:5000/api/tool/my-exec-sql-tool-read-only/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf("{\"sql\":\"SELECT * FROM %s WHERE id = 3 OR name = 'Alice'\"}", tableNameParam))),
			want:          invokeParamWant,
			isErr:         false,
		},
		{
			name:          "invoke my-exec-sql-tool-read-only create table",
			api:           "http://127.0.0.1:5000/api/tool/my-exec-sql-tool-read-only/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"sql":"CREATE TABLE t (id SERIAL PRIMARY KEY, name TEXT)"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-exec-sql-tool-read-only drop table",
			api:           "http://127.0.0.1:5000/api/tool/my-exec-sql-tool-read-only/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"sql":"DROP TABLE t"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-exec-sql-tool-read-only insert entry",
			api:           "http://127.0.0.1:5000/api/tool/my-exec-sql-tool-read-only/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf("{\"sql\":\"INSERT INTO %s (id, name) VALUES (4, 'test_name')\"}", tableNameParam))),
			isErr:         true,
		},
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
			isErr:         true,
		},
		{
			name:          "invoke my-exec-sql-tool drop table",
			api:           "http://127.0.0.1:5000/api/tool/my-exec-sql-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"sql":"DROP TABLE t"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-exec-sql-tool insert entry",
			api:           "http://127.0.0.1:5000/api/tool/my-exec-sql-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf("{\"sql\":\"INSERT INTO %s (id, name) VALUES (5, 'test_name')\"}", tableNameParam))),
			want:          "null",
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

func runSpannerSchemaToolInvokeTest(t *testing.T, accessSchemaWant string) {
	invokeTcs := []struct {
		name          string
		api           string
		requestHeader map[string]string
		requestBody   io.Reader
		want          string
		isErr         bool
	}{
		{
			name:          "invoke list-tables-read-only",
			api:           "http://127.0.0.1:5000/api/tool/access-schema-read-only/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			want:          accessSchemaWant,
			isErr:         false,
		},
		{
			name:          "invoke list-tables",
			api:           "http://127.0.0.1:5000/api/tool/access-schema/invoke",
			requestHeader: map[string]string{},
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

			if got != tc.want {
				t.Fatalf("unexpected value: got %q, want %q", got, tc.want)
			}
		})
	}
}
