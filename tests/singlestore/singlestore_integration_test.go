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

package singlestore

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
	SingleStoreSourceKind = "singlestore"
	SingleStoreToolKind   = "singlestore-sql"
	SingleStoreDatabase   = os.Getenv("SINGLESTORE_DATABASE")
	SingleStoreHost       = os.Getenv("SINGLESTORE_HOST")
	SingleStorePort       = os.Getenv("SINGLESTORE_PORT")
	SingleStoreUser       = os.Getenv("SINGLESTORE_USER")
	SingleStorePass       = os.Getenv("SINGLESTORE_PASSWORD")
)

func getSingleStoreVars(t *testing.T) map[string]any {
	switch "" {
	case SingleStoreDatabase:
		t.Fatal("'SINGLESTORE_DATABASE' not set")
	case SingleStoreHost:
		t.Fatal("'SINGLESTORE_HOST' not set")
	case SingleStorePort:
		t.Fatal("'SINGLESTORE_PORT' not set")
	case SingleStoreUser:
		t.Fatal("'SINGLESTORE_USER' not set")
	case SingleStorePass:
		t.Fatal("'SINGLESTORE_PASSWORD' not set")
	}

	return map[string]any{
		"kind":     SingleStoreSourceKind,
		"host":     SingleStoreHost,
		"port":     SingleStorePort,
		"database": SingleStoreDatabase,
		"user":     SingleStoreUser,
		"password": SingleStorePass,
	}
}

// getSingleStoreParamToolInfo returns statements and params for my-tool
func getSingleStoreParamToolInfo(tableName string) (string, string, string, string, string, string, []any) {
	createStatement := fmt.Sprintf("CREATE TABLE %s (id BIGINT NOT NULL PRIMARY KEY, name VARCHAR(255));", tableName)
	insertStatement := fmt.Sprintf("INSERT INTO %s (id, name) VALUES (?, ?), (?, ?), (?, ?), (?, ?);", tableName)
	toolStatement := fmt.Sprintf("SELECT * FROM %s WHERE id = ? OR name = ? ORDER BY id;", tableName)
	idParamStatement := fmt.Sprintf("SELECT * FROM %s WHERE id = ? ORDER BY id;", tableName)
	nameParamStatement := fmt.Sprintf("SELECT * FROM %s WHERE name = ? ORDER BY id;", tableName)
	// SingleStore doesn't support array parameters in IN clause unlike some other databases
	arrayToolStmt := ""
	insertParams := []any{1, "Alice", 2, "Jane", 3, "Sid", 4, nil}
	return createStatement, insertStatement, toolStatement, idParamStatement, nameParamStatement, arrayToolStmt, insertParams
}

// getSingleStoreAuthToolInfo returns statements and param of my-auth-tool
func getSingleStoreAuthToolInfo(tableName string) (string, string, string, []any) {
	createStatement := fmt.Sprintf("CREATE TABLE %s (id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY, name VARCHAR(255), email VARCHAR(255));", tableName)
	insertStatement := fmt.Sprintf("INSERT INTO %s (name, email) VALUES (?, ?), (?, ?)", tableName)
	toolStatement := fmt.Sprintf("SELECT name FROM %s WHERE email = ?;", tableName)
	params := []any{"Alice", tests.ServiceAccountEmail, "Jane", "janedoe@gmail.com"}
	return createStatement, insertStatement, toolStatement, params
}

// getSingleStoreTmplToolStatement returns statements and param for template parameter test cases for singlestore-sql kind
func getSingleStoreTmplToolStatement() (string, string) {
	tmplSelectCombined := "SELECT * FROM {{.tableName}} WHERE id = ?"
	tmplSelectFilterCombined := "SELECT * FROM {{.tableName}} WHERE {{.columnFilter}} = ?"
	return tmplSelectCombined, tmplSelectFilterCombined
}

// getSingleStoreWants return the expected wants for singlestore
func getSingleStoreWants() (string, string, string, string) {
	select1Want := "[{\"1\":1}]"
	mcpMyFailToolWant := `{"jsonrpc":"2.0","id":"invoke-fail-tool","result":{"content":[{"type":"text","text":"unable to execute query: Error 1064 (42000): You have an error in your SQL syntax; check the manual that corresponds to your MySQL server version for the right syntax to use near 'SELEC 1' at line 1"}],"isError":true}}`
	createTableStatement := `"CREATE TABLE t (id BIGINT PRIMARY KEY, name TEXT)"`
	mcpSelect1Want := `{"jsonrpc":"2.0","id":"invoke my-auth-required-tool","result":{"content":[{"type":"text","text":"{\"1\":1}"}]}}`
	return select1Want, mcpMyFailToolWant, createTableStatement, mcpSelect1Want
}

