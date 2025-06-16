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
	ALLOYDB_POSTGRES_SOURCE_KIND = "alloydb-postgres"
	ALLOYDB_POSTGRES_TOOL_KIND   = "postgres-sql"
	ALLOYDB_POSTGRES_PROJECT     = os.Getenv("ALLOYDB_POSTGRES_PROJECT")
	ALLOYDB_POSTGRES_REGION      = os.Getenv("ALLOYDB_POSTGRES_REGION")
	ALLOYDB_POSTGRES_CLUSTER     = os.Getenv("ALLOYDB_POSTGRES_CLUSTER")
	ALLOYDB_POSTGRES_INSTANCE    = os.Getenv("ALLOYDB_POSTGRES_INSTANCE")
	ALLOYDB_POSTGRES_DATABASE    = os.Getenv("ALLOYDB_POSTGRES_DATABASE")
	ALLOYDB_POSTGRES_USER        = os.Getenv("ALLOYDB_POSTGRES_USER")
	ALLOYDB_POSTGRES_PASS        = os.Getenv("ALLOYDB_POSTGRES_PASS")
)

func getAlloyDBPgVars(t *testing.T) map[string]any {
	switch "" {
	case ALLOYDB_POSTGRES_PROJECT:
		t.Fatal("'ALLOYDB_POSTGRES_PROJECT' not set")
	case ALLOYDB_POSTGRES_REGION:
		t.Fatal("'ALLOYDB_POSTGRES_REGION' not set")
	case ALLOYDB_POSTGRES_CLUSTER:
		t.Fatal("'ALLOYDB_POSTGRES_CLUSTER' not set")
	case ALLOYDB_POSTGRES_INSTANCE:
		t.Fatal("'ALLOYDB_POSTGRES_INSTANCE' not set")
	case ALLOYDB_POSTGRES_DATABASE:
		t.Fatal("'ALLOYDB_POSTGRES_DATABASE' not set")
	case ALLOYDB_POSTGRES_USER:
		t.Fatal("'ALLOYDB_POSTGRES_USER' not set")
	case ALLOYDB_POSTGRES_PASS:
		t.Fatal("'ALLOYDB_POSTGRES_PASS' not set")
	}
	return map[string]any{
		"kind":     ALLOYDB_POSTGRES_SOURCE_KIND,
		"project":  ALLOYDB_POSTGRES_PROJECT,
		"cluster":  ALLOYDB_POSTGRES_CLUSTER,
		"instance": ALLOYDB_POSTGRES_INSTANCE,
		"region":   ALLOYDB_POSTGRES_REGION,
		"database": ALLOYDB_POSTGRES_DATABASE,
		"user":     ALLOYDB_POSTGRES_USER,
		"password": ALLOYDB_POSTGRES_PASS,
	}
}

// Copied over from  alloydb_pg.go
func getAlloyDBDialOpts(ip_type string) ([]alloydbconn.DialOption, error) {
	switch strings.ToLower(ip_type) {
	case "private":
		return []alloydbconn.DialOption{alloydbconn.WithPrivateIP()}, nil
	case "public":
		return []alloydbconn.DialOption{alloydbconn.WithPublicIP()}, nil
	default:
		return nil, fmt.Errorf("invalid ip_type %s", ip_type)
	}
}

