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

package mysql

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"
)

var (
	MySQLSourceKind            = "mysql"
	MySQLToolKind              = "mysql-sql"
	MySQLListActiveQueriesKind = "mysql-list-active-queries"
	MySQLListTablesToolKind    = "mysql-list-tables"
	MySQLDatabase              = os.Getenv("MYSQL_DATABASE")
	MySQLHost                  = os.Getenv("MYSQL_HOST")
	MySQLPort                  = os.Getenv("MYSQL_PORT")
	MySQLUser                  = os.Getenv("MYSQL_USER")
	MySQLPass                  = os.Getenv("MYSQL_PASS")
)

func getMySQLVars(t *testing.T) map[string]any {
	switch "" {
	case MySQLDatabase:
		t.Fatal("'MYSQL_DATABASE' not set")
	case MySQLHost:
		t.Fatal("'MYSQL_HOST' not set")
	case MySQLPort:
		t.Fatal("'MYSQL_PORT' not set")
	case MySQLUser:
		t.Fatal("'MYSQL_USER' not set")
	case MySQLPass:
		t.Fatal("'MYSQL_PASS' not set")
	}

	return map[string]any{
		"kind":     MySQLSourceKind,
		"host":     MySQLHost,
		"port":     MySQLPort,
		"database": MySQLDatabase,
		"user":     MySQLUser,
		"password": MySQLPass,
	}
}

func addPrebuiltToolConfig(t *testing.T, config map[string]any) map[string]any {
	tools, ok := config["tools"].(map[string]any)
	if !ok {
		t.Fatalf("unable to get tools from config")
	}
	tools["list_tables"] = map[string]any{
		"kind":        MySQLListTablesToolKind,
		"source":      "my-instance",
		"description": "Lists tables in the database.",
	}
	tools["list_active_queries"] = map[string]any{
		"kind":        MySQLListActiveQueriesKind,
		"source":      "my-instance",
		"description": "Lists active queries in the database.",
	}
	config["tools"] = tools
	return config
}

// Copied over from mysql.go
func initMySQLConnectionPool(host, port, user, pass, dbname string) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", user, pass, host, port, dbname)

	// Interact with the driver directly as you normally would
	pool, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}
	return pool, nil
}

func TestMySQLToolEndpoints(t *testing.T) {
	sourceConfig := getMySQLVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	pool, err := initMySQLConnectionPool(MySQLHost, MySQLPort, MySQLUser, MySQLPass, MySQLDatabase)
	if err != nil {
		t.Fatalf("unable to create MySQL connection pool: %s", err)
	}

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
	toolsFile := tests.GetToolsConfig(sourceConfig, MySQLToolKind, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, authToolStmt)
	toolsFile = tests.AddMySqlExecuteSqlConfig(t, toolsFile)
	tmplSelectCombined, tmplSelectFilterCombined := tests.GetMySQLTmplToolStatement()
	toolsFile = tests.AddTemplateParamConfig(t, toolsFile, MySQLToolKind, tmplSelectCombined, tmplSelectFilterCombined, "")

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
	select1Want, mcpMyFailToolWant, createTableStatement, mcpSelect1Want := tests.GetMySQLWants()

	// Run tests
	tests.RunToolGetTest(t)
	tests.RunToolInvokeTest(t, select1Want, tests.DisableArrayTest())
	tests.RunMCPToolCallMethod(t, mcpMyFailToolWant, mcpSelect1Want)
	tests.RunExecuteSqlToolInvokeTest(t, createTableStatement, select1Want)
	tests.RunToolInvokeWithTemplateParameters(t, tableNameTemplateParam)

	// Run specific MySQL tool tests
	runMySQLListTablesTest(t, tableNameParam, tableNameAuth)
	runMySQLListActiveQueriesTest(t, ctx, pool)
}

