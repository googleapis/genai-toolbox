//go:build integration && bigquery

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

package tests

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
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

var (
	BIGQUERY_SOURCE_KIND = "bigquery"
	BIGQUERY_TOOL_KIND   = "bigquery-sql"
	BIGQUERY_PROJECT     = os.Getenv("BIGQUERY_PROJECT")
)

func getBigQueryVars(t *testing.T, location *string) map[string]any {
	switch "" {
	case BIGQUERY_PROJECT:
		t.Fatal("'BIGQUERY_PROJECT' not set")
	}

	return map[string]any{
		"kind":     BIGQUERY_SOURCE_KIND,
		"project":  BIGQUERY_PROJECT,
		"location": location,
	}
}

func initBigQueryConnection(project string) (*bigqueryapi.Client, error) {
	ctx := context.Background()
	cred, err := google.FindDefaultCredentials(ctx, bigqueryapi.Scope)
	if err != nil {
		return nil, fmt.Errorf("failed to find default Google Cloud credentials with scope %q: %w", bigqueryapi.Scope, err)
	}

	client, err := bigqueryapi.NewClient(ctx, project, option.WithCredentials(cred))
	if err != nil {
		return nil, fmt.Errorf("failed to create BigQuery client for project %q: %w", project, err)
	}
	return client, nil
}

func TestBigQueryToolEndpoints(t *testing.T) {
	sourceConfig := getBigQueryVars(t, nil)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	client, err := initBigQueryConnection(BIGQUERY_PROJECT)
	if err != nil {
		t.Fatalf("unable to create Cloud SQL connection pool: %s", err)
	}

	// create table name with UUID
	datasetName := fmt.Sprintf("temp_dataset_%s", strings.Replace(uuid.New().String(), "-", "", -1))
	tableNameParam := fmt.Sprintf("`%s.%s.param_table_%s`",
		BIGQUERY_PROJECT,
		datasetName,
		strings.Replace(uuid.New().String(), "-", "", -1),
	)
	tableNameAuth := fmt.Sprintf("`%s.%s.auth_table_%s`",
		BIGQUERY_PROJECT,
		datasetName,
		strings.Replace(uuid.New().String(), "-", "", -1),
	)
	// set up data for param tool
	create_statement1, insert_statement1, tool_statement1, params1 := GetBigQueryParamToolInfo(BIGQUERY_PROJECT, datasetName, tableNameParam)
	teardownTable1 := SetupBigQueryTable(t, ctx, client, create_statement1, insert_statement1, datasetName, tableNameParam, params1)
	defer teardownTable1(t)

	// set up data for auth tool
	create_statement2, insert_statement2, tool_statement2, params2 := GetBigQueryAuthToolInfo(BIGQUERY_PROJECT, datasetName, tableNameAuth)
	teardownTable2 := SetupBigQueryTable(t, ctx, client, create_statement2, insert_statement2, datasetName, tableNameAuth, params2)
	defer teardownTable2(t)

	// Write config into a file and pass it to command
	toolsFile := GetToolsConfig(sourceConfig, BIGQUERY_TOOL_KIND, tool_statement1, tool_statement2)

	cmd, cleanup, err := StartCmd(ctx, toolsFile, args...)
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

	RunToolGetTest(t)

	select_1_want := "[{\"f0_\":1}]"
	// Partial message; the full error message is too long.
	fail_invocation_want := "{jsonrpc:2.0,id:invoke-fail-tool,result:{content:[{type:text,text:unable to execute query: googleapi: Error 400: Syntax error: Unexpected identifier SELEC at [1:1]"
	RunToolInvokeTest(t, select_1_want)
	RunMCPToolCallMethod(t, fail_invocation_want)
}
