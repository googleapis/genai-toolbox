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

package snowflake

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"
	"github.com/jmoiron/sqlx"
	_ "github.com/snowflakedb/gosnowflake"
)

var (
	SnowflakeSourceKind = "snowflake"
	SnowflakeToolKind   = "snowflake-sql"
	SnowflakeAccount    = os.Getenv("SNOWFLAKE_ACCOUNT")
	SnowflakeUser       = os.Getenv("SNOWFLAKE_USER")
	SnowflakePassword   = os.Getenv("SNOWFLAKE_PASSWORD")
	SnowflakeDatabase   = os.Getenv("SNOWFLAKE_DATABASE")
	SnowflakeSchema     = os.Getenv("SNOWFLAKE_SCHEMA")
	SnowflakeWarehouse  = os.Getenv("SNOWFLAKE_WAREHOUSE")
	SnowflakeRole       = os.Getenv("SNOWFLAKE_ROLE")
)

func getSnowflakeVars(t *testing.T) map[string]any {
	switch "" {
	case SnowflakeAccount:
		t.Fatal("'SNOWFLAKE_ACCOUNT' not set")
	case SnowflakeUser:
		t.Fatal("'SNOWFLAKE_USER' not set")
	case SnowflakePassword:
		t.Fatal("'SNOWFLAKE_PASSWORD' not set")
	case SnowflakeDatabase:
		t.Fatal("'SNOWFLAKE_DATABASE' not set")
	case SnowflakeSchema:
		t.Fatal("'SNOWFLAKE_SCHEMA' not set")
	}

	// Set defaults for optional parameters
	if SnowflakeWarehouse == "" {
		SnowflakeWarehouse = "COMPUTE_WH"
	}
	if SnowflakeRole == "" {
		SnowflakeRole = "ACCOUNTADMIN"
	}

	return map[string]any{
		"kind":      SnowflakeSourceKind,
		"account":   SnowflakeAccount,
		"user":      SnowflakeUser,
		"password":  SnowflakePassword,
		"database":  SnowflakeDatabase,
		"schema":    SnowflakeSchema,
		"warehouse": SnowflakeWarehouse,
		"role":      SnowflakeRole,
	}
}

// Copied over from snowflake.go
func initSnowflakeConnectionPool(account, user, password, database, schema, warehouse, role string) (*sqlx.DB, error) {
	// Set defaults for optional parameters
	if warehouse == "" {
		warehouse = "COMPUTE_WH"
	}
	if role == "" {
		role = "ACCOUNTADMIN"
	}

	// Snowflake DSN format: user:password@account/database/schema?warehouse=warehouse&role=role
	dsn := fmt.Sprintf("%s:%s@%s/%s/%s?warehouse=%s&role=%s&protocol=https&timeout=60", user, password, account, database, schema, warehouse, role)
	db, err := sqlx.Connect("snowflake", dsn)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection: %w", err)
	}

	return db, nil
}

func TestSnowflake(t *testing.T) {
	sourceConfig := getSnowflakeVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	db, err := initSnowflakeConnectionPool(SnowflakeAccount, SnowflakeUser, SnowflakePassword, SnowflakeDatabase, SnowflakeSchema, SnowflakeWarehouse, SnowflakeRole)
	if err != nil {
		t.Fatalf("unable to create snowflake connection pool: %s", err)
	}

	// create table name with UUID
	tableNameParam := "param_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableNameAuth := "auth_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableNameTemplateParam := "template_param_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")

	// set up data for param tool
	createParamTableStmt, insertParamTableStmt, paramToolStmt, paramToolStmt2, arrayToolStmt, paramTestParams := tests.GetSnowflakeParamToolInfo(tableNameParam)
	teardownTable1 := tests.SetupSnowflakeTable(t, ctx, db, createParamTableStmt, insertParamTableStmt, tableNameParam, paramTestParams)
	defer teardownTable1(t)

	// set up data for auth tool
	createAuthTableStmt, insertAuthTableStmt, authToolStmt, authTestParams := tests.GetSnowflakeAuthToolInfo(tableNameAuth)
	teardownTable2 := tests.SetupSnowflakeTable(t, ctx, db, createAuthTableStmt, insertAuthTableStmt, tableNameAuth, authTestParams)
	defer teardownTable2(t)

	// Write config into a file and pass it to command
	toolsFile := tests.GetToolsConfig(sourceConfig, SnowflakeToolKind, paramToolStmt, paramToolStmt2, arrayToolStmt, authToolStmt)
	toolsFile = tests.AddSnowflakeExecuteSqlConfig(t, toolsFile)
	tmplSelectCombined, tmplSelectFilterCombined := tests.GetSnowflakeTmplToolStatement()
	toolsFile = tests.AddTemplateParamConfig(t, toolsFile, SnowflakeToolKind, tmplSelectCombined, tmplSelectFilterCombined, "")

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

	select1Want, failInvocationWant, createTableStatement := tests.GetSnowflakeWants()
	invokeParamWant, invokeParamWantNull, mcpInvokeParamWant := tests.GetNonSpannerInvokeParamWant()
	tests.RunToolInvokeTest(t, select1Want, invokeParamWant, invokeParamWantNull, true)
	tests.RunExecuteSqlToolInvokeTest(t, createTableStatement, select1Want)
	tests.RunMCPToolCallMethod(t, mcpInvokeParamWant, failInvocationWant)
	tests.RunToolInvokeWithTemplateParameters(t, tableNameTemplateParam, tests.NewTemplateParameterTestConfig())
}
