// Copyright © 2025, Oracle and/or its affiliates.

package oracle

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
	OracleSourceKind = "oracle"
	OracleToolKind   = "oracle-sql"
	OracleHost       = os.Getenv("ORACLE_HOST")
	OracleUser       = os.Getenv("ORACLE_USER")
	OraclePass       = os.Getenv("ORACLE_PASS")
	OracleServerName = os.Getenv("ORACLE_SERVER_NAME")
	OracleConnStr    = fmt.Sprintf(
		"%s:%s/%s", OracleHost, "1521", OracleServerName)
)

func getOracleVars(t *testing.T) map[string]any {
	switch "" {
	case OracleHost:
		t.Fatal("'ORACLE_HOST not set")
	case OracleUser:
		t.Fatal("'ORACLE_USER' not set")
	case OraclePass:
		t.Fatal("'ORACLE_PASS' not set")
	case OracleServerName:
		t.Fatal("'ORACLE_SERVER_NAME' not set")
	}

	return map[string]any{
		"kind":             OracleSourceKind,
		"connectionString": OracleConnStr,
		"user":             OracleUser,
		"password":         OraclePass,
	}
}

// Copied over from oracle.go
func initOracleConnection(ctx context.Context, user, pass, connStr string) (*sql.DB, error) {
	fullConnStr := fmt.Sprintf("oracle://%s:%s@%s", user, pass, connStr)

	db, err := sql.Open("oracle", fullConnStr)
	if err != nil {
		return nil, fmt.Errorf("unable to open Oracle connection: %w", err)
	}

	err = db.PingContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to ping Oracle connection: %w", err)
	}

	return db, nil
}

func TestOracleSimpleToolEndpoints(t *testing.T) {
	sourceConfig := getOracleVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	db, err := initOracleConnection(ctx, OracleUser, OraclePass, OracleConnStr)
	if err != nil {
		t.Fatalf("unable to create Oracle connection pool: %s", err)
	}

	dropAllUserTables(t, ctx, db)

	// create table name with UUID
	tableNameParam := "param_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableNameAuth := "auth_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableNameTemplateParam := "template_param_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")

	// set up data for param tool
	createParamTableStmt, insertParamTableStmt, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, paramTestParams := getOracleParamToolInfo(tableNameParam)
	teardownTable1 := setupOracleTable(t, ctx, db, createParamTableStmt, insertParamTableStmt, tableNameParam, paramTestParams)
	defer teardownTable1(t)

	// set up data for auth tool
	createAuthTableStmt, insertAuthTableStmt, authToolStmt, authTestParams := getOracleAuthToolInfo(tableNameAuth)
	teardownTable2 := setupOracleTable(t, ctx, db, createAuthTableStmt, insertAuthTableStmt, tableNameAuth, authTestParams)
	defer teardownTable2(t)

	// Write config into a file and pass it to command
	toolsFile := tests.GetToolsConfig(sourceConfig, OracleToolKind, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, authToolStmt)
	toolsFile = tests.AddExecuteSqlConfig(t, toolsFile, "oracle-execute-sql")
	tmplSelectCombined, tmplSelectFilterCombined := tests.GetMySQLTmplToolStatement()
	toolsFile = tests.AddTemplateParamConfig(t, toolsFile, OracleToolKind, tmplSelectCombined, tmplSelectFilterCombined, "")

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
	mcpMyFailToolWant := `{"jsonrpc":"2.0","id":"invoke-fail-tool","result":{"content":[{"type":"text","text":"unable to execute query: ORA-00900: invalid SQL statement\n error occur at position: 0"}],"isError":true}}`
	createTableStatement := `"CREATE TABLE t (id NUMBER GENERATED AS IDENTITY PRIMARY KEY, name VARCHAR2(255))"`
	mcpSelect1Want := `{"jsonrpc":"2.0","id":"invoke my-auth-required-tool","result":{"content":[{"type":"text","text":"{\"1\":1}"}]}}`

	// Run tests
	tests.RunToolGetTest(t)
	tests.RunToolInvokeTest(t, select1Want,
		tests.DisableArrayTest(),
	)
	tests.RunMCPToolCallMethod(t, mcpMyFailToolWant, mcpSelect1Want)
	tests.RunExecuteSqlToolInvokeTest(t, createTableStatement, select1Want)
	tests.RunToolInvokeWithTemplateParameters(t, tableNameTemplateParam)
}

