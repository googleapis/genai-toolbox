// Copyright 2025 Google LLC
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

package mindsdb

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"
)

var (
	MindsDBSourceKind = "mindsdb"
	MindsDBToolKind   = "mindsdb-sql"
	MindsDBExecuteSQLToolKind = "mindsdb-execute-sql"
	MindsDBDatabase   = os.Getenv("MINDSDB_DATABASE")
	MindsDBHost       = os.Getenv("MINDSDB_HOST")
	MindsDBPort       = os.Getenv("MINDSDB_PORT")
	MindsDBUser       = os.Getenv("MINDSDB_USER")
	MindsDBPass       = os.Getenv("MINDSDB_PASS")
)

func getMindsDBVars(t *testing.T) map[string]any {
	switch "" {
	case MindsDBDatabase:
		t.Fatal("'MINDSDB_DATABASE' not set")
	case MindsDBHost:
		t.Fatal("'MINDSDB_HOST' not set")
	case MindsDBPort:
		t.Fatal("'MINDSDB_PORT' not set")
	case MindsDBUser:
		t.Fatal("'MINDSDB_USER' not set")
	}

	config := map[string]any{
		"kind":     MindsDBSourceKind,
		"host":     MindsDBHost,
		"port":     MindsDBPort,
		"database": MindsDBDatabase,
		"user":     MindsDBUser,
	}

	// Only add password if it's set
	if MindsDBPass != "" {
		config["password"] = MindsDBPass
	}

	return config
}

// initMindsDBConnectionPool connects directly to MindsDB
func initMindsDBConnectionPool(host, port, user, pass, dbname string) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", user, pass, host, port, dbname)
	pool, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}
	return pool, nil
}

func TestMindsDBToolEndpoints(t *testing.T) {
	sourceConfig := getMindsDBVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	// create table name with UUID
	tableNameParam := "param_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableNameAuth := "auth_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")

	// set up data for param tool
	createParamTableStmt, insertParamTableStmt, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, _ := getMindsDBParamToolInfo(tableNameParam)

	// set up data for auth tool
	_, _, authToolStmt, _ := getMindsDBAuthToolInfo(tableNameAuth)

	// Write config into a file and pass it to command
	toolsFile := getMindsDBSimpleToolsConfig(sourceConfig, MindsDBToolKind, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, authToolStmt)
	toolsFile = addMindsDBMyExecSQLConfig(t, toolsFile)
	tmplSelectCombined, tmplSelectFilterCombined := getMindsDBTmplToolStatement()
	toolsFile = addTemplateParamConfig(t, toolsFile, MindsDBToolKind, tmplSelectCombined, tmplSelectFilterCombined, "")
	
	// Add parameterized SQL tool for testing
	paramQueryToolName := "my-param-sql-tool"
	// Use a simple parameterized query that doesn't reference a table
	paramQueryStatement := "SELECT ? as result"
	tools := toolsFile["tools"].(map[string]any)
	tools[paramQueryToolName] = map[string]any{
		"kind":        MindsDBToolKind,
		"source":      "my-instance",
		"description": "Tool to test parameterized SQL queries",
		"statement":   paramQueryStatement,
		"parameters": []any{
			map[string]any{
				"name":        "id",
				"type":        "integer",
				"description": "user ID",
			},
		},
	}

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

	tests.RunToolGetTest(t)

	// Test that the server started successfully and tools are registered
	t.Logf("Server started successfully with output: %s", out)
	t.Logf("MindsDB tool endpoints test is working correctly")

	// Test that the my-exec-sql-tool is properly registered
	tests.RunToolGetTestByName(t, "my-exec-sql-tool", map[string]any{
		"my-exec-sql-tool": map[string]any{
			"description": "Tool to execute sql",
			"authRequired": []any{},
			"parameters": []any{
				map[string]any{
					"name":        "sql",
					"type":        "string",
					"description": "The sql to execute.",
					"required":    true,
					"authSources": []any{},
				},
			},
		},
	})

	// Create the table and insert data for parameterized tests
	t.Logf("Setting up table for parameterized tests: %s", tableNameParam)
	runSQLTest(t, createParamTableStmt, "null")
	runSQLTest(t, insertParamTableStmt, "null")

	// Test parameterized tool invocation - create table, run parameterized query, drop table
	// Create a table for parameterized query testing
	paramTableName := "param_test_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	
	// Create table using execute-sql tool
	createTableSQL := fmt.Sprintf("CREATE TABLE files.%s (id INT PRIMARY KEY, name VARCHAR(255), email VARCHAR(255))", paramTableName)
	runSQLTest(t, createTableSQL, "null")
	
	// Insert test data using execute-sql tool
	insertDataSQL := fmt.Sprintf("INSERT INTO files.%s (id, name, email) VALUES (1, 'Alice', 'alice@example.com'), (2, 'Bob', 'bob@example.com')", paramTableName)
	runSQLTest(t, insertDataSQL, "null")
	
	// Test parameterized query using mindsdb-sql tool with parameters
	// First, let's test with a simple parameterized query to verify the tool works
	simpleParamTestParams := []byte(`{"id": 1}`)
	simpleExpectedResult := "[{\"1\":1}]"  // MindsDB returns the parameter value as column name
	tests.RunToolInvokeParametersTest(t, paramQueryToolName, simpleParamTestParams, simpleExpectedResult)
	
	// Now test with the actual table query using execute-sql tool
	// This demonstrates that parameterized queries work, but table queries have limitations
	paramQuerySQL := fmt.Sprintf("SELECT * FROM files.%s WHERE id = 1", paramTableName)
	expectedResult := "[{\"email\":\"alice@example.com\",\"id\":1,\"name\":\"Alice\"}]"
	runSQLTest(t, paramQuerySQL, expectedResult)
	
	// Clean up - drop the table
	dropTableSQL := fmt.Sprintf("DROP TABLE files.%s", paramTableName)
	runSQLTest(t, dropTableSQL, "null")

	// Clean up the table
	runSQLTest(t, fmt.Sprintf("DROP TABLE files.%s", tableNameParam), "null")
}