// Copied over from  alloydb_pg.go
func initAlloyDBPgConnectionPool(project, region, cluster, instance, ip_type, user, pass, dbname string) (*pgxpool.Pool, error) {
	// Configure the driver to connect to the database
	dsn := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", user, pass, dbname)
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("unable to parse connection uri: %w", err)
	}

	// Create a new dialer with options
	dialOpts, err := getAlloyDBDialOpts(ip_type)
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

	pool, err := initAlloyDBPgConnectionPool(ALLOYDB_POSTGRES_PROJECT, ALLOYDB_POSTGRES_REGION, ALLOYDB_POSTGRES_CLUSTER, ALLOYDB_POSTGRES_INSTANCE, "public", ALLOYDB_POSTGRES_USER, ALLOYDB_POSTGRES_PASS, ALLOYDB_POSTGRES_DATABASE)
	if err != nil {
		t.Fatalf("unable to create AlloyDB connection pool: %s", err)
	}

	// create table name with UUID
	tableNameParam := "param_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableNameAuth := "auth_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableNameTemplateParam := "template_param_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")

	// set up data for param tool
	create_statement1, insert_statement1, tool_statement1, params1 := tests.GetPostgresSQLParamToolInfo(tableNameParam)
	teardownTable1 := tests.SetupPostgresSQLTable(t, ctx, pool, create_statement1, insert_statement1, tableNameParam, params1)
	defer teardownTable1(t)

	// set up data for auth tool
	create_statement2, insert_statement2, tool_statement2, params2 := tests.GetPostgresSQLAuthToolInfo(tableNameAuth)
	teardownTable2 := tests.SetupPostgresSQLTable(t, ctx, pool, create_statement2, insert_statement2, tableNameAuth, params2)
	defer teardownTable2(t)

	// Write config into a file and pass it to command
	toolsFile := tests.GetToolsConfig(sourceConfig, ALLOYDB_POSTGRES_TOOL_KIND, tool_statement1, tool_statement2)
	toolsFile = tests.AddPgExecuteSqlConfig(t, toolsFile)
	tmplSelectCombined, tmplSelectFilterCombined := tests.GetPostgresSQLTmplToolStatement()
	toolsFile = tests.AddTemplateParamConfig(t, toolsFile, ALLOYDB_POSTGRES_TOOL_KIND, tmplSelectCombined, tmplSelectFilterCombined, "")

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

	select1Want, failInvocationWant, createTableStatement := tests.GetPostgresWants()
	invokeParamWant, mcpInvokeParamWant := tests.GetNonSpannerInvokeParamWant()
	tests.RunToolInvokeTest(t, select1Want, invokeParamWant)
	tests.RunExecuteSqlToolInvokeTest(t, createTableStatement, select1Want)
	tests.RunMCPToolCallMethod(t, mcpInvokeParamWant, failInvocationWant)
	tests.RunToolInvokeWithTemplateParameters(t, tableNameTemplateParam, tests.NewTemplateParameterTestConfig())
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
			err := tests.RunSourceConnectionTest(t, sourceConfig, ALLOYDB_POSTGRES_TOOL_KIND)
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
	serviceAccountEmail := strings.TrimSuffix(tests.SERVICE_ACCOUNT_EMAIL, ".gserviceaccount.com")

	noPassSourceConfig := map[string]any{
		"kind":     ALLOYDB_POSTGRES_SOURCE_KIND,
		"project":  ALLOYDB_POSTGRES_PROJECT,
		"cluster":  ALLOYDB_POSTGRES_CLUSTER,
		"instance": ALLOYDB_POSTGRES_INSTANCE,
		"region":   ALLOYDB_POSTGRES_REGION,
		"database": ALLOYDB_POSTGRES_DATABASE,
		"user":     serviceAccountEmail,
	}

	noUserSourceConfig := map[string]any{
		"kind":     ALLOYDB_POSTGRES_SOURCE_KIND,
		"project":  ALLOYDB_POSTGRES_PROJECT,
		"cluster":  ALLOYDB_POSTGRES_CLUSTER,
		"instance": ALLOYDB_POSTGRES_INSTANCE,
		"region":   ALLOYDB_POSTGRES_REGION,
		"database": ALLOYDB_POSTGRES_DATABASE,
		"password": "random",
	}

	noUserNoPassSourceConfig := map[string]any{
		"kind":     ALLOYDB_POSTGRES_SOURCE_KIND,
		"project":  ALLOYDB_POSTGRES_PROJECT,
		"cluster":  ALLOYDB_POSTGRES_CLUSTER,
		"instance": ALLOYDB_POSTGRES_INSTANCE,
		"region":   ALLOYDB_POSTGRES_REGION,
		"database": ALLOYDB_POSTGRES_DATABASE,
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
			err := tests.RunSourceConnectionTest(t, tc.sourceConfig, ALLOYDB_POSTGRES_TOOL_KIND)
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
