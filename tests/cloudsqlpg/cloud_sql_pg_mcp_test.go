// Copyright 2026 Google LLC
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

package cloudsqlpg

import (
	"context"
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/cloudsqlconn"
	"github.com/google/uuid"
	"github.com/googleapis/mcp-toolbox/internal/testutils"
	"github.com/googleapis/mcp-toolbox/tests"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	CloudSQLPostgresSourceType = "cloud-sql-postgres"
	CloudSQLPostgresToolType   = "postgres-sql"
	CloudSQLPostgresProject    = os.Getenv("CLOUD_SQL_POSTGRES_PROJECT")
	CloudSQLPostgresRegion     = os.Getenv("CLOUD_SQL_POSTGRES_REGION")
	CloudSQLPostgresInstance   = os.Getenv("CLOUD_SQL_POSTGRES_INSTANCE")
	CloudSQLPostgresDatabase   = os.Getenv("CLOUD_SQL_POSTGRES_DATABASE")
	CloudSQLPostgresUser       = os.Getenv("CLOUD_SQL_POSTGRES_USER")
	CloudSQLPostgresPass       = os.Getenv("CLOUD_SQL_POSTGRES_PASS")
)

func getCloudSQLPgVars(t *testing.T) map[string]any {
	switch "" {
	case CloudSQLPostgresProject:
		t.Fatal("'CLOUD_SQL_POSTGRES_PROJECT' not set")
	case CloudSQLPostgresRegion:
		t.Fatal("'CLOUD_SQL_POSTGRES_REGION' not set")
	case CloudSQLPostgresInstance:
		t.Fatal("'CLOUD_SQL_POSTGRES_INSTANCE' not set")
	case CloudSQLPostgresDatabase:
		t.Fatal("'CLOUD_SQL_POSTGRES_DATABASE' not set")
	case CloudSQLPostgresUser:
		t.Fatal("'CLOUD_SQL_POSTGRES_USER' not set")
	case CloudSQLPostgresPass:
		t.Fatal("'CLOUD_SQL_POSTGRES_PASS' not set")
	}

	return map[string]any{
		"type":     CloudSQLPostgresSourceType,
		"project":  CloudSQLPostgresProject,
		"instance": CloudSQLPostgresInstance,
		"region":   CloudSQLPostgresRegion,
		"database": CloudSQLPostgresDatabase,
		"user":     CloudSQLPostgresUser,
		"password": CloudSQLPostgresPass,
	}
}

func initCloudSQLPgConnectionPool(ctx context.Context, project, region, instance, ip_type, user, pass, dbname string) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", user, pass, dbname)
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("unable to parse connection uri: %w", err)
	}

	// Create a new dialer with options
	dialOpts, err := tests.GetCloudSQLDialOpts(ip_type)
	if err != nil {
		return nil, err
	}
	d, err := cloudsqlconn.NewDialer(ctx, cloudsqlconn.WithDefaultDialOptions(dialOpts...))
	if err != nil {
		return nil, fmt.Errorf("unable to parse connection uri: %w", err)
	}

	// Tell the driver to use the Cloud SQL Go Connector to create connections
	i := fmt.Sprintf("%s:%s:%s", project, region, instance)
	config.ConnConfig.DialFunc = func(ctx context.Context, _ string, instance string) (net.Conn, error) {
		return d.Dial(ctx, i)
	}

	// Interact with the driver directly as you normally would
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}
	return pool, nil
}

func TestCloudSQLPgListTools(t *testing.T) {
	getCloudSQLPgVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	args := []string{"--prebuilt", "cloud-sql-postgres"}

	cmd, cleanup, err := tests.StartCmd(ctx, map[string]any{}, args...)
	if err != nil {
		t.Fatalf("command initialization returned an error: %v", err)
	}
	defer cleanup()

	waitCtx, cancelWait := context.WithTimeout(ctx, 10*time.Second)
	defer cancelWait()
	out, err := testutils.WaitForString(waitCtx, regexp.MustCompile(`Server ready to serve`), cmd.Out)
	if err != nil {
		t.Logf("toolbox command logs: \n%s", out)
		t.Fatalf("toolbox didn't start successfully: %v", err)
	}

	// We expect standard Postgres tools to be listed
	_, tools, err := tests.GetMCPToolsList(t, ctx, nil)
	if err != nil {
		t.Fatalf("failed to get tools list: %v", err)
	}

	if len(tools) == 0 {
		t.Errorf("expected tools to be listed, got none")
	}
}