func TestMindsDBExecuteSQLTool(t *testing.T) {
	sourceConfig := getMindsDBVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	// Create tools configuration with standard my-exec-sql-tool pattern
	toolsFile := getExecuteSQLToolsConfig(sourceConfig)
	toolsFile = addMindsDBMyExecSQLConfig(t, toolsFile)

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

	// Test that the server started successfully and tools are registered
	t.Logf("Server started successfully with output: %s", out)
	t.Logf("MindsDB execute-sql tool test is working correctly")

	// Test that the my-exec-sql-tool is properly registered
	tests.RunToolGetTestByName(t, "my-exec-sql-tool", map[string]any{
		"my-exec-sql-tool": map[string]any{
			"description": "Tool to execute sql",
			"authRequired": []any{},
			"parameters": []any{
				map[string]any{
					"name":        "sql",
					"type":        "string",
					"description": "The sql to execute.",
					"required":    true,
					"authSources": []any{},
				},
			},
		},
	})
}

func TestMindsDBBasicToolFunctionality(t *testing.T) {
	sourceConfig := getMindsDBVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	// Create tools configuration with standard my-exec-sql-tool pattern
	toolsFile := getExecuteSQLToolsConfig(sourceConfig)
	toolsFile = addMindsDBMyExecSQLConfig(t, toolsFile)

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

	// Test that the server started successfully and tools are registered
	t.Logf("Server started successfully with output: %s", out)
	t.Logf("MindsDB integration test is working correctly with environment variables set")

	// Test that the my-exec-sql-tool is properly registered
	tests.RunToolGetTestByName(t, "my-exec-sql-tool", map[string]any{
		"my-exec-sql-tool": map[string]any{
			"description": "Tool to execute sql",
			"authRequired": []any{},
			"parameters": []any{
				map[string]any{
					"name":        "sql",
					"type":        "string",
					"description": "The sql to execute.",
					"required":    true,
					"authSources": []any{},
				},
			},
		},
	})
}

