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

package tidb

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
	TiDBSourceKind = "tidb"
	TiDBToolKind   = "tidb-sql"
	TiDBDatabase   = os.Getenv("TIDB_DATABASE")
	TiDBHost       = os.Getenv("TIDB_HOST")
	TiDBPort       = os.Getenv("TIDB_PORT")
	TiDBUser       = os.Getenv("TIDB_USERNAME")
	TiDBPass       = os.Getenv("TIDB_PASSWORD")
)

func getTiDBVars(t *testing.T) map[string]any {
	switch "" {
	case TiDBDatabase:
		t.Fatal("'TIDB_DATABASE' not set")
	case TiDBHost:
		t.Fatal("'TIDB_HOST' not set")
	case TiDBPort:
		t.Fatal("'TIDB_PORT' not set")
	case TiDBUser:
		t.Fatal("'TIDB_USERNAME' not set")
	case TiDBPass:
		t.Fatal("'TIDB_PASSWORD' not set")
	}

	return map[string]any{
		"kind":     TiDBSourceKind,
		"host":     TiDBHost,
		"port":     TiDBPort,
		"database": TiDBDatabase,
		"user":     TiDBUser,
		"password": TiDBPass,
	}
}

// Copied over from tidb.go
func initTiDBConnectionPool(host, port, user, pass, dbname string, useSSL bool) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&tls=%t", user, pass, host, port, dbname, useSSL)

	// Interact with the driver directly as you normally would
	pool, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}
	return pool, nil
}

func TestTiDBToolEndpoints(t *testing.T) {
	sourceConfig := getTiDBVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	pool, err := initTiDBConnectionPool(TiDBHost, TiDBPort, TiDBUser, TiDBPass, TiDBDatabase, true)
	if err != nil {
		t.Fatalf("unable to create TiDB connection pool: %s", err)
	}

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
	toolsFile := tests.GetToolsConfig(sourceConfig, TiDBToolKind, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, authToolStmt)
	toolsFile = tests.AddTiDBExecuteSqlConfig(t, toolsFile)
	tmplSelectCombined, tmplSelectFilterCombined := tests.GetMySQLTmplToolStatement()
	toolsFile = tests.AddTemplateParamConfig(t, toolsFile, TiDBToolKind, tmplSelectCombined, tmplSelectFilterCombined, "")

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

	select1Want, failInvocationWant, createTableStatement := tests.GetTiDBWants()
	invokeParamWant, invokeIdNullWant, nullWant, mcpInvokeParamWant := tests.GetNonSpannerInvokeParamWant()
	tests.RunToolInvokeTest(t, select1Want, invokeParamWant, invokeIdNullWant, nullWant, true, false)
	tests.RunExecuteSqlToolInvokeTest(t, createTableStatement, select1Want)
	tests.RunMCPToolCallMethod(t, mcpInvokeParamWant, failInvocationWant)
	tests.RunToolInvokeWithTemplateParameters(t, tableNameTemplateParam, tests.NewTemplateParameterTestConfig())
}
