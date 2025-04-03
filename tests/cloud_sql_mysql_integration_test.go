//go:build integration && cloudsqlmysql

//
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
	"slices"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/cloudsqlconn"
	"cloud.google.com/go/cloudsqlconn/mysql/mysql"
	"github.com/google/uuid"
)

var (
	CLOUD_SQL_MYSQL_SOURCE_KIND = "cloud-sql-mysql"
	CLOUD_SQL_MYSQL_TOOL_KIND   = "mysql-sql"
	CLOUD_SQL_MYSQL_PROJECT     = os.Getenv("CLOUD_SQL_MYSQL_PROJECT")
	CLOUD_SQL_MYSQL_REGION      = os.Getenv("CLOUD_SQL_MYSQL_REGION")
	CLOUD_SQL_MYSQL_INSTANCE    = os.Getenv("CLOUD_SQL_MYSQL_INSTANCE")
	CLOUD_SQL_MYSQL_DATABASE    = os.Getenv("CLOUD_SQL_MYSQL_DATABASE")
	CLOUD_SQL_MYSQL_USER        = os.Getenv("CLOUD_SQL_MYSQL_USER")
	CLOUD_SQL_MYSQL_PASS        = os.Getenv("CLOUD_SQL_MYSQL_PASS")
)

func getCloudSQLMySQLVars(t *testing.T) map[string]any {
	switch "" {
	case CLOUD_SQL_MYSQL_PROJECT:
		t.Fatal("'CLOUD_SQL_MYSQL_PROJECT' not set")
	case CLOUD_SQL_MYSQL_REGION:
		t.Fatal("'CLOUD_SQL_MYSQL_REGION' not set")
	case CLOUD_SQL_MYSQL_INSTANCE:
		t.Fatal("'CLOUD_SQL_MYSQL_INSTANCE' not set")
	case CLOUD_SQL_MYSQL_DATABASE:
		t.Fatal("'CLOUD_SQL_MYSQL_DATABASE' not set")
	case CLOUD_SQL_MYSQL_USER:
		t.Fatal("'CLOUD_SQL_MYSQL_USER' not set")
	case CLOUD_SQL_MYSQL_PASS:
		t.Fatal("'CLOUD_SQL_MYSQL_PASS' not set")
	}

	return map[string]any{
		"kind":     CLOUD_SQL_MYSQL_SOURCE_KIND,
		"project":  CLOUD_SQL_MYSQL_PROJECT,
		"instance": CLOUD_SQL_MYSQL_INSTANCE,
		"region":   CLOUD_SQL_MYSQL_REGION,
		"database": CLOUD_SQL_MYSQL_DATABASE,
		"user":     CLOUD_SQL_MYSQL_USER,
		"password": CLOUD_SQL_MYSQL_PASS,
	}
}

// Copied over from cloud_sql_mysql.go
func initCloudSQLMySQLConnectionPool(project, region, instance, ipType, user, pass, dbname string) (*sql.DB, error) {

	// Create a new dialer with options
	dialOpts, err := GetCloudSQLDialOpts(ipType)
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

	pool, err := initCloudSQLMySQLConnectionPool(CLOUD_SQL_MYSQL_PROJECT, CLOUD_SQL_MYSQL_REGION, CLOUD_SQL_MYSQL_INSTANCE, "public", CLOUD_SQL_MYSQL_USER, CLOUD_SQL_MYSQL_PASS, CLOUD_SQL_MYSQL_DATABASE)
	if err != nil {
		t.Fatalf("unable to create Cloud SQL connection pool: %s", err)
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
	toolsFile := GetToolsConfig(sourceConfig, CLOUD_SQL_MYSQL_TOOL_KIND, tool_statement1, tool_statement2)

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
	fail_invocation_want := `{"jsonrpc":"2.0","id":"invoke-fail-tool","result":{"content":[{"type":"text","text":"unable to execute query: Error 1064 (42000): You have an error in your SQL syntax; check the manual that corresponds to your MySQL server version for the right syntax to use near 'SELEC 1' at line 1"}],"isError":true}}`
	RunToolInvokeTest(t, select_1_want)
	RunMCPToolCallMethod(t, fail_invocation_want)
}

// Test connection with different IP type
func TestCloudSQLMysqlIpConnection(t *testing.T) {
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
			RunSourceConnectionTest(t, sourceConfig, CLOUD_SQL_MYSQL_TOOL_KIND)
		})
	}
}
