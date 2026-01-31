// Copyright © 2025, Oracle and/or its affiliates.

package oracle

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"regexp"
	"strings"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/googleapis/genai-toolbox/internal/sources/oracle"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"
)


var (
	OracleSourceType = "oracle"
	OracleToolKind   = "oracle-execute-sql"
	OracleHost       = os.Getenv("ORACLE_HOST")
	OraclePort       = os.Getenv("ORACLE_PORT")
	OracleUser       = os.Getenv("ORACLE_USER")
	OraclePass       = os.Getenv("ORACLE_PASS")
	OracleServerName = os.Getenv("ORACLE_SERVER_NAME")
	OracleUseOCI     = os.Getenv("ORACLE_USE_OCI")
	OracleWalletLocation = os.Getenv("ORACLE_WALLET_LOCATION")
	OracleTnsAdmin   = os.Getenv("ORACLE_TNS_ADMIN")

	OracleConnStr    = fmt.Sprintf(
		"%s:%s/%s", OracleHost, "1521", OracleServerName) // Default port 1521??
)

func getOracleVars(t *testing.T) map[string]any {
	switch "" {
	case OracleHost:
		t.Skip("'ORACLE_HOST' not set, skipping integration test")
	case OracleUser:
		t.Skip("'ORACLE_USER' not set, skipping integration test")
	case OraclePass:
		t.Skip("'ORACLE_PASS' not set, skipping integration test")
	case OracleServerName:
		t.Skip("'ORACLE_SERVER_NAME' not set, skipping integration test")
	}

	return map[string]any{
		"kind":             OracleToolKind,
		"connectionString": OracleConnStr,
		"useOCI":           OracleUseOCI,
		"walletLocation":   OracleWalletLocation,
		"tnsAdmin":         OracleTnsAdmin,
		"host":             OracleHost,
		"port":             OraclePort,
		"service":          OracleServerName,
		"user":             OracleUser,
		"password":         OraclePass,
	}
}

// getOracleConfigFromEnv constructs an oracle.Config from environment variables.
func getOracleConfigFromEnv(t *testing.T) oracle.Config {
	t.Helper()
	vars := getOracleVars(t)
	
	port, err := strconv.Atoi(vars["port"].(string))
	if err != nil && vars["port"].(string) != "" {
		t.Fatalf("invalid ORACLE_PORT: %v", err)
	}

	useOCI, err := strconv.ParseBool(vars["useOCI"].(string))
	if err != nil && vars["useOCI"].(string) != "" {
		useOCI = false
	}

	return oracle.Config{
		Name:             "test-oracle-instance",
		Kind:             vars["kind"].(string),
		User:             vars["user"].(string),
		Password:         vars["password"].(string),
		Host:             vars["host"].(string),
		Port:             port,
		ServiceName:      vars["service"].(string),
		WalletLocation:   vars["walletLocation"].(string),
		TnsAdmin:         vars["tnsAdmin"].(string),
		UseOCI:           useOCI,
	}
}

// setOracleEnv sets Oracle-related environment variables for testing and returns a cleanup function.
func setOracleEnv(t *testing.T, host, user, password, service, port, connStr, tnsAlias, tnsAdmin, walletLocation string, useOCI bool) func() {
	t.Helper()

	original := map[string]string{
		"ORACLE_HOST":            os.Getenv("ORACLE_HOST"),
		"ORACLE_USER":            os.Getenv("ORACLE_USER"),
		"ORACLE_PASSWORD":        os.Getenv("ORACLE_PASSWORD"),
		"ORACLE_SERVICE":         os.Getenv("ORACLE_SERVICE"),
		"ORACLE_PORT":            os.Getenv("ORACLE_PORT"),
		"ORACLE_TNS_ADMIN":       os.Getenv("ORACLE_TNS_ADMIN"),
		"ORACLE_WALLET_LOCATION": os.Getenv("ORACLE_WALLET_LOCATION"),
		"ORACLE_USE_OCI":         os.Getenv("ORACLE_USE_OCI"),
	}

	os.Setenv("ORACLE_HOST", host)
	os.Setenv("ORACLE_USER", user)
	os.Setenv("ORACLE_PASSWORD", password)
	os.Setenv("ORACLE_SERVICE", service)
	os.Setenv("ORACLE_PORT", port)
	os.Setenv("ORACLE_TNS_ADMIN", tnsAdmin)
	os.Setenv("ORACLE_WALLET_LOCATION", walletLocation)
	os.Setenv("ORACLE_USE_OCI", fmt.Sprintf("%v", useOCI))

	return func() {
		for k, v := range original {
			os.Setenv(k, v)
		}
	}
}

