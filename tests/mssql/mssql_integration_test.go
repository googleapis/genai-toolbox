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

package mssql

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"
	_ "github.com/microsoft/go-mssqldb"
)

var (
	MSSQLSourceKind = "mssql"
	MSSQLToolKind   = "mssql-sql"
	MSSQLListTablesToolKind = "mssql-list-tables"
	MSSQLDatabase   = os.Getenv("MSSQL_DATABASE")
	MSSQLHost       = os.Getenv("MSSQL_HOST")
	MSSQLPort       = os.Getenv("MSSQL_PORT")
	MSSQLUser       = os.Getenv("MSSQL_USER")
	MSSQLPass       = os.Getenv("MSSQL_PASS")
)

func getMsSQLVars(t *testing.T) map[string]any {
	switch "" {
	case MSSQLDatabase:
		t.Fatal("'MSSQL_DATABASE' not set")
	case MSSQLHost:
		t.Fatal("'MSSQL_HOST' not set")
	case MSSQLPort:
		t.Fatal("'MSSQL_PORT' not set")
	case MSSQLUser:
		t.Fatal("'MSSQL_USER' not set")
	case MSSQLPass:
		t.Fatal("'MSSQL_PASS' not set")
	}

	return map[string]any{
		"kind":     MSSQLSourceKind,
		"host":     MSSQLHost,
		"port":     MSSQLPort,
		"database": MSSQLDatabase,
		"user":     MSSQLUser,
		"password": MSSQLPass,
	}
}

func addPrebuiltToolConfig(t *testing.T, config map[string]any) map[string]any {
	tools, ok := config["tools"].(map[string]any)
	if !ok {
		t.Fatalf("unable to get tools from config")
	}
	tools["list_tables"] = map[string]any{
		"kind":        MSSQLListTablesToolKind,
		"source":      "my-instance",
		"description": "Lists tables in the database.",
	}
	config["tools"] = tools
	return config
}

// Copied over from mssql.go
func initMSSQLConnection(host, port, user, pass, dbname string) (*sql.DB, error) {
	// Create dsn
	query := url.Values{}
	query.Add("database", dbname)
	url := &url.URL{
		Scheme:   "sqlserver",
		User:     url.UserPassword(user, pass),
		Host:     fmt.Sprintf("%s:%s", host, port),
		RawQuery: query.Encode(),
	}

	// Open database connection
	db, err := sql.Open("sqlserver", url.String())
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}
	return db, nil
}

func TestMSSQLToolEndpoints(t *testing.T) {
	sourceConfig := getMsSQLVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	pool, err := initMSSQLConnection(MSSQLHost, MSSQLPort, MSSQLUser, MSSQLPass, MSSQLDatabase)
	if err != nil {
		t.Fatalf("unable to create SQL Server connection pool: %s", err)
	}

	// create table name with UUID
	tableNameParam := "param_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableNameAuth := "auth_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableNameTemplateParam := "template_param_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")

	// set up data for param tool
	createParamTableStmt, insertParamTableStmt, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, paramTestParams := tests.GetMSSQLParamToolInfo(tableNameParam)
	teardownTable1 := tests.SetupMsSQLTable(t, ctx, pool, createParamTableStmt, insertParamTableStmt, tableNameParam, paramTestParams)
	defer teardownTable1(t)

	// set up data for auth tool
	createAuthTableStmt, insertAuthTableStmt, authToolStmt, authTestParams := tests.GetMSSQLAuthToolInfo(tableNameAuth)
	teardownTable2 := tests.SetupMsSQLTable(t, ctx, pool, createAuthTableStmt, insertAuthTableStmt, tableNameAuth, authTestParams)
	defer teardownTable2(t)

	// Write config into a file and pass it to command
	toolsFile := tests.GetToolsConfig(sourceConfig, MSSQLToolKind, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, authToolStmt)
	toolsFile = tests.AddMSSQLExecuteSqlConfig(t, toolsFile)
	tmplSelectCombined, tmplSelectFilterCombined := tests.GetMSSQLTmplToolStatement()
	toolsFile = tests.AddTemplateParamConfig(t, toolsFile, MSSQLToolKind, tmplSelectCombined, tmplSelectFilterCombined, "")

	toolsFile = addPrebuiltToolConfig(t, toolsFile)

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
	select1Want, mcpMyFailToolWant, createTableStatement, mcpSelect1Want := tests.GetMSSQLWants()

	// Run tests
	tests.RunToolGetTest(t)
	tests.RunToolInvokeTest(t, select1Want, tests.DisableArrayTest())
	tests.RunMCPToolCallMethod(t, mcpMyFailToolWant, mcpSelect1Want)
	tests.RunExecuteSqlToolInvokeTest(t, createTableStatement, select1Want)
	tests.RunToolInvokeWithTemplateParameters(t, tableNameTemplateParam)

	// Run specific MSSQL tool tests
	runMSSQLListTablesTest(t, tableNameParam, tableNameAuth)
}