// setupSingleStoreTable creates and inserts data into a table of tool
// compatible with singlestore-sql tool
func setupSingleStoreTable(t *testing.T, ctx context.Context, pool *sql.DB, createStatement, insertStatement, tableName string, params []any) func(*testing.T) {
	err := pool.PingContext(ctx)
	if err != nil {
		t.Fatalf("unable to connect to test database: %s", err)
	}

	// Create table
	_, err = pool.QueryContext(ctx, createStatement)
	if err != nil {
		t.Fatalf("unable to create test table %s: %s", tableName, err)
	}

	// Insert test data
	_, err = pool.QueryContext(ctx, insertStatement, params...)
	if err != nil {
		t.Fatalf("unable to insert test data: %s", err)
	}

	return func(t *testing.T) {
		// tear down test
		_, err = pool.ExecContext(ctx, fmt.Sprintf("DROP TABLE %s;", tableName))
		if err != nil {
			t.Errorf("Teardown failed: %s", err)
		}
	}
}

func getSingleStoreToolsConfig(sourceConfig map[string]any, toolKind, paramToolStatement, idParamToolStmt, nameParamToolStmt, arrayToolStatement, authToolStatement string) map[string]any {
	toolsFile := tests.GetToolsConfig(sourceConfig, toolKind, paramToolStatement, idParamToolStmt, nameParamToolStmt, arrayToolStatement, authToolStatement)

	toolsMap, ok := toolsFile["tools"].(map[string]any)
	if !ok {
		return toolsFile
	}
	// Remove tools that are not supported
	delete(toolsMap, "my-array-tool")

	toolsFile["tools"] = toolsMap
	return toolsFile
}

// addSingleStoreExecuteSQLConfig gets the tools config for `singlestore-execute-sql`
func addSingleStoreExecuteSQLConfig(t *testing.T, config map[string]any) map[string]any {
	tools, ok := config["tools"].(map[string]any)
	if !ok {
		t.Fatalf("unable to get tools from config")
	}
	tools["my-exec-sql-tool"] = map[string]any{
		"kind":        "singlestore-execute-sql",
		"source":      "my-instance",
		"description": "Tool to execute sql",
	}
	tools["my-auth-exec-sql-tool"] = map[string]any{
		"kind":        "singlestore-execute-sql",
		"source":      "my-instance",
		"description": "Tool to execute sql",
		"authRequired": []string{
			"my-google-auth",
		},
	}
	config["tools"] = tools
	return config
}

// Copied over from singlestore.go
func initSingleStoreConnectionPool(host, port, user, pass, dbname string) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", user, pass, host, port, dbname)

	// Interact with the driver directly as you normally would
	pool, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}
	return pool, nil
}

func TestSingleStoreToolEndpoints(t *testing.T) {
	sourceConfig := getSingleStoreVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	pool, err := initSingleStoreConnectionPool(SingleStoreHost, SingleStorePort, SingleStoreUser, SingleStorePass, SingleStoreDatabase)
	if err != nil {
		t.Fatalf("unable to create SingleStore connection pool: %s", err)
	}

	// create table name with UUID
	tableNameParam := "param_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableNameAuth := "auth_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableNameTemplateParam := "template_param_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")

	// set up data for param tool
	createParamTableStmt, insertParamTableStmt, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, paramTestParams := getSingleStoreParamToolInfo(tableNameParam)
	teardownTable1 := setupSingleStoreTable(t, ctx, pool, createParamTableStmt, insertParamTableStmt, tableNameParam, paramTestParams)
	defer teardownTable1(t)

	// set up data for auth tool
	createAuthTableStmt, insertAuthTableStmt, authToolStmt, authTestParams := getSingleStoreAuthToolInfo(tableNameAuth)
	teardownTable2 := setupSingleStoreTable(t, ctx, pool, createAuthTableStmt, insertAuthTableStmt, tableNameAuth, authTestParams)
	defer teardownTable2(t)

	// Write config into a file and pass it to command
	toolsFile := getSingleStoreToolsConfig(sourceConfig, SingleStoreToolKind, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, authToolStmt)
	toolsFile = addSingleStoreExecuteSQLConfig(t, toolsFile)
	tmplSelectCombined, tmplSelectFilterCombined := getSingleStoreTmplToolStatement()
	toolsFile = tests.AddTemplateParamConfig(t, toolsFile, SingleStoreToolKind, tmplSelectCombined, tmplSelectFilterCombined, "")

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
	select1Want, mcpMyFailToolWant, createTableStatement, mcpSelect1Want := getSingleStoreWants()

	// Run tests
	tests.RunToolGetTest(t)
	tests.RunToolInvokeTest(t, select1Want, tests.DisableArrayTest())
	tests.RunMCPToolCallMethod(t, mcpMyFailToolWant, mcpSelect1Want)
	tests.RunExecuteSqlToolInvokeTest(t, createTableStatement, select1Want)
	tests.RunToolInvokeWithTemplateParameters(t, tableNameTemplateParam)
}