func setupOracleTable(t *testing.T, ctx context.Context, pool *sql.DB, createStatement, insertStatement, tableName string, params []any) func(*testing.T) {
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
		_, err = pool.ExecContext(ctx, fmt.Sprintf("DROP TABLE %s", tableName))
		if err != nil {
			t.Errorf("Teardown failed: %s", err)
		}
	}
}

func getOracleParamToolInfo(tableName string) (string, string, string, string, string, string, []any) {
	// Use GENERATED AS IDENTITY for auto-incrementing primary keys.
	// VARCHAR2 is the standard string type in Oracle.
	createStatement := fmt.Sprintf(`CREATE TABLE %s ("id" NUMBER GENERATED AS IDENTITY PRIMARY KEY, "name" VARCHAR2(255))`, tableName)

	// MODIFIED: Use a PL/SQL block for multiple inserts
	insertStatement := fmt.Sprintf(`
		BEGIN
			INSERT INTO %s ("name") VALUES (:1);
			INSERT INTO %s ("name") VALUES (:2);
			INSERT INTO %s ("name") VALUES (:3);
			INSERT INTO %s ("name") VALUES (:4);
		END;`, tableName, tableName, tableName, tableName)

	toolStatement := fmt.Sprintf(`SELECT * FROM %s WHERE "id" = :1 OR "name" = :2`, tableName)
	idParamStatement := fmt.Sprintf(`SELECT * FROM %s WHERE "id" = :1`, tableName)
	nameParamStatement := fmt.Sprintf(`SELECT * FROM %s WHERE "name" = :1`, tableName)

	// Oracle's equivalent for array parameters is using the 'MEMBER OF' operator
	// with a collection type defined in the database schema.
	arrayToolStatement := fmt.Sprintf(`SELECT * FROM %s WHERE "id" MEMBER OF :1 AND "name" MEMBER OF :2`, tableName)

	params := []any{"Alice", "Jane", "Sid", nil}

	return createStatement, insertStatement, toolStatement, idParamStatement, nameParamStatement, arrayToolStatement, params
}

// getOracleAuthToolInfo returns statements and params for my-auth-tool for Oracle SQL
func getOracleAuthToolInfo(tableName string) (string, string, string, []any) {
	createStatement := fmt.Sprintf(`CREATE TABLE %s ("id" NUMBER GENERATED AS IDENTITY PRIMARY KEY, "name" VARCHAR2(255), "email" VARCHAR2(255))`, tableName)

	// MODIFIED: Use a PL/SQL block for multiple inserts
	insertStatement := fmt.Sprintf(`
		BEGIN
			INSERT INTO %s ("name", "email") VALUES (:1, :2);
			INSERT INTO %s ("name", "email") VALUES (:3, :4);
		END;`, tableName, tableName)

	toolStatement := fmt.Sprintf(`SELECT "name" FROM %s WHERE "email" = :1`, tableName)

	params := []any{"Alice", tests.ServiceAccountEmail, "Jane", "janedoe@gmail.com"}

	return createStatement, insertStatement, toolStatement, params
}

// dropAllUserTables finds and drops all tables owned by the current user.
func dropAllUserTables(t *testing.T, ctx context.Context, db *sql.DB) {
	// Query for only the tables we know are created by this test suite.
	const query = `
		SELECT table_name FROM user_tables
		WHERE table_name LIKE 'param_table_%'
		   OR table_name LIKE 'auth_table_%'
		   OR table_name LIKE 'template_param_table_%'`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		t.Fatalf("failed to query for user tables: %v", err)
	}
	defer rows.Close()

	var tablesToDrop []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			t.Fatalf("failed to scan table name: %v", err)
		}
		tablesToDrop = append(tablesToDrop, tableName)
	}

	if err := rows.Err(); err != nil {
		t.Fatalf("error iterating over tables: %v", err)
	}

	for _, tableName := range tablesToDrop {
		_, err := db.ExecContext(ctx, fmt.Sprintf("DROP TABLE %s CASCADE CONSTRAINTS", tableName))
		if err != nil {
			t.Logf("failed to drop table %s: %v", tableName, err)
		}
	}
}
