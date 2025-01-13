//go:build integration && cloudsql

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
	"encoding/json"
	"io"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"testing"
	"time"

	"github.com/googleapis/genai-toolbox/internal/sources/cloudsqlpg"
)

var (
	CLOUD_SQL_POSTGRES_PROJECT  = os.Getenv("CLOUD_SQL_POSTGRES_PROJECT")
	CLOUD_SQL_POSTGRES_REGION   = os.Getenv("CLOUD_SQL_POSTGRES_REGION")
	CLOUD_SQL_POSTGRES_INSTANCE = os.Getenv("CLOUD_SQL_POSTGRES_INSTANCE")
	CLOUD_SQL_POSTGRES_DATABASE = os.Getenv("CLOUD_SQL_POSTGRES_DATABASE")
	CLOUD_SQL_POSTGRES_USER     = os.Getenv("CLOUD_SQL_POSTGRES_USER")
	CLOUD_SQL_POSTGRES_PASS     = os.Getenv("CLOUD_SQL_POSTGRES_PASS")
)

func requireCloudSQLPgVars(t *testing.T) map[string]any {
	switch "" {
	case CLOUD_SQL_POSTGRES_PROJECT:
		t.Fatal("'CLOUD_SQL_POSTGRES_PROJECT' not set")
	case CLOUD_SQL_POSTGRES_REGION:
		t.Fatal("'CLOUD_SQL_POSTGRES_REGION' not set")
	case CLOUD_SQL_POSTGRES_INSTANCE:
		t.Fatal("'CLOUD_SQL_POSTGRES_INSTANCE' not set")
	case CLOUD_SQL_POSTGRES_DATABASE:
		t.Fatal("'CLOUD_SQL_POSTGRES_DATABASE' not set")
	case CLOUD_SQL_POSTGRES_USER:
		t.Fatal("'CLOUD_SQL_POSTGRES_USER' not set")
	case CLOUD_SQL_POSTGRES_PASS:
		t.Fatal("'CLOUD_SQL_POSTGRES_PASS' not set")
	}

	return map[string]any{
		"kind":     "cloud-sql-postgres",
		"project":  CLOUD_SQL_POSTGRES_PROJECT,
		"instance": CLOUD_SQL_POSTGRES_INSTANCE,
		"region":   CLOUD_SQL_POSTGRES_REGION,
		"database": CLOUD_SQL_POSTGRES_DATABASE,
		"user":     CLOUD_SQL_POSTGRES_USER,
		"password": CLOUD_SQL_POSTGRES_PASS,
	}
}

func TestCloudSQLPostgres(t *testing.T) {
	sourceConfig := requireCloudSQLPgVars(t)
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
func setupAuthTest(t *testing.T) func(*testing.T) {
	// set up testt
	pool, err := cloudsqlpg.InitCloudSQLPgConnectionPool(CLOUD_SQL_POSTGRES_PROJECT, CLOUD_SQL_POSTGRES_REGION, CLOUD_SQL_POSTGRES_INSTANCE, "public", CLOUD_SQL_POSTGRES_USER, CLOUD_SQL_POSTGRES_PASS, CLOUD_SQL_POSTGRES_DATABASE)
	if err != nil {
		t.Fatalf("unable to create Cloud SQL connection pool: %s", err)
	}

	err = pool.Ping(context.Background())
	if err != nil {
		t.Fatalf("unable to connect to test database: %s", err)
	}

	_, err = pool.Query(context.Background(), `
		CREATE TABLE auth_table (
			id SERIAL PRIMARY KEY,
			name TEXT,
			email TEXT
		);
	`)
	if err != nil {
		t.Fatalf("unable to create test table: %s", err)
	}

	// Insert test data
	statement := `
		INSERT INTO auth_table (name, email) 
		VALUES ($1, $2), ($3, $4)
	`
	params := []any{"Alice", SERVICE_ACCOUNT_EMAIL, "Jane", "janedoe@gmail.com"}
	_, err = pool.Query(context.Background(), statement, params...)
	if err != nil {
		t.Fatalf("unable to insert test data: %s", err)
	}

	return func(t *testing.T) {
		// tear down test
		pool.Exec(context.Background(), `DROP TABLE auth_table;`)
	}
}

func TestGoogleAuthenticatedParameter(t *testing.T) {
	// create test configs
	sourceConfig := requireCloudSQLPgVars(t)
	teardownTest := setupAuthTest(t)
	defer teardownTest(t)

	// call generic auth test helper
	RunGoogleAuthenticatedParameterTest(t, sourceConfig, "postgres-sql")

}

func TestAuthRequiredToolInvocation(t *testing.T) {
	// create test configs
	sourceConfig := requireCloudSQLPgVars(t)

	// call generic auth test helper
	RunAuthRequiredToolInvocationTest(t, sourceConfig, "postgres-sql")

}
