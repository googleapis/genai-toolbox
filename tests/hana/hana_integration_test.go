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

package hana

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	_ "github.com/SAP/go-hdb/driver"
	"github.com/google/uuid"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"
)

var (
	HanaSourceKind = "hana"
	HanaToolKind   = "hana-sql"
	HanaDatabase   = os.Getenv("HANA_DATABASE")
	HanaHost       = os.Getenv("HANA_HOST")
	HanaPort       = os.Getenv("HANA_PORT")
	HanaUser       = os.Getenv("HANA_USER")
	HanaPass       = os.Getenv("HANA_PASS")
)

func getHanaVars(t *testing.T) map[string]any {
	switch "" {
	case HanaDatabase:
		t.Fatal("'HANA_DATABASE' not set")
	case HanaHost:
		t.Fatal("'HANA_HOST' not set")
	case HanaPort:
		t.Fatal("'HANA_PORT' not set")
	case HanaUser:
		t.Fatal("'HANA_USER' not set")
	case HanaPass:
		t.Fatal("'HANA_PASSWORD' not set")
	}

	return map[string]any{
		"kind":     HanaSourceKind,
		"host":     HanaHost,
		"port":     HanaPort,
		"database": HanaDatabase,
		"user":     HanaUser,
		"password": HanaPass,
	}
}

// initHanaConnection creates a connection using the go-hdb driver.
func initHanaConnection(host, port, user, pass, dbname string) (*sql.DB, error) {
	// Construct DSN: hdb://user:pass@host:port?databaseName=<dbname>
	dsnURL := &url.URL{
		Scheme: "hdb",
		User:   url.UserPassword(user, pass),
		Host:   fmt.Sprintf("%s:%s", host, port),
	}

	qs := url.Values{}
	if dbname != "" {
		qs.Add("databaseName", dbname)
	}
	if len(qs) > 0 {
		dsnURL.RawQuery = qs.Encode()
	}

	db, err := sql.Open("hdb", dsnURL.String())
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}
	return db, nil
}

func TestHanaToolEndpoints(t *testing.T) {
	sourceConfig := getHanaVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	db, err := initHanaConnection(HanaHost, HanaPort, HanaUser, HanaPass, HanaDatabase)
	if err != nil {
		t.Fatalf("unable to create HANA connection: %s", err)
	}
	defer db.Close()

	// Test connection
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("unable to ping HANA database: %s", err)
	}

	// create table name with UUID
	tableNameParam := "PARAM_TABLE_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableNameAuth := "AUTH_TABLE_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableNameTemplateParam := "TEMPLATE_PARAM_TABLE_" + strings.ReplaceAll(uuid.New().String(), "-", "")

	// set up data for param tool
	createParamTableStmt, insertParamTableStmt, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, paramTestParams := getHanaParamToolInfo(tableNameParam)
	teardownTable1 := setupHanaTable(t, ctx, db, createParamTableStmt, insertParamTableStmt, tableNameParam, paramTestParams)
	defer teardownTable1(t)

	// set up data for auth tool
	createAuthTableStmt, insertAuthTableStmt, authToolStmt, authTestParams := getHanaAuthToolInfo(tableNameAuth)
	teardownTable2 := setupHanaTable(t, ctx, db, createAuthTableStmt, insertAuthTableStmt, tableNameAuth, authTestParams)
	defer teardownTable2(t)

	// Write config into a file and pass it to command
	toolsFile := tests.GetToolsConfig(sourceConfig, HanaToolKind, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, authToolStmt)
	toolsFile = tests.AddPgExecuteSqlConfig(t, toolsFile)
	tmplSelectCombined, tmplSelectFilterCombined := tests.GetPostgresSQLTmplToolStatement()
	toolsFile = tests.AddTemplateParamConfig(t, toolsFile, HanaToolKind, tmplSelectCombined, tmplSelectFilterCombined, "")

	// Start test server
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
	select1Want, mcpMyFailToolWant, createTableStatement, mcpSelect1Want := getHanaWants()

	// Run tests
	tests.RunToolGetTest(t)
	tests.RunToolInvokeTest(t, select1Want)
	tests.RunMCPToolCallMethod(t, mcpMyFailToolWant, mcpSelect1Want)
	tests.RunExecuteSqlToolInvokeTest(t, createTableStatement, select1Want)
	tests.RunToolInvokeWithTemplateParameters(t, tableNameTemplateParam)
}

