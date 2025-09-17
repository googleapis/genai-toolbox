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
	"context"
	"database/sql"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"
)

var (
	MySQLSourceKind = "mysql"
	MySQLToolKind   = "mysql-sql"
	MySQLDatabase   = os.Getenv("MYSQL_DATABASE")
	MySQLHost       = os.Getenv("MYSQL_HOST")
	MySQLPort       = os.Getenv("MYSQL_PORT")
	MySQLUser       = os.Getenv("MYSQL_USER")
	MySQLPass       = os.Getenv("MYSQL_PASS")
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

func createTableHelper(t *testing.T, tableName string, primaryKey, uniqueKey, nonUniqueKey bool, ctx context.Context, pool *sql.DB) func() {
	t.Helper()

	var stmt strings.Builder
	stmt.WriteString(fmt.Sprintf("CREATE TABLE %s (", tableName))
	stmt.WriteString("c1 INT")
	if primaryKey {
		stmt.WriteString(" PRIMARY KEY")
	}
	stmt.WriteString(", c2 INT, c3 CHAR(8)")
	if uniqueKey {
		stmt.WriteString(", UNIQUE(c2)")
	}
	if nonUniqueKey {
		stmt.WriteString(", INDEX(c3)")
	}
	stmt.WriteString(")")

	t.Logf("Creating table: %s", stmt.String())
	if _, err := pool.ExecContext(ctx, stmt.String()); err != nil {
		t.Fatalf("failed executing %s: %v", stmt.String(), err)
	}

	return func() {
		t.Logf("Dropping table: %s", tableName)
		if _, err := pool.ExecContext(ctx, fmt.Sprintf("DROP TABLE %s", tableName)); err != nil {
			t.Errorf("failed to drop table %s: %v", tableName, err)
		}
	}
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

	toolsFile = tests.AddMySQLPrebuiltToolConfig(t, toolsFile)

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
	tests.RunMySQLListTablesTest(t, MySQLDatabase, tableNameParam, tableNameAuth)
	tests.RunMySQLListActiveQueriesTest(t, ctx, pool)
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
>>>>>>> c696e8066b (init implementation)
}

func runMySQLListTablesMissingIndex(t *testing.T, ctx context.Context, pool *sql.DB) {
	type listDetails struct {
		TableSchema string `json:"table_schema"`
		TableName   string `json:"table_name"`
	}

	// bunch of wanted
	nonUniqueKeyTableName := "t03_non_unqiue_key_table"
	noKeyTableName := "t04_no_key_table"
	nonUniqueKeyTableWant := listDetails{
		TableSchema: MySQLDatabase,
		TableName:   nonUniqueKeyTableName,
	}
	noKeyTableWant := listDetails{
		TableSchema: MySQLDatabase,
		TableName:   noKeyTableName,
	}

	invokeTcs := []struct {
		name                 string
		requestBody          io.Reader
		newTableName         string
		newTablePrimaryKey   bool
		newTableUniqueKey    bool
		newTableNonUniqueKey bool
		wantStatusCode       int
		want                 any
	}{
		{
			name:                 "invoke list_tables_missing_index when nothing to be found",
			requestBody:          bytes.NewBufferString(`{}`),
			newTableName:         "",
			newTablePrimaryKey:   false,
			newTableUniqueKey:    false,
			newTableNonUniqueKey: false,
			wantStatusCode:       http.StatusOK,
			want:                 []listDetails(nil),
		},
		{
			name:                 "invoke list_tables_missing_index pk table will not show",
			requestBody:          bytes.NewBufferString(`{}`),
			newTableName:         "t01",
			newTablePrimaryKey:   true,
			newTableUniqueKey:    false,
			newTableNonUniqueKey: false,
			wantStatusCode:       http.StatusOK,
			want:                 []listDetails(nil),
		},
		{
			name:                 "invoke list_tables_missing_index uk table will not show",
			requestBody:          bytes.NewBufferString(`{}`),
			newTableName:         "t02",
			newTablePrimaryKey:   false,
			newTableUniqueKey:    true,
			newTableNonUniqueKey: false,
			wantStatusCode:       http.StatusOK,
			want:                 []listDetails(nil),
		},
		{
			name:                 "invoke list_tables_missing_index non-unique key only table will show",
			requestBody:          bytes.NewBufferString(`{}`),
			newTableName:         nonUniqueKeyTableName,
			newTablePrimaryKey:   false,
			newTableUniqueKey:    false,
			newTableNonUniqueKey: true,
			wantStatusCode:       http.StatusOK,
			want:                 []listDetails{nonUniqueKeyTableWant},
		},
		{
			name:                 "invoke list_tables_missing_index table with no key at all will show",
			requestBody:          bytes.NewBufferString(`{}`),
			newTableName:         noKeyTableName,
			newTablePrimaryKey:   false,
			newTableUniqueKey:    false,
			newTableNonUniqueKey: false,
			wantStatusCode:       http.StatusOK,
			want:                 []listDetails{nonUniqueKeyTableWant, noKeyTableWant},
		},
		{
			name:                 "invoke list_tables_missing_index table w/ both pk & uk will not show",
			requestBody:          bytes.NewBufferString(`{}`),
			newTableName:         "t05",
			newTablePrimaryKey:   true,
			newTableUniqueKey:    true,
			newTableNonUniqueKey: false,
			wantStatusCode:       http.StatusOK,
			want:                 []listDetails{nonUniqueKeyTableWant, noKeyTableWant},
		},
		{
			name:                 "invoke list_tables_missing_index table w/ uk & nk will not show",
			requestBody:          bytes.NewBufferString(`{}`),
			newTableName:         "t06",
			newTablePrimaryKey:   false,
			newTableUniqueKey:    true,
			newTableNonUniqueKey: true,
			wantStatusCode:       http.StatusOK,
			want:                 []listDetails{nonUniqueKeyTableWant, noKeyTableWant},
		},
		{
			name:                 "invoke list_tables_missing_index table w/ pk & nk will not show",
			requestBody:          bytes.NewBufferString(`{}`),
			newTableName:         "t07",
			newTablePrimaryKey:   true,
			newTableUniqueKey:    false,
			newTableNonUniqueKey: true,
			wantStatusCode:       http.StatusOK,
			want:                 []listDetails{nonUniqueKeyTableWant, noKeyTableWant},
		},
		{
			name:                 "invoke list_tables_missing_index with a non-exist database, nothing to show",
			requestBody:          bytes.NewBufferString(`{"table_schema": "non-exist-database"}`),
			newTableName:         "",
			newTablePrimaryKey:   false,
			newTableUniqueKey:    false,
			newTableNonUniqueKey: false,
			wantStatusCode:       http.StatusOK,
			want:                 []listDetails(nil),
		},
		{
			name:                 "invoke list_tables_missing_index with the right database, show everything",
			requestBody:          bytes.NewBufferString(fmt.Sprintf(`{"table_schema": "%s"}`, MySQLDatabase)),
			newTableName:         "",
			newTablePrimaryKey:   false,
			newTableUniqueKey:    false,
			newTableNonUniqueKey: false,
			wantStatusCode:       http.StatusOK,
			want:                 []listDetails{nonUniqueKeyTableWant, noKeyTableWant},
		},
		{
			name:                 "invoke list_tables_missing_index with limited output",
			requestBody:          bytes.NewBufferString(`{"limit": 1}`),
			newTableName:         "",
			newTablePrimaryKey:   false,
			newTableUniqueKey:    false,
			newTableNonUniqueKey: false,
			wantStatusCode:       http.StatusOK,
			want:                 []listDetails{nonUniqueKeyTableWant},
		},
	}

	var cleanups []func()
	defer func() {
		for i := len(cleanups) - 1; i >= 0; i-- {
			cleanups[i]()
		}
	}()

	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			if tc.newTableName != "" {
				cleanup := createTableHelper(t, tc.newTableName, tc.newTablePrimaryKey, tc.newTableUniqueKey, tc.newTableNonUniqueKey, ctx, pool)
				cleanups = append(cleanups, cleanup)
			}

			const api = "http://127.0.0.1:5000/api/tool/list_tables_missing_index/invoke"
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
			var details []listDetails
			if err := json.Unmarshal([]byte(resultString), &details); err != nil {
				t.Fatalf("failed to unmarshal nested listDetails string: %v", err)
			}
			got = details

			if diff := cmp.Diff(tc.want, got, cmp.Comparer(func(a, b listDetails) bool {
				return a.TableSchema == b.TableSchema && a.TableName == b.TableName
			})); diff != "" {
				t.Errorf("Unexpected result: got %#v, want: %#v", got, tc.want)
			}
		})
	}
}
