//go:build integration && spanner

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

package tests

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/googleapis/go-sql-spanner"
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

func initSpannerDb(ctx context.Context, project, instance, dbname string) (*sql.DB, error) {
	// Create DSN
	dsn := fmt.Sprintf("projects/%s/instances/%s/databases/%s", project, instance, dbname)

	// Open DB connection
	db, err := sql.Open("spanner", dsn)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func TestSpannerToolEndpoints(t *testing.T) {
	sourceConfig := getSpannerVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	// Create Spanner DB
	pool, err := initSpannerDb(ctx, SPANNER_PROJECT, SPANNER_INSTANCE, SPANNER_DATABASE)
	if err != nil {
		t.Fatalf("unable to create Spanner db: %s", err)
	}

	// create table name with UUID
	tableNameParam := "param_table_" + strings.Replace(uuid.New().String(), "-", "", -1)
	tableNameAuth := "auth_table_" + strings.Replace(uuid.New().String(), "-", "", -1)

	// set up data for param tool
	create_statement1, insert_statement1, tool_statement1, params1 := GetSpannerParamToolInfo(tableNameParam)
	teardownTable1 := SetupSpannerTable(t, ctx, pool, create_statement1, insert_statement1, tableNameParam, params1)
	defer teardownTable1(t)

	// set up data for auth tool
	create_statement2, insert_statement2, tool_statement2, params2 := GetSpannerAuthToolInfo(tableNameAuth)
	teardownTable2 := SetupSpannerTable(t, ctx, pool, create_statement2, insert_statement2, tableNameAuth, params2)
	defer teardownTable2(t)

	// Write config into a file and pass it to command
	toolsFile := GetToolsConfig(sourceConfig, SPANNER_TOOL_KIND, tool_statement1, tool_statement2)

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

	select_1_want := "[{\"\":1}]"
	RunToolInvokeTest(t, select_1_want)
}
