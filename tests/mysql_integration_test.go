//go:build integration && mysql

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
	"database/sql"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

var (
	MYSQL_SOURCE_KIND = "mysql"
	MYSQL_TOOL_KIND   = "mysql-sql"
	MYSQL_DATABASE    = os.Getenv("MYSQL_DATABASE")
	MYSQL_HOST        = os.Getenv("MYSQL_HOST")
	MYSQL_PORT        = os.Getenv("MYSQL_PORT")
	MYSQL_USER        = os.Getenv("MYSQL_USER")
	MYSQL_PASS        = os.Getenv("MYSQL_PASS")
)

func getMySQLVars(t *testing.T) map[string]any {
	switch "" {
	case MYSQL_DATABASE:
		t.Fatal("'MYSQL_DATABASE' not set")
	case MYSQL_HOST:
		t.Fatal("'MYSQL_HOST' not set")
	case MYSQL_PORT:
		t.Fatal("'MYSQL_PORT' not set")
	case MYSQL_USER:
		t.Fatal("'MYSQL_USER' not set")
	case MYSQL_PASS:
		t.Fatal("'MYSQL_PASS' not set")
	}

	return map[string]any{
		"kind":     MYSQL_SOURCE_KIND,
		"host":     MYSQL_HOST,
		"port":     MYSQL_PORT,
		"database": MYSQL_DATABASE,
		"user":     MYSQL_USER,
		"password": MYSQL_PASS,
	}
}

// Copied over from mysql.go
func initMySQLConnectionPool(host, port, user, pass, dbname string) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", user, pass, host, port, dbname)

	// Interact with the driver directly as you normally would
	pool, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}
	return pool, nil
}

func TestMySQLToolEndpoints(t *testing.T) {
	sourceConfig := getMySQLVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	pool, err := initMySQLConnectionPool(MYSQL_HOST, MYSQL_PORT, MYSQL_USER, MYSQL_PASS, MYSQL_DATABASE)
	if err != nil {
		t.Fatalf("unable to create MySQL connection pool: %s", err)
	}

	// create table name with UUID
	tableNameParam := "param_table_" + strings.Replace(uuid.New().String(), "-", "", -1)
	tableNameAuth := "auth_table_" + strings.Replace(uuid.New().String(), "-", "", -1)

	// set up data for param tool
	create_statement1, insert_statement1, tool_statement1, params1 := GetMysqlParamToolInfo(tableNameParam)
	teardownTable1 := SetupMySQLTable(t, ctx, pool, create_statement1, insert_statement1, tableNameParam, params1)
	defer teardownTable1(t)

	// set up data for auth tool
	create_statement2, insert_statement2, tool_statement2, params2 := GetMysqlLAuthToolInfo(tableNameAuth)
	teardownTable2 := SetupMySQLTable(t, ctx, pool, create_statement2, insert_statement2, tableNameAuth, params2)
	defer teardownTable2(t)

	// Write config into a file and pass it to command
	toolsFile := GetToolsConfig(sourceConfig, MYSQL_TOOL_KIND, tool_statement1, tool_statement2)

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

	select_1_want := "[{\"1\":1}]"
	RunToolInvokeTest(t, select_1_want)
	RunMCPToolCallMethod(t)
}
