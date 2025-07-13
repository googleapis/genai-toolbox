package duckdb

import (
	"context"
	"database/sql"
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
	dbPath     = "/tmp/hotel_test.db"
)

func getDuckDbVars() map[string]any {
	return map[string]any{
		"kind":       "duckdb",
		"dbFilePath": dbPath,
		"configurations": map[string]any{
			"access_mode": "READ_ONLY",
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

	createParamTableStmt, insertParamTableStmt, paramToolStmt, paramToolStmt2, arrayToolStmt, paramTestParams := tests.GetDuckDbParamToolInfo(tableNameParam)
	createAuthTableStmt, insertAuthTableStmt, authToolStmt, authTestParams := tests.GetDuckDbAuthToolInfo(tableNameAuth)
	setupDuckDb(t, createParamTableStmt, insertParamTableStmt, createAuthTableStmt, insertAuthTableStmt, paramTestParams, authTestParams)

	toolsFile := tests.GetToolsConfig(sourceConfig, DuckDbKind, paramToolStmt, paramToolStmt2, arrayToolStmt, authToolStmt)

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

	select1Want, failInvocationWant, _ := tests.GetDuckDbWants()

	invokeParamWant, invokeParamWantNull, mcpInvokeParamWant := tests.GetDuckDbInvokeParamWant()
	tests.RunToolInvokeTest(t, select1Want, invokeParamWant, invokeParamWantNull, false)
	tests.RunMCPToolCallMethod(t, mcpInvokeParamWant, failInvocationWant)
}