// Copied over from oracle.go
// Copied over from oracle.go
func initOracleConnection(ctx context.Context, user, pass, connStr string, useOCI bool) (*sql.DB, error) {
	var driverName string
	var finalConnStr string

	if useOCI {
		driverName = "godror"
		// Build the full Oracle connection string for godror driver
		finalConnStr = fmt.Sprintf(`user="%s" password="%s" connectString="%s"`,
			user, pass, connStr)
	} else {
		driverName = "oracle"
		// Standard go-ora connection
		finalConnStr = fmt.Sprintf("oracle://%s:%s@%s",
			user, pass, connStr)
	}

	db, err := sql.Open(driverName, finalConnStr)
	if err != nil {
		return nil, fmt.Errorf("unable to open Oracle connection: %w", err)
	}

	err = db.PingContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to ping Oracle connection: %w", err)
	}

	return db, nil
}

// TestOracleSimpleToolEndpoints tests Oracle SQL tool endpoints
func TestOracleSimpleToolEndpoints(t *testing.T) {
	
	sourceConfig := getOracleVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	// Parse useOCI from config map to ensure consistency
	useOCI, _ := strconv.ParseBool(fmt.Sprintf("%v", sourceConfig["useOCI"]))

	db, err := initOracleConnection(ctx, OracleUser, OraclePass, OracleConnStr, useOCI)
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
	mcpMyFailToolWant := `{"jsonrpc":"2.0","id":"invoke-fail-tool","result":{"content":[{"type":"text","text":"unable to execute query: dpiStmt_execute: ORA-00900: invalid SQL statement"}],"isError":true}}`
	createTableStatement := `"CREATE TABLE t (id NUMBER GENERATED AS IDENTITY PRIMARY KEY, name VARCHAR2(255))"`
	mcpSelect1Want := `{"jsonrpc":"2.0","id":"invoke my-auth-required-tool","result":{"content":[{"type":"text","text":"{\"1\":1}"}]}}`

	// Run tests
	tests.RunToolGetTest(t)
	tests.RunToolInvokeTest(t, select1Want,
		tests.DisableOptionalNullParamTest(),
		tests.WithMyToolById4Want("[{\"id\":4,\"name\":\"\"}]"),
		tests.DisableArrayTest(),
	)
	tests.RunMCPToolCallMethod(t, mcpMyFailToolWant, mcpSelect1Want)
	tests.RunExecuteSqlToolInvokeTest(t, createTableStatement, select1Want)
	tests.RunToolInvokeWithTemplateParameters(t, tableNameTemplateParam)
}


// TestOracleConnectionOCIWithWallet tests OCI driver connection with TNS Admin and Wallet
func TestOracleConnectionOCIWithWallet(t *testing.T) {
    t.Parallel()
    // This test verifies that useOCI=true and tnsAdmin parameters are correctly passed for OCI wallet.
    // It will likely fail due to missing tnsnames.ora and wallet files.

    // Save original env vars and restore them at the end
    cleanup := setOracleEnv(t,
        "", OracleUser, OraclePass, "", "", // Unset host/port/service for TNS alias, but keep user/pass
        "", // connectionString
        "MY_TNS_ALIAS", // tnsAlias
        "/tmp/nonexistent_tns_admin", // tnsAdmin
        "", // walletLocation
        true, // useOCI
    )
    defer cleanup()

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    cfg := getOracleConfigFromEnv(t)
    _, err := cfg.Initialize(ctx, nil)

    if err == nil {
        t.Fatalf("Expected connection to fail (OCI driver with TNS Admin/Wallet), but it succeeded")
    }

    // Check for error message indicating TNS Admin/Wallet usage or connection failure.
    expectedErrorSubstrings := []string{"tns", "wallet", "oci", "driver", "connection"}
    foundExpectedError := false
    for _, sub := range expectedErrorSubstrings {
        if strings.Contains(strings.ToLower(err.Error()), sub) {
            foundExpectedError = true
            break
        }
    }
    if !foundExpectedError {
        t.Errorf("Expected error message to contain one of %v (case-insensitive) but got: %v", expectedErrorSubstrings, err)
    }
    t.Logf("Connection failed as expected (OCI Driver with TNS Admin/Wallet): %v", err)
}

