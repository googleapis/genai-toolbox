//go:build integration && cloudsqlmssql

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
	"cloud.google.com/go/cloudsqlconn/sqlserver/mssql"
	"github.com/google/uuid"
)

var (
	CLOUD_SQL_MSSQL_PROJECT  = os.Getenv("CLOUD_SQL_MSSQL_PROJECT")
	CLOUD_SQL_MSSQL_REGION   = os.Getenv("CLOUD_SQL_MSSQL_REGION")
	CLOUD_SQL_MSSQL_INSTANCE = os.Getenv("CLOUD_SQL_MSSQL_INSTANCE")
	CLOUD_SQL_MSSQL_DATABASE = os.Getenv("CLOUD_SQL_MSSQL_DATABASE")
	CLOUD_SQL_MSSQL_IP       = os.Getenv("CLOUD_SQL_MSSQL_IP")
	CLOUD_SQL_MSSQL_USER     = os.Getenv("CLOUD_SQL_MSSQL_USER")
	CLOUD_SQL_MSSQL_PASS     = os.Getenv("CLOUD_SQL_MSSQL_PASS")
)

func requireCloudSQLMssqlVars(t *testing.T) map[string]any {
	switch "" {
	case CLOUD_SQL_MSSQL_PROJECT:
		t.Fatal("'CLOUD_SQL_MSSQL_PROJECT' not set")
	case CLOUD_SQL_MSSQL_REGION:
		t.Fatal("'CLOUD_SQL_MSSQL_REGION' not set")
	case CLOUD_SQL_MSSQL_INSTANCE:
		t.Fatal("'CLOUD_SQL_MSSQL_INSTANCE' not set")
	case CLOUD_SQL_MSSQL_IP:
		t.Fatal("'CLOUD_SQL_MSSQL_IP' not set")
	case CLOUD_SQL_MSSQL_DATABASE:
		t.Fatal("'CLOUD_SQL_MSSQL_DATABASE' not set")
	case CLOUD_SQL_MSSQL_USER:
		t.Fatal("'CLOUD_SQL_MSSQL_USER' not set")
	case CLOUD_SQL_MSSQL_PASS:
		t.Fatal("'CLOUD_SQL_MSSQL_PASS' not set")
	}

	return map[string]any{
		"kind":      "cloud-sql-mssql",
		"project":   CLOUD_SQL_MSSQL_PROJECT,
		"instance":  CLOUD_SQL_MSSQL_INSTANCE,
		"ipType":    "public",
		"ipAddress": CLOUD_SQL_MSSQL_IP,
		"region":    CLOUD_SQL_MSSQL_REGION,
		"database":  CLOUD_SQL_MSSQL_DATABASE,
		"user":      CLOUD_SQL_MSSQL_USER,
		"password":  CLOUD_SQL_MSSQL_PASS,
	}
}

func getDialOpts(ipType string) ([]cloudsqlconn.DialOption, error) {
	switch strings.ToLower(ipType) {
	case "private":
		return []cloudsqlconn.DialOption{cloudsqlconn.WithPrivateIP()}, nil
	case "public":
		return []cloudsqlconn.DialOption{cloudsqlconn.WithPublicIP()}, nil
	default:
		return nil, fmt.Errorf("invalid ipType %s", ipType)
	}
}

