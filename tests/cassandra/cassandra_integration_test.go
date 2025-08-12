// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cassandra

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/gocql/gocql"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"
)

var (
	CassandraSourceKind = "cassandra"
	CassandraToolKind   = "cassandra-cql"
	Hosts               = os.Getenv("CASSANDRA_HOSTS") //Comma separated string with host IPs (default: []string{"localhost"})
	Port                = os.Getenv("CASSANDRA_PORT")  // 9042
	tableName           = "example_keyspace.users"
	Keyspace            = "example_keyspace"
	Username            = os.Getenv("CASSANDRA_USERNAME")
	Password            = os.Getenv("CASSANDRA_PASSWORD")
)

func initCassandraSession() (*gocql.Session, error) {
	hosts := strings.Split(Hosts, ",")
	// Configure cluster connection
	cluster := gocql.NewCluster(hosts...)
	cluster.Consistency = gocql.Quorum
	cluster.ProtoVersion = 4
	cluster.ConnectTimeout = 10 * time.Second
	cluster.NumConns = 2
	cluster.RetryPolicy = &gocql.ExponentialBackoffRetryPolicy{
		NumRetries: 3,
		Min:        200 * time.Millisecond,
		Max:        2 * time.Second,
	}

	// Create session
	session, err := cluster.CreateSession()
	if err != nil {
		return nil, fmt.Errorf("Failed to create session: %v", err)
	}

	// Create keyspace
	err = session.Query(`
		CREATE KEYSPACE IF NOT EXISTS example_keyspace
		WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1}
	`).Exec()
	if err != nil {
		return nil, fmt.Errorf("Failed to create keyspace: %v", err)
	}

	// Create table with additional columns
	err = session.Query(`
		CREATE TABLE IF NOT EXISTS example_keyspace.users (
			id text PRIMARY KEY,
			name text,
			email text,
			age int,
			is_active boolean,
			created_at timestamp
		)
	`).Exec()
	if err != nil {
		return nil, fmt.Errorf("Failed to create table: %v", err)
	}

	// Use fixed timestamps for reproducibility
	fixedTime, _ := time.Parse(time.RFC3339, "2025-07-25T12:00:00Z")
	dayAgo := fixedTime.Add(-24 * time.Hour)
	twelveHoursAgo := fixedTime.Add(-12 * time.Hour)

	// Insert minimal diverse data with fixed time.Time for timestamps
	err = session.Query(`
		INSERT INTO example_keyspace.users (id, name,email, age, is_active, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		"1", "Alice", tests.ServiceAccountEmail, 25, true, dayAgo,
	).Exec()
	if err != nil {
		return nil, fmt.Errorf("Failed to insert user: %v", err)
	}
	err = session.Query(`
		INSERT INTO example_keyspace.users (id, name,email, age, is_active, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		"2", "Jane", "janedoe@gmail.com", 30, false, twelveHoursAgo,
	).Exec()
	if err != nil {
		return nil, fmt.Errorf("Failed to insert user: %v", err)
	}
	err = session.Query(`
		INSERT INTO example_keyspace.users (id, name,email, age, is_active, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		"3", "Sid", "sid@gmail.com", 0, true, fixedTime,
	).Exec()
	if err != nil {
		return nil, fmt.Errorf("Failed to insert user: %v", err)
	}
	err = session.Query(`
		INSERT INTO example_keyspace.users (id, name,email, age, is_active, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		"4", nil, "a@gmail.com", 40, false, fixedTime,
	).Exec()
	if err != nil {
		return nil, fmt.Errorf("Failed to insert user: %v", err)
	}

	return session, nil
}

func getCassandraVars() map[string]any {
	return map[string]any{
		"kind":     CassandraSourceKind,
		"hosts":    strings.Split(Hosts, ","),
		"keyspace": Keyspace,
		"username": Username,
		"password": Password,
	}
}

func TestCassandra(t *testing.T) {
	session, err := initCassandraSession()
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	defer session.Query(fmt.Sprintf("drop table %s", tableName)).Exec()
	sourceConfig := getCassandraVars()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string
	byIdStmt, selectAllStmt, selectAllTemplateStmt, selectByIdTemplateStmt := createParamToolInfo()
	createAuthToolStmt, insertAuthToolStmt, selectAuthToolStmt := getCassandraAuthToolInfo("example_keyspace.auth_tool_table")
	toolsFile := getToolsConfig(sourceConfig, CassandraToolKind, byIdStmt, selectAllStmt, selectAllTemplateStmt, selectByIdTemplateStmt, createAuthToolStmt, insertAuthToolStmt, selectAuthToolStmt)
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
	selectByIdWant, selectAllWant, selectAllTemplateWant, selectByIdTemplateWant := getCassandraWants()
	runToolInvokeTest(t, selectByIdWant, selectAllWant)
	RunToolInvokeWithTemplateParameters(t, tableName, selectAllTemplateWant, selectByIdTemplateWant)
}

func createParamToolInfo() (string, string, string, string) {
	byIdStmt := fmt.Sprintf("SELECT id, name, age, is_active, created_at FROM %s WHERE id = ?;", tableName)
	selectAllStmt := fmt.Sprintf("SELECT id, name, age, is_active, created_at FROM %s;", tableName)
	selectAllTemplateStmt := "SELECT id, name, age, is_active, created_at FROM {{.tableName}};"
	selectByIdTemplateStmt := "SELECT id, name, age, is_active, created_at FROM {{.tableName}} WHERE id = ?;"
	return byIdStmt, selectAllStmt, selectAllTemplateStmt, selectByIdTemplateStmt
}

func getCassandraAuthToolInfo(tableName string) (string, string, string) {
	createStatement := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (name TEXT, email TEXT PRIMARY KEY);", tableName)
	insertStatement := fmt.Sprintf("INSERT INTO %s (name, email) VALUES (?, ?)", tableName)
	toolStatement := fmt.Sprintf("SELECT name FROM %s WHERE email = ?;", tableName)
	return createStatement, insertStatement, toolStatement
}

func getCassandraWants() (string, string, string, string) {
	fixedTime, _ := time.Parse(time.RFC3339, "2025-07-25T12:00:00Z")
	dayAgo := fixedTime.Add(-24 * time.Hour).Format(time.RFC3339)
	twelveHoursAgo := fixedTime.Add(-12 * time.Hour).Format(time.RFC3339)
	fixedTimeStr := fixedTime.Format(time.RFC3339)

	selectByIdWant := fmt.Sprintf(`[{"age":25,"created_at":"%s","id":"1","is_active":true,"name":"Alice"}]`, dayAgo)
	selectAllWant := fmt.Sprintf(`[{"age":40,"created_at":"%s","id":"4","is_active":false,"name":""},{"age":0,"created_at":"%s","id":"3","is_active":true,"name":"Sid"},{"age":30,"created_at":"%s","id":"2","is_active":false,"name":"Jane"},{"age":25,"created_at":"%s","id":"1","is_active":true,"name":"Alice"}]`, fixedTimeStr, fixedTimeStr, twelveHoursAgo, dayAgo)
	selectAllTemplateWant := selectAllWant
	selectByIdTemplateWant := selectByIdWant
	return selectByIdWant, selectAllWant, selectAllTemplateWant, selectByIdTemplateWant
}

func getToolsConfig(sourceConfig map[string]any, toolKind string, statements ...string) map[string]any {
	return map[string]any{
		"sources": map[string]any{
			"my-instance": sourceConfig,
		},
		"authServices": map[string]any{
			"my-google-auth": map[string]any{
				"kind":     "google",
				"clientId": tests.ClientId,
			},
		},
		"tools": map[string]any{
			"my-simple-tool": map[string]any{
				"kind":        toolKind,
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
				"statement":   "select id from example_keyspace.users where id='1';",
			},
			"select-by-id": map[string]any{
				"kind":        toolKind,
				"source":      "my-instance",
				"description": "Select user by ID",
				"statement":   statements[0],
				"parameters": []map[string]any{
					{
						"name":        "id",
						"type":        "string",
						"description": "user ID",
						"required":    true,
					},
				},
			},
			"select-all": map[string]any{
				"kind":        toolKind,
				"source":      "my-instance",
				"description": "Select all users",
				"statement":   statements[1],
			},
			"select-all-template": map[string]any{
				"kind":        toolKind,
				"source":      "my-instance",
				"description": "Select all users from table specified by template",
				"statement":   statements[2],
				"templateParameters": []map[string]any{
					{
						"name":        "tableName",
						"type":        "string",
						"description": "table name",
						"required":    true,
					},
				},
			},
			"select-by-id-template": map[string]any{
				"kind":        toolKind,
				"source":      "my-instance",
				"description": "Select user by ID from table specified by template",
				"statement":   statements[3],
				"parameters": []map[string]any{
					{
						"name":        "id",
						"type":        "string",
						"description": "user ID",
						"required":    true,
					},
				},
				"templateParameters": []map[string]any{
					{
						"name":        "tableName",
						"type":        "string",
						"description": "table name",
						"required":    true,
					},
				},
			},
			"create-table": map[string]any{
				"kind":        toolKind,
				"source":      "my-instance",
				"description": "Tool to test authenticated parameters.",
				// statement to auto-fill authenticated parameter
				"statement": statements[4],
			},
			"insert-into-table": map[string]any{
				"kind":        toolKind,
				"source":      "my-instance",
				"description": "Tool to test authenticated parameters.",
				// statement to auto-fill authenticated parameter
				"statement": statements[5],
				"parameters": []map[string]any{
					{
						"name":        "name",
						"type":        "string",
						"description": "user name",
					},
					{
						"name":        "email",
						"type":        "string",
						"description": "user email",
					},
				},
			},
			"my-auth-select-tool": map[string]any{
				"kind":        toolKind,
				"source":      "my-instance",
				"description": "Tool to test authenticated parameters.",
				// statement to auto-fill authenticated parameter
				"statement": statements[6],
				"parameters": []map[string]any{
					{
						"name":        "email",
						"type":        "string",
						"description": "user email",
						"authServices": []map[string]string{
							{
								"name":  "my-google-auth",
								"field": "email",
							},
						},
					},
				},
			},
		},
	}
}

func runToolInvokeTest(t *testing.T, selectByIdWant, selectAllWant string) {
	idToken, err := tests.GetGoogleIdToken(tests.ClientId)
	if err != nil {
		t.Fatalf("error getting Google ID token: %s", err)
	}
	invokeTcs := []struct {
		name          string
		api           string
		requestHeader map[string]string
		requestBody   io.Reader
		want          string
		isErr         bool
	}{
		{
			name:          "invoke select-by-id",
			api:           "http://127.0.0.1:5000/api/tool/select-by-id/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"id": "1"}`)),
			want:          selectByIdWant,
			isErr:         false,
		},
		{
			name:          "invoke select-all",
			api:           "http://127.0.0.1:5000/api/tool/select-all/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			want:          selectAllWant,
			isErr:         false,
		},
		{
			name:          "invoke select-by-id without parameters",
			api:           "http://127.0.0.1:5000/api/tool/select-by-id/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			isErr:         true,
		},
		{
			name:          "invoke auth-create-tool",
			api:           "http://127.0.0.1:5000/api/tool/create-table/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			want:          "null",
			isErr:         false,
		}, {
			name:          "invoke auth-insert",
			api:           "http://127.0.0.1:5000/api/tool/insert-into-table/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf(`{"name": "Alice", "email": "%s"}`, tests.ServiceAccountEmail))),
			want:          "null",
			isErr:         false,
		}, {
			name:          "invoke auth-select-by-email",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-select-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			want:          "[{\"name\":\"Alice\"}]",
			isErr:         false,
		},
	}

	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost, tc.api, tc.requestBody)
			if err != nil {
				t.Fatalf("unable to create request: %s", err)
			}
			req.Header.Add("Content-type", "application/json")
			for k, v := range tc.requestHeader {
				req.Header.Add(k, v)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("unable to send request: %s", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				if tc.isErr {
					return
				}
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

			if !tc.isErr && got != tc.want {
				t.Fatalf("unexpected value: got %q, want %q", got, tc.want)
			}
		})
	}
}

