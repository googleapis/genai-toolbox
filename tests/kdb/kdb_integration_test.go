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

package kdb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"
	kdbgo "github.com/sv/kdbgo"
)

var (
	KdbSourceKind = "kdb"
	KdbToolKind   = "kdb-sql"
	KDBHost       = os.Getenv("KDB_HOST")
	KDBPort       = os.Getenv("KDB_PORT")
	KDBUser       = os.Getenv("KDB_USER")
	KDBPass       = os.Getenv("KDB_PASS")
)

func getKDBVars(t *testing.T) map[string]any {
	switch "" {
	case KDBHost:
		t.Fatal("'KDB_HOST' not set")
	case KDBPort:
		t.Fatal("'KDB_PORT' not set")
	}

	return map[string]any{
		"kind":     KdbSourceKind,
		"host":     KDBHost,
		"port":     KDBPort,
	}
}

func initKDBConnection(host, port, user, pass string) (*kdbgo.KDBConn, error) {
	var portInt, err = strconv.Atoi(port)
	if err != nil {
		return nil, fmt.Errorf("unable to parse port: %w", err)
	}

	db, err := kdbgo.DialKDB(host, portInt, auth)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to kdb: %w", err)
	}

	return db, nil
}

func TestKDBToolEndpoints(t *testing.T) {
	sourceConfig := getKDBVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	var args []string

	kdbConn, err := initKDBConnection(KDBHost, KDBPort, KDBUser, KDBPass)
	if err != nil {
		t.Fatalf("unable to create kdb connection: %s", err)
	}

	// create table name with UUID
	tableNameParam := "param_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableNameAuth := "auth_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableNameTemplateParam := "template_param_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableNameDataType := "datatype_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")

	// set up data for param tool
	createParamTableStmt, insertParamTableStmt, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, paramTestParams := getKDBParamToolInfo(tableNameParam)
	teardownTable1 := setupKDBTable(t, ctx, kdbConn, createParamTableStmt, insertParamTableStmt, tableNameParam, paramTestParams)
	defer teardownTable1(t)

	// set up data for auth tool
	createAuthTableStmt, insertAuthTableStmt, authToolStmt, authTestParams := getKDBAuthToolInfo(tableNameAuth)
	teardownTable2 := setupKDBTable(t, ctx, kdbConn, createAuthTableStmt, insertAuthTableStmt, tableNameAuth, authTestParams)
	defer teardownTable2(t)

	// set up data for data type test tool
	createDataTypeTableStmt, insertDataTypeTableStmt, dataTypeToolStmt, arrayDataTypeToolStmt, dataTypeTestParams := getKDBDataTypeTestInfo(tableNameDataType)
	teardownTable3 := setupKDBTable(t, ctx, kdbConn, createDataTypeTableStmt, insertDataTypeTableStmt, tableNameDataType, dataTypeTestParams)
	defer teardownTable3(t)

	// Write config into a file and pass it to command
	toolsFile := tests.GetToolsConfig(sourceConfig, KdbToolKind, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, authToolStmt)
	toolsFile = addKdbSqlToolConfig(t, toolsFile, dataTypeToolStmt, arrayDataTypeToolStmt)
	tmplSelectCombined, tmplSelectFilterCombined := getKDBTmplToolStatement()
	toolsFile = tests.AddTemplateParamConfig(t, toolsFile, KdbToolKind, tmplSelectCombined, tmplSelectFilterCombined, "")

	cmd, cleanup, err := tests.StartCmd(ctx, toolsFile, args...)
	if err != nil {
		t.Fatalf("command initialization returned an error: %s", err)
	}
	defer cleanup()

	waitCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	out, err := testutils.WaitForString(waitCtx, regexp.MustCompile("Server ready to serve"), cmd.Out)
	if err != nil {
		t.Logf("toolbox command logs: %s", out)
		t.Fatalf("toolbox didn't start successfully: %s", err)
	}

	// Get configs for tests
	select1Want := `[{"x":1}]`
	invokeParamWant := `[{"id":1,"name":"Alice"},{"id":3,"name":"Sid"}]`
	ddlWant := "null"
	mcpMyFailToolWant := `{"jsonrpc":"2.0","id":"invoke-fail-tool","result":{"content":[{"type":"text","text":"unable to execute query: SELEC 1; ('rank)"}],"isError":true}}`
	mcpSelect1Want := `{"jsonrpc":"2.0","id":"invoke my-auth-required-tool","result":{"content":[{"type":"text","text":"{"x":1}"}]}}`
	createColArray := `["id: ` + "`int" + `", "name: ` + "`symbol" + `", "age: ` + "`int" + `"]`
	selectEmptyWant := "null"

	// Run tests
	tests.RunToolGetTest(t)
	tests.RunToolInvokeTest(t, select1Want)
	tests.RunMCPToolCallMethod(t, mcpMyFailToolWant, mcpSelect1Want)
	tests.RunToolInvokeWithTemplateParameters(t, tableNameTemplateParam,
		tests.WithCreateColArray(createColArray),
		tests.WithDdlWant(ddlWant),
		tests.WithSelectEmptyWant(selectEmptyWant),
		tests.WithInsert1Want(ddlWant),
	)

	runKdbExecuteSqlToolInvokeTest(t, select1Want, invokeParamWant, tableNameParam, ddlWant)
	runKdbDataTypeTests(t)
}