func runMySQLListTablesTest(t *testing.T, tableNameParam, tableNameAuth string) {
	type tableInfo struct {
		ObjectName    string `json:"object_name"`
		SchemaName    string `json:"schema_name"`
		ObjectDetails string `json:"object_details"`
	}

	type column struct {
		DataType        string `json:"data_type"`
		ColumnName      string `json:"column_name"`
		ColumnComment   string `json:"column_comment"`
		ColumnDefault   any    `json:"column_default"`
		IsNotNullable   int    `json:"is_not_nullable"`
		OrdinalPosition int    `json:"ordinal_position"`
	}

	type objectDetails struct {
		Owner       any      `json:"owner"`
		Columns     []column `json:"columns"`
		Comment     string   `json:"comment"`
		Indexes     []any    `json:"indexes"`
		Triggers    []any    `json:"triggers"`
		Constraints []any    `json:"constraints"`
		ObjectName  string   `json:"object_name"`
		ObjectType  string   `json:"object_type"`
		SchemaName  string   `json:"schema_name"`
	}

	paramTableWant := objectDetails{
		ObjectName: tableNameParam,
		SchemaName: MySQLDatabase,
		ObjectType: "TABLE",
		Columns: []column{
			{DataType: "int", ColumnName: "id", IsNotNullable: 1, OrdinalPosition: 1},
			{DataType: "varchar(255)", ColumnName: "name", OrdinalPosition: 2},
		},
		Indexes:     []any{map[string]any{"index_columns": []any{"id"}, "index_name": "PRIMARY", "is_primary": float64(1), "is_unique": float64(1)}},
		Triggers:    []any{},
		Constraints: []any{map[string]any{"constraint_columns": []any{"id"}, "constraint_name": "PRIMARY", "constraint_type": "PRIMARY KEY", "foreign_key_referenced_columns": any(nil), "foreign_key_referenced_table": any(nil), "constraint_definition": ""}},
	}

	authTableWant := objectDetails{
		ObjectName: tableNameAuth,
		SchemaName: MySQLDatabase,
		ObjectType: "TABLE",
		Columns: []column{
			{DataType: "int", ColumnName: "id", IsNotNullable: 1, OrdinalPosition: 1},
			{DataType: "varchar(255)", ColumnName: "name", OrdinalPosition: 2},
			{DataType: "varchar(255)", ColumnName: "email", OrdinalPosition: 3},
		},
		Indexes:     []any{map[string]any{"index_columns": []any{"id"}, "index_name": "PRIMARY", "is_primary": float64(1), "is_unique": float64(1)}},
		Triggers:    []any{},
		Constraints: []any{map[string]any{"constraint_columns": []any{"id"}, "constraint_name": "PRIMARY", "constraint_type": "PRIMARY KEY", "foreign_key_referenced_columns": any(nil), "foreign_key_referenced_table": any(nil), "constraint_definition": ""}},
	}

	invokeTcs := []struct {
		name           string
		requestBody    io.Reader
		wantStatusCode int
		want           any
		isSimple       bool
	}{
		{
			name:           "invoke list_tables detailed output",
			requestBody:    bytes.NewBufferString(fmt.Sprintf(`{"table_names": "%s"}`, tableNameAuth)),
			wantStatusCode: http.StatusOK,
			want:           []objectDetails{authTableWant},
		},
		{
			name:           "invoke list_tables simple output",
			requestBody:    bytes.NewBufferString(fmt.Sprintf(`{"table_names": "%s", "output_format": "simple"}`, tableNameAuth)),
			wantStatusCode: http.StatusOK,
			want:           []map[string]any{{"name": tableNameAuth}},
			isSimple:       true,
		},
		{
			name:           "invoke list_tables with multiple table names",
			requestBody:    bytes.NewBufferString(fmt.Sprintf(`{"table_names": "%s,%s"}`, tableNameParam, tableNameAuth)),
			wantStatusCode: http.StatusOK,
			want:           []objectDetails{authTableWant, paramTableWant},
		},
		{
			name:           "invoke list_tables with one existing and one non-existent table",
			requestBody:    bytes.NewBufferString(fmt.Sprintf(`{"table_names": "%s,non_existent_table"}`, tableNameAuth)),
			wantStatusCode: http.StatusOK,
			want:           []objectDetails{authTableWant},
		},
		{
			name:           "invoke list_tables with non-existent table",
			requestBody:    bytes.NewBufferString(`{"table_names": "non_existent_table"}`),
			wantStatusCode: http.StatusOK,
			want:           nil,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			const api = "http://127.0.0.1:5000/api/tool/list_tables/invoke"
			req, err := http.NewRequest(http.MethodPost, api, tc.requestBody)
			if err != nil {
				t.Fatalf("unable to create request: %v", err)
			}
			req.Header.Add("Content-type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("unable to send request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.wantStatusCode {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("wrong status code: got %d, want %d, body: %s", resp.StatusCode, tc.wantStatusCode, string(body))
			}
			if tc.wantStatusCode != http.StatusOK {
				return
			}

			var bodyWrapper struct {
				Result json.RawMessage `json:"result"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&bodyWrapper); err != nil {
				t.Fatalf("error decoding response wrapper: %v", err)
			}

			var resultString string
			if err := json.Unmarshal(bodyWrapper.Result, &resultString); err != nil {
				resultString = string(bodyWrapper.Result)
			}

			var got any
			if tc.isSimple {
				var tables []tableInfo
				if err := json.Unmarshal([]byte(resultString), &tables); err != nil {
					t.Fatalf("failed to unmarshal outer JSON array into []tableInfo: %v", err)
				}
				var details []map[string]any
				for _, table := range tables {
					var d map[string]any
					if err := json.Unmarshal([]byte(table.ObjectDetails), &d); err != nil {
						t.Fatalf("failed to unmarshal nested ObjectDetails string: %v", err)
					}
					details = append(details, d)
				}
				got = details
			} else {
				if resultString == "null" {
					got = nil
				} else {
					var tables []tableInfo
					if err := json.Unmarshal([]byte(resultString), &tables); err != nil {
						t.Fatalf("failed to unmarshal outer JSON array into []tableInfo: %v", err)
					}
					var details []objectDetails
					for _, table := range tables {
						var d objectDetails
						if err := json.Unmarshal([]byte(table.ObjectDetails), &d); err != nil {
							t.Fatalf("failed to unmarshal nested ObjectDetails string: %v", err)
						}
						details = append(details, d)
					}
					got = details
				}
			}

			opts := []cmp.Option{
				cmpopts.SortSlices(func(a, b objectDetails) bool { return a.ObjectName < b.ObjectName }),
				cmpopts.SortSlices(func(a, b column) bool { return a.ColumnName < b.ColumnName }),
				cmpopts.SortSlices(func(a, b map[string]any) bool { return a["name"].(string) < b["name"].(string) }),
			}

			if diff := cmp.Diff(tc.want, got, opts...); diff != "" {
				t.Errorf("Unexpected result: got %#v, want: %#v", got, tc.want)
			}
		})
	}
}

func runMySQLListActiveQueriesTest(t *testing.T, ctx context.Context, pool *sql.DB) {
	type queryListDetails struct {
		ProcessId       any    `json:"process_id"`
		Query           string `json:"query"`
		TrxStarted      any    `json:"trx_started"`
		TrxDuration     any    `json:"trx_duration_seconds"`
		TrxWaitDuration any    `json:"trx_wait_duration_seconds"`
		QueryTime       any    `json:"query_time"`
		TrxState        string `json:"trx_state"`
		ProcessState    string `json:"process_state"`
		User            string `json:"user"`
		TrxRowsLocked   any    `json:"trx_rows_locked"`
		TrxRowsModified any    `json:"trx_rows_modified"`
		Db              string `json:"db"`
	}

	singleQueryWanted := queryListDetails{
		ProcessId:       any(nil),
		Query:           "SELECT sleep(10)",
		TrxStarted:      any(nil),
		TrxDuration:     any(nil),
		TrxWaitDuration: any(nil),
		QueryTime:       any(nil),
		TrxState:        "",
		ProcessState:    "User sleep",
		User:            "",
		TrxRowsLocked:   any(nil),
		TrxRowsModified: any(nil),
		Db:              "",
	}

	invokeTcs := []struct {
		name                string
		requestBody         io.Reader
		clientSleepSecs     int
		waitSecsBeforeCheck int
		wantStatusCode      int
		want                any
	}{
		{
			name:                "invoke list_active_queries when the system is idle",
			requestBody:         bytes.NewBufferString(`{}`),
			clientSleepSecs:     0,
			waitSecsBeforeCheck: 0,
			wantStatusCode:      http.StatusOK,
			want:                []queryListDetails(nil),
		},
		{
			name:                "invoke list_active_queries when there is 1 ongoing but lower than the threshold",
			requestBody:         bytes.NewBufferString(`{"min_duration_secs": 100}`),
			clientSleepSecs:     10,
			waitSecsBeforeCheck: 1,
			wantStatusCode:      http.StatusOK,
			want:                []queryListDetails(nil),
		},
		{
			name:                "invoke list_active_queries when 1 ongoing query should show up",
			requestBody:         bytes.NewBufferString(`{"min_duration_secs": 5}`),
			clientSleepSecs:     0,
			waitSecsBeforeCheck: 5,
			wantStatusCode:      http.StatusOK,
			want:                []queryListDetails{singleQueryWanted},
		},
	}

	var wg sync.WaitGroup
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			if tc.clientSleepSecs > 0 {
				wg.Add(1)

				go func() {
					defer wg.Done()

					err := pool.PingContext(ctx)
					if err != nil {
						t.Errorf("unable to connect to test database: %s", err)
						return
					}
					_, err = pool.ExecContext(ctx, fmt.Sprintf("SELECT sleep(%d);", tc.clientSleepSecs))
					if err != nil {
						t.Errorf("Executing 'SELECT sleep' failed: %s", err)
					}
				}()
			}

			if tc.waitSecsBeforeCheck > 0 {
				time.Sleep(time.Duration(tc.waitSecsBeforeCheck) * time.Second)
			}

			const api = "http://127.0.0.1:5000/api/tool/list_active_queries/invoke"
			req, err := http.NewRequest(http.MethodPost, api, tc.requestBody)
			if err != nil {
				t.Fatalf("unable to create request: %v", err)
			}
			req.Header.Add("Content-type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("unable to send request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.wantStatusCode {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("wrong status code: got %d, want %d, body: %s", resp.StatusCode, tc.wantStatusCode, string(body))
			}
			if tc.wantStatusCode != http.StatusOK {
				return
			}

			var bodyWrapper struct {
				Result json.RawMessage `json:"result"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&bodyWrapper); err != nil {
				t.Fatalf("error decoding response wrapper: %v", err)
			}

			var resultString string
			if err := json.Unmarshal(bodyWrapper.Result, &resultString); err != nil {
				resultString = string(bodyWrapper.Result)
			}

			var got any
			var details []queryListDetails
			if err := json.Unmarshal([]byte(resultString), &details); err != nil {
				t.Fatalf("failed to unmarshal nested ObjectDetails string: %v", err)
			}
			got = details

			if diff := cmp.Diff(tc.want, got, cmp.Comparer(func(a, b queryListDetails) bool {
				return a.Query == b.Query && a.ProcessState == b.ProcessState
			})); diff != "" {
				t.Errorf("Unexpected result: got %#v, want: %#v", got, tc.want)
			}
		})
	}
	wg.Wait()
}
