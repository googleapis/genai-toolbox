// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mariadb

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

	"github.com/google/uuid"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"
)

var (
	MariaDBSourceKind = "mariadb"
	MariaDBToolKind   = "mysql-sql"
	MariaDBDatabase   = os.Getenv("MARIADB_DATABASE")
	MariaDBHost       = os.Getenv("MARIADB_HOST")
	MariaDBPort       = os.Getenv("MARIADB_PORT")
	MariaDBUser       = os.Getenv("MARIADB_USER")
	MariaDBPass       = os.Getenv("MARIADB_PASS")
)

func getMariaDBVars(t *testing.T) map[string]any {
	switch "" {
	case MariaDBDatabase:
		t.Fatal("'MARIADB_DATABASE' not set")
	case MariaDBHost:
		t.Fatal("'MARIADB_HOST' not set")
	case MariaDBPort:
		t.Fatal("'MARIADB_PORT' not set")
	case MariaDBUser:
		t.Fatal("'MARIADB_USER' not set")
	case MariaDBPass:
		t.Fatal("'MARIADB_PASS' not set")
	}

	return map[string]any{
		"kind":     MariaDBSourceKind,
		"host":     MariaDBHost,
		"port":     MariaDBPort,
		"database": MariaDBDatabase,
		"user":     MariaDBUser,
		"password": MariaDBPass,
	}
}

// Copied over from mariadb.go
func initMariaDB(host, port, user, password, database string) (*sql.DB, error) {

	q := url.Values{}

	// defaults
	q.Set("parseTime", "true")
	q.Set("clientFoundRows", "true")

	// driver accepts additional params in DSN, including "interpolateParams=true&tls=skip-verify"
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?%s",
		user,
		password,
		host,
		port,
		database,
		q.Encode(),
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}

	return db, nil
}

func TestMySQLToolEndpoints(t *testing.T) {
	sourceConfig := getMariaDBVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	pool, err := initMariaDB(MariaDBHost, MariaDBPort, MariaDBUser, MariaDBPass, MariaDBDatabase)
	if err != nil {
		t.Fatalf("unable to create MySQL connection pool: %s", err)
	}

	// cleanup test environment
	tests.CleanupMySQLTables(t, ctx, pool)

	// create table name with UUID
	tableNameParam := "param_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableNameAuth := "auth_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableNameTemplateParam := "template_param_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")

	// set up data for param tool
	createParamTableStmt, insertParamTableStmt, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, paramTestParams := tests.GetMySQLParamToolInfo(tableNameParam)
	teardownTable1 := tests.SetupMySQLTable(t, ctx, pool, createParamTableStmt, insertParamTableStmt, tableNameParam, paramTestParams)
	defer teardownTable1(t)

	// set up data for auth tool
	createAuthTableStmt, insertAuthTableStmt, authToolStmt, authTestParams := tests.GetMySQLAuthToolInfo(tableNameAuth)
	teardownTable2 := tests.SetupMySQLTable(t, ctx, pool, createAuthTableStmt, insertAuthTableStmt, tableNameAuth, authTestParams)
	defer teardownTable2(t)

	// Write config into a file and pass it to command
	toolsFile := tests.GetToolsConfig(sourceConfig, MariaDBToolKind, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, authToolStmt)
	toolsFile = tests.AddMySqlExecuteSqlConfig(t, toolsFile)
	tmplSelectCombined, tmplSelectFilterCombined := tests.GetMySQLTmplToolStatement()
	toolsFile = tests.AddTemplateParamConfig(t, toolsFile, MariaDBToolKind, tmplSelectCombined, tmplSelectFilterCombined, "")

	toolsFile = tests.AddMySQLPrebuiltToolConfig(t, toolsFile)

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
	select1Want, mcpMyFailToolWant, createTableStatement, mcpSelect1Want := tests.GetMariaDBWants()

	// Run tests
	tests.RunToolGetTest(t)
	tests.RunToolInvokeTest(t, select1Want, tests.DisableArrayTest())
	tests.RunMCPToolCallMethod(t, mcpMyFailToolWant, mcpSelect1Want)
	tests.RunExecuteSqlToolInvokeTest(t, createTableStatement, select1Want)
	tests.RunToolInvokeWithTemplateParameters(t, tableNameTemplateParam)

	// Run specific MySQL tool tests
	tests.RunMariDBListTablesTest(t, MariaDBDatabase, tableNameParam, tableNameAuth)
	tests.RunMySQLListActiveQueriesTest(t, ctx, pool)
	tests.RunMySQLListTablesMissingUniqueIndexes(t, ctx, pool, MariaDBDatabase)
	tests.RunMySQLListTableFragmentationTest(t, MariaDBDatabase, tableNameParam, tableNameAuth)
}
