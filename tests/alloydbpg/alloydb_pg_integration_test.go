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

package alloydbpg

import (
	"context"
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/alloydbconn"
	"github.com/google/uuid"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	AlloyDBPostgresSourceKind = "alloydb-postgres"
	AlloyDBPostgresToolKind   = "postgres-sql"
	AlloyDBPostgresProject    = os.Getenv("ALLOYDB_POSTGRES_PROJECT")
	AlloyDBPostgresRegion     = os.Getenv("ALLOYDB_POSTGRES_REGION")
	AlloyDBPostgresCluster    = os.Getenv("ALLOYDB_POSTGRES_CLUSTER")
	AlloyDBPostgresInstance   = os.Getenv("ALLOYDB_POSTGRES_INSTANCE")
	AlloyDBPostgresDatabase   = os.Getenv("ALLOYDB_POSTGRES_DATABASE")
	AlloyDBPostgresUser       = os.Getenv("ALLOYDB_POSTGRES_USER")
	AlloyDBPostgresPass       = os.Getenv("ALLOYDB_POSTGRES_PASS")
)

func getAlloyDBPgVars(t *testing.T) map[string]any {
	switch "" {
	case AlloyDBPostgresProject:
		t.Fatal("'ALLOYDB_POSTGRES_PROJECT' not set")
	case AlloyDBPostgresRegion:
		t.Fatal("'ALLOYDB_POSTGRES_REGION' not set")
	case AlloyDBPostgresCluster:
		t.Fatal("'ALLOYDB_POSTGRES_CLUSTER' not set")
	case AlloyDBPostgresInstance:
		t.Fatal("'ALLOYDB_POSTGRES_INSTANCE' not set")
	case AlloyDBPostgresDatabase:
		t.Fatal("'ALLOYDB_POSTGRES_DATABASE' not set")
	case AlloyDBPostgresUser:
		t.Fatal("'ALLOYDB_POSTGRES_USER' not set")
	case AlloyDBPostgresPass:
		t.Fatal("'ALLOYDB_POSTGRES_PASS' not set")
	}
	return map[string]any{
		"kind":     AlloyDBPostgresSourceKind,
		"project":  AlloyDBPostgresProject,
		"cluster":  AlloyDBPostgresCluster,
		"instance": AlloyDBPostgresInstance,
		"region":   AlloyDBPostgresRegion,
		"database": AlloyDBPostgresDatabase,
		"user":     AlloyDBPostgresUser,
		"password": AlloyDBPostgresPass,
	}
}

// Copied over from  alloydb_pg.go
func getAlloyDBDialOpts(ipType string) ([]alloydbconn.DialOption, error) {
	switch strings.ToLower(ipType) {
	case "private":
		return []alloydbconn.DialOption{alloydbconn.WithPrivateIP()}, nil
	case "public":
		return []alloydbconn.DialOption{alloydbconn.WithPublicIP()}, nil
	default:
		return nil, fmt.Errorf("invalid ipType %s", ipType)
	}
}

// Copied over from  alloydb_pg.go
func initAlloyDBPgConnectionPool(project, region, cluster, instance, ipType, user, pass, dbname string) (*pgxpool.Pool, error) {
	// Configure the driver to connect to the database
	dsn := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", user, pass, dbname)
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("unable to parse connection uri: %w", err)
	}

	// Create a new dialer with options
	dialOpts, err := getAlloyDBDialOpts(ipType)
	if err != nil {
		return nil, err
	}
	d, err := alloydbconn.NewDialer(context.Background(), alloydbconn.WithDefaultDialOptions(dialOpts...))
	if err != nil {
		return nil, fmt.Errorf("unable to parse connection uri: %w", err)
	}

	// Tell the driver to use the AlloyDB Go Connector to create connections
	i := fmt.Sprintf("projects/%s/locations/%s/clusters/%s/instances/%s", project, region, cluster, instance)
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

func TestAlloyDBPgToolEndpoints(t *testing.T) {
	sourceConfig := getAlloyDBPgVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	pool, err := initAlloyDBPgConnectionPool(AlloyDBPostgresProject, AlloyDBPostgresRegion, AlloyDBPostgresCluster, AlloyDBPostgresInstance, "public", AlloyDBPostgresUser, AlloyDBPostgresPass, AlloyDBPostgresDatabase)
	if err != nil {
		t.Fatalf("unable to create AlloyDB connection pool: %s", err)
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
	toolsFile := tests.GetToolsConfig(sourceConfig, AlloyDBPostgresToolKind, paramToolStmt, idParamToolStmt, nameParamToolStmt, arrayToolStmt, authToolStmt)
	toolsFile = tests.AddExecuteSqlConfig(t, toolsFile, "postgres-execute-sql")
	tmplSelectCombined, tmplSelectFilterCombined := tests.GetPostgresSQLTmplToolStatement()
	toolsFile = tests.AddTemplateParamConfig(t, toolsFile, AlloyDBPostgresToolKind, tmplSelectCombined, tmplSelectFilterCombined, "")

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
	select1Want, failInvocationWant, createTableStatement, mcpSelect1Want := tests.GetPostgresWants()

	// Run tests
	tests.RunToolGetTest(t)
	tests.RunToolInvokeTest(t, select1Want)
	tests.RunMCPToolCallMethod(t, failInvocationWant, mcpSelect1Want)
	tests.RunExecuteSqlToolInvokeTest(t, createTableStatement, select1Want)
	tests.RunToolInvokeWithTemplateParameters(t, tableNameTemplateParam)

	// Run Postgres prebuilt tool tests
	tests.RunPostgresListTablesTest(t, tableNameParam, tableNameAuth, AlloyDBPostgresUser)
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
func TestAlloyDBPgIpConnection(t *testing.T) {
	sourceConfig := getAlloyDBPgVars(t)

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
			err := tests.RunSourceConnectionTest(t, sourceConfig, AlloyDBPostgresToolKind)
			if err != nil {
				t.Fatalf("Connection test failure: %s", err)
			}
		})
	}
}

// Test IAM connection
func TestAlloyDBPgIAMConnection(t *testing.T) {
	getAlloyDBPgVars(t)
	// service account email used for IAM should trim the suffix
	serviceAccountEmail := strings.TrimSuffix(tests.ServiceAccountEmail, ".gserviceaccount.com")

	noPassSourceConfig := map[string]any{
		"kind":     AlloyDBPostgresSourceKind,
		"project":  AlloyDBPostgresProject,
		"cluster":  AlloyDBPostgresCluster,
		"instance": AlloyDBPostgresInstance,
		"region":   AlloyDBPostgresRegion,
		"database": AlloyDBPostgresDatabase,
		"user":     serviceAccountEmail,
	}

	noUserSourceConfig := map[string]any{
		"kind":     AlloyDBPostgresSourceKind,
		"project":  AlloyDBPostgresProject,
		"cluster":  AlloyDBPostgresCluster,
		"instance": AlloyDBPostgresInstance,
		"region":   AlloyDBPostgresRegion,
		"database": AlloyDBPostgresDatabase,
		"password": "random",
	}

	noUserNoPassSourceConfig := map[string]any{
		"kind":     AlloyDBPostgresSourceKind,
		"project":  AlloyDBPostgresProject,
		"cluster":  AlloyDBPostgresCluster,
		"instance": AlloyDBPostgresInstance,
		"region":   AlloyDBPostgresRegion,
		"database": AlloyDBPostgresDatabase,
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
			err := tests.RunSourceConnectionTest(t, tc.sourceConfig, AlloyDBPostgresToolKind)
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
