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

package postgres

import (
	"context"
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
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	PostgresSourceKind = "postgres"
	PostgresToolKind   = "postgres-sql"
	PostgresDatabase   = os.Getenv("POSTGRES_DATABASE")
	PostgresHost       = os.Getenv("POSTGRES_HOST")
	PostgresPort       = os.Getenv("POSTGRES_PORT")
	PostgresUser       = os.Getenv("POSTGRES_USER")
	PostgresPass       = os.Getenv("POSTGRES_PASS")
)

func getPostgresVars(t *testing.T) map[string]any {
	switch "" {
	case PostgresDatabase:
		t.Fatal("'POSTGRES_DATABASE' not set")
	case PostgresHost:
		t.Fatal("'POSTGRES_HOST' not set")
	case PostgresPort:
		t.Fatal("'POSTGRES_PORT' not set")
	case PostgresUser:
		t.Fatal("'POSTGRES_USER' not set")
	case PostgresPass:
		t.Fatal("'POSTGRES_PASS' not set")
	}

	return map[string]any{
		"kind":     PostgresSourceKind,
		"host":     PostgresHost,
		"port":     PostgresPort,
		"database": PostgresDatabase,
		"user":     PostgresUser,
		"password": PostgresPass,
	}
}

// Copied over from postgres.go
func initPostgresConnectionPool(host, port, user, pass, dbname string) (*pgxpool.Pool, error) {
	// urlExample := "postgres:dd//username:password@localhost:5432/database_name"
	url := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(user, pass),
		Host:   fmt.Sprintf("%s:%s", host, port),
		Path:   dbname,
	}
	pool, err := pgxpool.New(context.Background(), url.String())
	if err != nil {
		return nil, fmt.Errorf("Unable to create connection pool: %w", err)
	}

	return pool, nil
}

func TestPostgres(t *testing.T) {
	sourceConfig := getPostgresVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	pool, err := initPostgresConnectionPool(PostgresHost, PostgresPort, PostgresUser, PostgresPass, PostgresDatabase)
	if err != nil {
		t.Fatalf("unable to create postgres connection pool: %s", err)
	}

	// cleanup test environment
	tests.CleanupPostgresTables(t, ctx, pool)

	// create table name with UUID
	tableNameParam := "param_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableNameAuth := "auth_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableNameTemplateParam := "template_param_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")

	// set up data for param tool
	createParamTableStmt, insertParamTableStmt, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, paramTestParams := tests.GetPostgresSQLParamToolInfo(tableNameParam)
	teardownTable1 := tests.SetupPostgresSQLTable(t, ctx, pool, createParamTableStmt, insertParamTableStmt, tableNameParam, paramTestParams)
	defer teardownTable1(t)

	// set up data for auth tool
	createAuthTableStmt, insertAuthTableStmt, authToolStmt, authTestParams := tests.GetPostgresSQLAuthToolInfo(tableNameAuth)
	teardownTable2 := tests.SetupPostgresSQLTable(t, ctx, pool, createAuthTableStmt, insertAuthTableStmt, tableNameAuth, authTestParams)
	defer teardownTable2(t)

	// Write config into a file and pass it to command
	toolsFile := tests.GetToolsConfig(sourceConfig, PostgresToolKind, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, authToolStmt)
	toolsFile = tests.AddExecuteSqlConfig(t, toolsFile, "postgres-execute-sql")
	tmplSelectCombined, tmplSelectFilterCombined := tests.GetPostgresSQLTmplToolStatement()
	toolsFile = tests.AddTemplateParamConfig(t, toolsFile, PostgresToolKind, tmplSelectCombined, tmplSelectFilterCombined, "")
	toolsFile = tests.AddPostgresPrebuiltConfig(t, toolsFile)

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
	select1Want, mcpMyFailToolWant, createTableStatement, mcpSelect1Want := tests.GetPostgresWants()

	// Run tests
	tests.RunToolGetTest(t)
	tests.RunToolInvokeTest(t, select1Want)
	tests.RunMCPToolCallMethod(t, mcpMyFailToolWant, mcpSelect1Want)
	tests.RunExecuteSqlToolInvokeTest(t, createTableStatement, select1Want)
	tests.RunToolInvokeWithTemplateParameters(t, tableNameTemplateParam)

	// Run Postgres prebuilt tool tests
	tests.RunPostgresListTablesTest(t, tableNameParam, tableNameAuth, PostgresUser)
	tests.RunPostgresListViewsTest(t, ctx, pool, tableNameParam)
	tests.RunPostgresListSchemasTest(t, ctx, pool)
	tests.RunPostgresListActiveQueriesTest(t, ctx, pool)
	tests.RunPostgresListAvailableExtensionsTest(t)
	tests.RunPostgresListInstalledExtensionsTest(t)
	tests.RunPostgresDatabaseOverviewTest(t, ctx, pool)
	tests.RunPostgresListTriggersTest(t, ctx, pool)
	tests.RunPostgresListIndexesTest(t, ctx, pool)
	tests.RunPostgresListSequencesTest(t, ctx, pool)
	tests.RunPostgresLongRunningTransactionsTest(t, ctx, pool)
	tests.RunPostgresListLocksTest(t, ctx, pool)
	tests.RunPostgresReplicationStatsTest(t, ctx, pool)
	tests.RunPostgresListQueryStatsTest(t, ctx, pool)
	tests.RunPostgresGetColumnCardinalityTest(t, ctx, pool)
}
