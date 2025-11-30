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

package cloudsqlpg

import (
	"context"
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/cloudsqlconn"
	"github.com/google/uuid"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	CloudSQLPostgresSourceKind = "cloud-sql-postgres"
	CloudSQLPostgresToolKind   = "postgres-sql"
	CloudSQLPostgresProject    = os.Getenv("CLOUD_SQL_POSTGRES_PROJECT")
	CloudSQLPostgresRegion     = os.Getenv("CLOUD_SQL_POSTGRES_REGION")
	CloudSQLPostgresInstance   = os.Getenv("CLOUD_SQL_POSTGRES_INSTANCE")
	CloudSQLPostgresDatabase   = os.Getenv("CLOUD_SQL_POSTGRES_DATABASE")
	CloudSQLPostgresUser       = os.Getenv("CLOUD_SQL_POSTGRES_USER")
	CloudSQLPostgresPass       = os.Getenv("CLOUD_SQL_POSTGRES_PASS")
)

func getCloudSQLPgVars(t *testing.T) map[string]any {
	switch "" {
	case CloudSQLPostgresProject:
		t.Fatal("'CLOUD_SQL_POSTGRES_PROJECT' not set")
	case CloudSQLPostgresRegion:
		t.Fatal("'CLOUD_SQL_POSTGRES_REGION' not set")
	case CloudSQLPostgresInstance:
		t.Fatal("'CLOUD_SQL_POSTGRES_INSTANCE' not set")
	case CloudSQLPostgresDatabase:
		t.Fatal("'CLOUD_SQL_POSTGRES_DATABASE' not set")
	case CloudSQLPostgresUser:
		t.Fatal("'CLOUD_SQL_POSTGRES_USER' not set")
	case CloudSQLPostgresPass:
		t.Fatal("'CLOUD_SQL_POSTGRES_PASS' not set")
	}

	return map[string]any{
		"kind":     CloudSQLPostgresSourceKind,
		"project":  CloudSQLPostgresProject,
		"instance": CloudSQLPostgresInstance,
		"region":   CloudSQLPostgresRegion,
		"database": CloudSQLPostgresDatabase,
		"user":     CloudSQLPostgresUser,
		"password": CloudSQLPostgresPass,
	}
}

// Copied over from cloud_sql_pg.go
func initCloudSQLPgConnectionPool(project, region, instance, ip_type, user, pass, dbname string) (*pgxpool.Pool, error) {
	// Configure the driver to connect to the database
	dsn := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", user, pass, dbname)
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("unable to parse connection uri: %w", err)
	}

	// Create a new dialer with options
	dialOpts, err := tests.GetCloudSQLDialOpts(ip_type)
	if err != nil {
		return nil, err
	}
	d, err := cloudsqlconn.NewDialer(context.Background(), cloudsqlconn.WithDefaultDialOptions(dialOpts...))
	if err != nil {
		return nil, fmt.Errorf("unable to parse connection uri: %w", err)
	}

	// Tell the driver to use the Cloud SQL Go Connector to create connections
	i := fmt.Sprintf("%s:%s:%s", project, region, instance)
	config.ConnConfig.DialFunc = func(ctx context.Context, _ string, instance string) (net.Conn, error) {
		return d.Dial(ctx, i)
	}

	// Interact with the driver directly as you normally would
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, err
	}
	return pool, nil
}

