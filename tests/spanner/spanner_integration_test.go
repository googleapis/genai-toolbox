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

package spanner

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	database "cloud.google.com/go/spanner/admin/database/apiv1"
	"cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	"github.com/google/uuid"
	"github.com/googleapis/genai-toolbox/tests"
)

var (
	SPANNER_SOURCE_KIND = "spanner"
	SPANNER_TOOL_KIND   = "spanner-sql"
	SPANNER_PROJECT     = os.Getenv("SPANNER_PROJECT")
	SPANNER_DATABASE    = os.Getenv("SPANNER_DATABASE")
	SPANNER_INSTANCE    = os.Getenv("SPANNER_INSTANCE")
)

func getSpannerVars(t *testing.T) map[string]any {
	switch "" {
	case SPANNER_PROJECT:
		t.Fatal("'SPANNER_PROJECT' not set")
	case SPANNER_DATABASE:
		t.Fatal("'SPANNER_DATABASE' not set")
	case SPANNER_INSTANCE:
		t.Fatal("'SPANNER_INSTANCE' not set")
	}

	return map[string]any{
		"kind":     SPANNER_SOURCE_KIND,
		"project":  SPANNER_PROJECT,
		"instance": SPANNER_INSTANCE,
		"database": SPANNER_DATABASE,
	}
}

func initSpannerClients(ctx context.Context, project, instance, dbname string) (*spanner.Client, *database.DatabaseAdminClient, error) {
	// Configure the connection to the database
	db := fmt.Sprintf("projects/%s/instances/%s/databases/%s", project, instance, dbname)

	// Configure session pool to automatically clean inactive transactions
	sessionPoolConfig := spanner.SessionPoolConfig{
		TrackSessionHandles: true,
		InactiveTransactionRemovalOptions: spanner.InactiveTransactionRemovalOptions{
			ActionOnInactiveTransaction: spanner.WarnAndClose,
		},
	}

	// Create Spanner client (for queries)
	dataClient, err := spanner.NewClientWithConfig(context.Background(), db, spanner.ClientConfig{SessionPoolConfig: sessionPoolConfig})
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create new Spanner client: %w", err)
	}

	// Create Spanner admin client (for creating databases)
	adminClient, err := database.NewDatabaseAdminClient(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create new Spanner admin client: %w", err)
	}

	return dataClient, adminClient, nil
}

func TestSpannerToolEndpoints(t *testing.T) {
	sourceConfig := getSpannerVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	// Create Spanner client
	dataClient, adminClient, err := initSpannerClients(ctx, SPANNER_PROJECT, SPANNER_INSTANCE, SPANNER_DATABASE)
	if err != nil {
		t.Fatalf("unable to create Spanner client: %s", err)
	}

	// create table name with UUID
	tableNameParam := "param_table_" + strings.Replace(uuid.New().String(), "-", "", -1)
	tableNameAuth := "auth_table_" + strings.Replace(uuid.New().String(), "-", "", -1)

	// set up data for param tool
	create_statement1, insert_statement1, tool_statement1, params1 := getSpannerParamToolInfo(tableNameParam)
	dbString := fmt.Sprintf(
		"projects/%s/instances/%s/databases/%s",
		SPANNER_PROJECT,
		SPANNER_INSTANCE,
		SPANNER_DATABASE,
	)
	teardownTable1 := setupSpannerTable(t, ctx, adminClient, dataClient, create_statement1, insert_statement1, tableNameParam, dbString, params1)
	defer teardownTable1(t)

	// set up data for auth tool
	create_statement2, insert_statement2, tool_statement2, params2 := getSpannerAuthToolInfo(tableNameAuth)
	teardownTable2 := setupSpannerTable(t, ctx, adminClient, dataClient, create_statement2, insert_statement2, tableNameAuth, dbString, params2)
	defer teardownTable2(t)

	// Write config into a file and pass it to command
	toolsFile := tests.GetToolsConfig(sourceConfig, SPANNER_TOOL_KIND, tool_statement1, tool_statement2)

	cmd, cleanup, err := tests.StartCmd(ctx, toolsFile, args...)
	if err != nil {
		t.Fatalf("command initialization returned an error: %s", err)
	}
	defer cleanup()

	waitCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	out, err := cmd.WaitForString(waitCtx, regexp.MustCompile(`Server ready to serve`))
	if err != nil {
		t.Logf("toolbox command logs: \n%s", out)
		t.Fatalf("toolbox didn't start successfully: %s", err)
	}

	tests.RunToolGetTest(t)

	select1Want := "[{\"\":\"1\"}]"
	invokeParamWant := "[{\"id\":\"1\",\"name\":\"Alice\"},{\"id\":\"3\",\"name\":\"Sid\"}]"
	mcpInvokeParamWant := `{"jsonrpc":"2.0","id":"my-param-tool","result":{"content":[{"type":"text","text":"{\"id\":\"1\",\"name\":\"Alice\"}"},{"type":"text","text":"{\"id\":\"3\",\"name\":\"Sid\"}"}]}}`
	failInvocationWant := `"jsonrpc":"2.0","id":"invoke-fail-tool","result":{"content":[{"type":"text","text":"unable to execute client: unable to parse row: spanner: code = \"InvalidArgument\", desc = \"Syntax error: Unexpected identifier \\\\\\\"SELEC\\\\\\\" [at 1:1]\\\\nSELEC 1;\\\\n^\"`

	tests.RunToolInvokeTest(t, select1Want, invokeParamWant)
	tests.RunMCPToolCallMethod(t, mcpInvokeParamWant, failInvocationWant)
}