// getKDBParamToolInfo returns statements and param for my-tool for kdb kind
func getKDBParamToolInfo(tableName string) (string, string, string, string, string, string, *kdbgo.K) {
	createStatement := fmt.Sprintf("%s:([] id:`int$(); name:`symbol$())", tableName)
	insertStatement := fmt.Sprintf("`%s insert(1; `Alice); `%s insert(2; `Jane); `%s insert(3; `Sid); `%s insert(4; `)", tableName, tableName, tableName, tableName)
	toolStatement := fmt.Sprintf("select from %s where id = ?, name = ?", tableName)
	idToolStatement := fmt.Sprintf("select from %s where id = ?", tableName)
	nameToolStatement := fmt.Sprintf("select from %s where name = ?", tableName)
	arrayToolStatememt := fmt.Sprintf("select from %s where id in ?, name in ?", tableName)
	params := &kdbgo.K{}
	return createStatement, insertStatement, toolStatement, idToolStatement, nameToolStatement, arrayToolStatememt, params
}

// getKDBAuthToolInfo returns statements and param of my-auth-tool for kdb kind
func getKDBAuthToolInfo(tableName string) (string, string, string, *kdbgo.K) {
	createStatement := fmt.Sprintf("%s:([] id:`int$(); name:`symbol$(); email:`symbol$())", tableName)
	insertStatement := fmt.Sprintf("`%s insert(1; `Alice; `%s); `%s insert(2; `Jane; `janedoe@gmail.com)", tableName, tests.ServiceAccountEmail, tableName)
	toolStatement := fmt.Sprintf("select name from %s where email = ?", tableName)
	params := &kdbgo.K{}
	return createStatement, insertStatement, toolStatement, params
}

// getKDBDataTypeTestInfo returns statements and params for data type tests.
func getKDBDataTypeTestInfo(tableName string) (string, string, string, string, *kdbgo.K) {
	createStatement := fmt.Sprintf("%s:([] id:`int$(); int_val:`int$(); string_val:`symbol$(); float_val:`float$(); bool_val:`boolean$())", tableName)
	insertStatement := fmt.Sprintf("`%s insert(1; 123; `hello; 3.14; `boolean$1); `%s insert(2; -456; `world; -0.55; `boolean$0); `%s insert(3; 789; `test; 100.1; `boolean$1)", tableName, tableName, tableName)
	toolStatement := "select from %s where int_val = ?, string_val = ?, float_val = ?, bool_val = ?"
	arrayToolStatement := "select from %s where int_val in ?, string_val in ?, float_val in ?, bool_val in ?"
	params := &kdbgo.K{}
	return createStatement, insertStatement, toolStatement, arrayToolStatement, params
}