func TestCloudSQLPgSimpleToolEndpoints(t *testing.T) {
	sourceConfig := getCloudSQLPgVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	pool, err := initCloudSQLPgConnectionPool(CloudSQLPostgresProject, CloudSQLPostgresRegion, CloudSQLPostgresInstance, "public", CloudSQLPostgresUser, CloudSQLPostgresPass, CloudSQLPostgresDatabase)
	if err != nil {
		t.Fatalf("unable to create Cloud SQL connection pool: %s", err)
	}

	// cleanup test environment
	tests.CleanupPostgresTables(t, ctx, pool)

	// create table name with UUID
	tableNameParam := "param_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableNameAuth := "auth_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableNameTemplateParam := "template_param_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")

	// set up data for param tool
	createParamTableStmt, insertParamTableStmt, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, paramTestParams := tests.GetPostgresSQLParamToolInfo(tableNameParam)
	teardownTable1 := tests.SetupPostgresSQLTable(t, ctx, pool, createParamTableStmt, insertParamTableStmt, tableNameParam, paramTestParams)
	defer teardownTable1(t)

	// set up data for auth tool
	createAuthTableStmt, insertAuthTableStmt, authToolStmt, authTestParams := tests.GetPostgresSQLAuthToolInfo(tableNameAuth)
	teardownTable2 := tests.SetupPostgresSQLTable(t, ctx, pool, createAuthTableStmt, insertAuthTableStmt, tableNameAuth, authTestParams)
	defer teardownTable2(t)

	// Write config into a file and pass it to command
	toolsFile := tests.GetToolsConfig(sourceConfig, CloudSQLPostgresToolKind, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, authToolStmt)
	toolsFile = tests.AddExecuteSqlConfig(t, toolsFile, "postgres-execute-sql")
	tmplSelectCombined, tmplSelectFilterCombined := tests.GetPostgresSQLTmplToolStatement()
	toolsFile = tests.AddTemplateParamConfig(t, toolsFile, CloudSQLPostgresToolKind, tmplSelectCombined, tmplSelectFilterCombined, "")

	toolsFile = tests.AddPostgresPrebuiltConfig(t, toolsFile)
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
	select1Want, mcpMyFailToolWant, createTableStatement, mcpSelect1Want := tests.GetPostgresWants()

	// Run tests
	tests.RunToolGetTest(t)
	tests.RunToolInvokeTest(t, select1Want)
	tests.RunMCPToolCallMethod(t, mcpMyFailToolWant, mcpSelect1Want)
	tests.RunExecuteSqlToolInvokeTest(t, createTableStatement, select1Want)
	tests.RunToolInvokeWithTemplateParameters(t, tableNameTemplateParam)

	// Run Postgres prebuilt tool tests
	tests.RunPostgresListTablesTest(t, tableNameParam, tableNameAuth, CloudSQLPostgresUser)
	tests.RunPostgresListViewsTest(t, ctx, pool, tableNameParam)
	tests.RunPostgresListSchemasTest(t, ctx, pool)
	tests.RunPostgresListActiveQueriesTest(t, ctx, pool)
	tests.RunPostgresListAvailableExtensionsTest(t)
	tests.RunPostgresListInstalledExtensionsTest(t)
	tests.RunPostgresDatabaseOverviewTest(t, ctx, pool)
	tests.RunPostgresListTriggersTest(t, ctx, pool)
	tests.RunPostgresListIndexesTest(t, ctx, pool)
	tests.RunPostgresListSequencesTest(t, ctx, pool)
	tests.RunPostgresListLocksTest(t, ctx, pool)
	tests.RunPostgresReplicationStatsTest(t, ctx, pool)
	tests.RunPostgresLongRunningTransactionsTest(t, ctx, pool)
	tests.RunPostgresListQueryStatsTest(t, ctx, pool)
	tests.RunPostgresGetColumnCardinalityTest(t, ctx, pool)
}

// Test connection with different IP type
func TestCloudSQLPgIpConnection(t *testing.T) {
	sourceConfig := getCloudSQLPgVars(t)

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
			err := tests.RunSourceConnectionTest(t, sourceConfig, CloudSQLPostgresToolKind)
			if err != nil {
				t.Fatalf("Connection test failure: %s", err)
			}
		})
	}
}

func TestCloudSQLPgIAMConnection(t *testing.T) {
	getCloudSQLPgVars(t)
	// service account email used for IAM should trim the suffix
	serviceAccountEmail := strings.TrimSuffix(tests.ServiceAccountEmail, ".gserviceaccount.com")

	noPassSourceConfig := map[string]any{
		"kind":     CloudSQLPostgresSourceKind,
		"project":  CloudSQLPostgresProject,
		"instance": CloudSQLPostgresInstance,
		"region":   CloudSQLPostgresRegion,
		"database": CloudSQLPostgresDatabase,
		"user":     serviceAccountEmail,
	}

	noUserSourceConfig := map[string]any{
		"kind":     CloudSQLPostgresSourceKind,
		"project":  CloudSQLPostgresProject,
		"instance": CloudSQLPostgresInstance,
		"region":   CloudSQLPostgresRegion,
		"database": CloudSQLPostgresDatabase,
		"password": "random",
	}

	noUserNoPassSourceConfig := map[string]any{
		"kind":     CloudSQLPostgresSourceKind,
		"project":  CloudSQLPostgresProject,
		"instance": CloudSQLPostgresInstance,
		"region":   CloudSQLPostgresRegion,
		"database": CloudSQLPostgresDatabase,
	}
	tcs := []struct {
		name         string
		sourceConfig map[string]any
		isErr        bool
	}{
		{
			name:         "no user no pass",
			sourceConfig: noUserNoPassSourceConfig,
			isErr:        false,
		},
		{
			name:         "no password",
			sourceConfig: noPassSourceConfig,
			isErr:        false,
		},
		{
			name:         "no user",
			sourceConfig: noUserSourceConfig,
			isErr:        true,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			err := tests.RunSourceConnectionTest(t, tc.sourceConfig, CloudSQLPostgresToolKind)
			if err != nil {
				if tc.isErr {
					return
				}
				t.Fatalf("Connection test failure: %s", err)
			}
			if tc.isErr {
				t.Fatalf("Expected error but test passed.")
			}
		})
	}
}
