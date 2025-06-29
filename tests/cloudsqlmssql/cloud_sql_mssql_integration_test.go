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

package cloudsqlmssql

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/cloudsqlconn"
	"cloud.google.com/go/cloudsqlconn/sqlserver/mssql"
	"github.com/google/uuid"
	"github.com/googleapis/genai-toolbox/tests"
)

var (
	CloudSQLMssqlSourceKind = "cloud-sql-mssql"
	CloudSQLMssqlToolKind   = "mssql-sql"
	CloudSQLMssqlProject    = os.Getenv("CLOUD_SQL_MSSQL_PROJECT")
	CloudSQLMssqlRegion     = os.Getenv("CLOUD_SQL_MSSQL_REGION")
	CloudSQLMssqlInstance   = os.Getenv("CLOUD_SQL_MSSQL_INSTANCE")
	CloudSQLMssqlDatabase   = os.Getenv("CLOUD_SQL_MSSQL_DATABASE")
	CloudSQLMssqlIp         = os.Getenv("CLOUD_SQL_MSSQL_IP")
	CloudSQLMssqlUser       = os.Getenv("CLOUD_SQL_MSSQL_USER")
	CloudSQLMssqlPass       = os.Getenv("CLOUD_SQL_MSSQL_PASS")
)

func getCloudSQLMssqlVars(t *testing.T) map[string]any {
	switch "" {
	case CloudSQLMssqlProject:
		t.Fatal("'CLOUD_SQL_MSSQL_PROJECT' not set")
	case CloudSQLMssqlRegion:
		t.Fatal("'CLOUD_SQL_MSSQL_REGION' not set")
	case CloudSQLMssqlInstance:
		t.Fatal("'CLOUD_SQL_MSSQL_INSTANCE' not set")
	case CloudSQLMssqlIp:
		t.Fatal("'CLOUD_SQL_MSSQL_IP' not set")
	case CloudSQLMssqlDatabase:
		t.Fatal("'CLOUD_SQL_MSSQL_DATABASE' not set")
	case CloudSQLMssqlUser:
		t.Fatal("'CLOUD_SQL_MSSQL_USER' not set")
	case CloudSQLMssqlPass:
		t.Fatal("'CLOUD_SQL_MSSQL_PASS' not set")
	}

	return map[string]any{
		"kind":      CloudSQLMssqlSourceKind,
		"project":   CloudSQLMssqlProject,
		"instance":  CloudSQLMssqlInstance,
		"ipAddress": CloudSQLMssqlIp,
		"region":    CloudSQLMssqlRegion,
		"database":  CloudSQLMssqlDatabase,
		"user":      CloudSQLMssqlUser,
		"password":  CloudSQLMssqlPass,
	}
}

// Copied over from cloud_sql_mssql.go
func initCloudSQLMssqlConnection(project, region, instance, ipAddress, ipType, user, pass, dbname string) (*sql.DB, error) {
	// Create dsn
	query := fmt.Sprintf("database=%s&cloudsql=%s:%s:%s", dbname, project, region, instance)
	url := &url.URL{
		Scheme:   "sqlserver",
		User:     url.UserPassword(user, pass),
		Host:     ipAddress,
		RawQuery: query,
	}

	// Get dial options
	dialOpts, err := tests.GetCloudSQLDialOpts(ipType)
	if err != nil {
		return nil, err
	}

	// Register sql server driver
	if !slices.Contains(sql.Drivers(), "cloudsql-sqlserver-driver") {
		_, err := mssql.RegisterDriver("cloudsql-sqlserver-driver", cloudsqlconn.WithDefaultDialOptions(dialOpts...))
		if err != nil {
			return nil, err
		}
	}

	// Open database connection
	db, err := sql.Open(
		"cloudsql-sqlserver-driver",
		url.String(),
	)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func TestCloudSQLMssqlToolEndpoints(t *testing.T) {
	sourceConfig := getCloudSQLMssqlVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	db, err := initCloudSQLMssqlConnection(CloudSQLMssqlProject, CloudSQLMssqlRegion, CloudSQLMssqlInstance, CloudSQLMssqlIp, "public", CloudSQLMssqlUser, CloudSQLMssqlPass, CloudSQLMssqlDatabase)
	if err != nil {
		t.Fatalf("unable to create Cloud SQL connection pool: %s", err)
	}

	// create table name with UUID
	tableNameParam := "param_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableNameAuth := "auth_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableNameTemplateParam := "template_param_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")

	// set up data for param tool
	createStatement1, insertStatement1, toolStatement1, params1 := tests.GetMssqlParamToolInfo(tableNameParam)
	teardownTable1 := tests.SetupMsSQLTable(t, ctx, db, createStatement1, insertStatement1, tableNameParam, params1)
	defer teardownTable1(t)

	// set up data for auth tool
	createStatement2, insertStatement2, toolStatement2, params2 := tests.GetMssqlAuthToolInfo(tableNameAuth)
	teardownTable2 := tests.SetupMsSQLTable(t, ctx, db, createStatement2, insertStatement2, tableNameAuth, params2)
	defer teardownTable2(t)

	// Write config into a file and pass it to command
	toolsFile := tests.GetToolsConfig(sourceConfig, CloudSQLMssqlToolKind, toolStatement1, toolStatement2)
	toolsFile = tests.AddMssqlExecuteSqlConfig(t, toolsFile)
	tmplSelectCombined, tmplSelectFilterCombined := tests.GetMssqlTmplToolStatement()
	toolsFile = tests.AddTemplateParamConfig(t, toolsFile, CloudSQLMssqlToolKind, tmplSelectCombined, tmplSelectFilterCombined, "")

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

	select1Want, failInvocationWant, createTableStatement := tests.GetMssqlWants()
	invokeParamWant, mcpInvokeParamWant := tests.GetNonSpannerInvokeParamWant()
	tests.RunToolInvokeTest(t, select1Want, invokeParamWant)
	tests.RunExecuteSqlToolInvokeTest(t, createTableStatement, select1Want)
	tests.RunMCPToolCallMethod(t, mcpInvokeParamWant, failInvocationWant)
	tests.RunToolInvokeWithTemplateParameters(t, tableNameTemplateParam, tests.NewTemplateParameterTestConfig())
}

// Test connection with different IP type
func TestCloudSQLMssqlIpConnection(t *testing.T) {
	sourceConfig := getCloudSQLMssqlVars(t)

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
			err := tests.RunSourceConnectionTest(t, sourceConfig, CloudSQLMssqlToolKind)
			if err != nil {
				t.Fatalf("Connection test failure: %s", err)
			}
		})
	}
}
