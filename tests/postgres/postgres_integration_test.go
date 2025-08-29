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
	"net/url"
	"net/http"
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
	PostgresListTablesToolKind = "postgres-list-tables"
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

func addToolConfig(t *testing.T, config map[string]any, toolName string, toolConfig map[string]any) map[string]any {
	tools, ok := config["tools"].(map[string]any)
	if !ok {
		t.Fatalf("unable to get tools from config")
	}
	tools[toolName] = toolConfig
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

	toolsFile = addToolConfig(t, toolsFile, "list_tables", map[string]any{
		"kind":        PostgresListTablesToolKind,
		"source":      "my-instance", 
		"description": "Lists tables in the database.",
	})

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
}

func runPostgresListTablesTest(t *testing.T, tableNameParam, tableNameAuth string) {
	invokeTcs := []struct {
		name          string
		api           string
		requestHeader map[string]string
		requestBody   io.Reader
		isErr         bool
		validation    func(*testing.T, []byte)
	}{
		{
			name:        "invoke list_tables detailed output",
			api:         "http://127.0.0.1:5000/api/tool/list_tables/invoke",
			requestBody: bytes.NewBuffer([]byte(`{"table_names": ""}`)),
			isErr:       false,
			validation: func(t *testing.T, body []byte) {
				var result []map[string]any
				if err := json.Unmarshal(body, &result); err != nil {
					t.Fatalf("failed to unmarshal response: %s", err)
				}
				if len(result) < 2 {
					t.Fatalf("expected at least 2 tables, got %d", len(result))
				}
				foundTables := map[string]bool{}
				for _, item := range result {
					details := item["object_details"].(map[string]any)
					foundTables[details["object_name"].(string)] = true
				}
				for _, expected := range []string{tableNameParam, tableNameAuth} {
					if !foundTables[expected] {
						t.Errorf("expected to find table %q, but it was missing", expected)
					}
				}
			},
		},
		{
			name:        "invoke list_tables simple output",
			api:         "http://127.0.0.1:5000/api/tool/list_tables/invoke",
			requestBody: bytes.NewBuffer([]byte(`{"table_names": "", "output_format": "simple"}`)),
			isErr:       false,
			validation: func(t *testing.T, body []byte) {
				var result []map[string]any
				if err := json.Unmarshal(body, &result); err != nil {
					t.Fatalf("failed to unmarshal response: %s", err)
				}
				details := result[0]["object_details"].(map[string]any)
				if _, ok := details["name"]; !ok {
					t.Error("expected 'name' field in simple output")
				}
				if _, ok := details["columns"]; ok {
					t.Error("did not expect 'columns' field in simple output")
				}
			},
		},
		{
			name:        "invoke list_tables with invalid output format",
			api:         "http://127.0.0.1:5000/api/tool/list_tables/invoke",
			requestBody: bytes.NewBuffer([]byte(`{"table_names": "", "output_format": "abcd"}`)),
			isErr:       true,
			validation: nil,
		},
		{
			name:        "invoke list_tables with missing table_names parameter",
			api:         "http://127.0.0.1:5000/api/tool/list_tables/invoke",
			requestBody: bytes.NewBuffer([]byte(`{}`)),
			isErr:       true,
			validation: nil,
		},
		{
			name:        "invoke list_tables with malformed table_names parameter",
			api:         "http://127.0.0.1:5000/api/tool/list_tables/invoke",
			requestBody: bytes.NewBuffer([]byte(`{"table_names": 12345, "output_format": "detailed"}`)),
			isErr:       true,
			validation: nil,
		},
		{
			name:        "invoke list_tables with multiple table names",
			api:         "http://127.0.0.1:5000/api/tool/list_tables/invoke",
			requestBody: bytes.NewBuffer([]byte(fmt.Sprintf(`{"table_names": "%s,%s"}`, tableNameParam, tableNameAuth))),
			isErr:       false,
			validation: func(t *testing.T, body []byte) {
				var result []map[string]any
				if err := json.Unmarshal(body, &result); err != nil {
					t.Fatalf("failed to unmarshal response: %s", err)
				}
				if len(result) != 2 {
					t.Fatalf("expected exactly 2 tables, got %d", len(result))
				}
			},
		},
		{
			name:        "invoke list_tables with non-existent table",
			api:         "http://127.0.0.1:5000/api/tool/list_tables/invoke",
			requestBody: bytes.NewBuffer([]byte(`{"table_names": "non_existent_table"}`)),
			isErr:       false,
			validation: func(t *testing.T, body []byte) {
				var result []map[string]any
				if err := json.Unmarshal(body, &result); err != nil {
					t.Fatalf("failed to unmarshal response: %s", err)
				}
				if len(result) != 0 {
					t.Fatalf("expected 0 tables for a non-existent table, got %d", len(result))
				}
			},
		},
		{
			name:        "invoke list_tables with one existing and one non-existent table",
			api:         "http://127.0.0.1:5000/api/tool/list_tables/invoke",
			requestBody: bytes.NewBuffer([]byte(fmt.Sprintf(`{"table_names": "%s,non_existent_table"}`, tableNameParam))),
			isErr:       false,
			validation: func(t *testing.T, body []byte) {
				var result []map[string]any
				if err := json.Unmarshal(body, &result); err != nil {
					t.Fatalf("failed to unmarshal response: %s", err)
				}
				if len(result) != 1 {
					t.Fatalf("expected 1 table, got %d", len(result))
				}
				details := result[0]["object_details"].(map[string]any)
				if details["object_name"] != tableNameParam {
					t.Errorf("expected table %q, got %q", tableNameParam, details["object_name"])
				}
			},
		},
		{
			name:        "invoke list_tables with a table name and simple output",
			api:         "http://127.0.0.1:5000/api/tool/list_tables/invoke",
			requestBody: bytes.NewBuffer([]byte(fmt.Sprintf(`{"table_names": "%s", "output_format": "simple"}`, tableNameAuth))),
			isErr:       false,
			validation: func(t *testing.T, body []byte) {
				var result []map[string]any
				if err := json.Unmarshal(body, &result); err != nil {
					t.Fatalf("failed to unmarshal response: %s", err)
				}
				if len(result) != 1 {
					t.Fatalf("expected 1 table, got %d", len(result))
				}
				details := result[0]["object_details"].(map[string]any)
				if details["name"] != tableNameAuth {
					t.Errorf("expected table name %q, got %q", tableNameAuth, details["name"])
				}
				if _, ok := details["columns"]; ok {
					t.Error("did not expect 'columns' field in simple output")
				}
			},
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost, tc.api, tc.requestBody)
			if err != nil {
				t.Fatalf("unable to create request: %s", err)
			}
			req.Header.Add("Content-type", "application/json")
			for k, v := range tc.requestHeader {
				req.Header.Add(k, v)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("unable to send request: %s", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				if tc.isErr {
					return
				}
				bodyBytes, _ := io.ReadAll(resp.Body)
				t.Fatalf("response status code is not 200, got %d: %s", resp.StatusCode, string(bodyBytes))
			}
			if tc.isErr {
				t.Fatal("expected an error, but got status 200")
			}

			var bodyWrapper map[string]json.RawMessage
			respBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("error reading response body: %s", err)
			}

			if err := json.Unmarshal(respBytes, &bodyWrapper); err != nil {
				t.Fatalf("error parsing response wrapper: %s", err)
			}

			resultJSON, ok := bodyWrapper["result"]
			if !ok {
				t.Fatal("unable to find result in response body")
			}

			var resultString string
			if err := json.Unmarshal(resultJSON, &resultString); err != nil {
				t.Fatalf("result is not a JSON-encoded string: %s", err)
			}

			if tc.validation != nil {
				tc.validation(t, []byte(resultString))
			}
		})
	}
}