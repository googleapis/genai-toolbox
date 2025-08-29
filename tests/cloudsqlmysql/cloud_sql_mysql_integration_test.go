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

package cloudsqlmysql

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/cloudsqlconn"
	"cloud.google.com/go/cloudsqlconn/mysql/mysql"
	"github.com/google/uuid"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"
)

var (
	CloudSQLMySQLSourceKind         = "cloud-sql-mysql"
	CloudSQLMySQLToolKind           = "mysql-sql"
	CloudSQLMySQLListTablesToolKind = "cloud-sql-mysql-list-tables"
	CloudSQLMySQLProject            = os.Getenv("CLOUD_SQL_MYSQL_PROJECT")
	CloudSQLMySQLRegion             = os.Getenv("CLOUD_SQL_MYSQL_REGION")
	CloudSQLMySQLInstance           = os.Getenv("CLOUD_SQL_MYSQL_INSTANCE")
	CloudSQLMySQLDatabase           = os.Getenv("CLOUD_SQL_MYSQL_DATABASE")
	CloudSQLMySQLUser               = os.Getenv("CLOUD_SQL_MYSQL_USER")
	CloudSQLMySQLPass               = os.Getenv("CLOUD_SQL_MYSQL_PASS")
)

func getCloudSQLMySQLVars(t *testing.T) map[string]any {
	switch "" {
	case CloudSQLMySQLProject:
		t.Fatal("'CLOUD_SQL_MYSQL_PROJECT' not set")
	case CloudSQLMySQLRegion:
		t.Fatal("'CLOUD_SQL_MYSQL_REGION' not set")
	case CloudSQLMySQLInstance:
		t.Fatal("'CLOUD_SQL_MYSQL_INSTANCE' not set")
	case CloudSQLMySQLDatabase:
		t.Fatal("'CLOUD_SQL_MYSQL_DATABASE' not set")
	case CloudSQLMySQLUser:
		t.Fatal("'CLOUD_SQL_MYSQL_USER' not set")
	case CloudSQLMySQLPass:
		t.Fatal("'CLOUD_SQL_MYSQL_PASS' not set")
	}

	return map[string]any{
		"kind":     CloudSQLMySQLSourceKind,
		"project":  CloudSQLMySQLProject,
		"instance": CloudSQLMySQLInstance,
		"region":   CloudSQLMySQLRegion,
		"database": CloudSQLMySQLDatabase,
		"user":     CloudSQLMySQLUser,
		"password": CloudSQLMySQLPass,
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

// Copied over from cloud_sql_mysql.go
func initCloudSQLMySQLConnectionPool(project, region, instance, ipType, user, pass, dbname string) (*sql.DB, error) {

	// Create a new dialer with options
	dialOpts, err := tests.GetCloudSQLDialOpts(ipType)
	if err != nil {
		return nil, err
	}

	if !slices.Contains(sql.Drivers(), "cloudsql-mysql") {
		_, err = mysql.RegisterDriver("cloudsql-mysql", cloudsqlconn.WithDefaultDialOptions(dialOpts...))
		if err != nil {
			return nil, fmt.Errorf("unable to register driver: %w", err)
		}
	}

	// Tell the driver to use the Cloud SQL Go Connector to create connections
	dsn := fmt.Sprintf("%s:%s@cloudsql-mysql(%s:%s:%s)/%s", user, pass, project, region, instance, dbname)
	db, err := sql.Open(
		"cloudsql-mysql",
		dsn,
	)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func TestCloudSQLMySQLToolEndpoints(t *testing.T) {
	sourceConfig := getCloudSQLMySQLVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	pool, err := initCloudSQLMySQLConnectionPool(CloudSQLMySQLProject, CloudSQLMySQLRegion, CloudSQLMySQLInstance, "public", CloudSQLMySQLUser, CloudSQLMySQLPass, CloudSQLMySQLDatabase)
	if err != nil {
		t.Fatalf("unable to create Cloud SQL connection pool: %s", err)
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
	toolsFile := tests.GetToolsConfig(sourceConfig, CloudSQLMySQLToolKind, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, authToolStmt)
	toolsFile = tests.AddMySqlExecuteSqlConfig(t, toolsFile)
	tmplSelectCombined, tmplSelectFilterCombined := tests.GetMySQLTmplToolStatement()
	toolsFile = tests.AddTemplateParamConfig(t, toolsFile, CloudSQLMySQLToolKind, tmplSelectCombined, tmplSelectFilterCombined, "")

	toolsFile = addToolConfig(t, toolsFile, "list_tables", map[string]any{
		"kind":        CloudSQLMySQLListTablesToolKind,
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
	select1Want, mcpMyFailToolWant, createTableStatement, mcpSelect1Want := tests.GetMySQLWants()

	// Run tests
	tests.RunToolGetTest(t)
	tests.RunToolInvokeTest(t, select1Want, tests.DisableArrayTest())
	tests.RunMCPToolCallMethod(t, mcpMyFailToolWant, mcpSelect1Want)
	tests.RunExecuteSqlToolInvokeTest(t, createTableStatement, select1Want)
	tests.RunToolInvokeWithTemplateParameters(t, tableNameTemplateParam)

	// Run specific CloudSQLMySQL tool tests
	runCloudSQLMySQLListTablesTest(t, tableNameParam, tableNameAuth)
}

func runCloudSQLMySQLListTablesTest(t *testing.T, tableNameParam, tableNameAuth string) {
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
				foundTables := make(map[string]bool)
				for _, item := range result {
					// The result is a map with a single key(object_details) containing the JSON string
					detailsStr, ok := item["object_details"].(string)
					if !ok {
						t.Fatalf("expected 'object_details' to be a string, but it's not")
					}
					
					var details map[string]any
					if err := json.Unmarshal([]byte(detailsStr), &details); err != nil {
						t.Fatalf("failed to unmarshal object_details JSON string: %s", err)
					}
					
					tableName, ok := details["object_name"].(string)
					if !ok {
						t.Fatalf("table name not found or not a string in object_details")
					}
					foundTables[tableName] = true

					// Check for a field that only exists in detailed output
					if _, ok := details["columns"]; !ok {
						t.Errorf("expected 'columns' field in detailed output for table %s, but it was missing", tableName)
					}
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
				detailsStr, ok := result[0]["object_details"].(string)
				if !ok {
					t.Fatal("expected 'object_details' to be a string")
				}

				var details map[string]any
				if err := json.Unmarshal([]byte(detailsStr), &details); err != nil {
					t.Fatalf("failed to unmarshal object_details JSON string: %s", err)
				}
				if _, ok := details["name"]; !ok {
					t.Error("expected 'name' field in simple output")
				}
				if _, ok := details["columns"]; ok {
					t.Error("did not expect 'columns' field in simple output")
				}
			},
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
				detailsStr := result[0]["object_details"].(string)
				var details map[string]any

				if err := json.Unmarshal([]byte(detailsStr), &details); err != nil {
					t.Fatalf("failed to unmarshal object_details JSON string: %s", err)
				}

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
				detailsStr := result[0]["object_details"].(string)
				var details map[string]any

				if err := json.Unmarshal([]byte(detailsStr), &details); err != nil {
					t.Fatalf("failed to unmarshal object_details JSON string: %s", err)
				}

				if details["name"] != tableNameAuth {
					t.Errorf("expected table name %q, got %q", tableNameAuth, details["name"])
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

// Test connection with different IP type
func TestCloudSQLMySQLIpConnection(t *testing.T) {
	sourceConfig := getCloudSQLMySQLVars(t)

	tcs := []struct {
		name   string
		ipType string
	}{
		{
			name:   "public ip",
			ipType: "public",
		},
		{
			name:   "private ip",
			ipType: "private",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			sourceConfig["ipType"] = tc.ipType
			err := tests.RunSourceConnectionTest(t, sourceConfig, CloudSQLMySQLToolKind)
			if err != nil {
				t.Fatalf("Connection test failure: %s", err)
			}
		})
	}
}
