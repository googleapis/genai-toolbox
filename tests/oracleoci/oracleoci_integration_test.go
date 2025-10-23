//go:build oracleoci
// +build oracleoci

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

package oracleoci

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
	OracleOCISourceKind = "oracle-oci"
	OracleToolKind      = "oracle-sql"
	OracleTnsAlias      = os.Getenv("ORACLE_TNS_ALIAS")
	OracleTnsAdmin      = os.Getenv("ORACLE_TNS_ADMIN")
	OracleUser          = os.Getenv("ORACLE_USER")
	OraclePass          = os.Getenv("ORACLE_PASS")
)

func getOracleOCIVars(t *testing.T) map[string]any {
	switch "" {
	case OracleTnsAlias:
		t.Fatal("'ORACLE_TNS_ALIAS' not set")
	case OracleUser:
		t.Fatal("'ORACLE_USER' not set")
	case OraclePass:
		t.Fatal("'ORACLE_PASS' not set")
	}

	config := map[string]any{
		"kind":     OracleOCISourceKind,
		"tnsAlias": OracleTnsAlias,
		"user":     OracleUser,
		"password": OraclePass,
	}

	// Only add tnsAdmin if it's set
	if OracleTnsAdmin != "" {
		config["tnsAdmin"] = OracleTnsAdmin
	}

	return config
}

// Initialize Oracle OCI connection using godror driver
func initOracleOCIConnection(ctx context.Context, user, pass, tnsAlias string) (*sql.DB, error) {
	// Godror connection string format: user/password@tnsalias
	connStr := fmt.Sprintf("%s/%s@%s", user, pass, tnsAlias)

	db, err := sql.Open("godror", connStr)
	if err != nil {
		return nil, fmt.Errorf("unable to open Oracle OCI connection: %w", err)
	}

	err = db.PingContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to ping Oracle OCI connection: %w", err)
	}

	return db, nil
}

func TestOracleOCISimpleToolEndpoints(t *testing.T) {
	sourceConfig := getOracleOCIVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	db, err := initOracleOCIConnection(ctx, OracleUser, OraclePass, OracleTnsAlias)
	if err != nil {
		t.Fatalf("unable to create Oracle OCI connection: %s", err)
	}
	defer db.Close()

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
	createStatement := fmt.Sprintf(`CREATE TABLE %s ("id" NUMBER GENERATED AS IDENTITY PRIMARY KEY, "name" VARCHAR2(255))`, tableName)

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
	arrayToolStatement := fmt.Sprintf(`SELECT * FROM %s WHERE "id" MEMBER OF :1 AND "name" MEMBER OF :2`, tableName)

	params := []any{"Alice", "Jane", "Sid", nil}

	return createStatement, insertStatement, toolStatement, idParamStatement, nameParamStatement, arrayToolStatement, params
}

func getOracleAuthToolInfo(tableName string) (string, string, string, []any) {
	createStatement := fmt.Sprintf(`CREATE TABLE %s ("id" NUMBER GENERATED AS IDENTITY PRIMARY KEY, "name" VARCHAR2(255), "email" VARCHAR2(255))`, tableName)

	insertStatement := fmt.Sprintf(`
		BEGIN
			INSERT INTO %s ("name", "email") VALUES (:1, :2);
			INSERT INTO %s ("name", "email") VALUES (:3, :4);
		END;`, tableName, tableName)

	toolStatement := fmt.Sprintf(`SELECT "name" FROM %s WHERE "email" = :1`, tableName)

	params := []any{"Alice", tests.ServiceAccountEmail, "Jane", "janedoe@gmail.com"}

	return createStatement, insertStatement, toolStatement, params
}

func dropAllUserTables(t *testing.T, ctx context.Context, db *sql.DB) {
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
