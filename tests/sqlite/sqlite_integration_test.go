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

package sqlite

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
	SQLiteSourceKind = "sqlite"
	SQLiteToolKind   = "sqlite-sql"
	SQLiteDatabase   = os.Getenv("SQLITE_DATABASE")
)

func getSQLiteVars(t *testing.T) map[string]any {
	return map[string]any{
		"kind":     SQLiteSourceKind,
		"database": SQLiteDatabase,
	}
}

func initSQLiteDb(t *testing.T, sqliteDb string) (*sql.DB, func(t *testing.T), string, error) {
	if sqliteDb == "" {
		// Create a temporary database file
		tmpFile, err := os.CreateTemp("", "test-*.db")
		if err != nil {
			return nil, nil, "", fmt.Errorf("failed to create temp file: %v", err)
		}
		sqliteDb = tmpFile.Name()
	}

	// Open database connection
	db, err := sql.Open("sqlite", sqliteDb)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to open database: %v", err)
	}

	cleanup := func(t *testing.T) {
		if err := os.Remove(sqliteDb); err != nil {
			t.Errorf("Failed to remove test database: %s", err)
		}
	}

	return db, cleanup, sqliteDb, nil
}

// setupSQLiteTestDB creates a temporary SQLite database for testing
func setupSQLiteTestDB(t *testing.T, ctx context.Context, db *sql.DB, createStatement string, insertStatement string, tableName string, params []any) {
	// Create test table
	_, err := db.ExecContext(ctx, createStatement)
	if err != nil {
		t.Fatalf("unable to connect to create test table %s: %s", tableName, err)
	}

	_, err = db.ExecContext(ctx, insertStatement, params...)
	if err != nil {
		t.Fatalf("unable to insert test data: %s", err)
	}
}

func getSQLiteParamToolInfo(tableName string) (string, string, string, string, string, string, []any) {
	createStatement := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (id INTEGER PRIMARY KEY, name TEXT);", tableName)
	insertStatement := fmt.Sprintf("INSERT INTO %s (name) VALUES (?), (?), (?), (?);", tableName)
	toolStatement := fmt.Sprintf("SELECT * FROM %s WHERE id = ? OR name = ?;", tableName)
	idToolStatement := fmt.Sprintf("SELECT * FROM %s WHERE id = ?;", tableName)
	nameToolStatement := fmt.Sprintf("SELECT * FROM %s WHERE name = ?;", tableName)
	arrayToolStatement := fmt.Sprintf("SELECT * FROM %s WHERE id = ANY({{.idArray}}) AND name = ANY({{.nameArray}});", tableName)
	params := []any{"Alice", "Jane", "Sid", nil}
	return createStatement, insertStatement, toolStatement, idToolStatement, nameToolStatement, arrayToolStatement, params
}

func getSQLiteAuthToolInfo(tableName string) (string, string, string, []any) {
	createStatement := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (id INTEGER PRIMARY KEY, name TEXT NOT NULL, email TEXT)", tableName)
	insertStatement := fmt.Sprintf("INSERT INTO %s (name, email) VALUES (?, ?), (?,?) RETURNING id, name, email;", tableName)
	toolStatement := fmt.Sprintf("SELECT name FROM %s WHERE email = ?;", tableName)
	params := []any{"Alice", tests.ServiceAccountEmail, "Jane", "janedoe@gmail.com"}
	return createStatement, insertStatement, toolStatement, params
}

func getSQLiteTmplToolStatement() (string, string) {
	tmplSelectCombined := "SELECT * FROM {{.tableName}} WHERE id = ?"
	tmplSelectFilterCombined := "SELECT * FROM {{.tableName}} WHERE {{.columnFilter}} = ?"
	return tmplSelectCombined, tmplSelectFilterCombined
}

func TestSQLiteToolEndpoint(t *testing.T) {
	db, teardownDb, sqliteDb, err := initSQLiteDb(t, SQLiteDatabase)
	if err != nil {
		t.Fatal(err)
	}
	defer teardownDb(t)
	defer db.Close()

	sourceConfig := getSQLiteVars(t)
	sourceConfig["database"] = sqliteDb
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	// create table name with UUID
	tableNameParam := "param_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableNameAuth := "auth_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableNameTemplateParam := "template_param_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")

	// set up data for param tool
	createParamTableStmt, insertParamTableStmt, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, paramTestParams := getSQLiteParamToolInfo(tableNameParam)
	setupSQLiteTestDB(t, ctx, db, createParamTableStmt, insertParamTableStmt, tableNameParam, paramTestParams)

	// set up data for auth tool
	createAuthTableStmt, insertAuthTableStmt, authToolStmt, authTestParams := getSQLiteAuthToolInfo(tableNameAuth)
	setupSQLiteTestDB(t, ctx, db, createAuthTableStmt, insertAuthTableStmt, tableNameAuth, authTestParams)

	// Write config into a file and pass it to command
	toolsFile := tests.GetToolsConfig(sourceConfig, SQLiteToolKind, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, authToolStmt)
	tmplSelectCombined, tmplSelectFilterCombined := getSQLiteTmplToolStatement()
	toolsFile = tests.AddTemplateParamConfig(t, toolsFile, SQLiteToolKind, tmplSelectCombined, tmplSelectFilterCombined, "")

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
	select1Want := "[{\"1\":1}]"
	failInvocationWant := `{"jsonrpc":"2.0","id":"invoke-fail-tool","result":{"content":[{"type":"text","text":"unable to execute query: SQL logic error: near \"SELEC\": syntax error (1)"}],"isError":true}}`
	toolInvokeConfig := tests.NewInvokeTestConfig(
		tests.WithInvoketestSelect1Want(select1Want),
		tests.WithDisableArray(),
	)
	mcpConfig := tests.NewMCPTestConfig(tests.WithFailInvocationWant(failInvocationWant))
	templateParamTestConfig := tests.NewTemplateParameterTestConfig()

	// Run tests
	tests.RunToolGetTest(t)
	tests.RunToolInvokeTest(t, toolInvokeConfig)
	tests.RunMCPToolCallMethod(t, mcpConfig)
	tests.RunToolInvokeWithTemplateParameters(t, tableNameTemplateParam, templateParamTestConfig)
}
