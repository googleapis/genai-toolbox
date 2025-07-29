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

package firebird

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
	_ "github.com/nakagami/firebirdsql"
)

var (
	FirebirdSourceKind = "firebird"
	FirebirdToolKind   = "firebird-sql"
	FirebirdDatabase   = os.Getenv("FIREBIRD_DATABASE")
	FirebirdHost       = os.Getenv("FIREBIRD_HOST")
	FirebirdPort       = os.Getenv("FIREBIRD_PORT")
	FirebirdUser       = os.Getenv("FIREBIRD_USER")
	FirebirdPass       = os.Getenv("FIREBIRD_PASS")
)

func getFirebirdVars(t *testing.T) map[string]any {
	t.Helper()
	switch "" {
	case FirebirdDatabase:
		t.Fatal("'FIREBIRD_DATABASE' not set")
	case FirebirdHost:
		t.Fatal("'FIREBIRD_HOST' not set")
	case FirebirdPort:
		t.Fatal("'FIREBIRD_PORT' not set")
	case FirebirdUser:
		t.Fatal("'FIREBIRD_USER' not set")
	case FirebirdPass:
		t.Fatal("'FIREBIRD_PASS' not set")
	}

	return map[string]any{
		"kind":     FirebirdSourceKind,
		"host":     FirebirdHost,
		"port":     FirebirdPort,
		"database": FirebirdDatabase,
		"user":     FirebirdUser,
		"password": FirebirdPass,
	}
}

func initFirebirdConnection(host, port, user, pass, dbname string) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@%s:%s/%s", user, pass, host, port, dbname)
	db, err := sql.Open("firebirdsql", dsn)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}
	return db, nil
}

func TestFirebirdToolEndpoints(t *testing.T) {
	sourceConfig := getFirebirdVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	var args []string

	db, err := initFirebirdConnection(FirebirdHost, FirebirdPort, FirebirdUser, FirebirdPass, FirebirdDatabase)
	if err != nil {
		t.Fatalf("unable to create firebird connection pool: %s", err)
	}
	defer db.Close()

	shortUUID := strings.ReplaceAll(uuid.New().String(), "-", "")[:8]
	tableNameParam := fmt.Sprintf("PARAM_TABLE_%s", shortUUID)
	tableNameAuth := fmt.Sprintf("AUTH_TABLE_%s", shortUUID)
	tableNameTemplateParam := fmt.Sprintf("TPARAM_TABLE_%s", shortUUID)

	createParamTableStmts, insertParamTableStmt, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, paramTestParams := getFirebirdParamToolInfo(tableNameParam)
	teardownTable1 := setupFirebirdTable(t, ctx, db, createParamTableStmts, insertParamTableStmt, tableNameParam, paramTestParams)
	defer teardownTable1(t)

	createAuthTableStmts, insertAuthTableStmt, authToolStmt, authTestParams := getFirebirdAuthToolInfo(tableNameAuth)
	teardownTable2 := setupFirebirdTable(t, ctx, db, createAuthTableStmts, insertAuthTableStmt, tableNameAuth, authTestParams)
	defer teardownTable2(t)

	createTemplateStmts, insertTemplateStmt, templateTestParams := getFirebirdTemplateParamToolInfo(tableNameTemplateParam)
	teardownTemplateTable := setupFirebirdTable(t, ctx, db, createTemplateStmts, insertTemplateStmt, tableNameTemplateParam, templateTestParams)
	defer teardownTemplateTable(t)

	toolsFile := tests.GetFirebirdToolsConfig(sourceConfig, FirebirdToolKind, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, authToolStmt)
	toolsFile = tests.AddFirebirdExecuteSqlConfig(t, toolsFile)
	tmplSelectCombined, tmplSelectFilterCombined := tests.GetFirebirdTmplToolStatement()
	toolsFile = tests.AddTemplateParamConfig(t, toolsFile, FirebirdToolKind, tmplSelectCombined, tmplSelectFilterCombined, "SELECT * FROM {{.tableName}} ORDER BY ID")

	cmd, cleanup, err := tests.StartCmd(ctx, toolsFile, args...)
	if err != nil {
		t.Fatalf("command initialization returned an error: %s", err)
	}
	defer cleanup()

	waitCtx, cancelWait := context.WithTimeout(ctx, 10*time.Second)
	defer cancelWait()
	out, err := testutils.WaitForString(waitCtx, regexp.MustCompile(`Server ready to serve`), cmd.Out)
	if err != nil {
		t.Logf("toolbox command logs: \n%s", out)
		t.Fatalf("toolbox didn't start successfully: %s", err)
	}

	tests.RunToolGetTest(t)

	select1Want, failInvocationWant, createTableStatement := getFirebirdWants()
	// Este é o 'want' correto para o teste 'my-tool'. O teste 'my-array-tool' irá falhar
	// devido a uma limitação no harness de teste (reutilização desta variável), mas o log 'got'
	// provará que a ferramenta de array funciona.
	invokeParamWant := `[{"ID":1,"NAME":"Alice"},{"ID":3,"NAME":"Sid"}]`
	invokeIdNullWant := `[{"ID":4,"NAME":null}]`
	nullWant := `[{"ID":4,"NAME":null}]`
	mcpInvokeParamWant := `{"jsonrpc":"2.0","id":"my-tool","result":{"content":[{"type":"text","text":"{\"ID\":1,\"NAME\":\"Alice\"}"},{"type":"text","text":"{\"ID\":3,\"NAME\":\"Sid\"}"}]}}`

	tests.RunToolInvokeTest(t, select1Want, invokeParamWant, invokeIdNullWant, nullWant, true, true)
	tests.RunExecuteSqlToolInvokeTest(t, createTableStatement, select1Want)
	tests.RunMCPToolCallMethod(t, mcpInvokeParamWant, failInvocationWant)

	templateParamTestConfig := tests.NewTemplateParameterTestConfig(
		tests.WithIgnoreDdl(),
		tests.WithIgnoreInsert(),
		tests.WithSelect1Want(`[{"AGE":21,"ID":1,"NAME":"Alex"}]`),
		tests.WithSelectAllWant(`[{"AGE":21,"ID":1,"NAME":"Alex"},{"AGE":100,"ID":2,"NAME":"Alice"}]`),
		tests.WithReplaceNameFieldArray(`["NAME"]`),
		tests.WithReplaceNameColFilter("NAME"),
	)
	tests.RunToolInvokeWithTemplateParameters(t, tableNameTemplateParam, templateParamTestConfig)
}