func RunToolInvokeWithTemplateParameters(t *testing.T, tableName, selectAllTemplateWant, selectByIdTemplateWant string) {
	invokeTcs := []struct {
		name          string
		api           string
		requestHeader map[string]string
		requestBody   io.Reader
		want          string
		isErr         bool
	}{
		{
			name:          "invoke select-all-template",
			api:           "http://127.0.0.1:5000/api/tool/select-all-template/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf(`{"tableName": "%s"}`, tableName))),
			want:          selectAllTemplateWant,
			isErr:         false,
		},
		{
			name:          "invoke select-by-id-template",
			api:           "http://127.0.0.1:5000/api/tool/select-by-id-template/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf(`{"id": "1", "tableName": "%s"}`, tableName))),
			want:          selectByIdTemplateWant,
			isErr:         false,
		},
	}

	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost, tc.api, tc.requestBody)
			if err != nil {
				t.Fatalf("unable to create request: %s", err)
			}
			req.Header.Add("Content-type", "application/json")
			for k, v := range tc.requestHeader {
				req.Header.Add(k, v)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("unable to send request: %s", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				if tc.isErr {
					return
				}
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

			if !tc.isErr && got != tc.want {
				t.Fatalf("unexpected value: got %q, want %q", got, tc.want)
			}
		})
	}
}
