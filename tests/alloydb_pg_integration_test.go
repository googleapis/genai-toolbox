//go:build integration && alloydb

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
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/alloydbconn"
	"github.com/google/uuid"
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

func TestAlloyDBToolEndpoints(t *testing.T) {
	sourceConfig := getAlloyDBPgVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	pool, err := initAlloyDBPgConnectionPool(ALLOYDB_POSTGRES_PROJECT, ALLOYDB_POSTGRES_REGION, ALLOYDB_POSTGRES_CLUSTER, ALLOYDB_POSTGRES_INSTANCE, "public", ALLOYDB_POSTGRES_USER, ALLOYDB_POSTGRES_PASS, ALLOYDB_POSTGRES_DATABASE)
	if err != nil {
		t.Fatalf("unable to create AlloyDB connection pool: %s", err)
	}

	// create table name with UUID
	tableNameParam := "param_table_" + strings.Replace(uuid.New().String(), "-", "", -1)
	tableNameAuth := "auth_table_" + strings.Replace(uuid.New().String(), "-", "", -1)

	// set up data for param tool
	create_statement1, insert_statement1, tool_statement1, params1 := GetPostgresSQLParamToolInfo(tableNameParam)
	teardownTable1 := SetupPostgresSQLTable(t, ctx, pool, create_statement1, insert_statement1, tableNameParam, params1)
	defer teardownTable1(t)

	// set up data for auth tool
	create_statement2, insert_statement2, tool_statement2, params2 := GetPostgresSQLAuthToolInfo(tableNameAuth)
	teardownTable2 := SetupPostgresSQLTable(t, ctx, pool, create_statement2, insert_statement2, tableNameAuth, params2)
	defer teardownTable2(t)

	// Write config into a file and pass it to command
	toolsFile := GetToolsConfig(sourceConfig, ALLOYDB_POSTGRES_TOOL_KIND, tool_statement1, tool_statement2)

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

	select_1_want := "[{\"?column?\":1}]"
	RunToolInvokeTest(t, select_1_want)
}

// Test connection to public IP
func TestAlloyDBPublicIpConnection(t *testing.T) {
	sourceConfig := getAlloyDBPgVars(t)
	sourceConfig["ipType"] = "public"
	RunSourceConnectionTest(t, sourceConfig, ALLOYDB_POSTGRES_TOOL_KIND)
}

// Test connection to private IP
func TestAlloyDBPrivateIpConnection(t *testing.T) {
	sourceConfig := getAlloyDBPgVars(t)
	sourceConfig["ipType"] = "private"
	RunSourceConnectionTest(t, sourceConfig, ALLOYDB_POSTGRES_TOOL_KIND)
}