// TestOracleConnectionPureGoWithWallet tests pure Go driver connection with wallet
func TestOracleConnectionPureGoWithWallet(t *testing.T) {
    t.Parallel()
    // This test expects the connection to fail because the wallet file won't exist.
    // It verifies that the walletLocation parameter is correctly passed to the pure Go driver.

    // Save original env vars and restore them at the end
    cleanup := setOracleEnv(t,
        OracleHost, OracleUser, OraclePass, OracleServerName, OraclePort, // Use existing base connection details
        "", // connectionString
        "", // tnsAlias
        "",                        // tnsAdmin
        "/tmp/nonexistent_wallet", // walletLocation
        false,                     // useOCI
    )
    defer cleanup()

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    cfg := getOracleConfigFromEnv(t)
    _, err := cfg.Initialize(ctx, nil) // Pass nil for tracer as it's not critical for this test

    if err == nil {
        t.Fatalf("Expected connection to fail with non-existent wallet, but it succeeded")
    }

    // Check for error message indicating wallet usage or connection failure related to wallet
    // The exact error message might vary depending on the go-ora version and OS.
    // We are looking for an error that suggests the wallet path was attempted.
    expectedErrorSubstring := "wallet"
    if !strings.Contains(strings.ToLower(err.Error()), expectedErrorSubstring) {
        t.Errorf("Expected error message to contain '%s' (case-insensitive) but got: %v", expectedErrorSubstring, err)
    }
    t.Logf("Connection failed as expected (Pure Go with Wallet): %v", err)
}

// TestOracleConnectionOCI tests OCI driver connection without wallet
func TestOracleConnectionOCI(t *testing.T) {
    t.Parallel()
    // This test verifies that the useOCI=true parameter is correctly passed to the OCI driver.
    // It will likely fail if Oracle Instant Client is not installed or configured.

    // Save original env vars and restore them at the end
    cleanup := setOracleEnv(t,
        OracleHost, OracleUser, OraclePass, OracleServerName, OraclePort, // Use existing base connection details
        "",    // connectionString
        "",    // tnsAlias
        "",    // tnsAdmin
        "", // walletLocation
        true,  // useOCI
    )
    defer cleanup()

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    cfg := getOracleConfigFromEnv(t)
    _, err := cfg.Initialize(ctx, nil)

    	// Fix: Do not fail if connection succeeds.
	// If err is nil, it means OCI is set up correctly, which is good!
	if err == nil {
		t.Log("Connection succeeded (OCI Driver configured correctly)")
		return
	}

	// Check for error message indicating OCI driver usage or connection failure related to OCI.
	// Common errors include "OCI environment not initialized", "driver: bad connection", etc.
	expectedErrorSubstrings := []string{"oci", "driver", "connection", "cannot load"}
	foundExpectedError := false
	for _, sub := range expectedErrorSubstrings {
		if strings.Contains(strings.ToLower(err.Error()), sub) {
			foundExpectedError = true
			break
		}
	}
	if !foundExpectedError {
		t.Errorf("Expected error message to contain one of %v (case-insensitive) but got: %v", expectedErrorSubstrings, err)
	}
	t.Logf("Connection failed (OCI Driver issues expected if not configured): %v", err)
}

//test utils
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