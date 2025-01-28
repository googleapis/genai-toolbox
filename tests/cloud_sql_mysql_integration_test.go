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
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
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
	CLOUD_SQL_MYSQL_PROJECT  = os.Getenv("CLOUD_SQL_MYSQL_PROJECT")
	CLOUD_SQL_MYSQL_REGION   = os.Getenv("CLOUD_SQL_MYSQL_REGION")
	CLOUD_SQL_MYSQL_INSTANCE = os.Getenv("CLOUD_SQL_MYSQL_INSTANCE")
	CLOUD_SQL_MYSQL_DATABASE = os.Getenv("CLOUD_SQL_MYSQL_DATABASE")
	CLOUD_SQL_MYSQL_USER     = os.Getenv("CLOUD_SQL_MYSQL_USER")
	CLOUD_SQL_MYSQL_PASS     = os.Getenv("CLOUD_SQL_MYSQL_PASS")
)

func requireCloudSQLMySQLVars(t *testing.T) map[string]any {
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
		"kind":     "cloud-sql-mysql",
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

func TestCloudSQLMySQL(t *testing.T) {
	sourceConfig := requireCloudSQLMySQLVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	// Write config into a file and pass it to command
	toolsFile := map[string]any{
		"sources": map[string]any{
			"my-mysql-instance": sourceConfig,
		},
		"tools": map[string]any{
			"my-simple-tool": map[string]any{
				"kind":        "mysql",
				"source":      "my-mysql-instance",
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
			if resp.StatusCode != http.StatusOK {
				bodyBytes, _ := io.ReadAll(resp.Body)
				t.Fatalf("response status code is not 200, got %d: %s", resp.StatusCode, string(bodyBytes))
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
			want:        "Stub tool call for \"my-simple-tool\"! Parameters parsed: [] \n Output: [%!s(int64=1)]",
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := http.Post(tc.api, "application/json", tc.requestBody)
			if err != nil {
				t.Fatalf("error when sending a request: %s", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				bodyBytes, _ := io.ReadAll(resp.Body)
				t.Fatalf("response status code is not 200, got %d: %s", resp.StatusCode, string(bodyBytes))
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

// Set up auth test database table
func setupCloudSQLMySQLAuthTest(t *testing.T, ctx context.Context, tableName string) func(*testing.T) {
	// set up testt
	pool, err := initCloudSQLMySQLConnectionPool(CLOUD_SQL_MYSQL_PROJECT, CLOUD_SQL_MYSQL_REGION, CLOUD_SQL_MYSQL_INSTANCE, "public", CLOUD_SQL_MYSQL_USER, CLOUD_SQL_MYSQL_PASS, CLOUD_SQL_MYSQL_DATABASE)
	if err != nil {
		t.Fatalf("unable to create Cloud SQL connection pool: %s", err)
	}

	err = pool.PingContext(ctx)
	if err != nil {
		t.Fatalf("unable to connect to test database: %s", err)
	}

	_, err = pool.QueryContext(ctx, fmt.Sprintf(`
		CREATE TABLE %s (
			id INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(255),
			email VARCHAR(255)
		);
	`, tableName))
	if err != nil {
		t.Fatalf("unable to create test table: %s", err)
	}

	// Insert test data
	statement := fmt.Sprintf(`
		INSERT INTO %s (name, email) 
		VALUES (?, ?), (?, ?)
	`, tableName)

	params := []any{"Alice", SERVICE_ACCOUNT_EMAIL, "Jane", "janedoe@gmail.com"}
	_, err = pool.QueryContext(ctx, statement, params...)
	if err != nil {
		t.Fatalf("unable to insert test data: %s", err)
	}

	return func(t *testing.T) {
		// tear down test
		_, err := pool.ExecContext(ctx, fmt.Sprintf(`DROP TABLE %s;`, tableName))
		if err != nil {
			t.Errorf("Teardown failed: %s", err)
		}
	}
}

func TestCloudSQLMySQLGoogleAuthenticatedParameter(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	// create test configs
	sourceConfig := requireCloudSQLMySQLVars(t)

	// create table name with UUID
	tableName := "auth_table_" + strings.Replace(uuid.New().String(), "-", "", -1)

	// test setup function reterns teardown function
	teardownTest := setupCloudSQLMySQLAuthTest(t, ctx, tableName)
	defer teardownTest(t)

	// call generic auth test helper
	RunGoogleAuthenticatedParameterTest(t, sourceConfig, "mysql", tableName)

}

func TestCloudSQLMySQLAuthRequiredToolInvocation(t *testing.T) {
	// create test configs
	sourceConfig := requireCloudSQLMySQLVars(t)

	// call generic auth test helper
	RunAuthRequiredToolInvocationTest(t, sourceConfig, "mysql")

}
