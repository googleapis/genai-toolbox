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

package mindsdb

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
	MindsDBSourceKind = "mindsdb"
	MindsDBToolKind   = "mindsdb-sql"
	MindsDBDatabase   = os.Getenv("MINDSDB_DATABASE")
	MindsDBHost       = os.Getenv("MINDSDB_HOST")
	MindsDBPort       = os.Getenv("MINDSDB_PORT")
	MindsDBUser       = os.Getenv("MINDSDB_USER")
	MindsDBPass       = os.Getenv("MINDSDB_PASS")
	MySQLPort         = os.Getenv("MYSQL_PORT")
	MySQLUser         = os.Getenv("MINDSDB_MYSQL_USER")
	MySQLPass         = os.Getenv("MINDSDB_MYSQL_PASS")
	MySQLDatabase     = os.Getenv("MYSQL_DATABASE")
	MySQLHost         = "mysql-server"
)

func getMindsDBVars(t *testing.T) map[string]any {
	switch "" {
	case MindsDBDatabase:
		t.Fatal("'MINDSDB_DATABASE' not set")
	case MindsDBHost:
		t.Fatal("'MINDSDB_HOST' not set")
	case MindsDBPort:
		t.Fatal("'MINDSDB_PORT' not set")
	case MindsDBUser:
		t.Fatal("'MINDSDB_USER' not set")
	case MindsDBPass:
		t.Fatal("'MINDSDB_PASS' not set")
	case MySQLUser:
		t.Fatal("'MYSQL_USER' not set")
	case MySQLPass:
		t.Fatal("'MYSQL_PASS' not set")
	case MySQLDatabase:
		t.Fatal("'MYSQL_DATABASE' not set")
	}

	return map[string]any{
		"kind":     MindsDBSourceKind,
		"host":     MindsDBHost,
		"port":     MindsDBPort,
		"database": MindsDBDatabase,
		"user":     MindsDBUser,
		"password": MindsDBPass,
	}
}

// Copied over from mysql.go
func initMySQLConnectionPool(host, port, user, pass, dbname string) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", user, pass, host, port, dbname)
	pool, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}
	return pool, nil
}

func setupMindsDBIntegration(t *testing.T, ctx context.Context) {
	// Connect to mindsdb's own `mindsdb` database to run CREATE DATABASE.
	mindsdbPool, err := initMySQLConnectionPool(MindsDBHost, MindsDBPort, MindsDBUser, MindsDBPass, "mindsdb")
	if err != nil {
		t.Fatalf("unable to connect to mindsdb for setup: %s", err)
	}
	defer mindsdbPool.Close()

	// The SQL command to connect MindsDB to the MySQL test database.
	createStatement := fmt.Sprintf(`
        CREATE DATABASE IF NOT EXISTS %s
        WITH ENGINE = 'mysql',
        PARAMETERS = {
            "user": "%s",
            "password": "%s",
            "host": "%s",
            "port": %s,
            "database": "%s"
        }`, MindsDBDatabase, MySQLUser, MySQLPass, MySQLHost, MySQLPort, MySQLDatabase)

	_, err = mindsdbPool.ExecContext(ctx, createStatement)
	if err != nil {
		t.Fatalf("failed to create mindsdb integration: %v", err)
	}

	// Clean up
	t.Cleanup(func() {
		dropStatement := fmt.Sprintf("DROP DATABASE %s", MindsDBDatabase)
		_, err := mindsdbPool.ExecContext(context.Background(), dropStatement)
		if err != nil {
			t.Logf("failed to drop mindsdb integration, may require manual cleanup: %v", err)
		}
	})
}

func TestMindsDBToolEndpoints(t *testing.T) {
	sourceConfig := getMindsDBVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	setupMindsDBIntegration(t, ctx)

	// Give MindsDB integration time to establish connection to MySQL
	time.Sleep(2 * time.Second)

	var args []string

	pool, err := initMySQLConnectionPool(MindsDBHost, MySQLPort, MySQLUser, MySQLPass, MySQLDatabase)
	if err != nil {
		t.Fatalf("unable to create MindsDB connection pool: %s", err)
	}

	// create table name with UUID
	tableNameParam := "param_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableNameAuth := "auth_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableNameTemplateParam := "template_param_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")

	// set up data for param tool - create tables in the underlying MySQL database
	createParamTableStmt, insertParamTableStmt, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, paramTestParams := tests.GetMySQLParamToolInfo(tableNameParam)
	t.Logf("Creating param table: %s", tableNameParam)
	teardownTable1 := tests.SetupMySQLTable(t, ctx, pool, createParamTableStmt, insertParamTableStmt, tableNameParam, paramTestParams)
	defer teardownTable1(t)

	// set up data for auth tool - create tables in the underlying MySQL database
	createAuthTableStmt, insertAuthTableStmt, authToolStmt, authTestParams := tests.GetMySQLAuthToolInfo(tableNameAuth)
	t.Logf("Creating auth table: %s", tableNameAuth)
	teardownTable2 := tests.SetupMySQLTable(t, ctx, pool, createAuthTableStmt, insertAuthTableStmt, tableNameAuth, authTestParams)
	defer teardownTable2(t)

	// Allow time for MindsDB to detect the new tables
	time.Sleep(1 * time.Second)

	// Write config into a file and pass it to command
	toolsFile := tests.GetToolsConfig(sourceConfig, MindsDBToolKind, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, authToolStmt)
	toolsFile = tests.AddMySqlExecuteSqlConfig(t, toolsFile)
	tmplSelectCombined, tmplSelectFilterCombined := tests.GetMySQLTmplToolStatement()
	toolsFile = tests.AddTemplateParamConfig(t, toolsFile, MindsDBToolKind, tmplSelectCombined, tmplSelectFilterCombined, "")

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

	select1Want, mcpMyFailToolWant, createTableStatement := tests.GetMindsDBWants()

	// Run basic tests - these should work once table setup is fixed
	tests.RunToolInvokeTest(t, select1Want, tests.DisableArrayTest())
	tests.RunExecuteSqlToolInvokeTest(t, createTableStatement, select1Want)
	tests.RunMCPToolCallMethod(t, mcpMyFailToolWant)

	// Skip template parameter tests as MindsDB doesn't support standard DDL operations
	tests.RunToolInvokeWithTemplateParameters(t, tableNameTemplateParam,
		tests.DisableDdlTest(),
		tests.DisableInsertTest())
}
