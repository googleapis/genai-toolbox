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
	"reflect"
	"strings"
	"testing"

	bigqueryapi "cloud.google.com/go/bigquery"
	"cloud.google.com/go/spanner"
	database "cloud.google.com/go/spanner/admin/database/apiv1"
	"cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	"github.com/googleapis/genai-toolbox/internal/server/mcp"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
)

// SetupPostgresSQLTable creates and inserts data into a table of tool
// compatible with postgres-sql tool
func SetupPostgresSQLTable(t *testing.T, ctx context.Context, pool *pgxpool.Pool, create_statement, insert_statement, tableName string, params []any) func(*testing.T) {
	err := pool.Ping(ctx)
	if err != nil {
		t.Fatalf("unable to connect to test database: %s", err)
	}

	// Create table
	_, err = pool.Query(ctx, create_statement)
	if err != nil {
		t.Fatalf("unable to create test table %s: %s", tableName, err)
	}

	// Insert test data
	_, err = pool.Query(ctx, insert_statement, params...)
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

// SetupMsSQLTable creates and inserts data into a table of tool
// compatible with mssql-sql tool
func SetupMsSQLTable(t *testing.T, ctx context.Context, pool *sql.DB, create_statement, insert_statement, tableName string, params []any) func(*testing.T) {
	err := pool.PingContext(ctx)
	if err != nil {
		t.Fatalf("unable to connect to test database: %s", err)
	}

	// Create table
	_, err = pool.QueryContext(ctx, create_statement)
	if err != nil {
		t.Fatalf("unable to create test table %s: %s", tableName, err)
	}

	// Insert test data
	_, err = pool.QueryContext(ctx, insert_statement, params...)
	if err != nil {
		t.Fatalf("unable to insert test data: %s", err)
	}

	return func(t *testing.T) {
		// tear down test
		_, err = pool.ExecContext(ctx, fmt.Sprintf("DROP TABLE %s;", tableName))
		if err != nil {
			t.Errorf("Teardown failed: %s", err)
		}
	}
}

// SetupMySQLTable creates and inserts data into a table of tool
// compatible with mssql-sql tool
func SetupMySQLTable(t *testing.T, ctx context.Context, pool *sql.DB, create_statement, insert_statement, tableName string, params []any) func(*testing.T) {
	err := pool.PingContext(ctx)
	if err != nil {
		t.Fatalf("unable to connect to test database: %s", err)
	}

	// Create table
	_, err = pool.QueryContext(ctx, create_statement)
	if err != nil {
		t.Fatalf("unable to create test table %s: %s", tableName, err)
	}

	// Insert test data
	_, err = pool.QueryContext(ctx, insert_statement, params...)
	if err != nil {
		t.Fatalf("unable to insert test data: %s", err)
	}

	return func(t *testing.T) {
		// tear down test
		_, err = pool.ExecContext(ctx, fmt.Sprintf("DROP TABLE %s;", tableName))
		if err != nil {
			t.Errorf("Teardown failed: %s", err)
		}
	}
}

// SetupSpannerTable creates and inserts data into a table of tool
// compatible with spanner-sql tool
func SetupSpannerTable(t *testing.T, ctx context.Context, adminClient *database.DatabaseAdminClient, dataClient *spanner.Client, create_statement, insert_statement, tableName, dbString string, params map[string]any) func(*testing.T) {

	// Create table
	op, err := adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
		Database:   dbString,
		Statements: []string{create_statement},
	})
	if err != nil {
		t.Fatalf("unable to start create table operation %s: %s", tableName, err)
	}
	err = op.Wait(ctx)
	if err != nil {
		t.Fatalf("unable to create test table %s: %s", tableName, err)
	}

	// Insert test data
	_, err = dataClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.Statement{
			SQL:    insert_statement,
			Params: params,
		}
		_, err := txn.Update(ctx, stmt)
		return err
	})
	if err != nil {
		t.Fatalf("unable to insert test data: %s", err)
	}

	return func(t *testing.T) {
		// tear down test
		op, err = adminClient.UpdateDatabaseDdl(ctx, &databasepb.UpdateDatabaseDdlRequest{
			Database:   dbString,
			Statements: []string{fmt.Sprintf("DROP TABLE %s", tableName)},
		})
		if err != nil {
			t.Errorf("unable to start drop table operation: %s", err)
			return
		}

		err = op.Wait(ctx)
		if err != nil {
			t.Errorf("Teardown failed: %s", err)
		}
	}
}

