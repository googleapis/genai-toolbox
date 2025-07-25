package cassandra

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"testing"
	"time"

	"github.com/gocql/gocql"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"
)

var (
	CassandraSourceKind = "cassandra"
	CassandraToolKind   = "cassandra-cql"
	Hosts               = []string{}
	PORT                = 9042
	tableName           = "example_keyspace.users"
)

func initCassandraSession() (*gocql.Session, error) {
	// Configure cluster connection
	cluster := gocql.NewCluster("localhost")
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
		INSERT INTO example_keyspace.users (id, name, age, is_active, created_at)
		VALUES (?, ?, ?, ?, ?)`,
		"1", "Alice", 25, true, dayAgo,
	).Exec()
	if err != nil {
		return nil, fmt.Errorf("Failed to insert user: %v", err)
	}
	err = session.Query(`
		INSERT INTO example_keyspace.users (id, name, age, is_active, created_at)
		VALUES (?, ?, ?, ?, ?)`,
		"2", "Jane", 30, false, twelveHoursAgo,
	).Exec()
	if err != nil {
		return nil, fmt.Errorf("Failed to insert user: %v", err)
	}
	err = session.Query(`
		INSERT INTO example_keyspace.users (id, name, age, is_active, created_at)
		VALUES (?, ?, ?, ?, ?)`,
		"3", "Sid", 0, true, fixedTime,
	).Exec()
	if err != nil {
		return nil, fmt.Errorf("Failed to insert user: %v", err)
	}
	err = session.Query(`
		INSERT INTO example_keyspace.users (id, name, age, is_active, created_at)
		VALUES (?, ?, ?, ?, ?)`,
		"4", nil, 40, false, fixedTime,
	).Exec()
	if err != nil {
		return nil, fmt.Errorf("Failed to insert user: %v", err)
	}

	return session, nil
}

func getCassandraVars(t *testing.T) map[string]any {
	return map[string]any{
		"kind": CassandraSourceKind,
		"host": Hosts,
	}
}

func TestCassandra(t *testing.T) {
	session, err := initCassandraSession()
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	defer session.Query(fmt.Sprintf("drop table %s", tableName)).Exec()
	sourceConfig := getCassandraVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string
	byIdStmt, nonExistentIdStmt, nullNameStmt, ageFilterStmt, isActiveStmt, createdAtStmt, selectAllStmt, limitStmt, countAllStmt := createParamToolInfo()
	toolsFile := getToolsConfig(sourceConfig, CassandraToolKind, byIdStmt, nonExistentIdStmt, nullNameStmt, ageFilterStmt, isActiveStmt, createdAtStmt, selectAllStmt, limitStmt, countAllStmt)
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
	selectByIdWant, invokeNonExistentIdWant, invokeNullNameWant, ageFilterWant, isActiveWant, selectAllWant, limitWant, countAllWant := getCassandraWants()
	runToolInvokeTest(t, selectByIdWant, invokeNonExistentIdWant, invokeNullNameWant, ageFilterWant, isActiveWant, selectAllWant, limitWant, countAllWant)
}

func createParamToolInfo() (string, string, string, string, string, string, string, string, string) {
	byIdStmt := fmt.Sprintf("SELECT id, name, age, is_active, created_at FROM %s WHERE id = ?;", tableName)
	nonExistentIdStmt := fmt.Sprintf("SELECT id FROM %s WHERE id = ?;", tableName)
	nullNameStmt := fmt.Sprintf("SELECT id, name FROM %s WHERE id = ?;", tableName)
	ageFilterStmt := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE age > ? ALLOW FILTERING;", tableName)
	isActiveStmt := fmt.Sprintf("SELECT id FROM %s WHERE is_active = ? ALLOW FILTERING;", tableName)
	createdAtStmt := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE created_at >= ? ALLOW FILTERING;", tableName)
	selectAllStmt := fmt.Sprintf("SELECT id, name, age, is_active, created_at FROM %s;", tableName)
	limitStmt := fmt.Sprintf("SELECT id FROM %s LIMIT ?;", tableName)
	countAllStmt := fmt.Sprintf("SELECT COUNT(*) FROM %s;", tableName)
	return byIdStmt, nonExistentIdStmt, nullNameStmt, ageFilterStmt, isActiveStmt, createdAtStmt, selectAllStmt, limitStmt, countAllStmt
}

func getCassandraWants() (string, string, string, string, string, string, string, string) {
	// Use fixed timestamps matching initCassandraSession
	fixedTime, _ := time.Parse(time.RFC3339, "2025-07-25T12:00:00Z")
	dayAgo := fixedTime.Add(-24 * time.Hour).Format(time.RFC3339)
	twelveHoursAgo := fixedTime.Add(-12 * time.Hour).Format(time.RFC3339)
	fixedTimeStr := fixedTime.Format(time.RFC3339)

	selectByIdWant := fmt.Sprintf(`[{"age":25,"created_at":"%s","id":"1","is_active":true,"name":"Alice"}]`, dayAgo)
	invokeNonExistentIdWant := "null"
	invokeNullNameWant := `[{"id":"4","name":""}]`
	ageFilterWant := `[{"count":3}]`
	isActiveWant := `[{"id":"3"},{"id":"1"}]`
	selectAllWant := fmt.Sprintf(`[{"age":40,"created_at":"%s","id":"4","is_active":false,"name":""},{"age":0,"created_at":"%s","id":"3","is_active":true,"name":"Sid"},{"age":30,"created_at":"%s","id":"2","is_active":false,"name":"Jane"},{"age":25,"created_at":"%s","id":"1","is_active":true,"name":"Alice"}]`, fixedTimeStr, fixedTimeStr, twelveHoursAgo, dayAgo)
	limitWant := `[{"id":"4"},{"id":"3"}]`
	countAllWant := `[{"count":4}]`
	return selectByIdWant, invokeNonExistentIdWant, invokeNullNameWant, ageFilterWant, isActiveWant, selectAllWant, limitWant, countAllWant
}

func runToolInvokeTest(t *testing.T, selectByIdWant, invokeNonExistentIdWant, invokeNullNameWant, ageFilterWant, isActiveWant, selectAllWant, limitWant, countAllWant string) {
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
			name:          "invoke non-existent-id",
			api:           "http://127.0.0.1:5000/api/tool/non-existent-id/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"id": "999"}`)),
			want:          invokeNonExistentIdWant,
			isErr:         false,
		},
		{
			name:          "invoke null-name",
			api:           "http://127.0.0.1:5000/api/tool/null-name/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"id": "4"}`)),
			want:          invokeNullNameWant,
			isErr:         false,
		},
		{
			name:          "invoke age-filter",
			api:           "http://127.0.0.1:5000/api/tool/age-filter/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"age": 20}`)),
			want:          ageFilterWant,
			isErr:         false,
		},
		{
			name:          "invoke is-active",
			api:           "http://127.0.0.1:5000/api/tool/is-active/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"is_active": true}`)),
			want:          isActiveWant,
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
			name:          "invoke limit",
			api:           "http://127.0.0.1:5000/api/tool/limit/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"limit": 2}`)),
			want:          limitWant,
			isErr:         false,
		},
		{
			name:          "invoke count-all",
			api:           "http://127.0.0.1:5000/api/tool/count-all/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			want:          countAllWant,
			isErr:         false,
		},
		{
			name:          "invoke select-by-id without parameters",
			api:           "http://127.0.0.1:5000/api/tool/select-by-id/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			isErr:         true,
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

func getToolsConfig(sourceConfig map[string]any, toolKind string, statements ...string) map[string]any {
	return map[string]any{
		"sources": map[string]any{
			"my-instance": sourceConfig,
		},
		"tools": map[string]any{
			"my-simple-tool": map[string]any{
				"kind":        toolKind,
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
				"statement":   "SELECT 1;",
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
			"non-existent-id": map[string]any{
				"kind":        toolKind,
				"source":      "my-instance",
				"description": "Select non-existent ID",
				"statement":   statements[1],
				"parameters": []map[string]any{
					{
						"name":        "id",
						"type":        "string",
						"description": "user ID",
						"required":    true,
					},
				},
			},
			"null-name": map[string]any{
				"kind":        toolKind,
				"source":      "my-instance",
				"description": "Select user with null name",
				"statement":   statements[2],
				"parameters": []map[string]any{
					{
						"name":        "id",
						"type":        "string",
						"description": "user ID",
						"required":    true,
					},
				},
			},
			"age-filter": map[string]any{
				"kind":        toolKind,
				"source":      "my-instance",
				"description": "Count users with age greater than specified",
				"statement":   statements[3],
				"parameters": []map[string]any{
					{
						"name":        "age",
						"type":        "integer",
						"description": "minimum age",
						"required":    true,
					},
				},
			},
			"is-active": map[string]any{
				"kind":        toolKind,
				"source":      "my-instance",
				"description": "Select users by active status",
				"statement":   statements[4],
				"parameters": []map[string]any{
					{
						"name":        "is_active",
						"type":        "boolean",
						"description": "active status",
						"required":    true,
					},
				},
			},
			"created-at": map[string]any{
				"kind":        toolKind,
				"source":      "my-instance",
				"description": "Count users created after specified time",
				"statement":   statements[5],
				"parameters": []map[string]any{
					{
						"name":        "created_at",
						"type":        "string",
						"description": "creation timestamp",
						"required":    true,
					},
				},
			},
			"select-all": map[string]any{
				"kind":        toolKind,
				"source":      "my-instance",
				"description": "Select all users",
				"statement":   statements[6],
			},
			"limit": map[string]any{
				"kind":        toolKind,
				"source":      "my-instance",
				"description": "Select limited number of user IDs",
				"statement":   statements[7],
				"parameters": []map[string]any{
					{
						"name":        "limit",
						"type":        "integer",
						"description": "number of rows to return",
						"required":    true,
					},
				},
			},
			"count-all": map[string]any{
				"kind":        toolKind,
				"source":      "my-instance",
				"description": "Count all users",
				"statement":   statements[8],
			},
		},
	}
}
