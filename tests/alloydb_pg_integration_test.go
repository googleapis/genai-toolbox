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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/alloydbconn"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ALLOYDB_POSTGRES_PROJECT  = os.Getenv("ALLOYDB_POSTGRES_PROJECT")
	ALLOYDB_POSTGRES_REGION   = os.Getenv("ALLOYDB_POSTGRES_REGION")
	ALLOYDB_POSTGRES_CLUSTER  = os.Getenv("ALLOYDB_POSTGRES_CLUSTER")
	ALLOYDB_POSTGRES_INSTANCE = os.Getenv("ALLOYDB_POSTGRES_INSTANCE")
	ALLOYDB_POSTGRES_DATABASE = os.Getenv("ALLOYDB_POSTGRES_DATABASE")
	ALLOYDB_POSTGRES_USER     = os.Getenv("ALLOYDB_POSTGRES_USER")
	ALLOYDB_POSTGRES_PASS     = os.Getenv("ALLOYDB_POSTGRES_PASS")
)

func requireAlloyDBPgVars(t *testing.T) map[string]any {
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
		"kind":     "alloydb-postgres",
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

func TestAlloyDBSimpleToolEndpoints(t *testing.T) {
	sourceConfig := requireAlloyDBPgVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	// Write config into a file and pass it to command
	toolsFile := map[string]any{
		"sources": map[string]any{
			"my-pg-instance": sourceConfig,
		},
		"tools": map[string]any{
			"my-simple-tool": map[string]any{
				"kind":        "postgres-sql",
				"source":      "my-pg-instance",
				"description": "Simple tool to test end to end functionality.",
				"statement":   "SELECT 1;",
			},
		},
	}
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

	// Test tool get endpoint
	tcs := []struct {
		name string
		api  string
		want map[string]any
	}{
		{
			name: "get my-simple-tool",
			api:  "http://127.0.0.1:5000/api/tool/my-simple-tool/",
			want: map[string]any{
				"my-simple-tool": map[string]any{
					"description": "Simple tool to test end to end functionality.",
					"parameters":  []any{},
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := http.Get(tc.api)
			if err != nil {
				t.Fatalf("error when sending a request: %s", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != 200 {
				t.Fatalf("response status code is not 200")
			}

			var body map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&body)
			if err != nil {
				t.Fatalf("error parsing response body")
			}

			got, ok := body["tools"]
			if !ok {
				t.Fatalf("unable to find tools in response body")
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}

	// Test tool invoke endpoint
	invokeTcs := []struct {
		name        string
		api         string
		requestBody io.Reader
		want        string
	}{
		{
			name:        "invoke my-simple-tool",
			api:         "http://127.0.0.1:5000/api/tool/my-simple-tool/invoke",
			requestBody: bytes.NewBuffer([]byte(`{}`)),
			want:        "Stub tool call for \"my-simple-tool\"! Parameters parsed: [] \n Output: [%!s(int32=1)]",
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := http.Post(tc.api, "application/json", tc.requestBody)
			if err != nil {
				t.Fatalf("error when sending a request: %s", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != 200 {
				t.Fatalf("response status code is not 200")
			}

			var body map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&body)
			if err != nil {
				t.Fatalf("error parsing response body")
			}
			got, ok := body["result"].(string)
			if !ok {
				t.Fatalf("unable to find result in response body")
			}

			if got != tc.want {
				t.Fatalf("unexpected value: got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestPublicIpConnection(t *testing.T) {
	// Test connecting to an AlloyDB source's public IP
	sourceConfig := requireAlloyDBPgVars(t)
	sourceConfig["ipType"] = "public"
	RunSourceConnectionTest(t, sourceConfig, "postgres-sql")
}

func TestPrivateIpConnection(t *testing.T) {
	// Test connecting to an AlloyDB source's private IP
	sourceConfig := requireAlloyDBPgVars(t)
	sourceConfig["ipType"] = "private"
	RunSourceConnectionTest(t, sourceConfig, "postgres-sql")
}

// Set up tool calling with parameters test table
func setupParamTest(t *testing.T, ctx context.Context, tableName string) func(*testing.T) {
	// Set up Tool invocation with parameters test
	pool, err := initAlloyDBPgConnectionPool(ALLOYDB_POSTGRES_PROJECT, ALLOYDB_POSTGRES_REGION, ALLOYDB_POSTGRES_CLUSTER, ALLOYDB_POSTGRES_INSTANCE, "public", ALLOYDB_POSTGRES_USER, ALLOYDB_POSTGRES_PASS, ALLOYDB_POSTGRES_DATABASE)
	if err != nil {
		t.Fatalf("unable to create AlloyDB connection pool: %s", err)
	}

	err = pool.Ping(ctx)
	if err != nil {
		t.Fatalf("unable to connect to test database: %s", err)
	}

	_, err = pool.Query(ctx, fmt.Sprintf(`
		CREATE TABLE %s (
			id SERIAL PRIMARY KEY,
			name TEXT
		);
	`, tableName))
	if err != nil {
		t.Fatalf("unable to create test table %s: %s", tableName, err)
	}

	// Insert test data
	statement := fmt.Sprintf(`
		INSERT INTO %s (name)
		VALUES ($1), ($2), ($3);
	`, tableName)

	params := []any{"Alice", "Jane", "Sid"}
	_, err = pool.Query(ctx, statement, params...)
	if err != nil {
		t.Fatalf("unable to insert test data: %s", err)
	}

	return func(t *testing.T) {
		// tear down test
		_, err = pool.Exec(ctx, fmt.Sprintf("DROP TABLE %s;", tableName))
		if err != nil {
			t.Errorf("Teardown failed: %s", err)
		}
	}
}

func TestToolInvocationWithParams(t *testing.T) {
	// create test configs
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	// create source config
	sourceConfig := requireAlloyDBPgVars(t)

	// create table name with UUID
	tableName := "param_test_table_" + strings.Replace(uuid.New().String(), "-", "", -1)

	// test setup function reterns teardown function
	teardownTest := setupParamTest(t, ctx, tableName)
	defer teardownTest(t)

	// call generic invocation test helper
	RunToolInvocationWithParamsTest(t, sourceConfig, "postgres-sql", tableName)
}

// Set up auth test database table
func setupAlloyDBAuthTest(t *testing.T, ctx context.Context, tableName string) func(*testing.T) {
	// set up db connection pool
	pool, err := initAlloyDBPgConnectionPool(ALLOYDB_POSTGRES_PROJECT, ALLOYDB_POSTGRES_REGION, ALLOYDB_POSTGRES_CLUSTER, ALLOYDB_POSTGRES_INSTANCE, "public", ALLOYDB_POSTGRES_USER, ALLOYDB_POSTGRES_PASS, ALLOYDB_POSTGRES_DATABASE)
	if err != nil {
		t.Fatalf("unable to create AlloyDB connection pool: %s", err)
	}

	err = pool.Ping(ctx)
	if err != nil {
		t.Fatalf("unable to connect to test database: %s", err)
	}

	_, err = pool.Query(ctx, fmt.Sprintf(`
		CREATE TABLE %s (
			id SERIAL PRIMARY KEY,
			name TEXT
		);
	`, tableName))
	if err != nil {
		t.Fatalf("unable to create test table %s: %s", tableName, err)
	}

	// Insert test data
	statement := fmt.Sprintf(`
		INSERT INTO %s (name, email) 
		VALUES ($1, $2), ($3, $4)
	`, tableName)
	params := []any{"Alice", SERVICE_ACCOUNT_EMAIL, "Jane", "janedoe@gmail.com"}
	_, err = pool.Query(ctx, statement, params...)
	if err != nil {
		t.Fatalf("unable to insert test data: %s", err)
	}

	return func(t *testing.T) {
		// tear down test
		_, err = pool.Exec(ctx, fmt.Sprintf("DROP TABLE %s;", tableName))
		if err != nil {
			t.Errorf("Teardown failed: %s", err)
		}
	}
}

func TestAlloyDBGoogleAuthenticatedParameter(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	// create test configs
	sourceConfig := requireAlloyDBPgVars(t)

	// create table name with UUID
	tableName := "auth_table_" + strings.Replace(uuid.New().String(), "-", "", -1)

	// test setup function returns teardown funtion
	teardownTest := setupAlloyDBAuthTest(t, ctx, tableName)
	defer teardownTest(t)

	// call generic auth test helper
	RunGoogleAuthenticatedParameterTest(t, sourceConfig, "postgres-sql", tableName)

}

func TestAlloyDBAuthRequiredToolInvocation(t *testing.T) {
	// create test configs
	sourceConfig := requireAlloyDBPgVars(t)

	// call generic auth test helper
	RunAuthRequiredToolInvocationTest(t, sourceConfig, "postgres-sql")

}
