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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
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

func addPrebuiltToolConfig(t *testing.T, config map[string]any) map[string]any {
	tools, ok := config["tools"].(map[string]any)
	if !ok {
		t.Fatalf("unable to get tools from config")
	}
	tools["list_tables"] = map[string]any{
		"kind":        PostgresListTablesToolKind,
		"source":      "my-instance",
		"description": "Lists tables in the database.",
	}
	tools["list_active_queries"] = map[string]any{
		"kind":        PostgresListActiveQueriesToolKind,
		"source":      "my-instance",
		"description": "Lists active queries in the database.",
	}

	tools["list_installed_extensions"] = map[string]any{
		"kind":        PostgresListInstalledExtensionsToolKind,
		"source":      "my-instance",
		"description": "Lists installed extensions in the database.",
	}

	tools["list_available_extensions"] = map[string]any{
		"kind":        PostgresListAvailableExtensionsToolKind,
		"source":      "my-instance",
		"description": "Lists available extensions in the database.",
	}

	config["tools"] = tools
	return config
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
	tests.RunPostgresLongRunningTransactionsTest(t, ctx, pool)
	tests.RunPostgresListLocksTest(t, ctx, pool)
	tests.RunPostgresReplicationStatsTest(t, ctx, pool)
	tests.RunPostgresListIndexesTest(t, ctx, pool)
	tests.RunPostgresListSequencesTest(t, ctx, pool)
}

func runPostgresListLocksTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tableName string) {
	invokeTcs := []struct {
		name                string
		requestBody         io.Reader
		clientHoldSecs      int
		waitSecsBeforeCheck int
		wantStatusCode      int
		expectLockPresent   bool
	}{
		{
			name:                "invoke list_locks when the system is idle",
			requestBody:         bytes.NewBufferString(`{}`),
			clientHoldSecs:      0,
			waitSecsBeforeCheck: 0,
			wantStatusCode:      http.StatusOK,
			expectLockPresent:   false,
		},
		{
			name:                "invoke list_locks when a transaction holds a FOR UPDATE lock",
			requestBody:         bytes.NewBufferString(`{}`),
			clientHoldSecs:      8,
			waitSecsBeforeCheck: 1,
			wantStatusCode:      http.StatusOK,
			expectLockPresent:   true,
		},
	}

	var wg sync.WaitGroup
	for _, tc := range invokeTcs {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if tc.clientHoldSecs > 0 {
				wg.Add(1)
				go func() {
					defer wg.Done()

					tx, err := pool.Begin(ctx)
					if err != nil {
						t.Errorf("unable to begin transaction: %s", err)
						return
					}
					defer func() { _ = tx.Rollback(ctx) }()

					// acquire a row-level lock and hold it
					rows, err := tx.Query(ctx, fmt.Sprintf("SELECT id FROM %s LIMIT 1 FOR UPDATE", tableName))
					if err != nil {
						t.Errorf("failed to execute FOR UPDATE: %v", err)
						return
					}
					// ensure rows are read so the query is executed
					for rows.Next() {
						var id any
						_ = rows.Scan(&id)
					}
					rows.Close()

					time.Sleep(time.Duration(tc.clientHoldSecs) * time.Second)
					// rollback to release lock
					if err := tx.Rollback(ctx); err != nil {
						// ignore errors from rollback during test shutdown
					}
				}()
			}

			if tc.waitSecsBeforeCheck > 0 {
				time.Sleep(time.Duration(tc.waitSecsBeforeCheck) * time.Second)
			}

			const api = "http://127.0.0.1:5000/api/tool/list_locks/invoke"
			req, err := http.NewRequest(http.MethodPost, api, tc.requestBody)
			if err != nil {
				t.Fatalf("unable to create request: %v", err)
			}
			req.Header.Add("Content-type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("unable to send request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.wantStatusCode {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("wrong status code: got %d, want %d, body: %s", resp.StatusCode, tc.wantStatusCode, string(body))
			}
			if tc.wantStatusCode != http.StatusOK {
				return
			}

			var bodyWrapper struct {
				Result json.RawMessage `json:"result"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&bodyWrapper); err != nil {
				t.Fatalf("error decoding response wrapper: %v", err)
			}

			var resultString string
			if err := json.Unmarshal(bodyWrapper.Result, &resultString); err != nil {
				resultString = string(bodyWrapper.Result)
			}

			var details []map[string]any
			if err := json.Unmarshal([]byte(resultString), &details); err != nil {
				t.Fatalf("failed to unmarshal nested locks result: %v", err)
			}

			// If we expect a lock present, verify at least one returned row's query contains FOR UPDATE
			found := false
			for _, item := range details {
				if qv, ok := item["query"]; ok {
					if qs, ok := qv.(string); ok && strings.Contains(strings.ToUpper(qs), "FOR UPDATE") {
						found = true
						break
					}
				}
			}
			if tc.expectLockPresent && !found {
				t.Errorf("expected to find a FOR UPDATE lock in list_locks result, got: %#v", details)
			}
			if !tc.expectLockPresent && found {
				t.Errorf("did not expect a FOR UPDATE lock, but found one in: %#v", details)
			}
		})
	}
	wg.Wait()
}

func runPostgresReplicationStatsTest(t *testing.T) {
	invokeTcs := []struct {
		name           string
		api            string
		requestBody    io.Reader
		wantStatusCode int
	}{
		{
			name:           "invoke replication_stats output",
			api:            "http://127.0.0.1:5000/api/tool/replication_stats/invoke",
			wantStatusCode: http.StatusOK,
			requestBody:    bytes.NewBuffer([]byte(`{}`)),
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost, tc.api, tc.requestBody)
			if err != nil {
				t.Fatalf("unable to create request: %s", err)
			}
			req.Header.Add("Content-type", "application/json")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("unable to send request: %s", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.wantStatusCode {
				bodyBytes, _ := io.ReadAll(resp.Body)
				t.Fatalf("response status code is not 200, got %d: %s", resp.StatusCode, string(bodyBytes))
			}

			// Intentionally not adding the output check as output depends on the postgres instance used where the functional test runs.
		})
	}
}
