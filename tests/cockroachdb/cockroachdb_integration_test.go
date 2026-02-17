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

package cockroachdb

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
	CockroachDBSourceType = "cockroachdb"
	CockroachDBToolType   = "cockroachdb-sql"
	CockroachDBDatabase   = getEnvOrDefault("COCKROACHDB_DATABASE", "defaultdb")
	CockroachDBHost       = getEnvOrDefault("COCKROACHDB_HOST", "localhost")
	CockroachDBPort       = getEnvOrDefault("COCKROACHDB_PORT", "26257")
	CockroachDBUser       = getEnvOrDefault("COCKROACHDB_USER", "root")
	CockroachDBPass       = getEnvOrDefault("COCKROACHDB_PASS", "")
)

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getCockroachDBVars(t *testing.T) map[string]any {
	if CockroachDBHost == "" {
		t.Skip("COCKROACHDB_HOST not set, skipping CockroachDB integration test")
	}

	return map[string]any{
		"type":           CockroachDBSourceType,
		"host":           CockroachDBHost,
		"port":           CockroachDBPort,
		"database":       CockroachDBDatabase,
		"user":           CockroachDBUser,
		"password":       CockroachDBPass,
		"maxRetries":     5,
		"retryBaseDelay": "500ms",
		"queryParams": map[string]string{
			"sslmode": "disable",
		},
	}
}

func initCockroachDBConnectionPool(host, port, user, pass, dbname string) (*pgxpool.Pool, error) {
	connURL := &url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(user, pass),
		Host:     fmt.Sprintf("%s:%s", host, port),
		Path:     dbname,
		RawQuery: "sslmode=disable&application_name=cockroachdb-integration-test",
	}
	pool, err := pgxpool.New(context.Background(), connURL.String())
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	return pool, nil
}

func TestCockroachDB(t *testing.T) {
	sourceConfig := getCockroachDBVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	var args []string

	pool, err := initCockroachDBConnectionPool(CockroachDBHost, CockroachDBPort, CockroachDBUser, CockroachDBPass, CockroachDBDatabase)
	if err != nil {
		t.Fatalf("unable to create cockroachdb connection pool: %s", err)
	}
	// Note: Don't defer pool.Close() here - the pool is only used for test setup/teardown.
	// Closing it explicitly can cause hangs if the server's pool is still active.
	// The pool will be cleaned up when the test exits.

	// Verify CockroachDB version
	var version string
	err = pool.QueryRow(ctx, "SELECT version()").Scan(&version)
	if err != nil {
		t.Fatalf("failed to query version: %s", err)
	}
	if !strings.Contains(version, "CockroachDB") {
		t.Fatalf("not connected to CockroachDB, got: %s", version)
	}
	t.Logf("✅ Connected to: %s", version)

	// Generate a unique ID
	uniqueID := strings.ReplaceAll(uuid.New().String(), "-", "")

	// This will execute after all tool tests complete (success, fail, or t.Fatal)
	t.Cleanup(func() {
		tests.CleanupPostgresTables(t, context.Background(), pool, uniqueID)
	})

	//Create table names using the UUID
	tableNameParam := "param_table_" + uniqueID
	tableNameAuth := "auth_table_" + uniqueID
	tableNameTemplateParam := "template_param_table_" + uniqueID

	// set up data for param tool (using CockroachDB explicit INT primary keys)
	createParamTableStmt, insertParamTableStmt, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, paramTestParams := tests.GetCockroachDBParamToolInfo(tableNameParam)
	teardownTable1 := tests.SetupPostgresSQLTable(t, ctx, pool, createParamTableStmt, insertParamTableStmt, tableNameParam, paramTestParams)
	defer teardownTable1(t)

	// set up data for auth tool
	createAuthTableStmt, insertAuthTableStmt, authToolStmt, authTestParams := tests.GetCockroachDBAuthToolInfo(tableNameAuth)
	teardownTable2 := tests.SetupPostgresSQLTable(t, ctx, pool, createAuthTableStmt, insertAuthTableStmt, tableNameAuth, authTestParams)
	defer teardownTable2(t)

	// Write config into a file and pass it to command
	toolsFile := tests.GetToolsConfig(sourceConfig, CockroachDBToolType, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, authToolStmt)

	// Add execute-sql tool with write-enabled source (CockroachDB MCP security requires explicit opt-in)
	toolsFile = addCockroachDBExecuteSqlConfig(t, toolsFile, sourceConfig)

	tmplSelectCombined, tmplSelectFilterCombined := tests.GetPostgresSQLTmplToolStatement()
	toolsFile = tests.AddTemplateParamConfig(t, toolsFile, CockroachDBToolType, tmplSelectCombined, tmplSelectFilterCombined, "")

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

	// Get configs for tests (use CockroachDB-specific expectations)
	select1Want, mcpMyFailToolWant, createTableStatement, mcpSelect1Want := tests.GetCockroachDBWants()

	// Run required integration test suites (per CONTRIBUTING.md)
	t.Run("ToolGetTest", func(t *testing.T) {
		tests.RunToolGetTest(t)
	})

	t.Run("ToolInvokeTest", func(t *testing.T) {
		tests.RunToolInvokeTest(t, select1Want)
	})

	t.Run("MCPToolCallMethod", func(t *testing.T) {
		tests.RunMCPToolCallMethod(t, mcpMyFailToolWant, mcpSelect1Want)
	})

	t.Run("ExecuteSqlToolInvokeTest", func(t *testing.T) {
		tests.RunExecuteSqlToolInvokeTest(t, createTableStatement, select1Want)
	})

	t.Run("ToolInvokeWithTemplateParameters", func(t *testing.T) {
		tests.RunToolInvokeWithTemplateParameters(t, tableNameTemplateParam)
	})

	t.Logf("✅✅✅ All CockroachDB integration tests passed!")
}

// addCockroachDBExecuteSqlConfig adds execute-sql tool with write-enabled source
// CockroachDB has MCP security enabled by default, so execute-sql needs a separate source with enableWriteMode
func addCockroachDBExecuteSqlConfig(t *testing.T, config map[string]any, baseSourceConfig map[string]any) map[string]any {
	// Add write-enabled source for execute-sql tool
	sources, ok := config["sources"].(map[string]any)
	if !ok {
		t.Fatalf("unable to get sources from config")
	}

	// Create a copy of the base source config with write mode enabled
	writeEnabledSource := make(map[string]any)
	for k, v := range baseSourceConfig {
		writeEnabledSource[k] = v
	}
	writeEnabledSource["enableWriteMode"] = true
	writeEnabledSource["readOnlyMode"] = false

	sources["my-write-instance"] = writeEnabledSource

	// Add tools using the write-enabled source
	tools, ok := config["tools"].(map[string]any)
	if !ok {
		t.Fatalf("unable to get tools from config")
	}

	tools["my-exec-sql-tool"] = map[string]any{
		"type":        "cockroachdb-execute-sql",
		"source":      "my-write-instance",
		"description": "Tool to execute sql",
	}
	tools["my-auth-exec-sql-tool"] = map[string]any{
		"type":        "cockroachdb-execute-sql",
		"source":      "my-write-instance",
		"description": "Tool to execute sql",
		"authRequired": []string{
			"my-google-auth",
		},
	}

	return config
}