func runMSSQLListTablesTest(t *testing.T, tableNameParam, tableNameAuth string) {
	// TableNameParam columns to construct want.
	const paramTableColumns = `[
        {"column_name": "id", "data_type": "INT", "column_ordinal_position": 1, "is_not_nullable": true},
        {"column_name": "name", "data_type": "VARCHAR(255)", "column_ordinal_position": 2, "is_not_nullable": false}
    ]`

	// TableNameAuth columns to construct want
	const authTableColumns = `[
		{"column_name": "id", "data_type": "INT", "column_ordinal_position": 1, "is_not_nullable": true},
		{"column_name": "name", "data_type": "VARCHAR(255)", "column_ordinal_position": 2, "is_not_nullable": false},
		{"column_name": "email", "data_type": "VARCHAR(255)", "column_ordinal_position": 3, "is_not_nullable": false}
    ]`

	const (
		// Template to construct detailed output want.
		detailedObjectTemplate = `{
            "schema_name": "dbo",
            "object_name": "%[1]s",
            "object_details": {
                "owner": "dbo",
                "triggers": [],
                "columns": %[2]s,
                "object_name": "%[1]s",
                "object_type": "TABLE",
                "schema_name": "dbo"
            }
        }`

		// Template to construct simple output want
		simpleObjectTemplate = `{"object_name":"%s", "schema_name":"dbo", "object_details":{"name":"%s"}}`
	)

	// Helper to build json for detailed want
	getDetailedWant := func(tableName, columnJSON string) string {
		return fmt.Sprintf(detailedObjectTemplate, tableName, columnJSON)
	}

	// Helper to build template for simple want
	getSimpleWant := func(tableName string) string {
		return fmt.Sprintf(simpleObjectTemplate, tableName, tableName)
	}

	invokeTcs := []struct {
		name           string
		api            string
		requestBody    io.Reader
		wantStatusCode int
		want           string
	}{
		{
			name:           "invoke list_tables detailed output",
			api:            "http://127.0.0.1:5000/api/tool/list_tables/invoke",
			requestBody:    bytes.NewBuffer([]byte(fmt.Sprintf(`{"table_names": "%s"}`, tableNameAuth))),
			wantStatusCode: http.StatusOK,
			want:           fmt.Sprintf("[%s]", getDetailedWant(tableNameAuth, authTableColumns)),
		},
		{
			name:           "invoke list_tables simple output",
			api:            "http://127.0.0.1:5000/api/tool/list_tables/invoke",
			requestBody:    bytes.NewBuffer([]byte(fmt.Sprintf(`{"table_names": "%s", "output_format": "simple"}`, tableNameAuth))),
			wantStatusCode: http.StatusOK,
			want:           fmt.Sprintf("[%s]", getSimpleWant(tableNameAuth)),
		},
		{
			name:           "invoke list_tables with invalid output format",
			api:            "http://127.0.0.1:5000/api/tool/list_tables/invoke",
			requestBody:    bytes.NewBuffer([]byte(`{"table_names": "", "output_format": "abcd"}`)),
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "invoke list_tables with malformed table_names parameter",
			api:            "http://127.0.0.1:5000/api/tool/list_tables/invoke",
			requestBody:    bytes.NewBuffer([]byte(`{"table_names": 12345, "output_format": "detailed"}`)),
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "invoke list_tables with multiple table names",
			api:            "http://127.0.0.1:5000/api/tool/list_tables/invoke",
			requestBody:    bytes.NewBuffer([]byte(fmt.Sprintf(`{"table_names": "%s,%s"}`, tableNameParam, tableNameAuth))),
			wantStatusCode: http.StatusOK,
			want:           fmt.Sprintf("[%s,%s]", getDetailedWant(tableNameAuth, authTableColumns), getDetailedWant(tableNameParam, paramTableColumns)),
		},
		{
			name:           "invoke list_tables with non-existent table",
			api:            "http://127.0.0.1:5000/api/tool/list_tables/invoke",
			requestBody:    bytes.NewBuffer([]byte(`{"table_names": "non_existent_table"}`)),
			wantStatusCode: http.StatusOK,
			want:           `null`,
		},
		{
			name:           "invoke list_tables with one existing and one non-existent table",
			api:            "http://127.0.0.1:5000/api/tool/list_tables/invoke",
			requestBody:    bytes.NewBuffer([]byte(fmt.Sprintf(`{"table_names": "%s,non_existent_table"}`, tableNameParam))),
			wantStatusCode: http.StatusOK,
			want:           fmt.Sprintf("[%s]", getDetailedWant(tableNameParam, paramTableColumns)),
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost, tc.api, tc.requestBody)
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
				t.Fatalf("response status code is not %d, got %d: %s", tc.wantStatusCode, resp.StatusCode, string(bodyBytes))
			}

			if tc.wantStatusCode == http.StatusOK {
				var bodyWrapper map[string]json.RawMessage
				respBytes, err := io.ReadAll(resp.Body)
				if err != nil {
					t.Fatalf("error reading response body: %s", err)
				}

				if err := json.Unmarshal(respBytes, &bodyWrapper); err != nil {
					t.Fatalf("error parsing response wrapper: %s, body: %s", err, string(respBytes))
				}

				resultJSON, ok := bodyWrapper["result"]
				if !ok {
					t.Fatal("unable to find 'result' in response body")
				}

				var resultString string
				if err := json.Unmarshal(resultJSON, &resultString); err != nil {
					if string(resultJSON) == "null" {
						resultString = "null"
					} else {
						t.Fatalf("'result' is not a JSON-encoded string: %s", err)
					}
				}

				var got, want []any

				if err := json.Unmarshal([]byte(resultString), &got); err != nil {
					t.Fatalf("failed to unmarshal actual result string: %v", err)
				}
				if err := json.Unmarshal([]byte(tc.want), &want); err != nil {
					t.Fatalf("failed to unmarshal expected want string: %v", err)
				}

				for _, item := range got {
					itemMap, ok := item.(map[string]any)
					if !ok {
						continue
					}

					detailsStr, ok := itemMap["object_details"].(string)
					if !ok {
						continue
					}

					var detailsMap map[string]any
					if err := json.Unmarshal([]byte(detailsStr), &detailsMap); err != nil {
						t.Fatalf("failed to unmarshal nested object_details string: %v", err)
					}

					// clean unpredictable fields
					delete(detailsMap, "constraints")
					delete(detailsMap, "indexes")

					itemMap["object_details"] = detailsMap
				}

				sort.SliceStable(got, func(i, j int) bool {
					return fmt.Sprintf("%v", got[i]) < fmt.Sprintf("%v", got[j])
				})
				sort.SliceStable(want, func(i, j int) bool {
					return fmt.Sprintf("%v", want[i]) < fmt.Sprintf("%v", want[j])
				})

				if !reflect.DeepEqual(got, want) {
					gotJSON, _ := json.MarshalIndent(got, "", "  ")
					wantJSON, _ := json.MarshalIndent(want, "", "  ")
					t.Errorf("Unexpected result:\ngot:\n%s\n\nwant:\n%s", string(gotJSON), string(wantJSON))
				}
			}
		})
	}
}