// Copied over from cloud_sql_mssql.go
func initCloudSQLMssqlConnection(project, region, instance, ipAddress, ipType, user, pass, dbname string) (*sql.DB, error) {
	// Create dsn
	dsn := fmt.Sprintf("sqlserver://%s:%s@%s?database=%s&cloudsql=%s:%s:%s", user, pass, ipAddress, dbname, project, region, instance)

	// Get dial options
	dialOpts, err := getDialOpts(ipType)
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
		dsn,
	)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func TestCloudSQLMssql(t *testing.T) {
	sourceConfig := requireCloudSQLMssqlVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	// Write config into a file and pass it to command
	toolsFile := map[string]any{
		"sources": map[string]any{
			"my-instance": sourceConfig,
		},
		"tools": map[string]any{
			"my-simple-tool": map[string]any{
				"kind":        "mssql-sql",
				"source":      "my-instance",
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

// Set up tool calling with parameters test table
func setupParamTest(t *testing.T, tableName string) (func(*testing.T), error) {
	// Set up Tool invocation with parameters test
	db, err := initCloudSQLMssqlConnection(CLOUD_SQL_MSSQL_PROJECT, CLOUD_SQL_MSSQL_REGION, CLOUD_SQL_MSSQL_INSTANCE, CLOUD_SQL_MSSQL_IP, "public", CLOUD_SQL_MSSQL_USER, CLOUD_SQL_MSSQL_PASS, CLOUD_SQL_MSSQL_DATABASE)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	_, err = db.Query(fmt.Sprintf(`
		CREATE TABLE %s (
			id INT IDENTITY(1,1) PRIMARY KEY,
			name VARCHAR(255),
		);
	`, tableName))
	if err != nil {
		return nil, err
	}

	// Insert test data
	statement := fmt.Sprintf(`
		INSERT INTO %s (name) 
		VALUES (@alice), (@jane), (@sid);
	`, tableName)
	params := []any{sql.Named("alice", "Alice"), sql.Named("jane", "Jane"), sql.Named("sid", "Sid")}
	_, err = db.Query(statement, params...)
	if err != nil {
		return nil, err
	}

	return func(t *testing.T) {
		// tear down test
		_, err := db.Exec(fmt.Sprintf(`DROP TABLE %s;`, tableName))
		if err != nil {
			t.Errorf("Teardown failed: %s", err)
		}
	}, nil
}

func TestToolInvocationWithParams(t *testing.T) {
	// create source config
	sourceConfig := requireCloudSQLMssqlVars(t)

	// create table name with UUID
	tableName := "param_test_table_" + strings.Replace(uuid.New().String(), "-", "", -1)

	// test setup function reterns teardown function
	teardownTest, err := setupParamTest(t, tableName)
	if err != nil {
		t.Fatalf("Unable to set up auth test: %s", err)
	}
	defer teardownTest(t)

	// call generic invocation test helper
	RunToolInvocationWithParamsTest(t, sourceConfig, "mssql-sql", tableName)
}

// Set up auth test database table
func setupCloudSQLMssqlAuthTest(t *testing.T, tableName string) (func(*testing.T), error) {
	// set up testt
	db, err := initCloudSQLMssqlConnection(CLOUD_SQL_MSSQL_PROJECT, CLOUD_SQL_MSSQL_REGION, CLOUD_SQL_MSSQL_INSTANCE, CLOUD_SQL_MSSQL_IP, "public", CLOUD_SQL_MSSQL_USER, CLOUD_SQL_MSSQL_PASS, CLOUD_SQL_MSSQL_DATABASE)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	_, err = db.Query(fmt.Sprintf(`
		CREATE TABLE %s (
			id INT IDENTITY(1,1) PRIMARY KEY,
			name VARCHAR(255),
			email VARCHAR(255)
		);
	`, tableName))
	if err != nil {
		return nil, err
	}

	// Insert test data
	statement := fmt.Sprintf(`
		INSERT INTO %s (name, email) 
		VALUES (@alice, @aliceemail), (@jane, @janeemail);
	`, tableName)
	params := []any{sql.Named("alice", "Alice"), sql.Named("aliceemail", SERVICE_ACCOUNT_EMAIL), sql.Named("jane", "Jane"), sql.Named("janeemail", "janedoe@gmail.com")}
	_, err = db.Query(statement, params...)
	if err != nil {
		return nil, err
	}

	return func(t *testing.T) {
		// tear down test
		_, err := db.Exec(fmt.Sprintf(`DROP TABLE %s;`, tableName))
		if err != nil {
			t.Errorf("Teardown failed: %s", err)
		}
	}, nil
}

func TestCloudSQLMssqlGoogleAuthenticatedParameter(t *testing.T) {
	// create test configs
	sourceConfig := requireCloudSQLMssqlVars(t)

	// create table name with UUID
	tableName := "auth_table_" + strings.Replace(uuid.New().String(), "-", "", -1)

	// test setup function reterns teardown function
	teardownTest, err := setupCloudSQLMssqlAuthTest(t, tableName)
	if err != nil {
		t.Fatalf("Unable to set up auth test: %s", err)
	}
	defer teardownTest(t)

	// call generic auth test helper
	RunGoogleAuthenticatedParameterTest(t, sourceConfig, "mssql-sql", tableName)

}

func TestCloudSQLMssqlAuthRequiredToolInvocation(t *testing.T) {
	// create test configs
	sourceConfig := requireCloudSQLMssqlVars(t)

	// call generic auth test helper
	RunAuthRequiredToolInvocationTest(t, sourceConfig, "mssql-sql")

}