// getHanaWants return the expected wants for SAP Hana
func getHanaWants() (string, string, string, string) {
	select1Want := "[{\"?column?\":1}]"
	mcpMyFailToolWant := `{"jsonrpc":"2.0","id":"invoke-fail-tool","result":{"content":[{"type":"text","text":"unable to execute query: ERROR: syntax error at or near \"SELEC\" (SQLSTATE 42601)"}],"isError":true}}`
	createTableStatement := `"CREATE TABLE t (id SERIAL PRIMARY KEY, name TEXT)"`
	mcpSelect1Want := `{"jsonrpc":"2.0","id":"invoke my-auth-required-tool","result":{"content":[{"type":"text","text":"{\"?column?\":1}"}]}}`
	return select1Want, mcpMyFailToolWant, createTableStatement, mcpSelect1Want
}

// getHanaParamToolInfo returns statements and param for my-tool hana-sql kind
func getHanaParamToolInfo(tableName string) (string, string, string, string, string, string, []any) {
	createStatement := fmt.Sprintf("CREATE TABLE %s (id INTEGER NOT NULL PRIMARY KEY, name NVARCHAR(255))", tableName)
	insertStatement := fmt.Sprintf("INSERT INTO %s (id, name) VALUES (?, ?), (?, ?), (?, ?), (?, ?)", tableName)
	toolStatement := fmt.Sprintf("SELECT * FROM %s WHERE id = ? OR name = ?", tableName)
	idParamStatement := fmt.Sprintf("SELECT * FROM %s WHERE id = ?", tableName)
	nameParamStatement := fmt.Sprintf("SELECT * FROM %s WHERE name = ?", tableName)
	arrayToolStatement := fmt.Sprintf("SELECT * FROM %s WHERE id IN (?, ?) AND name IN (?, ?)", tableName)
	params := []any{1, "Alice", 2, "Jane", 3, "Sid", 4, nil}
	return createStatement, insertStatement, toolStatement, idParamStatement, nameParamStatement, arrayToolStatement, params
}

// getHanaAuthToolInfo returns statements and param of my-auth-tool for hana-sql kind
func getHanaAuthToolInfo(tableName string) (string, string, string, []any) {
	createStatement := fmt.Sprintf("CREATE TABLE %s (id INTEGER NOT NULL PRIMARY KEY, name NVARCHAR(255), email NVARCHAR(255))", tableName)
	insertStatement := fmt.Sprintf("INSERT INTO %s (id, name, email) VALUES (?, ?, ?), (?, ?, ?)", tableName)
	toolStatement := fmt.Sprintf("SELECT name FROM %s WHERE email = ?", tableName)
	params := []any{1, "Alice", tests.ServiceAccountEmail, 2, "Jane", "janedoe@gmail.com"}
	return createStatement, insertStatement, toolStatement, params
}

// setupHanaTable sets up a table for testing HANA tools
func setupHanaTable(t *testing.T, ctx context.Context, db *sql.DB, createStatement, insertStatement, tableName string, params []any) func(*testing.T) {
	_, err := db.ExecContext(ctx, createStatement)
	if err != nil {
		t.Fatalf("failed to create table: %s", err)
	}

	if len(params) > 0 {
		_, err = db.ExecContext(ctx, insertStatement, params...)
		if err != nil {
			t.Fatalf("failed to insert into table: %s", err)
		}
	}

	return func(t *testing.T) {
		dropStatement := fmt.Sprintf("DROP TABLE %s", tableName)
		_, err := db.ExecContext(ctx, dropStatement)
		if err != nil {
			t.Logf("failed to drop table %s: %s", tableName, err)
		}
	}
}
