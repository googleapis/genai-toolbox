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
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	PostgresSourceKind                      = "postgres"
	PostgresToolKind                        = "postgres-sql"
	PostgresListTablesToolKind              = "postgres-list-tables"
	PostgresListActiveQueriesToolKind       = "postgres-list-active-queries"
	PostgresListInstalledExtensionsToolKind = "postgres-list-installed-extensions"
	PostgresListAvailableExtensionsToolKind = "postgres-list-available-extensions"
	PostgresDatabase                        = os.Getenv("POSTGRES_DATABASE")
	PostgresHost                            = os.Getenv("POSTGRES_HOST")
	PostgresPort                            = os.Getenv("POSTGRES_PORT")
	PostgresUser                            = os.Getenv("POSTGRES_USER")
	PostgresPass                            = os.Getenv("POSTGRES_PASS")
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
	tests.CleanupPostgresTables(t, ctx, pool);

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
	toolsFile = tests.AddPgExecuteSqlConfig(t, toolsFile)
	tmplSelectCombined, tmplSelectFilterCombined := tests.GetPostgresSQLTmplToolStatement()
	toolsFile = tests.AddTemplateParamConfig(t, toolsFile, PostgresToolKind, tmplSelectCombined, tmplSelectFilterCombined, "")

	toolsFile = addPrebuiltToolConfig(t, toolsFile)

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

	// Run specific Postgres tool tests
	runPostgresListTablesTest(t, tableNameParam, tableNameAuth)
	runPostgresListActiveQueriesTest(t, ctx, pool)
	runPostgresListAvailableExtensionsTest(t)
	runPostgresListInstalledExtensionsTest(t)
}

