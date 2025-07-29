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
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"
)

var (
	CloudSQLMySQLSourceKind = "cloud-sql-mysql"
	CloudSQLMySQLToolKind   = "mysql-sql"
	CloudSQLMySQLProject    = os.Getenv("CLOUD_SQL_MYSQL_PROJECT")
	CloudSQLMySQLRegion     = os.Getenv("CLOUD_SQL_MYSQL_REGION")
	CloudSQLMySQLInstance   = os.Getenv("CLOUD_SQL_MYSQL_INSTANCE")
	CloudSQLMySQLDatabase   = os.Getenv("CLOUD_SQL_MYSQL_DATABASE")
	CloudSQLMySQLUser       = os.Getenv("CLOUD_SQL_MYSQL_USER")
	CloudSQLMySQLPass       = os.Getenv("CLOUD_SQL_MYSQL_PASS")
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
	createParamTableStmt, insertParamTableStmt, paramToolStmt, paramToolStmt2, arrayToolStmt, paramTestParams := tests.GetMySQLParamToolInfo(tableNameParam)
	teardownTable1 := tests.SetupMySQLTable(t, ctx, pool, createParamTableStmt, insertParamTableStmt, tableNameParam, paramTestParams)
	defer teardownTable1(t)

	// set up data for auth tool
	createAuthTableStmt, insertAuthTableStmt, authToolStmt, authTestParams := tests.GetMySQLAuthToolInfo(tableNameAuth)
	teardownTable2 := tests.SetupMySQLTable(t, ctx, pool, createAuthTableStmt, insertAuthTableStmt, tableNameAuth, authTestParams)
	defer teardownTable2(t)

	// Write config into a file and pass it to command
	toolsFile := tests.GetToolsConfig(sourceConfig, CloudSQLMySQLToolKind, paramToolStmt, paramToolStmt2, arrayToolStmt, authToolStmt)
	toolsFile = tests.AddMySqlExecuteSqlConfig(t, toolsFile)
	tmplSelectCombined, tmplSelectFilterCombined := tests.GetMySQLTmplToolStatement()
	toolsFile = tests.AddTemplateParamConfig(t, toolsFile, CloudSQLMySQLToolKind, tmplSelectCombined, tmplSelectFilterCombined, "")

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

	select1Want, failInvocationWant, createTableStatement := tests.GetMySQLWants()
	invokeParamWant, invokeParamWantNull, mcpInvokeParamWant := tests.GetNonSpannerInvokeParamWant()
	tests.RunToolInvokeTest(t, select1Want, invokeParamWant, invokeParamWantNull, false)
	tests.RunExecuteSqlToolInvokeTest(t, createTableStatement, select1Want)
	tests.RunMCPToolCallMethod(t, mcpInvokeParamWant, failInvocationWant)
	tests.RunToolInvokeWithTemplateParameters(t, tableNameTemplateParam, tests.NewTemplateParameterTestConfig())
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
