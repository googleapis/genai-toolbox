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

package alloydbomni

import (
	"context"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/googleapis/mcp-toolbox/internal/testutils"
	"github.com/googleapis/mcp-toolbox/tests"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	AlloyDBUser     = "postgres"
	AlloyDBPass     = "mysecretpassword"
	AlloyDBDatabase = "postgres"
)

func setupAlloyDBContainer(ctx context.Context, t *testing.T) (string, string, func()) {
	t.Helper()

	req := testcontainers.ContainerRequest{
		Image:        "google/alloydbomni:16.9.0-ubi9", // Pinning version for stability
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_PASSWORD": AlloyDBPass,
		},
		WaitingFor: wait.ForAll(
			wait.ForLog("database system was shut down at"),
			wait.ForLog("database system is ready to accept connections"),
			wait.ForExposedPort(),
		),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start alloydb container: %s", err)
	}

	cleanup := func() {
		if err := container.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate container: %s", err)
		}
	}

	host, err := container.Host(ctx)
	if err != nil {
		cleanup()
		t.Fatalf("failed to get container host: %s", err)
	}

	mappedPort, err := container.MappedPort(ctx, "5432")
	if err != nil {
		cleanup()
		t.Fatalf("failed to get container mapped port: %s", err)
	}

	return host, mappedPort.Port(), cleanup
}

func TestAlloyDBOmniListTools(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	AlloyDBHost, AlloyDBPort, containerCleanup := setupAlloyDBContainer(ctx, t)
	defer containerCleanup()

	os.Setenv("ALLOYDB_OMNI_HOST", AlloyDBHost)
	os.Setenv("ALLOYDB_OMNI_PORT", AlloyDBPort)
	os.Setenv("ALLOYDB_OMNI_USER", AlloyDBUser)
	os.Setenv("ALLOYDB_OMNI_PASSWORD", AlloyDBPass)
	os.Setenv("ALLOYDB_OMNI_DATABASE", AlloyDBDatabase)

	args := []string{"--prebuilt", "alloydb-omni"}

	cmd, cleanup, err := tests.StartCmd(ctx, map[string]any{}, args...)
	if err != nil {
		t.Fatalf("command initialization returned an error: %s", err)
	}
	defer cleanup()

	waitCtx, waitCancel := context.WithTimeout(ctx, 30*time.Second)
	defer waitCancel()

	_, err = testutils.WaitForString(waitCtx, regexp.MustCompile(`Server ready to serve`), cmd.Out)
	if err != nil {
		t.Fatalf("toolbox didn't start successfully: %s", err)
	}

	statusCode, toolsList, err := tests.GetMCPToolsList(t, ctx, nil)
	if err != nil {
		t.Fatalf("native error executing tools/list: %s", err)
	}
	if statusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", statusCode)
	}

	found := false
	for _, tool := range toolsList {
		toolMap, ok := tool.(map[string]any)
		if !ok {
			continue
		}
		if toolMap["name"] == "list_autovacuum_configurations" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected tool 'list_autovacuum_configurations' not found in list")
	}
}

func TestAlloyDBOmniCallTool(t *testing.T) {
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

	pool, err := tests.InitPostgresConnectionPool(AlloyDBHost, AlloyDBPort, AlloyDBUser, AlloyDBPass, AlloyDBDatabase)
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

	t.Logf("AlloyDB Omni container started on %s:%s", AlloyDBHost, AlloyDBPort)
	t.Logf("Toolbox server started with output: %s", out)

	tests.RunMCPPostgresListViewsTest(t, ctx, pool)
	tests.RunMCPPostgresListSchemasTest(t, ctx, pool, AlloyDBUser, uniqueID)
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
		"list_columnar_configurations":   `{}`,
	}
	tests.RunMCPStatementToolsTest(t, ctx, toolsToTest)

	t.Run("list_columnar_recommended_columns_expect_error", func(t *testing.T) {
		statusCode, mcpResp, err := tests.InvokeMCPTool(t, ctx, "list_columnar_recommended_columns", nil, nil)
		if err != nil {
			t.Fatalf("native error: %s", err)
		}
		if statusCode != http.StatusOK {
			t.Fatalf("status code: %d", statusCode)
		}
		if !mcpResp.Result.IsError {
			t.Fatalf("expected error result but got success")
		}
		t.Logf("Got expected error result as expected: %v", mcpResp.Result)
	})
}