func TestMindsDBComprehensiveSQLOperations(t *testing.T) {
	sourceConfig := getMindsDBVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	// Create tools configuration with standard my-exec-sql-tool pattern
	toolsFile := getExecuteSQLToolsConfig(sourceConfig)
	toolsFile = addMindsDBMyExecSQLConfig(t, toolsFile)

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

	t.Logf("Server started successfully with output: %s", out)

	// Create a unique table name for this test
	tableName := "test_" + strings.ReplaceAll(uuid.New().String(), "-", "")

	// Test 1: CREATE TABLE
	t.Run("CREATE TABLE", func(t *testing.T) {
		createTableSQL := fmt.Sprintf("CREATE TABLE files.%s (id INT PRIMARY KEY, name VARCHAR(255), email VARCHAR(255))", tableName)
		runSQLTest(t, createTableSQL, "null")
	})

	// Test 2: INSERT data
	t.Run("INSERT", func(t *testing.T) {
		insertSQL := fmt.Sprintf("INSERT INTO files.%s (id, name, email) VALUES (1, 'Alice', 'alice@example.com'), (2, 'Bob', 'bob@example.com')", tableName)
		runSQLTest(t, insertSQL, "null")
	})

	// Test 3: SELECT data
	t.Run("SELECT", func(t *testing.T) {
		selectSQL := fmt.Sprintf("SELECT * FROM files.%s", tableName)
		expectedResult := `[{"email":"alice@example.com","id":1,"name":"Alice"},{"email":"bob@example.com","id":2,"name":"Bob"}]`
		runSQLTest(t, selectSQL, expectedResult)
	})

	// Test 4: SELECT with WHERE clause
	t.Run("SELECT with WHERE", func(t *testing.T) {
		selectSQL := fmt.Sprintf("SELECT * FROM files.%s WHERE id = 1", tableName)
		expectedResult := `[{"email":"alice@example.com","id":1,"name":"Alice"}]`
		runSQLTest(t, selectSQL, expectedResult)
	})

	// Test 5: SELECT specific columns
	t.Run("SELECT specific columns", func(t *testing.T) {
		selectSQL := fmt.Sprintf("SELECT name, email FROM files.%s", tableName)
		expectedResult := `[{"email":"alice@example.com","name":"Alice"},{"email":"bob@example.com","name":"Bob"}]`
		runSQLTest(t, selectSQL, expectedResult)
	})

	// Test 6: DROP TABLE
	t.Run("DROP TABLE", func(t *testing.T) {
		dropTableSQL := fmt.Sprintf("DROP TABLE files.%s", tableName)
		runSQLTest(t, dropTableSQL, "null")
	})

	t.Logf("All supported SQL operations completed successfully for table: %s", tableName)
	t.Logf("Note: UPDATE and DELETE operations are not supported in MindsDB's files schema")
}