// getKDBTmplToolStatement returns statements for template parameter test cases for kdb kind
func getKDBTmplToolStatement() (string, string) {
	tmplSelectCombined := "select from {{.tableName}} where id = ?"
	tmplSelectFilterCombined := "select from {{.tableName}} where {{.columnFilter}} = ?"
	return tmplSelectCombined, tmplSelectFilterCombined
}

func setupKDBTable(t *testing.T, ctx context.Context, db *kdbgo.KDBConn, createStatement, insertStatement, tableName string, params *kdbgo.K) func(*testing.T) {
	// Create table
	if _, err := db.Call(createStatement); err != nil {
		t.Fatalf("Failed to create table %s: %v", tableName, err)
	}

	// Insert test data
	if _, err := db.Call(insertStatement); err != nil {
		t.Fatalf("Failed to insert data into table %s: %v", tableName, err)
	}

	return func(t *testing.T) {
		// tear down table
		dropSQL := fmt.Sprintf("delete %s from `.", tableName)
		if _, err := db.Call(dropSQL); err != nil {
			t.Errorf("Failed to drop table %s: %v", tableName, err)
		}
	}
}

func addKdbSqlToolConfig(t *testing.T, config map[string]any, toolStatement, arrayToolStatement string) map[string]any {
	tools, ok := config["tools"].(map[string]any)
	if !ok {
		t.Fatalf("unable to get tools from config")
	}
	tools["my-scalar-datatype-tool"] = map[string]any{
		"kind":        "kdb-sql",
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
		"kind":        "kdb-sql",
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

func runKdbExecuteSqlToolInvokeTest(t *testing.T, select1Want, invokeParamWant, tableNameParam, ddlWant string) {
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
			requestBody:   bytes.NewBuffer([]byte(`{"sql":"select from ([] x:1)"}`)),
			want:          select1Want,
			isErr:         false,
		},
		{
			name:          "invoke my-exec-sql-tool create table",
			api:           "http://127.0.0.1:5000/api/tool/my-exec-sql-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"sql":"t:([] id:` + "`int$(); name:`symbol$())" + `"}`)),
			want:          ddlWant,
			isErr:         false,
		},
		{
			name:          "invoke my-exec-sql-tool with data present in table",
			api:           "http://127.0.0.1:5000/api/tool/my-exec-sql-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf(`{"sql":"select from %s where id = 3, name = Alice"}`, tableNameParam))),
			want:          invokeParamWant,
			isErr:         false,
		},
		{
			name:          "invoke my-exec-sql-tool with no matching rows",
			api:           "http://127.0.0.1:5000/api/tool/my-exec-sql-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf(`{"sql":"select from %s where id = 999"}`, tableNameParam))),
			want:          `"The query returned 0 rows."`,
			isErr:         false,
		},
		{
			name:          "invoke my-exec-sql-tool drop table",
			api:           "http://127.0.0.1:5000/api/tool/my-exec-sql-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"sql":"delete t from ` + "`.`" + `"}`)),
			want:          ddlWant,
			isErr:         false,
		},
		{
			name:          "invoke my-exec-sql-tool insert entry",
			api:           "http://127.0.0.1:5000/api/tool/my-exec-sql-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf(`{"sql":"%s insert(4; \"test_name\")"}`, tableNameParam))),
			want:          ddlWant,
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

func runKdbDataTypeTests(t *testing.T) {
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
			want:          `[{"id":1,"int_val":123,"string_val":"hello","float_val":3.14,"bool_val":true}]`,
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
			want:          `[{"id":1,"int_val":123,"string_val":"hello","float_val":3.14,"bool_val":true},{"id":3,"int_val":789,"string_val":"test","float_val":100.1,"bool_val":true}]`,
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