func TestCloudSQLPgCallTool(t *testing.T) {
	sourceConfig := getCloudSQLPgVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, err := initCloudSQLPgConnectionPool(ctx, CloudSQLPostgresProject, CloudSQLPostgresRegion, CloudSQLPostgresInstance, "public", CloudSQLPostgresUser, CloudSQLPostgresPass, CloudSQLPostgresDatabase)
	if err != nil {
		t.Fatalf("unable to create Cloud SQL connection pool: %s", err)
	}
	// Note: Don't defer pool.Close() here - the pool is only used for test setup/teardown.
	// Closing it causes indefinite hangs due to background goroutines in the driver.

	uniqueID := strings.ReplaceAll(uuid.New().String(), "-", "")

	t.Cleanup(func() {
		tests.CleanupPostgresTables(t, context.Background(), pool, uniqueID)
	})

	tableNameParam := "param_table_" + uniqueID
	tableNameAuth := "auth_table_" + uniqueID

	// set up data for param tool
	createParamTableStmt, insertParamTableStmt, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, paramTestParams := tests.GetPostgresSQLParamToolInfo(tableNameParam)
	teardownTable1 := tests.SetupPostgresSQLTable(t, ctx, pool, createParamTableStmt, insertParamTableStmt, tableNameParam, paramTestParams)
	defer teardownTable1(t)

	// set up data for auth tool
	createAuthTableStmt, insertAuthTableStmt, authToolStmt, authTestParams := tests.GetPostgresSQLAuthToolInfo(tableNameAuth)
	teardownTable2 := tests.SetupPostgresSQLTable(t, ctx, pool, createAuthTableStmt, insertAuthTableStmt, tableNameAuth, authTestParams)
	defer teardownTable2(t)

	// Set up table for semantic search
	vectorTableName, tearDownVectorTable := tests.SetupPostgresVectorTable(t, ctx, pool)
	defer tearDownVectorTable(t)

	// Write config into a file and pass it to command
	toolsFile := tests.GetToolsConfig(sourceConfig, CloudSQLPostgresToolType, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, authToolStmt)
	toolsFile = tests.AddExecuteSqlConfig(t, toolsFile, "postgres-execute-sql")
	tmplSelectCombined, tmplSelectFilterCombined := tests.GetPostgresSQLTmplToolStatement()
	toolsFile = tests.AddTemplateParamConfig(t, toolsFile, CloudSQLPostgresToolType, tmplSelectCombined, tmplSelectFilterCombined, "")

	// Add semantic search tool config
	insertStmt, searchStmt := tests.GetPostgresVectorSearchStmts(vectorTableName)
	toolsFile = tests.AddSemanticSearchConfig(t, toolsFile, CloudSQLPostgresToolType, insertStmt, searchStmt)

	toolsFile = tests.AddPostgresPrebuiltConfig(t, toolsFile)

	cmd, cleanup, err := tests.StartCmd(ctx, toolsFile)
	if err != nil {
		t.Fatalf("command initialization returned an error: %v", err)
	}
	defer cleanup()

	waitCtx, cancelWait := context.WithTimeout(ctx, 10*time.Second)
	defer cancelWait()
	out, err := testutils.WaitForString(waitCtx, regexp.MustCompile(`Server ready to serve`), cmd.Out)
	if err != nil {
		t.Logf("toolbox command logs: \n%s", out)
		t.Fatalf("toolbox didn't start successfully: %v", err)
	}

	// Get configs for tests
	select1Want, _, createTableStatement, _ := tests.GetPostgresWants()

	// Run custom tool tests via MCP
	tests.RunMCPToolInvokeTest(t, ctx, select1Want)

	// Run execute-sql tool tests via MCP
	tests.RunMCPExecuteSqlToolInvokeTest(t, ctx, createTableStatement, select1Want)

	// Run template parameters tool tests via MCP
	tableNameTemplateParam := "template_param_table_" + uniqueID
	tests.RunMCPToolInvokeWithTemplateParameters(t, ctx, tableNameTemplateParam)

	// Run shared Postgres tests
	tests.RunMCPPostgresListViewsTest(t, ctx, pool)
	tests.RunMCPPostgresListSchemasTest(t, ctx, pool, CloudSQLPostgresUser, uniqueID)
	tests.RunMCPPostgresListActiveQueriesTest(t, ctx, pool)
	tests.RunMCPPostgresListAvailableExtensionsTest(t, ctx)
	tests.RunMCPPostgresListInstalledExtensionsTest(t, ctx)
	tests.RunMCPPostgresDatabaseOverviewTest(t, ctx, pool)
	tests.RunMCPPostgresListTriggersTest(t, ctx, pool)
	tests.RunMCPPostgresListIndexesTest(t, ctx, pool)
	tests.RunMCPPostgresListSequencesTest(t, ctx, pool)
	tests.RunMCPPostgresLongRunningTransactionsTest(t, ctx, pool)
	tests.RunMCPPostgresListLocksTest(t, ctx, pool)
	tests.RunMCPPostgresReplicationStatsTest(t, ctx, pool)
	tests.RunMCPPostgresGetColumnCardinalityTest(t, ctx, pool)
	tests.RunMCPPostgresListTableStatsTest(t, ctx, pool)
	tests.RunMCPPostgresListPublicationTablesTest(t, ctx, pool)
	tests.RunMCPPostgresListTableSpacesTest(t, ctx)
	tests.RunMCPPostgresListPgSettingsTest(t, ctx, pool)
	tests.RunMCPPostgresListDatabaseStatsTest(t, ctx, pool)
	tests.RunMCPPostgresListRolesTest(t, ctx, pool)
	tests.RunMCPPostgresListStoredProcedureTest(t, ctx, pool)

	toolsToTest := map[string]string{
		"list_autovacuum_configurations": `{}`,
		"list_memory_configurations":     `{}`,
		"list_top_bloated_tables":        `{"limit": 10}`,
		"list_replication_slots":         `{}`,
		"list_invalid_indexes":           `{}`,
		"get_query_plan":                 `{"query": "SELECT 1"}`,
	}
	tests.RunMCPStatementToolsTest(t, ctx, toolsToTest)
}