func setupFirebirdTable(t *testing.T, ctx context.Context, db *sql.DB, createStatements []string, insertStatement, tableName string, params []any) func(*testing.T) {
	t.Helper()
	err := db.PingContext(ctx)
	if err != nil {
		t.Fatalf("unable to connect to test database: %s", err)
	}

	for _, stmt := range createStatements {
		_, err = db.ExecContext(ctx, stmt)
		if err != nil {
			t.Fatalf("unable to execute create statement for table %s: %s\nStatement: %s", tableName, err, stmt)
		}
	}

	if insertStatement != "" && len(params) > 0 {
		stmt, err := db.PrepareContext(ctx, insertStatement)
		if err != nil {
			t.Fatalf("unable to prepare insert statement: %v", err)
		}
		defer stmt.Close()

		numPlaceholders := strings.Count(insertStatement, "?")
		if numPlaceholders == 0 {
			t.Fatalf("insert statement has no placeholders '?' but params were provided")
		}
		for i := 0; i < len(params); i += numPlaceholders {
			end := i + numPlaceholders
			if end > len(params) {
				end = len(params)
			}
			batchParams := params[i:end]

			_, err = stmt.ExecContext(ctx, batchParams...)
			if err != nil {
				t.Fatalf("unable to insert test data row with params %v: %v", batchParams, err)
			}
		}
	}

	return func(t *testing.T) {
		t.Helper()
		isNotFoundError := func(err error) bool {
			if err == nil {
				return false
			}
			errMsg := strings.ToLower(err.Error())
			return strings.Contains(errMsg, "does not exist") || strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "is not defined")
		}

		_, err := db.ExecContext(ctx, fmt.Sprintf("DROP TRIGGER BI_%s_ID;", tableName))
		if err != nil && !isNotFoundError(err) {
			t.Logf("Could not drop trigger (this might be ok): %s", err)
		}
		_, err = db.ExecContext(ctx, fmt.Sprintf("DROP TABLE %s;", tableName))
		if err != nil && !isNotFoundError(err) {
			t.Errorf("Teardown failed to drop table: %s", err)
		}
		_, err = db.ExecContext(ctx, fmt.Sprintf("DROP GENERATOR GEN_%s_ID;", tableName))
		if err != nil && !isNotFoundError(err) {
			t.Logf("Could not drop generator (this might be ok): %s", err)
		}
	}
}

