package duckdb

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
	DuckDbKind = "duckdb-sql"
	dbPath     = "/tmp/users.db"
)

func getDuckDbVars() map[string]any {
	return map[string]any{
		"kind":       "duckdb",
		"dbFilePath": dbPath,
		"configurations": map[string]any{
			"access_mode": "READ_WRITE",
		},
	}
}

func setupDuckDb(t *testing.T, createParamStmt, insertParamStmt, createAuthStmt, insertAuthStmt string, params []any, authparams []any) {
	// Remove any existing database file to ensure a clean state
	os.Remove(dbPath)

	// Open a connection to DuckDB
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("Failed to open DuckDB connection: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(createParamStmt, params...)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}
	_, err = db.Exec(createAuthStmt, authparams...)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	_, err = db.Exec(insertParamStmt, params...)
	if err != nil {
		t.Fatalf("Failed to insert initial data: %v", err)
	}
	_, err = db.Exec(insertAuthStmt, authparams...)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}
}
func TestDuckDb(t *testing.T) {
	sourceConfig := getDuckDbVars()
	var args []string
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	tableNameParam := "param_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableNameAuth := "auth_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableNameTemplateParam := "template_param_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")

	createParamTableStmt, insertParamTableStmt, paramToolStmt, paramToolStmt2, arrayToolStmt, paramTestParams := GetDuckDbParamToolInfo(tableNameParam)
	createAuthTableStmt, insertAuthTableStmt, authToolStmt, authTestParams := GetDuckDbAuthToolInfo(tableNameAuth)
	setupDuckDb(t, createParamTableStmt, insertParamTableStmt, createAuthTableStmt, insertAuthTableStmt, paramTestParams, authTestParams)

	toolsFile := tests.GetToolsConfig(sourceConfig, DuckDbKind, paramToolStmt, paramToolStmt2, arrayToolStmt, authToolStmt)
	tmplSelectCombined, tmplSelectFilterCombined := tests.GetPostgresSQLTmplToolStatement()
	toolsFile = tests.AddTemplateParamConfig(t, toolsFile, DuckDbKind, tmplSelectCombined, tmplSelectFilterCombined, "")
	defer os.Remove(dbPath)
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

	select1Want, failInvocationWant, _ := GetDuckDbWants()

	invokeParamWant, invokeParamWantNull, mcpInvokeParamWant := GetDuckDbInvokeParamWant()
	tests.RunToolInvokeTest(t, select1Want, invokeParamWant, invokeParamWantNull, false)
	tests.RunMCPToolCallMethod(t, mcpInvokeParamWant, failInvocationWant)
	tests.RunToolInvokeWithTemplateParameters(t, tableNameTemplateParam, tests.DuckDBTemplateParameterTestConfig())
}

func GetDuckDbParamToolInfo(tableName string) (string, string, string, string, string, []any) {
	createStatement := fmt.Sprintf("CREATE TABLE %s (id INTEGER PRIMARY KEY, name TEXT);", tableName)
	insertStatement := fmt.Sprintf("INSERT INTO %s (id, name) VALUES (1, $1), (2, $2), (3, $3), (4, $4);", tableName)
	toolStatement := fmt.Sprintf("SELECT * EXCLUDE (id) FROM %s WHERE id = $1 OR name = $2 order by id;", tableName)
	toolStatement2 := fmt.Sprintf("SELECT * FROM %s WHERE id = $1;", tableName)
	arrayToolStatement := fmt.Sprintf("SELECT * FROM %s WHERE id = ANY($1) AND name = ANY($2);", tableName)
	params := []any{"Alice", "Jane", "Sid", nil}
	return createStatement, insertStatement, toolStatement, toolStatement2, arrayToolStatement, params
}

// GetDuckDbAuthToolInfo returns statements and param of my-auth-tool for duckdb-sql kind
func GetDuckDbAuthToolInfo(tableName string) (string, string, string, []any) {
	createStatement := fmt.Sprintf("CREATE TABLE %s (id INTEGER PRIMARY KEY, name TEXT, email TEXT);", tableName)
	insertStatement := fmt.Sprintf("INSERT INTO %s (id, name, email) VALUES (1, $1, $2), (2, $3, $4)", tableName)
	toolStatement := fmt.Sprintf("SELECT name FROM %s WHERE email = $1;", tableName)
	params := []any{"Alice", tests.ServiceAccountEmail, "Jane", "janedoe@gmail.com"}
	return createStatement, insertStatement, toolStatement, params
}

func GetDuckDbWants() (string, string, string) {
	select1Want := "[{\"1\":1}]"
	failInvocationWant := `{"jsonrpc":"2.0","id":"invoke-fail-tool","result":{"content":[{"type":"text","text":"unable to execute query: Parser Error: syntax error at or near \"SELEC\""}],"isError":true}}`
	createTableStatement := `"CREATE TABLE t (id SERIAL PRIMARY KEY, name TEXT)"`
	return select1Want, failInvocationWant, createTableStatement
}

func GetDuckDbInvokeParamWant() (string, string, string) {
	invokeParamWant := "[{\"name\":\"Alice\"},{\"name\":\"Sid\"}]"
	invokeParamWantNull := "[{\"id\":4,\"name\":null}]"
	mcpInvokeParamWant := `{"jsonrpc":"2.0","id":"my-param-tool","result":{"content":[{"type":"text","text":"{\"name\":\"Alice\"}"},{"type":"text","text":"{\"name\":\"Sid\"}"}]}}`
	return invokeParamWant, invokeParamWantNull, mcpInvokeParamWant
}