func SetupBigQueryTable(t *testing.T, ctx context.Context, client *bigqueryapi.Client, create_statement, insert_statement, datasetName string, tableName string, params []bigqueryapi.QueryParameter) func(*testing.T) {
	// Create dataset
	dataset := client.Dataset(datasetName)
	_, err := dataset.Metadata(ctx)

	if err != nil {
		apiErr, ok := err.(*googleapi.Error)
		if !ok || apiErr.Code != 404 {
			t.Fatalf("Failed to check dataset %q existence: %v", datasetName, err)
		}
		metadataToCreate := &bigqueryapi.DatasetMetadata{Name: datasetName}
		if err := dataset.Create(ctx, metadataToCreate); err != nil {
			t.Fatalf("Failed to create dataset %q: %v", datasetName, err)
		}
	}

	// Create table
	createJob, err := client.Query(create_statement).Run(ctx)

	if err != nil {
		t.Fatalf("Failed to start create table job for %s: %v", tableName, err)
	}
	createStatus, err := createJob.Wait(ctx)
	if err != nil {
		t.Fatalf("Failed to wait for create table job for %s: %v", tableName, err)
	}
	if err := createStatus.Err(); err != nil {
		t.Fatalf("Create table job for %s failed: %v", tableName, err)
	}

	// Insert test data
	insertQuery := client.Query(insert_statement)
	insertQuery.Parameters = params
	insertJob, err := insertQuery.Run(ctx)
	if err != nil {
		t.Fatalf("Failed to start insert job for %s: %v", tableName, err)
	}
	insertStatus, err := insertJob.Wait(ctx)
	if err != nil {
		t.Fatalf("Failed to wait for insert job for %s: %v", tableName, err)
	}
	if err := insertStatus.Err(); err != nil {
		t.Fatalf("Insert job for %s failed: %v", tableName, err)
	}

	return func(t *testing.T) {
		// tear down table
		dropSQL := fmt.Sprintf("drop table %s", tableName)
		dropJob, err := client.Query(dropSQL).Run(ctx)
		if err != nil {
			t.Errorf("Failed to start drop table job for %s: %v", tableName, err)
			return
		}
		dropStatus, err := dropJob.Wait(ctx)
		if err != nil {
			t.Errorf("Failed to wait for drop table job for %s: %v", tableName, err)
			return
		}
		if err := dropStatus.Err(); err != nil {
			t.Errorf("Error dropping table %s: %v", tableName, err)
		}

		// tear down dataset
		datasetToTeardown := client.Dataset(datasetName)
		tablesIterator := datasetToTeardown.Tables(ctx)
		_, err = tablesIterator.Next()

		if err == iterator.Done {
			if err := datasetToTeardown.Delete(ctx); err != nil {
				t.Errorf("Failed to delete dataset %s: %v", datasetName, err)
			}
		} else if err != nil {
			t.Errorf("Failed to list tables in dataset %s to check emptiness: %v.", datasetName, err)
		}
	}
}

// RunToolGet runs the tool get endpoint
func RunToolGetTest(t *testing.T) {
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
					"description":  "Simple tool to test end to end functionality.",
					"parameters":   []any{},
					"authRequired": []any{},
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
}

// RunToolInvoke runs the tool invoke endpoint
func RunToolInvokeTest(t *testing.T, select_1_want string) {
	// Get ID token
	idToken, err := GetGoogleIdToken(ClientId)
	if err != nil {
		t.Fatalf("error getting Google ID token: %s", err)
	}

	// Test tool invoke endpoint
	invokeTcs := []struct {
		name          string
		api           string
		requestHeader map[string]string
		requestBody   io.Reader
		want          string
		isErr         bool
	}{
		{
			name:          "invoke my-simple-tool",
			api:           "http://127.0.0.1:5000/api/tool/my-simple-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			want:          select_1_want,
			isErr:         false,
		},
		{
			name:          "invoke my-param-tool",
			api:           "http://127.0.0.1:5000/api/tool/my-param-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"id": 3, "name": "Alice"}`)),
			want:          "[{\"id\":1,\"name\":\"Alice\"},{\"id\":3,\"name\":\"Sid\"}]",
			isErr:         false,
		},
		{
			name:          "Invoke my-tool without parameters",
			api:           "http://127.0.0.1:5000/api/tool/my-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			isErr:         true,
		},
		{
			name:          "Invoke my-tool with insufficient parameters",
			api:           "http://127.0.0.1:5000/api/tool/my-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"id": 1}`)),
			isErr:         true,
		},
		{
			name:          "Invoke my-auth-tool with auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			want:          "[{\"name\":\"Alice\"}]",
			isErr:         false,
		},
		{
			name:          "Invoke my-auth-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": "INVALID_TOKEN"},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			isErr:         true,
		},
		{
			name:          "Invoke my-auth-tool without auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			isErr:         true,
		},
		{
			name:          "Invoke my-auth-required-tool with auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-required-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			isErr:         false,
			want:          select_1_want,
		},
		{
			name:          "Invoke my-auth-required-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-required-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": "INVALID_TOKEN"},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			isErr:         true,
		},
		{
			name:          "Invoke my-auth-required-tool without auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			isErr:         true,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			// Send Tool invocation request
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

			// Check response body
			var body map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&body)
			if err != nil {
				t.Fatalf("error parsing response body")
			}

			got, ok := body["result"].(string)
			if !ok {
				t.Fatalf("unable to find result in response body")
			}

			// Remove `\` and `"` for string comparison
			got = strings.ReplaceAll(got, "\\", "")
			want := strings.ReplaceAll(tc.want, "\\", "")
			got = strings.ReplaceAll(got, "\"", "")
			want = strings.ReplaceAll(want, "\"", "")

			if got != want {
				t.Fatalf("unexpected value: got %q, want %q", got, tc.want)
			}
		})
	}
}

