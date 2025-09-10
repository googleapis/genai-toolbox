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
	"reflect"
	"regexp"
	"sort"
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
	MySQLListTablesToolKind = "mysql-list-tables"
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
}

func runMySQLListTablesTest(t *testing.T, tableNameParam, tableNameAuth string) {
	// TableNameParam columns to construct want.
	paramTableColumns := `[{"data_type": "int", "column_name": "id", "column_comment": "", "column_default": null, "is_not_nullable": 1, "ordinal_position": 1}, {"data_type": "varchar(255)", "column_name": "name", "column_comment": "", "column_default": null, "is_not_nullable": 0, "ordinal_position": 2}]`

	// TableNameAuth columns to construct want.
	authTableColumns := `[{"data_type": "varchar(255)", "column_name": "email", "column_comment": "", "column_default": null, "is_not_nullable": 0, "ordinal_position": 3}, {"data_type": "int", "column_name": "id", "column_comment": "", "column_default": null, "is_not_nullable": 1, "ordinal_position": 1}, {"data_type": "varchar(255)", "column_name": "name", "column_comment": "", "column_default": null, "is_not_nullable": 0, "ordinal_position": 2}]`

	const (
		// Template to construct detailed output want
		detailedObjectDetailsTemplate = `{"owner": null, "columns": %s, "comment": "", "indexes": [{"is_unique": 1, "index_name": "PRIMARY", "is_primary": 1, "index_columns": ["id"]}], "triggers": [], "constraints": [{"constraint_name": "PRIMARY", "constraint_type": "PRIMARY KEY", "constraint_columns": ["id"], "constraint_definition": "", "foreign_key_referenced_table": null, "foreign_key_referenced_columns": null}], "object_name": "%s", "object_type": "TABLE", "schema_name": "%s"}`
		detailedObjectTemplate = `{"object_name": "%s", "schema_name": "%s", "object_details": %q}`

		// Template to construct simple output want
		simpleObjectDetailsTemplate = `{"name": "%s"}`
		simpleObjectTemplate = `{"object_name": "%s", "schema_name": "%s", "object_details": %q}`
	)

    // Helper to build JSON for detailed want.
    getDetailedWant := func(tableName, columnJSON string) string {
        objectDetailsContent := fmt.Sprintf(detailedObjectDetailsTemplate, columnJSON, tableName, MySQLDatabase)
        return fmt.Sprintf(detailedObjectTemplate, tableName, MySQLDatabase, objectDetailsContent)
    }

    // Helper to build template for simple want.
    getSimpleWant := func(tableName string) string {
        objectDetailsContent := fmt.Sprintf(simpleObjectDetailsTemplate, tableName)
        return fmt.Sprintf(simpleObjectTemplate, tableName, MySQLDatabase, objectDetailsContent)
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
            requestBody:    bytes.NewBuffer([]byte(fmt.Sprintf(`{"table_names": "%s,non_existent_table"}`, tableNameAuth))),
            wantStatusCode: http.StatusOK,
            want:           fmt.Sprintf("[%s]", getDetailedWant(tableNameAuth, authTableColumns)),
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
                t.Fatalf("response status code is not 200, got %d: %s", resp.StatusCode, string(bodyBytes))
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
                    t.Fatalf("'result' is not a JSON-encoded string: %s", err)
                }

                var got, want []any

                if err := json.Unmarshal([]byte(resultString), &got); err != nil {
                    t.Fatalf("failed to unmarshal actual result string: %v", err)
                }
                if err := json.Unmarshal([]byte(tc.want), &want); err != nil {
                    t.Fatalf("failed to unmarshal expected want string: %v", err)
                }

                sort.SliceStable(got, func(i, j int) bool {
                    return fmt.Sprintf("%v", got[i]) < fmt.Sprintf("%v", got[j])
                })
                sort.SliceStable(want, func(i, j int) bool {
                    return fmt.Sprintf("%v", want[i]) < fmt.Sprintf("%v", want[j])
                })

                if !reflect.DeepEqual(got, want) {
                    t.Errorf("Unexpected result: got  %#v, want: %#v", got, want)
                }
            }
        })
    }
}
