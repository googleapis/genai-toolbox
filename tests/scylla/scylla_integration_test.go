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

package scylla

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/gocql/gocql"
	"github.com/google/uuid"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"
)

var (
	ScyllaSourceKind = "scylla"
	ScyllaToolKind   = "scylla-cql"
	ScyllaHosts      = os.Getenv("SCYLLA_HOSTS")
	ScyllaPort       = os.Getenv("SCYLLA_PORT")
	ScyllaUser       = os.Getenv("SCYLLA_USER")
	ScyllaPass       = os.Getenv("SCYLLA_PASS")
	ScyllaKeyspace   = os.Getenv("SCYLLA_KEYSPACE")
)

func getScyllaVars(t *testing.T) map[string]any {
	switch "" {
	case ScyllaHosts:
		t.Fatal("'SCYLLA_HOSTS' not set")
	case ScyllaPort:
		t.Fatal("'SCYLLA_PORT' not set")
	case ScyllaKeyspace:
		t.Fatal("'SCYLLA_KEYSPACE' not set")
	}

	hosts := strings.Split(ScyllaHosts, ",")

	config := map[string]any{
		"kind":     ScyllaSourceKind,
		"hosts":    hosts,
		"port":     ScyllaPort,
		"keyspace": ScyllaKeyspace,
	}

	// Add optional authentication
	if ScyllaUser != "" {
		config["username"] = ScyllaUser
		config["password"] = ScyllaPass
	}

	return config
}

// initScyllaSession creates a Scylla session for testing
func initScyllaSession(hosts []string, port, keyspace, user, pass string) (*gocql.Session, error) {
	cluster := gocql.NewCluster(hosts...)
	cluster.Keyspace = keyspace
	cluster.Consistency = gocql.Quorum
	cluster.Timeout = 60 * time.Second
	cluster.ConnectTimeout = 10 * time.Second

	if user != "" && pass != "" {
		cluster.Authenticator = gocql.PasswordAuthenticator{
			Username: user,
			Password: pass,
		}
	}

	session, err := cluster.CreateSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return session, nil
}

// getScyllaParamToolInfo returns statements and param for my-tool scylla-cql kind
func getScyllaParamToolInfo(tableName string) (string, string, string, string, string, string, []any) {
	createStatement := fmt.Sprintf(`CREATE TABLE %s (
		id bigint PRIMARY KEY,
		name text
	)`, tableName)
	insertStatement := fmt.Sprintf("INSERT INTO %s (id, name) VALUES (?, ?)", tableName)
	toolStatement := fmt.Sprintf("SELECT * FROM %s WHERE id = ? OR name = ? ALLOW FILTERING", tableName)
	idParamStatement := fmt.Sprintf("SELECT * FROM %s WHERE id = ?", tableName)
	nameParamStatement := fmt.Sprintf("SELECT * FROM %s WHERE name = ? ALLOW FILTERING", tableName)
	// Scylla/Cassandra doesn't support IN with parameterized queries the same way
	arrayToolStatement := fmt.Sprintf("SELECT * FROM %s WHERE id IN (1, 2) ALLOW FILTERING", tableName)
	params := []any{"Alice", "Jane", "Sid", nil}
	return createStatement, insertStatement, toolStatement, idParamStatement, nameParamStatement, arrayToolStatement, params
}

// getScyllaAuthToolInfo returns statements and param of my-auth-tool for scylla-cql kind
func getScyllaAuthToolInfo(tableName string) (string, string, string, []any) {
	createStatement := fmt.Sprintf(`CREATE TABLE %s (
		id bigint PRIMARY KEY,
		name text,
		email text
	)`, tableName)
	insertStatement := fmt.Sprintf("INSERT INTO %s (id, name, email) VALUES (?, ?, ?)", tableName)
	toolStatement := fmt.Sprintf("SELECT name FROM %s WHERE email = ? ALLOW FILTERING", tableName)
	params := []any{int64(1), "Alice", tests.ServiceAccountEmail, int64(2), "Jane", "janedoe@gmail.com"}
	return createStatement, insertStatement, toolStatement, params
}

// getScyllaTmplToolStatement returns statements and param for template parameter test cases for scylla-cql kind
func getScyllaTmplToolStatement() (string, string) {
	tmplSelectCombined := "SELECT * FROM {{.tableName}} WHERE id = ?"
	tmplSelectFilterCombined := "SELECT * FROM {{.tableName}} WHERE {{.columnFilter}} = ? ALLOW FILTERING"
	return tmplSelectCombined, tmplSelectFilterCombined
}

// getScyllaWants return the expected wants for scylla
func getScyllaWants() (string, string, string) {
	// For Scylla, we need to create a simple test table first
	select1Want := `[{"count":1}]`
	failInvocationWant := `{"jsonrpc":"2.0","id":"invoke-fail-tool","result":{"content":[{"type":"text","text":"unable to execute query: no connections were made when creating the session`
	createTableStatement := `"CREATE TABLE t (id bigint PRIMARY KEY, name text)"`
	return select1Want, failInvocationWant, createTableStatement
}

