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

package alloydbpg

import (
	"context"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"
)

func TestAlloyDBPG_MCP(t *testing.T) {
	sourceConfig := getAlloyDBPgVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	pool, err := initAlloyDBPgConnectionPool(AlloyDBPostgresProject, AlloyDBPostgresRegion, AlloyDBPostgresCluster, AlloyDBPostgresInstance, "public", AlloyDBPostgresUser, AlloyDBPostgresPass, AlloyDBPostgresDatabase)
	if err != nil {
		t.Fatalf("unable to create AlloyDB connection pool: %s", err)
	}

	uniqueID := strings.ReplaceAll(uuid.New().String(), "-", "")

	t.Cleanup(func() {
		tests.CleanupPostgresTables(t, context.Background(), pool, uniqueID)
	})

	tableNameParam := "param_table_" + uniqueID
	tableNameAuth := "auth_table_" + uniqueID
	tableNameTemplateParam := "tmpl_table_" + uniqueID

	createParamTableStmt, insertParamTableStmt, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, paramTestParams := tests.GetPostgresSQLParamToolInfo(tableNameParam)
	teardownTable1 := tests.SetupPostgresSQLTable(t, ctx, pool, createParamTableStmt, insertParamTableStmt, tableNameParam, paramTestParams)
	defer teardownTable1(t)

	createAuthTableStmt, insertAuthTableStmt, authToolStmt, authTestParams := tests.GetPostgresSQLAuthToolInfo(tableNameAuth)
	teardownTable2 := tests.SetupPostgresSQLTable(t, ctx, pool, createAuthTableStmt, insertAuthTableStmt, tableNameAuth, authTestParams)
	defer teardownTable2(t)

	vectorTableName, tearDownVectorTable := tests.SetupPostgresVectorTable(t, ctx, pool)
	defer tearDownVectorTable(t)

	toolsFile := tests.GetToolsConfig(sourceConfig, AlloyDBPostgresToolType, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, authToolStmt)
	toolsFile = tests.AddExecuteSqlConfig(t, toolsFile, "postgres-execute-sql")
	tmplSelectCombined, tmplSelectFilterCombined := tests.GetPostgresSQLTmplToolStatement()
	toolsFile = tests.AddTemplateParamConfig(t, toolsFile, AlloyDBPostgresToolType, tmplSelectCombined, tmplSelectFilterCombined, "")

	insertStmt, searchStmt := tests.GetPostgresVectorSearchStmts(vectorTableName)
	toolsFile = tests.AddSemanticSearchConfig(t, toolsFile, AlloyDBPostgresToolType, insertStmt, searchStmt)

	toolsFile = tests.AddPostgresPrebuiltConfig(t, toolsFile)

	cmd, cleanup, err := tests.StartCmd(ctx, toolsFile, "--enable-api=false")
	if err != nil {
		t.Fatalf("command initialization returned an error: %s", err)
	}
	defer cleanup()

	waitCtx, cancelWait := context.WithTimeout(ctx, 20*time.Second)
	defer cancelWait()
	out, err := testutils.WaitForString(waitCtx, regexp.MustCompile(`Server ready to serve`), cmd.Out)
	if err != nil {
		t.Logf("toolbox command logs: \n%s", out)
		t.Fatalf("toolbox didn't start successfully: %s", err)
	}

	_, failInvocationWant, createTableStatement, mcpSelect1Want := tests.GetPostgresWants()

	// Generic Tool Tests
	t.Run("MCP tools/call", func(t *testing.T) {
		tests.RunMCPToolCallMethod(t, failInvocationWant, mcpSelect1Want)
	})

	// Execute SQL Tool Tests
	t.Run("MCP execute SQL tools", func(t *testing.T) {
		tests.RunMCPCustomToolCallMethod(t, "my-exec-sql-tool", map[string]any{"sql": "SELECT 1"}, `{"?column?":1}`)
		tests.RunMCPCustomToolCallMethod(t, "my-exec-sql-tool", map[string]any{"sql": createTableStatement}, "null")
		tests.RunMCPCustomToolCallMethod(t, "my-exec-sql-tool", map[string]any{"sql": "SELECT * FROM t"}, "null")
		tests.RunMCPCustomToolCallMethod(t, "my-exec-sql-tool", map[string]any{"sql": "DROP TABLE t"}, "null")
	})

	// Prebuilt Tool Tests
	tests.RunMCPPostgresListTablesTest(t, tableNameParam, tableNameAuth, AlloyDBPostgresUser)
	tests.RunMCPPostgresListViewsTest(t, ctx, pool)
	tests.RunMCPPostgresListSchemasTest(t, ctx, pool, AlloyDBPostgresUser, uniqueID)
	tests.RunMCPPostgresListActiveQueriesTest(t, ctx, pool)
	tests.RunMCPPostgresListAvailableExtensionsTest(t)
	tests.RunMCPPostgresListInstalledExtensionsTest(t)
	tests.RunMCPPostgresDatabaseOverviewTest(t)
	tests.RunMCPPostgresListTriggersTest(t, ctx, pool)
	tests.RunMCPPostgresListIndexesTest(t, ctx, pool)
	tests.RunMCPPostgresListSequencesTest(t, ctx, pool)
	tests.RunMCPPostgresListLocksTest(t, ctx, pool)
	tests.RunMCPPostgresReplicationStatsTest(t)
	tests.RunMCPPostgresLongRunningTransactionsTest(t)
	tests.RunMCPPostgresTemplateToolsTest(t, tableNameTemplateParam)
	tests.RunMCPPostgresListQueryStatsTest(t, ctx, pool)
	tests.RunMCPPostgresGetColumnCardinalityTest(t, ctx, pool)
	tests.RunMCPPostgresListTableStatsTest(t, ctx, pool)
	tests.RunMCPPostgresListPublicationTablesTest(t, ctx, pool)
	tests.RunMCPPostgresListTableSpacesTest(t)
	tests.RunMCPPostgresListPgSettingsTest(t)
	tests.RunMCPPostgresListDatabaseStatsTest(t, ctx, pool)
	tests.RunMCPPostgresListRolesTest(t, ctx, pool)
	tests.RunMCPPostgresListStoredProcedureTest(t, ctx, pool)

	// Statement Tools
	t.Run("MCP statement tools", func(t *testing.T) {
		tests.RunMCPCustomToolCallMethod(t, "list_autovacuum_configurations", map[string]any{}, "")
		tests.RunMCPCustomToolCallMethod(t, "list_memory_configurations", map[string]any{}, "")
		tests.RunMCPCustomToolCallMethod(t, "list_top_bloated_tables", map[string]any{"limit": 10}, "")
		tests.RunMCPCustomToolCallMethod(t, "list_replication_slots", map[string]any{}, "")
		tests.RunMCPCustomToolCallMethod(t, "list_invalid_indexes", map[string]any{}, "")
		tests.RunMCPCustomToolCallMethod(t, "get_query_plan", map[string]any{"query": "SELECT 1"}, "Plan")
	})
}
