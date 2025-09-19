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

	_ "github.com/go-sql-driver/mysql"
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
	MySQLHost         = os.Getenv("MINDSDB_HOST")
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
	case MySQLHost:
		t.Fatal("'MYSQL_HOST' not set")
	case MySQLUser:
		t.Fatal("'MYSQL_USER' not set")
	case MySQLPass:
		t.Fatal("'MYSQL_PASS' not set")
	case MySQLDatabase:
		t.Fatal("'MYSQL_DATABASE' not set")
	}

	// Handle MindsDB's no-password authentication for toolbox config
	mindsdbPassword := MindsDBPass
	if mindsdbPassword == "none" {
		mindsdbPassword = ""
	}

	return map[string]any{
		"kind":     MindsDBSourceKind,
		"host":     MindsDBHost,
		"port":     MindsDBPort,
		"database": MindsDBDatabase,
		"user":     MindsDBUser,
		"password": mindsdbPassword,
	}
}

// Copied over from mysql.go
func initMySQLConnectionPool(host, port, user, pass, dbname string) (*sql.DB, error) {
	// Handle MindsDB's no-password authentication: if pass is "none", use empty string
	if pass == "none" {
		pass = ""
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", user, pass, host, port, dbname)
	pool, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}
	return pool, nil
}

func setupMindsDBIntegration(t *testing.T, ctx context.Context) {
	// Connect to mindsdb's own `mindsdb` database to run CREATE DATABASE.
	mindsdbPool, err := initMySQLConnectionPool(MindsDBHost, MindsDBPort, MindsDBUser, MindsDBPass, MindsDBDatabase)
	if err != nil {
		t.Fatalf("unable to connect to mindsdb for setup: %s", err)
	}
	defer mindsdbPool.Close()

	// Use environment variables for MySQL connection
	// For Docker network: MYSQL_HOST=mysql-server, for host network: MYSQL_HOST=127.0.0.1

	// Debug: Print the values being used
	t.Logf("DEBUG: MindsDB MySQL connection params - user: %s, pass: %s, host: %s, port: %s, database: %s",
		MySQLUser, MySQLPass, MySQLHost, MySQLPort, MySQLDatabase)

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
        }`, MindsDBDatabase, MySQLUser, MySQLPass, "mysql-server", MySQLPort, MySQLDatabase)

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

	// Create connection pool to MySQL database for table creation
	// Use the actual MySQL credentials, not the MindsDB integration credentials
	mysqlPool, err := initMySQLConnectionPool(MySQLHost, MySQLPort, MySQLUser, MySQLPass, MySQLDatabase)
	if err != nil {
		t.Fatalf("unable to create MySQL connection pool: %s", err)
	}

	// create table name with UUID
	tableNameParam := "param_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableNameAuth := "auth_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableNameTemplateParam := "template_param_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")

	// set up data for param tool using standard MySQL functions
	createParamTableStmt, insertParamTableStmt, _, _, _, _, paramTestParams := tests.GetMySQLParamToolInfo(tableNameParam)

	// Create query statements with MindsDB database prefix for table references
	mindsdbTableName := fmt.Sprintf("%s.%s", MindsDBDatabase, tableNameParam)
	paramToolStmt := fmt.Sprintf("SELECT * FROM %s WHERE id = ? OR name = ?;", mindsdbTableName)
	idParamToolStmt := fmt.Sprintf("SELECT * FROM %s WHERE id = ?;", mindsdbTableName)
	nameParamToolStmt := fmt.Sprintf("SELECT * FROM %s WHERE name = ?;", mindsdbTableName)
	arrayToolStmt := fmt.Sprintf("SELECT * FROM %s WHERE id = ANY(?) AND name = ANY(?);", mindsdbTableName)
	teardownTable1 := tests.SetupMySQLTable(t, ctx, mysqlPool, createParamTableStmt, insertParamTableStmt, tableNameParam, paramTestParams)
	defer teardownTable1(t)

	// set up data for auth tool
	createAuthTableStmt, insertAuthTableStmt, _, authTestParams := tests.GetMySQLAuthToolInfo(tableNameAuth)

	// Create auth query statement with MindsDB database prefix for table references
	mindsdbAuthTableName := fmt.Sprintf("%s.%s", MindsDBDatabase, tableNameAuth)
	authToolStmt := fmt.Sprintf("SELECT name FROM %s WHERE email = ?;", mindsdbAuthTableName)
	teardownTable2 := tests.SetupMySQLTable(t, ctx, mysqlPool, createAuthTableStmt, insertAuthTableStmt, tableNameAuth, authTestParams)
	defer teardownTable2(t)

	// Allow time for MindsDB to detect the new tables
	time.Sleep(5 * time.Second)

	// Write config into a file and pass it to command - use standard tools config
	toolsFile := tests.GetToolsConfig(sourceConfig, MindsDBToolKind, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, authToolStmt)
	toolsFile = tests.AddMySqlExecuteSqlConfig(t, toolsFile)
	// Create MindsDB-specific template statements with database prefix
	tmplSelectCombined := fmt.Sprintf("SELECT * FROM %s.{{.tableName}} WHERE id = ?", MindsDBDatabase)
	tmplSelectFilterCombined := fmt.Sprintf("SELECT * FROM %s.{{.tableName}} WHERE {{.columnFilter}} = ?", MindsDBDatabase)
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

	// Get configs for tests - use standard MySQL expectations
	select1Want, mcpMyFailToolWant, createTableStatement, mcpSelect1Want := tests.GetMySQLWants()

	// Run tests without disabled options - MindsDB should support the same features as MySQL
	tests.RunToolGetTest(t)
	tests.RunToolInvokeTest(t, select1Want)
	tests.RunMCPToolCallMethod(t, mcpMyFailToolWant, mcpSelect1Want)
	tests.RunExecuteSqlToolInvokeTest(t, createTableStatement, select1Want)
	tests.RunToolInvokeWithTemplateParameters(t, tableNameTemplateParam)
}