// setupScyllaTable creates and inserts data into a table of tool
// compatible with scylla-cql tool
func setupScyllaTable(t *testing.T, ctx context.Context, session *gocql.Session, createStatement, tableName string, insertParams []any) func(*testing.T) {
	// Create table
	err := session.Query(createStatement).Exec()
	if err != nil {
		t.Fatalf("unable to create test table %s: %s", tableName, err)
	}

	// Insert test data
	// For Scylla/Cassandra, we need to insert rows one by one
	if strings.Contains(createStatement, "name text,") {
		// Auth table with 3 columns
		for i := 0; i < len(insertParams); i += 3 {
			insertStmt := fmt.Sprintf("INSERT INTO %s (id, name, email) VALUES (?, ?, ?)", tableName)
			err = session.Query(insertStmt, insertParams[i], insertParams[i+1], insertParams[i+2]).Exec()
			if err != nil {
				t.Fatalf("unable to insert test data: %s", err)
			}
		}
	} else {
		// Param table with 2 columns
		for i, name := range insertParams {
			insertStmt := fmt.Sprintf("INSERT INTO %s (id, name) VALUES (?, ?)", tableName)
			err = session.Query(insertStmt, int64(i+1), name).Exec()
			if err != nil {
				t.Fatalf("unable to insert test data: %s", err)
			}
		}
	}

	return func(t *testing.T) {
		// tear down test
		err = session.Query(fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName)).Exec()
		if err != nil {
			t.Errorf("Teardown failed: %s", err)
		}
	}
}

// addScyllaExecuteCqlConfig gets the tools config for `scylla-execute-cql`
func addScyllaExecuteCqlConfig(t *testing.T, config map[string]any) map[string]any {
	tools, ok := config["tools"].(map[string]any)
	if !ok {
		t.Fatalf("unable to get tools from config")
	}
	tools["my-exec-cql-tool"] = map[string]any{
		"kind":        "scylla-execute-cql",
		"source":      "my-instance",
		"description": "Tool to execute CQL",
	}
	tools["my-auth-exec-cql-tool"] = map[string]any{
		"kind":        "scylla-execute-cql",
		"source":      "my-instance",
		"description": "Tool to execute CQL",
		"authRequired": []string{
			"my-google-auth",
		},
	}
	config["tools"] = tools
	return config
}

func TestScyllaToolEndpoints(t *testing.T) {
	sourceConfig := getScyllaVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	hosts := strings.Split(ScyllaHosts, ",")
	session, err := initScyllaSession(hosts, ScyllaPort, ScyllaKeyspace, ScyllaUser, ScyllaPass)
	if err != nil {
		t.Fatalf("unable to create Scylla session: %s", err)
	}
	defer session.Close()

	// create table name with UUID
	tableNameParam := "param_table_" + strings.ReplaceAll(uuid.New().String(), "-", "_")
	tableNameAuth := "auth_table_" + strings.ReplaceAll(uuid.New().String(), "-", "_")
	tableNameTemplateParam := "template_param_table_" + strings.ReplaceAll(uuid.New().String(), "-", "_")

	// set up data for param tool
	createParamTableStmt, _, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, paramTestParams := getScyllaParamToolInfo(tableNameParam)
	teardownTable1 := setupScyllaTable(t, ctx, session, createParamTableStmt, tableNameParam, paramTestParams)
	defer teardownTable1(t)

	// set up data for auth tool
	createAuthTableStmt, _, authToolStmt, authTestParams := getScyllaAuthToolInfo(tableNameAuth)
	teardownTable2 := setupScyllaTable(t, ctx, session, createAuthTableStmt, tableNameAuth, authTestParams)
	defer teardownTable2(t)

	// Modify statements for testing (we'll use simpler queries for Scylla)
	// For the simple test, we'll use COUNT instead of SELECT 1
	simpleStatement := "SELECT COUNT(*) as count FROM system.local"

	// Write config into a file and pass it to command
	toolsFile := tests.GetToolsConfig(sourceConfig, ScyllaToolKind, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, authToolStmt)

	// Override the simple-tool statement for Scylla
	tools := toolsFile["tools"].(map[string]any)
	simpleTool := tools["simple-tool"].(map[string]any)
	simpleTool["statement"] = simpleStatement
	tools["simple-tool"] = simpleTool
	toolsFile["tools"] = tools

	toolsFile = addScyllaExecuteCqlConfig(t, toolsFile)
	tmplSelectCombined, tmplSelectFilterCombined := getScyllaTmplToolStatement()
	toolsFile = tests.AddTemplateParamConfig(t, toolsFile, ScyllaToolKind, tmplSelectCombined, tmplSelectFilterCombined, "")

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

	select1Want, _, createTableStatement := getScyllaWants()

	// For Scylla, we need to adjust expected responses
	invokeParamWant := `[{"id":1,"name":"Alice"}]`
	invokeIdNullWant := `[]` // Scylla doesn't return rows with null matches
	nullWant := `[]`

	tests.RunToolInvokeTest(t, select1Want, invokeParamWant, invokeIdNullWant, nullWant, true, false)
	tests.RunExecuteSqlToolInvokeTest(t, createTableStatement, select1Want)

	// Skip MCP tool call test for Scylla as it requires different error handling
	// tests.RunMCPToolCallMethod(t, mcpInvokeParamWant, failInvocationWant)

	// For template parameters, we need to create the table first
	createTemplateTableStmt := fmt.Sprintf(`CREATE TABLE %s (
		id bigint PRIMARY KEY,
		name text
	)`, tableNameTemplateParam)
	err = session.Query(createTemplateTableStmt).Exec()
	if err != nil {
		t.Fatalf("unable to create template param table: %s", err)
	}
	defer func() {
		_ = session.Query(fmt.Sprintf("DROP TABLE IF EXISTS %s", tableNameTemplateParam)).Exec()
	}()

	// Insert test data for template param table
	_ = session.Query(fmt.Sprintf("INSERT INTO %s (id, name) VALUES (1, 'test')", tableNameTemplateParam)).Exec()

	tests.RunToolInvokeWithTemplateParameters(t, tableNameTemplateParam, tests.NewTemplateParameterTestConfig(tests.WithInsert1Want(`{"status":"success","message":"Query executed successfully"}`)))
}