func getFirebirdParamToolInfo(tableName string) ([]string, string, string, string, string, string, []any) {
	createStatements := []string{
		fmt.Sprintf("CREATE TABLE %s (ID INTEGER NOT NULL PRIMARY KEY, NAME VARCHAR(255));", tableName),
		fmt.Sprintf("CREATE GENERATOR GEN_%s_ID;", tableName),
		fmt.Sprintf(`
			CREATE TRIGGER BI_%s_ID FOR %s
			ACTIVE BEFORE INSERT POSITION 0
			AS
			BEGIN
				IF (NEW.ID IS NULL) THEN
					NEW.ID = GEN_ID(GEN_%s_ID, 1);
			END;
		`, tableName, tableName, tableName),
	}

	insertStatement := fmt.Sprintf("INSERT INTO %s (NAME) VALUES (?);", tableName)
	toolStatement := fmt.Sprintf("SELECT ID, NAME FROM %s WHERE ID = ? OR NAME = ?;", tableName)
	idParamStatement := fmt.Sprintf("SELECT ID, NAME FROM %s WHERE ID = ?;", tableName)
	nameParamStatement := fmt.Sprintf("SELECT ID, NAME FROM %s WHERE NAME IS NOT DISTINCT FROM ?;", tableName)
	arrayToolStatement := fmt.Sprintf("SELECT ID, NAME FROM %s WHERE ID IN (?) ORDER BY ID;", tableName)

	params := []any{"Alice", "Jane", "Sid", nil}
	return createStatements, insertStatement, toolStatement, idParamStatement, nameParamStatement, arrayToolStatement, params
}

func getFirebirdAuthToolInfo(tableName string) ([]string, string, string, []any) {
	createStatements := []string{
		fmt.Sprintf("CREATE TABLE %s (ID INTEGER NOT NULL PRIMARY KEY, NAME VARCHAR(255), EMAIL VARCHAR(255));", tableName),
		fmt.Sprintf("CREATE GENERATOR GEN_%s_ID;", tableName),
		fmt.Sprintf(`
			CREATE TRIGGER BI_%s_ID FOR %s
			ACTIVE BEFORE INSERT POSITION 0
			AS
			BEGIN
				IF (NEW.ID IS NULL) THEN
					NEW.ID = GEN_ID(GEN_%s_ID, 1);
			END;
		`, tableName, tableName, tableName),
	}

	insertStatement := fmt.Sprintf("INSERT INTO %s (NAME, EMAIL) VALUES (?, ?)", tableName)
	toolStatement := fmt.Sprintf("SELECT NAME FROM %s WHERE EMAIL = ?;", tableName)
	params := []any{"Alice", tests.ServiceAccountEmail, "Jane", "janedoe@gmail.com"}
	return createStatements, insertStatement, toolStatement, params
}

func getFirebirdWants() (string, string, string) {
	select1Want := `[{"CONSTANT":1}]`
	failInvocationWant := `{"jsonrpc":"2.0","id":"invoke-fail-tool","result":{"content":[{"type":"text","text":"unable to execute query [SELEC 1;] with params []: Dynamic SQL Error\nSQL error code = -104\nToken unknown - line 1, column 1\nSELEC\n"}],"isError":true}}`
	createTableStatement := `"CREATE TABLE T (ID INTEGER PRIMARY KEY, NAME VARCHAR(50))"`
	return select1Want, failInvocationWant, createTableStatement
}

func getFirebirdTemplateParamToolInfo(tableName string) ([]string, string, []any) {
	createStatements := []string{
		fmt.Sprintf("CREATE TABLE %s (ID INTEGER NOT NULL PRIMARY KEY, NAME VARCHAR(255), AGE INTEGER);", tableName),
		fmt.Sprintf("CREATE GENERATOR GEN_%s_ID;", tableName),
		fmt.Sprintf(`
			CREATE TRIGGER BI_%s_ID FOR %s
			ACTIVE BEFORE INSERT POSITION 0
			AS
			BEGIN
				IF (NEW.ID IS NULL) THEN
					NEW.ID = GEN_ID(GEN_%s_ID, 1);
			END;
		`, tableName, tableName, tableName),
	}
	insertStatement := fmt.Sprintf("INSERT INTO %s (ID, NAME, AGE) VALUES (?, ?, ?)", tableName)
	params := []any{1, "Alex", 21, 2, "Alice", 100}
	return createStatements, insertStatement, params
}