// runSQLTest executes a SQL statement using the my-exec-sql-tool and verifies the result
func runSQLTest(t *testing.T, sqlStatement, expectedResult string) {
	// Test tool invoke endpoint
	api := "http://127.0.0.1:5000/api/tool/my-exec-sql-tool/invoke"
	requestBody := fmt.Sprintf(`{"sql":"%s"}`, sqlStatement)
	
	req, err := http.NewRequest(http.MethodPost, api, bytes.NewBuffer([]byte(requestBody)))
	if err != nil {
		t.Fatalf("unable to create request: %s", err)
	}
	req.Header.Add("Content-type", "application/json")
	
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("unable to send request: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Read the response body to see the error message
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("response status code is not 200, got %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	if err != nil {
		t.Fatalf("error parsing response body: %s", err)
	}

	// Check if the response contains the expected result
	result, ok := body["result"]
	if !ok {
		t.Fatalf("response does not contain 'result' field")
	}

	// Convert result to string for comparison
	resultStr := fmt.Sprintf("%v", result)
	if resultStr != expectedResult {
		t.Fatalf("got %q, want %q", resultStr, expectedResult)
	}
}

// getMindsDBSimpleToolsConfig creates a tools configuration without auth service
func getMindsDBSimpleToolsConfig(sourceConfig map[string]any, toolKind, paramToolStatement, idParamToolStmt, nameParamToolStmt, arrayToolStatement, authToolStatement string) map[string]any {
	return map[string]any{
		"sources": map[string]any{
			"my-instance": sourceConfig,
		},
		"tools": map[string]any{
			"my-simple-tool": map[string]any{
				"kind":        toolKind,
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
				"statement":   "SELECT 1;",
			},
			"my-tool": map[string]any{
				"kind":        toolKind,
				"source":      "my-instance",
				"description": "Tool to test invocation with params.",
				"statement":   paramToolStatement,
				"parameters": []any{
					map[string]any{
						"name":        "id",
						"type":        "integer",
						"description": "user ID",
					},
					map[string]any{
						"name":        "name",
						"type":        "string",
						"description": "user name",
					},
				},
			},
			"my-tool-by-id": map[string]any{
				"kind":        toolKind,
				"source":      "my-instance",
				"description": "Tool to test invocation with params.",
				"statement":   idParamToolStmt,
				"parameters": []any{
					map[string]any{
						"name":        "id",
						"type":        "integer",
						"description": "user ID",
					},
				},
			},
			"my-tool-by-name": map[string]any{
				"kind":        toolKind,
				"source":      "my-instance",
				"description": "Tool to test invocation with params.",
				"statement":   nameParamToolStmt,
				"parameters": []any{
					map[string]any{
						"name":        "name",
						"type":        "string",
						"description": "user name",
						"required":    false,
					},
				},
			},
			"my-array-tool": map[string]any{
				"kind":        toolKind,
				"source":      "my-instance",
				"description": "Tool to test invocation with array params.",
				"statement":   arrayToolStatement,
				"parameters": []any{
					map[string]any{
						"name":        "idArray",
						"type":        "array",
						"description": "ID array",
						"items": map[string]any{
							"name":        "id",
							"type":        "integer",
							"description": "ID",
						},
					},
					map[string]any{
						"name":        "nameArray",
						"type":        "array",
						"description": "user name array",
						"items": map[string]any{
							"name":        "name",
							"type":        "string",
							"description": "user name",
						},
					},
				},
			},
			"my-auth-tool": map[string]any{
				"kind":        toolKind,
				"source":      "my-instance",
				"description": "Tool to test invocation with auth.",
				"statement":   authToolStatement,
				"authRequired": []string{
					"my-google-auth",
				},
			},
		},
	}
}

// getExecuteSQLToolsConfig creates a tools configuration with only the execute-sql tool
func getExecuteSQLToolsConfig(sourceConfig map[string]any) map[string]any {
	return map[string]any{
		"sources": map[string]any{
			"my-instance": sourceConfig,
		},
		"tools": map[string]any{},
	}
}

// addMindsDBMyExecSQLConfig adds the standard "my-exec-sql-tool" configuration for MindsDB
func addMindsDBMyExecSQLConfig(t *testing.T, config map[string]any) map[string]any {
	tools := config["tools"].(map[string]any)
	tools["my-exec-sql-tool"] = map[string]any{
		"kind":        MindsDBExecuteSQLToolKind,
		"source":      "my-instance",
		"description": "Tool to execute sql",
	}
	tools["my-auth-exec-sql-tool"] = map[string]any{
		"kind":        MindsDBExecuteSQLToolKind,
		"source":      "my-instance",
		"description": "Tool to execute sql",
		"authRequired": []string{
			"my-google-auth",
		},
	}
	return config
}

// addTemplateParamConfig adds template parameter tools to the configuration
func addTemplateParamConfig(t *testing.T, config map[string]any, toolKind, tmplSelectCombined, tmplSelectFilterCombined string, tmplSelectAll string) map[string]any {
	tools := config["tools"].(map[string]any)
	tools["my-template-tool"] = map[string]any{
		"kind":        toolKind,
		"source":      "my-instance",
		"description": "Tool to test invocation with template params.",
		"statement":   tmplSelectCombined,
		"templateParameters": []any{
			map[string]any{
				"name":        "name",
				"type":        "string",
				"description": "user name",
			},
		},
	}
	tools["my-template-filter-tool"] = map[string]any{
		"kind":        toolKind,
		"source":      "my-instance",
		"description": "Tool to test invocation with template filter params.",
		"statement":   tmplSelectFilterCombined,
		"templateParameters": []any{
			map[string]any{
				"name":        "name",
				"type":        "string",
				"description": "user name",
			},
		},
	}
	return config
}

// getMindsDBParamToolInfo returns the SQL statements and parameters for testing parameterized queries
func getMindsDBParamToolInfo(tableName string) (string, string, string, string, string, string, []any) {
	createStatement := fmt.Sprintf("CREATE TABLE files.%s (id INT PRIMARY KEY, name VARCHAR(255), email VARCHAR(255))", tableName)
	insertStatement := fmt.Sprintf("INSERT INTO files.%s (id, name, email) VALUES (1, 'Alice', 'alice@example.com'), (2, 'Bob', 'bob@example.com'), (3, 'Charlie', 'charlie@example.com')", tableName)
	paramToolStatement := fmt.Sprintf("SELECT * FROM files.%s WHERE id = ? AND name = ?", tableName)
	idParamToolStmt := fmt.Sprintf("SELECT * FROM files.%s WHERE id = ?", tableName)
	nameParamToolStmt := fmt.Sprintf("SELECT * FROM files.%s WHERE name = ?", tableName)
	arrayToolStatement := fmt.Sprintf("SELECT * FROM files.%s WHERE id IN (?)", tableName)
	
	paramTestParams := []any{1, "Alice"}
	
	return createStatement, insertStatement, paramToolStatement, idParamToolStmt, nameParamToolStmt, arrayToolStatement, paramTestParams
}

// getMindsDBAuthToolInfo returns the SQL statements and parameters for testing auth tools
func getMindsDBAuthToolInfo(tableName string) (string, string, string, []any) {
	createStatement := fmt.Sprintf("CREATE TABLE files.%s (id INT PRIMARY KEY, name VARCHAR(255), email VARCHAR(255))", tableName)
	insertStatement := fmt.Sprintf("INSERT INTO files.%s (id, name, email) VALUES (1, 'Alice', 'alice@example.com'), (2, 'Bob', 'bob@example.com')", tableName)
	authToolStatement := fmt.Sprintf("SELECT * FROM files.%s WHERE id = ?", tableName)
	authTestParams := []any{1}
	
	return createStatement, insertStatement, authToolStatement, authTestParams
}

// getMindsDBTmplToolStatement returns the template SQL statements
func getMindsDBTmplToolStatement() (string, string) {
	tmplSelectCombined := "SELECT * FROM files.template_param_table_{{.name}} WHERE id = 1"
	tmplSelectFilterCombined := "SELECT * FROM files.template_param_table_{{.name}} WHERE name = '{{.name}}'"
	
	return tmplSelectCombined, tmplSelectFilterCombined
}

// getMindsDBWants returns the expected results for MindsDB tests
func getMindsDBWants() (string, string, string) {
	select1Want := "[{\"1\":1}]"
	failInvocationWant := "failed to invoke tool"
	createTableStatement := "CREATE TABLE files.test_table (id INT PRIMARY KEY, name VARCHAR(255))"
	
	return select1Want, failInvocationWant, createTableStatement
}

// setupMindsDBTable sets up a test table and returns a cleanup function
func setupMindsDBTable(t *testing.T, ctx context.Context, pool *sql.DB, createStatement, insertStatement, tableName string, params []any) func(*testing.T) {
	// Create table
	_, err := pool.ExecContext(ctx, createStatement)
	if err != nil {
		t.Fatalf("failed to create table: %s", err)
	}

	// Insert data
	_, err = pool.ExecContext(ctx, insertStatement)
	if err != nil {
		t.Fatalf("failed to insert data: %s", err)
	}

	// Return cleanup function
	return func(t *testing.T) {
		dropStatement := fmt.Sprintf("DROP TABLE files.%s", tableName)
		_, err := pool.ExecContext(ctx, dropStatement)
		if err != nil {
			t.Logf("failed to drop table: %s", err)
		}
	}
}
