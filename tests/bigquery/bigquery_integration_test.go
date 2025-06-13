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

package bigquery

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	bigqueryapi "cloud.google.com/go/bigquery"
	"github.com/google/uuid"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

var (
	BIGQUERY_SOURCE_KIND = "bigquery"
	BIGQUERY_TOOL_KIND   = "bigquery-sql"
	BIGQUERY_PROJECT     = os.Getenv("BIGQUERY_PROJECT")
	BIGQUERY_LOCATION    = os.Getenv("BIGQUERY_LOCATION")
)

func getBigQueryVars(t *testing.T) map[string]any {
	switch "" {
	case BIGQUERY_PROJECT:
		t.Fatal("'BIGQUERY_PROJECT' not set")
	}

	return map[string]any{
		"kind":     BIGQUERY_SOURCE_KIND,
		"project":  BIGQUERY_PROJECT,
		"location": BIGQUERY_LOCATION,
	}
}

// Copied over from bigquery.go
func initBigQueryConnection(project, location string) (*bigqueryapi.Client, error) {
	ctx := context.Background()
	cred, err := google.FindDefaultCredentials(ctx, bigqueryapi.Scope)
	if err != nil {
		return nil, fmt.Errorf("failed to find default Google Cloud credentials with scope %q: %w", bigqueryapi.Scope, err)
	}

	client, err := bigqueryapi.NewClient(ctx, project, option.WithCredentials(cred))
	client.Location = location
	if err != nil {
		return nil, fmt.Errorf("failed to create BigQuery client for project %q: %w", project, err)
	}
	return client, nil
}

func TestBigQueryToolEndpoints(t *testing.T) {
	sourceConfig := getBigQueryVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	client, err := initBigQueryConnection(BIGQUERY_PROJECT, BIGQUERY_LOCATION)
	if err != nil {
		t.Fatalf("unable to create Cloud SQL connection pool: %s", err)
	}

	// create table name with UUID
	datasetName := fmt.Sprintf("temp_toolbox_test_%s", strings.ReplaceAll(uuid.New().String(), "-", ""))
	tableNameParam := fmt.Sprintf("`%s.%s.param_table_%s`",
		BIGQUERY_PROJECT,
		datasetName,
		strings.ReplaceAll(uuid.New().String(), "-", ""),
	)
	tableNameAuth := fmt.Sprintf("`%s.%s.auth_table_%s`",
		BIGQUERY_PROJECT,
		datasetName,
		strings.ReplaceAll(uuid.New().String(), "-", ""),
	)
	// set up data for param tool
	create_statement1, insert_statement1, tool_statement1, params1 := getBigQueryParamToolInfo(BIGQUERY_PROJECT, datasetName, tableNameParam)
	teardownTable1 := setupBigQueryTable(t, ctx, client, create_statement1, insert_statement1, datasetName, tableNameParam, params1)
	defer teardownTable1(t)

	// set up data for auth tool
	create_statement2, insert_statement2, tool_statement2, params2 := getBigQueryAuthToolInfo(BIGQUERY_PROJECT, datasetName, tableNameAuth)
	teardownTable2 := setupBigQueryTable(t, ctx, client, create_statement2, insert_statement2, datasetName, tableNameAuth, params2)
	defer teardownTable2(t)

	// Write config into a file and pass it to command
	toolsFile := tests.GetToolsConfig(sourceConfig, BIGQUERY_TOOL_KIND, tool_statement1, tool_statement2)

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

	select1Want := "[{\"f0_\":1}]"
	// Partial message; the full error message is too long.
	failInvocationWant := `{"jsonrpc":"2.0","id":"invoke-fail-tool","result":{"content":[{"type":"text","text":"unable to execute query: googleapi: Error 400: Syntax error: Unexpected identifier \"SELEC\" at [1:1]`
	invokeParamWant, mcpInvokeParamWant := tests.GetNonSpannerInvokeParamWant()
	tests.RunToolInvokeTest(t, select1Want, invokeParamWant)
	tests.RunMCPToolCallMethod(t, mcpInvokeParamWant, failInvocationWant)
}