// getSpannerToolInfo returns statements and param for my-param-tool for spanner-sql kind
func getSpannerParamToolInfo(tableName string) (string, string, string, map[string]any) {
	create_statement := fmt.Sprintf("CREATE TABLE %s (id INT64, name STRING(MAX)) PRIMARY KEY (id)", tableName)
	insert_statement := fmt.Sprintf("INSERT INTO %s (id, name) VALUES (1, @name1), (2, @name2), (3, @name3)", tableName)
	tool_statement := fmt.Sprintf("SELECT * FROM %s WHERE id = @id OR name = @name", tableName)
	params := map[string]any{"name1": "Alice", "name2": "Jane", "name3": "Sid"}
	return create_statement, insert_statement, tool_statement, params
}

// getSpannerAuthToolInfo returns statements and param of my-auth-tool for spanner-sql kind
func getSpannerAuthToolInfo(tableName string) (string, string, string, map[string]any) {
	create_statement := fmt.Sprintf("CREATE TABLE %s (id INT64, name STRING(MAX), email STRING(MAX)) PRIMARY KEY (id)", tableName)
	insert_statement := fmt.Sprintf("INSERT INTO %s (id, name, email) VALUES (1, @name1, @email1), (2, @name2, @email2)", tableName)
	tool_statement := fmt.Sprintf("SELECT name FROM %s WHERE email = @email", tableName)
	params := map[string]any{
		"name1":  "Alice",
		"email1": tests.SERVICE_ACCOUNT_EMAIL,
		"name2":  "Jane",
		"email2": "janedoe@gmail.com",
	}
	return create_statement, insert_statement, tool_statement, params
}

// setupSpannerTable creates and inserts data into a table of tool
// compatible with spanner-sql tool
func setupSpannerTable(t *testing.T, ctx context.Context, adminClient *database.DatabaseAdminClient, dataClient *spanner.Client, create_statement, insert_statement, tableName, dbString string, params map[string]any) func(*testing.T) {

	// Create table
	op, err := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
		Database:   dbString,
		Statements: []string{create_statement},
	})
	if err != nil {
		t.Fatalf("unable to start create table operation %s: %s", tableName, err)
	}
	err = op.Wait(ctx)
	if err != nil {
		t.Fatalf("unable to create test table %s: %s", tableName, err)
	}

	// Insert test data
	_, err = dataClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.Statement{
			SQL:    insert_statement,
			Params: params,
		}
		_, err := txn.Update(ctx, stmt)
		return err
	})
	if err != nil {
		t.Fatalf("unable to insert test data: %s", err)
	}

	return func(t *testing.T) {
		// tear down test
		op, err = adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
			Database:   dbString,
			Statements: []string{fmt.Sprintf("DROP TABLE %s", tableName)},
		})
		if err != nil {
			t.Errorf("unable to start drop table operation: %s", err)
			return
		}

		err = op.Wait(ctx)
		if err != nil {
			t.Errorf("Teardown failed: %s", err)
		}
	}
}