// RunMCPToolCallMethod runs the tool/call for mcp endpoint
func RunMCPToolCallMethod(t *testing.T, fail_invocation_want string) {
	// Test tool invoke endpoint
	invokeTcs := []struct {
		name          string
		api           string
		requestBody   mcp.JSONRPCRequest
		requestHeader map[string]string
		want          string
	}{
		{
			name:          "MCP Invoke my-param-tool",
			api:           "http://127.0.0.1:5000/mcp",
			requestHeader: map[string]string{},
			requestBody: mcp.JSONRPCRequest{
				Jsonrpc: "2.0",
				Id:      "my-param-tool",
				Request: mcp.Request{
					Method: "tools/call",
				},
				Params: map[string]any{
					"name": "my-param-tool",
					"arguments": map[string]any{
						"id":   int(3),
						"name": "Alice",
					},
				},
			},
			want: `{"jsonrpc":"2.0","id":"my-param-tool","result":{"content":[{"type":"text","text":"{\"id\":1,\"name\":\"Alice\"}"},{"type":"text","text":"{\"id\":3,\"name\":\"Sid\"}"}]}}`,
		},
		{
			name:          "MCP Invoke invalid tool",
			api:           "http://127.0.0.1:5000/mcp",
			requestHeader: map[string]string{},
			requestBody: mcp.JSONRPCRequest{
				Jsonrpc: "2.0",
				Id:      "invalid-tool",
				Request: mcp.Request{
					Method: "tools/call",
				},
				Params: map[string]any{
					"name":      "foo",
					"arguments": map[string]any{},
				},
			},
			want: `{"jsonrpc":"2.0","id":"invalid-tool","error":{"code":-32602,"message":"invalid tool name: tool with name \"foo\" does not exist"}}`,
		},
		{
			name:          "MCP Invoke my-param-tool without parameters",
			api:           "http://127.0.0.1:5000/mcp",
			requestHeader: map[string]string{},
			requestBody: mcp.JSONRPCRequest{
				Jsonrpc: "2.0",
				Id:      "invoke-without-parameter",
				Request: mcp.Request{
					Method: "tools/call",
				},
				Params: map[string]any{
					"name":      "my-param-tool",
					"arguments": map[string]any{},
				},
			},
			want: `{"jsonrpc":"2.0","id":"invoke-without-parameter","error":{"code":-32602,"message":"provided parameters were invalid: parameter id is required"}}`,
		},
		{
			name:          "MCP Invoke my-param-tool with insufficient parameters",
			api:           "http://127.0.0.1:5000/mcp",
			requestHeader: map[string]string{},
			requestBody: mcp.JSONRPCRequest{
				Jsonrpc: "2.0",
				Id:      "invoke-insufficient-parameter",
				Request: mcp.Request{
					Method: "tools/call",
				},
				Params: map[string]any{
					"name":      "my-param-tool",
					"arguments": map[string]any{"id": 1},
				},
			},
			want: `{"jsonrpc":"2.0","id":"invoke-insufficient-parameter","error":{"code":-32602,"message":"provided parameters were invalid: parameter name is required"}}`,
		},
		{
			name:          "MCP Invoke my-fail-tool",
			api:           "http://127.0.0.1:5000/mcp",
			requestHeader: map[string]string{},
			requestBody: mcp.JSONRPCRequest{
				Jsonrpc: "2.0",
				Id:      "invoke-fail-tool",
				Request: mcp.Request{
					Method: "tools/call",
				},
				Params: map[string]any{
					"name":      "my-fail-tool",
					"arguments": map[string]any{"id": 1},
				},
			},
			want: fail_invocation_want,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			reqMarshal, err := json.Marshal(tc.requestBody)
			if err != nil {
				t.Fatalf("unexpected error during marshaling of request body")
			}
			// Send Tool invocation request
			req, err := http.NewRequest(http.MethodPost, tc.api, bytes.NewBuffer(reqMarshal))
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
			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("unable to read request body: %s", err)
			}
			defer resp.Body.Close()
			got := string(bytes.TrimSpace(respBody))

			// Remove `\` and `"` for string comparison
			got = strings.ReplaceAll(got, "\\", "")
			want := strings.ReplaceAll(tc.want, "\\", "")
			got = strings.ReplaceAll(got, "\"", "")
			want = strings.ReplaceAll(want, "\"", "")

			if !strings.Contains(got, want) {
				t.Fatalf("Expected substring not found:\ngot:  %q\nwant: %q (to be contained within got)", got, want)
			}
		})
	}
}
