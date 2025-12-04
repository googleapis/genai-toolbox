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
//  - "CLOUD_SQL_MYSQL_PROJECT=manukarahul-playground"
    //   - "CLOUD_SQL_MYSQL_INSTANCE=quickstart-instance"
    //   - "CLOUD_SQL_MYSQL_DATABASE=integration_test"
    //   - "CLOUD_SQL_MYSQL_REGION=us-central1"
    //   - "SERVICE_ACCOUNT_EMAIL=toolbox-identity@manukarahul-playground.iam.gserviceaccount.com"
    // secretEnv: ["postgres", "R@7075ahul", "573556258322-bdc30fafe1atq2pueddbcsrcf5q21k76.apps.googleusercontent.com"]
var (
	CloudSQLMySQLSourceKind = "cloud-sql-mysql"
	CloudSQLMySQLToolKind   = "mysql-sql"
	CloudSQLMySQLProject    = "manukarahul-playground"
	CloudSQLMySQLRegion     = "us-central1"
	CloudSQLMySQLInstance   = "integration-testing"
	CloudSQLMySQLDatabase   = "mysql"
	CloudSQLMySQLUser       = "integration-testing"
	CloudSQLMySQLPass       = "R@7075ahul"
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

	// cleanup test environment
	tests.CleanupMySQLTables(t, ctx, pool)

	// create table name with UUID
	tableNameParam := "param_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableNameAuth := "auth_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableNameTemplateParam := "template_param_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")

	// set up data for param tool
	createParamTableStmt, insertParamTableStmt, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, paramTestParams := tests.GetMySQLParamToolInfo(tableNameParam)
	teardownTable1 := tests.SetupMySQLTable(t, ctx, pool, createParamTableStmt, insertParamTableStmt, tableNameParam, paramTestParams)
	defer teardownTable1(t)

	// set up data for auth tool
	createAuthTableStmt, insertAuthTableStmt, authToolStmt, authTestParams := tests.GetMySQLAuthToolInfo(tableNameAuth)
	teardownTable2 := tests.SetupMySQLTable(t, ctx, pool, createAuthTableStmt, insertAuthTableStmt, tableNameAuth, authTestParams)
	defer teardownTable2(t)

	// Write config into a file and pass it to command
	toolsFile := tests.GetToolsConfig(sourceConfig, CloudSQLMySQLToolKind, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, authToolStmt)
	toolsFile = tests.AddMySqlExecuteSqlConfig(t, toolsFile)
	tmplSelectCombined, tmplSelectFilterCombined := tests.GetMySQLTmplToolStatement()
	toolsFile = tests.AddTemplateParamConfig(t, toolsFile, CloudSQLMySQLToolKind, tmplSelectCombined, tmplSelectFilterCombined, "")
	toolsFile = tests.AddMySQLPrebuiltToolConfig(t, toolsFile)

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

	// Get configs for tests
	select1Want, mcpMyFailToolWant, createTableStatement, mcpSelect1Want := tests.GetMySQLWants()

	// Run tests
	tests.RunToolGetTest(t)
	tests.RunToolInvokeTest(t, select1Want, tests.DisableArrayTest())
	tests.RunMCPToolCallMethod(t, mcpMyFailToolWant, mcpSelect1Want)
	tests.RunExecuteSqlToolInvokeTest(t, createTableStatement, select1Want)
	tests.RunToolInvokeWithTemplateParameters(t, tableNameTemplateParam)

	// Run specific MySQL tool tests
	tests.RunMySQLListTablesTest(t, CloudSQLMySQLDatabase, tableNameParam, tableNameAuth)
	tests.RunMySQLListActiveQueriesTest(t, ctx, pool)
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