// getBigQueryParamToolInfo returns statements and param for my-param-tool for bigquery kind
func getBigQueryParamToolInfo(projectID, datasetID, tableName string) (string, string, string, []bigqueryapi.QueryParameter) {
	createStatement := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (id INT64, name STRING);`, tableName)
	insertStatement := fmt.Sprintf(`
		INSERT INTO %s (id, name) VALUES (?, ?), (?, ?), (?, ?);`, tableName)
	toolStatement := fmt.Sprintf(`SELECT * FROM %s WHERE id = ? OR name = ? ORDER BY id;`, tableName)
	params := []bigqueryapi.QueryParameter{
		{Value: int64(1)}, {Value: "Alice"},
		{Value: int64(2)}, {Value: "Jane"},
		{Value: int64(3)}, {Value: "Sid"},
	}
	return createStatement, insertStatement, toolStatement, params
}

// getBigQueryAuthToolInfo returns statements and param of my-auth-tool for bigquery kind
func getBigQueryAuthToolInfo(projectID, datasetID, tableName string) (string, string, string, []bigqueryapi.QueryParameter) {
	createStatement := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (id INT64, name STRING, email STRING)`, tableName)
	insertStatement := fmt.Sprintf(`
		INSERT INTO %s (id, name, email) VALUES (?, ?, ?), (?, ?, ?)`, tableName)
	toolStatement := fmt.Sprintf(`
		SELECT name FROM %s WHERE email = ?`, tableName)
	params := []bigqueryapi.QueryParameter{
		{Value: int64(1)}, {Value: "Alice"}, {Value: tests.SERVICE_ACCOUNT_EMAIL},
		{Value: int64(2)}, {Value: "Jane"}, {Value: "janedoe@gmail.com"},
	}
	return createStatement, insertStatement, toolStatement, params
}

func setupBigQueryTable(t *testing.T, ctx context.Context, client *bigqueryapi.Client, create_statement, insert_statement, datasetName string, tableName string, params []bigqueryapi.QueryParameter) func(*testing.T) {
	// Create dataset
	dataset := client.Dataset(datasetName)
	_, err := dataset.Metadata(ctx)

	if err != nil {
		apiErr, ok := err.(*googleapi.Error)
		if !ok || apiErr.Code != 404 {
			t.Fatalf("Failed to check dataset %q existence: %v", datasetName, err)
		}
		metadataToCreate := &bigqueryapi.DatasetMetadata{Name: datasetName}
		if err := dataset.Create(ctx, metadataToCreate); err != nil {
			t.Fatalf("Failed to create dataset %q: %v", datasetName, err)
		}
	}

	// Create table
	createJob, err := client.Query(create_statement).Run(ctx)

	if err != nil {
		t.Fatalf("Failed to start create table job for %s: %v", tableName, err)
	}
	createStatus, err := createJob.Wait(ctx)
	if err != nil {
		t.Fatalf("Failed to wait for create table job for %s: %v", tableName, err)
	}
	if err := createStatus.Err(); err != nil {
		t.Fatalf("Create table job for %s failed: %v", tableName, err)
	}

	// Insert test data
	insertQuery := client.Query(insert_statement)
	insertQuery.Parameters = params
	insertJob, err := insertQuery.Run(ctx)
	if err != nil {
		t.Fatalf("Failed to start insert job for %s: %v", tableName, err)
	}
	insertStatus, err := insertJob.Wait(ctx)
	if err != nil {
		t.Fatalf("Failed to wait for insert job for %s: %v", tableName, err)
	}
	if err := insertStatus.Err(); err != nil {
		t.Fatalf("Insert job for %s failed: %v", tableName, err)
	}

	return func(t *testing.T) {
		// tear down table
		dropSQL := fmt.Sprintf("drop table %s", tableName)
		dropJob, err := client.Query(dropSQL).Run(ctx)
		if err != nil {
			t.Errorf("Failed to start drop table job for %s: %v", tableName, err)
			return
		}
		dropStatus, err := dropJob.Wait(ctx)
		if err != nil {
			t.Errorf("Failed to wait for drop table job for %s: %v", tableName, err)
			return
		}
		if err := dropStatus.Err(); err != nil {
			t.Errorf("Error dropping table %s: %v", tableName, err)
		}

		// tear down dataset
		datasetToTeardown := client.Dataset(datasetName)
		tablesIterator := datasetToTeardown.Tables(ctx)
		_, err = tablesIterator.Next()

		if err == iterator.Done {
			if err := datasetToTeardown.Delete(ctx); err != nil {
				t.Errorf("Failed to delete dataset %s: %v", datasetName, err)
			}
		} else if err != nil {
			t.Errorf("Failed to list tables in dataset %s to check emptiness: %v.", datasetName, err)
		}
	}
}