func runPostgresListTablesTest(t *testing.T, tableNameParam, tableNameAuth string) {
	// TableNameParam columns to construct want
	paramTableColumns := fmt.Sprintf(`[
		{"data_type": "integer", "column_name": "id", "column_default": "nextval('%s_id_seq'::regclass)", "is_not_nullable": true, "ordinal_position": 1, "column_comment": null},
		{"data_type": "text", "column_name": "name", "column_default": null, "is_not_nullable": false, "ordinal_position": 2, "column_comment": null}
	]`, tableNameParam)

	// TableNameAuth columns to construct want
	authTableColumns := fmt.Sprintf(`[
		{"data_type": "integer", "column_name": "id", "column_default": "nextval('%s_id_seq'::regclass)", "is_not_nullable": true, "ordinal_position": 1, "column_comment": null},
		{"data_type": "text", "column_name": "name", "column_default": null, "is_not_nullable": false, "ordinal_position": 2, "column_comment": null},
		{"data_type": "text", "column_name": "email", "column_default": null, "is_not_nullable": false, "ordinal_position": 3, "column_comment": null}
	]`, tableNameAuth)

	const (
		// Template to construct detailed output want
		detailedObjectTemplate = `{
            "object_name": "%[1]s", "schema_name": "public",
            "object_details": {
                "owner": "%[3]s", "comment": null,
                "indexes": [{"is_primary": true, "is_unique": true, "index_name": "%[1]s_pkey", "index_method": "btree", "index_columns": ["id"], "index_definition": "CREATE UNIQUE INDEX %[1]s_pkey ON public.%[1]s USING btree (id)"}],
                "triggers": [], "columns": %[2]s, "object_name": "%[1]s", "object_type": "TABLE", "schema_name": "public",
                "constraints": [{"constraint_name": "%[1]s_pkey", "constraint_type": "PRIMARY KEY", "constraint_columns": ["id"], "constraint_definition": "PRIMARY KEY (id)", "foreign_key_referenced_table": null, "foreign_key_referenced_columns": null}]
            }
        }`

		// Template to construct simple output want
		simpleObjectTemplate = `{"object_name":"%s", "schema_name":"public", "object_details":{"name":"%s"}}`
	)

	// Helper to build json for detailed want
	getDetailedWant := func(tableName, columnJSON string) string {
		return fmt.Sprintf(detailedObjectTemplate, tableName, columnJSON, PostgresUser)
	}

	// Helper to build template for simple want
	getSimpleWant := func(tableName string) string {
		return fmt.Sprintf(simpleObjectTemplate, tableName, tableName)
	}

	invokeTcs := []struct {
		name           string
		api            string
		requestBody    io.Reader
		wantStatusCode int
		want           string
	}{
		{
			name:           "invoke list_tables all tables detailed output",
			api:            "http://127.0.0.1:5000/api/tool/list_tables/invoke",
			requestBody:    bytes.NewBuffer([]byte(`{"table_names": ""}`)),
			wantStatusCode: http.StatusOK,
			want:           fmt.Sprintf("[%s,%s]", getDetailedWant(tableNameAuth, authTableColumns), getDetailedWant(tableNameParam, paramTableColumns)),
		},
		{
			name:           "invoke list_tables all tables simple output",
			api:            "http://127.0.0.1:5000/api/tool/list_tables/invoke",
			requestBody:    bytes.NewBuffer([]byte(`{"table_names": "", "output_format": "simple"}`)),
			wantStatusCode: http.StatusOK,
			want:           fmt.Sprintf("[%s,%s]", getSimpleWant(tableNameAuth), getSimpleWant(tableNameParam)),
		},
		{
			name:           "invoke list_tables detailed output",
			api:            "http://127.0.0.1:5000/api/tool/list_tables/invoke",
			requestBody:    bytes.NewBuffer([]byte(fmt.Sprintf(`{"table_names": "%s"}`, tableNameAuth))),
			wantStatusCode: http.StatusOK,
			want:           fmt.Sprintf("[%s]", getDetailedWant(tableNameAuth, authTableColumns)),
		},
		{
			name:           "invoke list_tables simple output",
			api:            "http://127.0.0.1:5000/api/tool/list_tables/invoke",
			requestBody:    bytes.NewBuffer([]byte(fmt.Sprintf(`{"table_names": "%s", "output_format": "simple"}`, tableNameAuth))),
			wantStatusCode: http.StatusOK,
			want:           fmt.Sprintf("[%s]", getSimpleWant(tableNameAuth)),
		},
		{
			name:           "invoke list_tables with invalid output format",
			api:            "http://127.0.0.1:5000/api/tool/list_tables/invoke",
			requestBody:    bytes.NewBuffer([]byte(`{"table_names": "", "output_format": "abcd"}`)),
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "invoke list_tables with malformed table_names parameter",
			api:            "http://127.0.0.1:5000/api/tool/list_tables/invoke",
			requestBody:    bytes.NewBuffer([]byte(`{"table_names": 12345, "output_format": "detailed"}`)),
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "invoke list_tables with multiple table names",
			api:            "http://127.0.0.1:5000/api/tool/list_tables/invoke",
			requestBody:    bytes.NewBuffer([]byte(fmt.Sprintf(`{"table_names": "%s,%s"}`, tableNameParam, tableNameAuth))),
			wantStatusCode: http.StatusOK,
			want:           fmt.Sprintf("[%s,%s]", getDetailedWant(tableNameAuth, authTableColumns), getDetailedWant(tableNameParam, paramTableColumns)),
		},
		{
			name:           "invoke list_tables with non-existent table",
			api:            "http://127.0.0.1:5000/api/tool/list_tables/invoke",
			requestBody:    bytes.NewBuffer([]byte(`{"table_names": "non_existent_table"}`)),
			wantStatusCode: http.StatusOK,
			want:           `null`,
		},
		{
			name:           "invoke list_tables with one existing and one non-existent table",
			api:            "http://127.0.0.1:5000/api/tool/list_tables/invoke",
			requestBody:    bytes.NewBuffer([]byte(fmt.Sprintf(`{"table_names": "%s,non_existent_table"}`, tableNameParam))),
			wantStatusCode: http.StatusOK,
			want:           fmt.Sprintf("[%s]", getDetailedWant(tableNameParam, paramTableColumns)),
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

			if tc.wantStatusCode == http.StatusOK {
				var bodyWrapper map[string]json.RawMessage
				respBytes, err := io.ReadAll(resp.Body)
				if err != nil {
					t.Fatalf("error reading response body: %s", err)
				}

				if err := json.Unmarshal(respBytes, &bodyWrapper); err != nil {
					t.Fatalf("error parsing response wrapper: %s, body: %s", err, string(respBytes))
				}

				resultJSON, ok := bodyWrapper["result"]
				if !ok {
					t.Fatal("unable to find 'result' in response body")
				}

				var resultString string
				if err := json.Unmarshal(resultJSON, &resultString); err != nil {
					t.Fatalf("'result' is not a JSON-encoded string: %s", err)
				}

				var got, want []any

				if err := json.Unmarshal([]byte(resultString), &got); err != nil {
					t.Fatalf("failed to unmarshal actual result string: %v", err)
				}
				if err := json.Unmarshal([]byte(tc.want), &want); err != nil {
					t.Fatalf("failed to unmarshal expected want string: %v", err)
				}

				sort.SliceStable(got, func(i, j int) bool {
					return fmt.Sprintf("%v", got[i]) < fmt.Sprintf("%v", got[j])
				})
				sort.SliceStable(want, func(i, j int) bool {
					return fmt.Sprintf("%v", want[i]) < fmt.Sprintf("%v", want[j])
				})

				if !reflect.DeepEqual(got, want) {
					t.Errorf("Unexpected result: got  %#v, want: %#v", got, want)
				}
			}
		})
	}
}

func runPostgresListActiveQueriesTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	type queryListDetails struct {
		ProcessId        any    `json:"pid"`
		User             string `json:"user"`
		Datname          string `json:"datname"`
		ApplicationName  string `json:"application_name"`
		ClientAddress    string `json:"client_addr"`
		State            string `json:"state"`
		WaitEventType    string `json:"wait_event_type"`
		WaitEvent        string `json:"wait_event"`
		BackendStart     any    `json:"backend_start"`
		TransactionStart any    `json:"xact_start"`
		QueryStart       any    `json:"query_start"`
		QueryDuration    any    `json:"query_duration"`
		Query            string `json:"query"`
	}

	singleQueryWanted := queryListDetails{
		ProcessId:        any(nil),
		User:             "",
		Datname:          "",
		ApplicationName:  "",
		ClientAddress:    "",
		State:            "",
		WaitEventType:    "",
		WaitEvent:        "",
		BackendStart:     any(nil),
		TransactionStart: any(nil),
		QueryStart:       any(nil),
		QueryDuration:    any(nil),
		Query:            "SELECT pg_sleep(10);",
	}

	invokeTcs := []struct {
		name                string
		requestBody         io.Reader
		clientSleepSecs     int
		waitSecsBeforeCheck int
		wantStatusCode      int
		want                any
	}{
		{
			name:                "invoke list_active_queries when the system is idle",
			requestBody:         bytes.NewBufferString(`{}`),
			clientSleepSecs:     0,
			waitSecsBeforeCheck: 0,
			wantStatusCode:      http.StatusOK,
			want:                []queryListDetails(nil),
		},
		{
			name:                "invoke list_active_queries when there is 1 ongoing but lower than the threshold",
			requestBody:         bytes.NewBufferString(`{"min_duration": "100 seconds"}`),
			clientSleepSecs:     1,
			waitSecsBeforeCheck: 1,
			wantStatusCode:      http.StatusOK,
			want:                []queryListDetails(nil),
		},
		{
			name:                "invoke list_active_queries when 1 ongoing query should show up",
			requestBody:         bytes.NewBufferString(`{"min_duration": "1 seconds"}`),
			clientSleepSecs:     10,
			waitSecsBeforeCheck: 5,
			wantStatusCode:      http.StatusOK,
			want:                []queryListDetails{singleQueryWanted},
		},
	}

	var wg sync.WaitGroup
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			if tc.clientSleepSecs > 0 {
				wg.Add(1)

				go func() {
					defer wg.Done()

					err := pool.Ping(ctx)
					if err != nil {
						t.Errorf("unable to connect to test database: %s", err)
						return
					}
					_, err = pool.Exec(ctx, fmt.Sprintf("SELECT pg_sleep(%d);", tc.clientSleepSecs))
					if err != nil {
						t.Errorf("Executing 'SELECT pg_sleep' failed: %s", err)
					}
				}()
			}

			if tc.waitSecsBeforeCheck > 0 {
				time.Sleep(time.Duration(tc.waitSecsBeforeCheck) * time.Second)
			}

			const api = "http://127.0.0.1:5000/api/tool/list_active_queries/invoke"
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

			var got any
			var details []queryListDetails
			if err := json.Unmarshal([]byte(resultString), &details); err != nil {
				t.Fatalf("failed to unmarshal nested ObjectDetails string: %v", err)
			}
			got = details

			if diff := cmp.Diff(tc.want, got, cmp.Comparer(func(a, b queryListDetails) bool {
				return a.Query == b.Query
			})); diff != "" {
				t.Errorf("Unexpected result: got %#v, want: %#v", got, tc.want)
			}
		})
	}
	wg.Wait()
}

func runPostgresListAvailableExtensionsTest(t *testing.T) {
	invokeTcs := []struct {
		name           string
		api            string
		requestBody    io.Reader
		wantStatusCode int
	}{
		{
			name:           "invoke list_available_extensions output",
			api:            "http://127.0.0.1:5000/api/tool/list_available_extensions/invoke",
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

			// Intentionally not adding the output check as output depends on the postgres instance used where the the functional test runs.
			// Adding the check will make the test flaky.
		})
	}
}

func runPostgresListInstalledExtensionsTest(t *testing.T) {
	invokeTcs := []struct {
		name           string
		api            string
		requestBody    io.Reader
		wantStatusCode int
	}{
		{
			name:           "invoke list_installed_extensions output",
			api:            "http://127.0.0.1:5000/api/tool/list_installed_extensions/invoke",
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

			// Intentionally not adding the output check as output depends on the postgres instance used where the the functional test runs.
			// Adding the check will make the test flaky.
		})
	}
}
