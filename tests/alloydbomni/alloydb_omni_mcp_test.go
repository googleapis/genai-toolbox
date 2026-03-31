package alloydbomni

import (
	"context"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"
)

func TestAlloyDBOmni_MCP(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	AlloyDBHost, AlloyDBPort, containerCleanup := setupAlloyDBContainer(ctx, t)
	defer containerCleanup()

	os.Setenv("ALLOYDB_OMNI_HOST", AlloyDBHost)
	os.Setenv("ALLOYDB_OMNI_PORT", AlloyDBPort)
	os.Setenv("ALLOYDB_OMNI_USER", AlloyDBUser)
	os.Setenv("ALLOYDB_OMNI_PASSWORD", AlloyDBPass)
	os.Setenv("ALLOYDB_OMNI_DATABASE", AlloyDBDatabase)

	// Generate a unique ID
	uniqueID := strings.ReplaceAll(uuid.New().String(), "-", "")

	args := []string{"--prebuilt", "alloydb-omni"}

	pool, err := initPostgresConnectionPool(AlloyDBHost, AlloyDBPort, AlloyDBUser, AlloyDBPass, AlloyDBDatabase)
	if err != nil {
		t.Fatalf("unable to create alloydb connection pool: %s", err)
	}

	cmd, cleanup, err := tests.StartCmd(ctx, map[string]any{}, args...)
	if err != nil {
		t.Fatalf("command initialization returned an error: %s", err)
	}
	defer cleanup()

	// Wait for server to be ready
	waitCtx, waitCancel := context.WithTimeout(ctx, 30*time.Second)
	defer waitCancel()

	out, err := testutils.WaitForString(waitCtx, regexp.MustCompile(`Server ready to serve`), cmd.Out)
	if err != nil {
		t.Logf("toolbox command logs: \n%s", out)
		t.Fatalf("toolbox didn't start successfully: %s", err)
	}

	// Run MCP tests
	tests.RunMCPPostgresListViewsTest(t, ctx, pool)
	tests.RunMCPPostgresListSchemasTest(t, ctx, pool, AlloyDBUser, uniqueID)
	tests.RunMCPPostgresListActiveQueriesTest(t, ctx, pool)
	tests.RunMCPPostgresListAvailableExtensionsTest(t)
	tests.RunMCPPostgresListInstalledExtensionsTest(t)
	tests.RunMCPPostgresDatabaseOverviewTest(t)
	tests.RunMCPPostgresListTriggersTest(t, ctx, pool)
	tests.RunMCPPostgresListIndexesTest(t, ctx, pool)
	tests.RunMCPPostgresListSequencesTest(t, ctx, pool)
	tests.RunMCPPostgresLongRunningTransactionsTest(t)
	tests.RunMCPPostgresListLocksTest(t, ctx, pool)
	tests.RunMCPPostgresReplicationStatsTest(t)
	tests.RunMCPPostgresGetColumnCardinalityTest(t, ctx, pool)
	tests.RunMCPPostgresListTableStatsTest(t, ctx, pool)
	tests.RunMCPPostgresListPublicationTablesTest(t, ctx, pool)
	tests.RunMCPPostgresListTableSpacesTest(t)
	tests.RunMCPPostgresListPgSettingsTest(t)
	tests.RunMCPPostgresListDatabaseStatsTest(t, ctx, pool)
	tests.RunMCPPostgresListRolesTest(t, ctx, pool)
	tests.RunMCPPostgresListStoredProcedureTest(t, ctx, pool)

	// Verify Postgres statement tools that return system/diagnostic information.
	t.Run("MCP list_autovacuum_configurations", func(t *testing.T) {
		tests.RunMCPCustomToolCallMethod(t, "list_autovacuum_configurations", map[string]any{}, "")
	})
	t.Run("MCP list_memory_configurations", func(t *testing.T) {
		tests.RunMCPCustomToolCallMethod(t, "list_memory_configurations", map[string]any{}, "")
	})
	t.Run("MCP list_top_bloated_tables", func(t *testing.T) {
		tests.RunMCPCustomToolCallMethod(t, "list_top_bloated_tables", map[string]any{"limit": 10}, "")
	})
	t.Run("MCP list_replication_slots", func(t *testing.T) {
		tests.RunMCPCustomToolCallMethod(t, "list_replication_slots", map[string]any{}, "")
	})
	t.Run("MCP list_invalid_indexes", func(t *testing.T) {
		tests.RunMCPCustomToolCallMethod(t, "list_invalid_indexes", map[string]any{}, "")
	})
	t.Run("MCP get_query_plan", func(t *testing.T) {
		tests.RunMCPCustomToolCallMethod(t, "get_query_plan", map[string]any{"query": "SELECT 1"}, "Plan")
	})
	t.Run("MCP list_columnar_configurations", func(t *testing.T) {
		tests.RunMCPCustomToolCallMethod(t, "list_columnar_configurations", map[string]any{}, "")
	})
	t.Run("MCP list_columnar_recommended_columns", func(t *testing.T) {
		tests.RunMCPCustomToolCallMethod(t, "list_columnar_recommended_columns", map[string]any{}, "")
	})
}
