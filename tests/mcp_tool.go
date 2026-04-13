// Copyright 2026 Google LLC
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

	"github.com/google/go-cmp/cmp"

	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/googleapis/mcp-toolbox/internal/server/mcp/jsonrpc"
	v20251125 "github.com/googleapis/mcp-toolbox/internal/server/mcp/v20251125"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RunRequest is a helper function to send HTTP requests and return the response
// TODO: In Go, context should normally be the first parameter (after *testing.T).
// We put it at the end as a variadic argument here to make it optional and avoid
// breaking legacy tests that don't pass it. Consider refactoring this to the first
// parameter when legacy tests are removed.
func RunRequest(t *testing.T, method, url string, body io.Reader, headers map[string]string, ctx ...context.Context) (*http.Response, []byte) {
	rCtx := context.Background()
	if len(ctx) > 0 && ctx[0] != nil {
		rCtx = ctx[0]
	}

	req, err := http.NewRequestWithContext(rCtx, method, url, body)
	if err != nil {
		t.Fatalf("unable to create request: %s", err)
	}

	req.Header.Set("Content-type", "application/json")

	for k, v := range headers {
		req.Header.Set(k, v)
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
	return resp, respBody
}

// RunInitialize runs the initialize lifecycle for mcp to set up client-server connection
func RunInitialize(t *testing.T, protocolVersion string) string {
	url := "http://127.0.0.1:5000/mcp"

	initializeRequestBody := map[string]any{
		"jsonrpc": "2.0",
		"id":      "mcp-initialize",
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": protocolVersion,
		},
	}
	reqMarshal, err := json.Marshal(initializeRequestBody)
	if err != nil {
		t.Fatalf("unexpected error during marshaling of body")
	}

	resp, _ := RunRequest(t, http.MethodPost, url, bytes.NewBuffer(reqMarshal), nil)
	if resp.StatusCode != 200 {
		t.Fatalf("response status code is not 200")
	}

	if contentType := resp.Header.Get("Content-type"); contentType != "application/json" {
		t.Fatalf("unexpected content-type header: want %s, got %s", "application/json", contentType)
	}

	sessionId := resp.Header.Get("Mcp-Session-Id")

	header := map[string]string{}
	if sessionId != "" {
		header["Mcp-Session-Id"] = sessionId
	}

	initializeNotificationBody := map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
	}
	notiMarshal, err := json.Marshal(initializeNotificationBody)
	if err != nil {
		t.Fatalf("unexpected error during marshaling of notifications body")
	}

	_, _ = RunRequest(t, http.MethodPost, url, bytes.NewBuffer(notiMarshal), header)
	return sessionId
}

// NewMCPRequestHeader takes custom headers and appends headers required for MCP.
func NewMCPRequestHeader(t *testing.T, customHeaders map[string]string) map[string]string {
	headers := make(map[string]string)
	for k, v := range customHeaders {
		headers[k] = v
	}
	headers["Content-Type"] = "application/json"
	headers["MCP-Protocol-Version"] = v20251125.PROTOCOL_VERSION
	return headers
}

// InvokeMCPTool is a transparent, native JSON-RPC execution harness for tests.
func InvokeMCPTool(t *testing.T, ctx context.Context, toolName string, arguments map[string]any, requestHeader map[string]string) (int, *MCPCallToolResponse, error) {
	headers := NewMCPRequestHeader(t, requestHeader)

	req := NewMCPCallToolRequest(uuid.New().String(), toolName, arguments)
	reqBody, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("error marshalling request body: %v", err)
	}

	resp, respBody := RunRequest(t, http.MethodPost, "http://127.0.0.1:5000/mcp", bytes.NewBuffer(reqBody), headers, ctx)

	var mcpResp MCPCallToolResponse
	if err := json.Unmarshal(respBody, &mcpResp); err != nil {
		if resp.StatusCode != http.StatusOK {
			return resp.StatusCode, nil, fmt.Errorf("%s", string(respBody))
		}
		t.Fatalf("error parsing mcp response body: %v\nraw body: %s", err, string(respBody))
	}

	return resp.StatusCode, &mcpResp, nil
}

// GetMCPResultText safely extracts the text from content blocks, unmarshaling them if they are valid JSON.
//
// TODO: For tests that need to strictly validate the exact schema or structure of the output,
// consider avoiding this helper and instead unmarshal the raw JSON directly into expected Go structs for comparison.
func GetMCPResultText(t *testing.T, resp *MCPCallToolResponse) []any {
	if len(resp.Result.Content) == 0 {
		// Return an initialized empty slice instead of nil to avoid false failures with cmp.Diff
		return []any{}
	}

	var res []any
	for _, content := range resp.Result.Content {
		var item any
		if err := json.Unmarshal([]byte(content.Text), &item); err != nil {
			res = append(res, content.Text)
		} else {
			if slice, ok := item.([]any); ok {
				res = append(res, slice...)
			} else if item != nil {
				res = append(res, item)
			}
		}

	}
	if res == nil {
		// Return an initialized empty slice instead of nil to avoid false failures with cmp.Diff
		return []any{}
	}
	return res
}

// GetMCPToolsList is a JSON-RPC harness that fetches the tools/list registry.
func GetMCPToolsList(t *testing.T, ctx context.Context, requestHeader map[string]string) (int, []any, error) {
	headers := NewMCPRequestHeader(t, requestHeader)

	req := MCPListToolsRequest{
		Jsonrpc: jsonrpc.JSONRPC_VERSION,
		Id:      uuid.New().String(),
		Method:  v20251125.TOOLS_LIST,
	}
	reqBody, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("error marshalling tools/list request body: %v", err)
	}

	resp, respBody := RunRequest(t, http.MethodPost, "http://127.0.0.1:5000/mcp", bytes.NewBuffer(reqBody), headers, ctx)

	var mcpResp jsonrpc.JSONRPCResponse
	if err := json.Unmarshal(respBody, &mcpResp); err != nil {
		if resp.StatusCode != http.StatusOK {
			return resp.StatusCode, nil, fmt.Errorf("%s", string(respBody))
		}
		t.Fatalf("error parsing tools/list response: %v\nraw body: %s", err, string(respBody))
	}

	resultMap, ok := mcpResp.Result.(map[string]any)
	if !ok {
		t.Fatalf("tools/list result is not a map: %v", mcpResp.Result)
	}

	toolsList, ok := resultMap["tools"].([]any)
	if !ok {
		t.Fatalf("tools/list did not contain tools array: %v", resultMap)
	}

	return resp.StatusCode, toolsList, nil
}

// AssertMCPError asserts that the response contains an error covering the expected message.
func AssertMCPError(t *testing.T, mcpResp *MCPCallToolResponse, wantErrMsg string) {
	t.Helper()
	var errText string
	if mcpResp.Error != nil {
		errText = mcpResp.Error.Message
	} else if mcpResp.Result.IsError {
		for _, content := range mcpResp.Result.Content {
			if content.Type == "text" {
				errText += content.Text
			}
		}
	} else {
		t.Fatalf("expected error containing %q, but got success result: %v", wantErrMsg, mcpResp.Result)
	}

	if !strings.Contains(errText, wantErrMsg) {
		t.Fatalf("expected error text containing %q, got %q", wantErrMsg, errText)
	}
}

// RunMCPToolsListMethod calls tools/list and verifies that the returned tools match the expected list.
func RunMCPToolsListMethod(t *testing.T, ctx context.Context, expectedOutput []MCPToolManifest) {
	t.Helper()
	statusCodeList, toolsList, errList := GetMCPToolsList(t, ctx, nil)
	if errList != nil {
		t.Fatalf("native error executing tools/list: %s", errList)
	}
	if statusCodeList != http.StatusOK {
		t.Fatalf("expected status 200 for tools/list, got %d", statusCodeList)
	}

	// Unmarshal toolsList into []MCPToolManifest
	toolsJSON, err := json.Marshal(toolsList)
	if err != nil {
		t.Fatalf("error marshalling tools list: %v", err)
	}

	var actualTools []MCPToolManifest
	if err := json.Unmarshal(toolsJSON, &actualTools); err != nil {
		t.Fatalf("error unmarshalling tools into MCPToolManifest: %v", err)
	}

	if len(actualTools) != len(expectedOutput) {
		t.Fatalf("expected %d tools, got %d. Actual tools: %+v", len(expectedOutput), len(actualTools), actualTools)
	}

	for _, expected := range expectedOutput {
		found := false
		for _, actual := range actualTools {
			if actual.Name == expected.Name {
				found = true
				if diff := cmp.Diff(expected, actual); diff != "" {
					t.Fatalf("tool %s mismatch (-want +got):\n%s", expected.Name, diff)
				}
				break
			}
		}
		if !found {
			t.Fatalf("tool %s was not found in the tools/list registry", expected.Name)
		}
	}
}

// RunMCPCustomToolCallMethod invokes a tool and compares the result with expected output.
func RunMCPCustomToolCallMethod(t *testing.T, ctx context.Context, toolName string, arguments map[string]any, want string) {
	t.Helper()
	statusCode, mcpResp, err := InvokeMCPTool(t, ctx, toolName, arguments, nil)
	if err != nil {
		t.Fatalf("native error executing %s: %s", toolName, err)
	}
	if statusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", statusCode)
	}
	if mcpResp.Result.IsError {
		t.Fatalf("%s returned error result: %v", toolName, mcpResp.Result)
	}
	got := GetMCPResultText(t, mcpResp)
	gotBytes, _ := json.Marshal(got)
	gotStr := string(gotBytes)
	if !strings.Contains(gotStr, want) {
		t.Fatalf(`expected %q to contain %q`, gotStr, want)
	}

}

// RunMCPToolInvokeTest runs the tool invoke test cases over MCP protocol.
func RunMCPToolInvokeTest(t *testing.T, ctx context.Context, select1Want string, options ...InvokeTestOption) {
	t.Helper()
	// Resolve options using existing InvokeTestOption and InvokeTestConfig from option.go
	configs := &InvokeTestConfig{
		myToolId3NameAliceWant:   "[{\"id\":1,\"name\":\"Alice\"},{\"id\":3,\"name\":\"Sid\"}]",
		myToolById4Want:          "[{\"id\":4,\"name\":null}]",
		myArrayToolWant:          "[{\"id\":1,\"name\":\"Alice\"},{\"id\":3,\"name\":\"Sid\"}]",
		nullWant:                 "[]",
		supportOptionalNullParam: true,
		supportArrayParam:        true,
		supportClientAuth:        false,
		supportSelect1Want:       true,
		supportSelect1Auth:       true,
	}

	for _, option := range options {
		option(configs)
	}

	invokeTcs := []struct {
		name       string
		toolName   string
		args       map[string]any
		headers    map[string]string
		enabled    bool
		wantResult string // for success cases
		wantError  string // for failure cases
	}{
		{
			name:       "invoke my-simple-tool",
			toolName:   "my-simple-tool",
			args:       map[string]any{},
			enabled:    configs.supportSelect1Want,
			wantResult: select1Want,
		},
		{
			name:       "invoke my-tool",
			toolName:   "my-tool",
			args:       map[string]any{"id": 3, "name": "Alice"},
			enabled:    true,
			wantResult: configs.myToolId3NameAliceWant,
		},
		{
			name:       "invoke my-tool-by-id with nil response",
			toolName:   "my-tool-by-id",
			args:       map[string]any{"id": 4},
			enabled:    true,
			wantResult: configs.myToolById4Want,
		},
		{
			name:       "invoke my-tool-by-name with nil response",
			toolName:   "my-tool-by-name",
			args:       map[string]any{},
			enabled:    configs.supportOptionalNullParam,
			wantResult: configs.nullWant,
		},
		{
			name:      "Invoke my-tool without parameters",
			toolName:  "my-tool",
			args:      map[string]any{},
			enabled:   true,
			wantError: `parameter "id" is required`,
		},
		{
			name:      "Invoke my-tool with insufficient parameters",
			toolName:  "my-tool",
			args:      map[string]any{"id": 1},
			enabled:   true,
			wantError: `parameter "name" is required`,
		},
	}

	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			if !tc.enabled {
				t.Skip("skipping disabled test case")
			}
			statusCode, mcpResp, err := InvokeMCPTool(t, ctx, tc.toolName, tc.args, tc.headers)
			if err != nil {
				t.Fatalf("native error executing %s: %s", tc.toolName, err)
			}
			if statusCode != http.StatusOK {
				t.Fatalf("expected status 200, got %d", statusCode)
			}
			if tc.wantError != "" {
				AssertMCPError(t, mcpResp, tc.wantError)
				return
			}
			if mcpResp.Result.IsError {
				t.Fatalf("%s returned error result: %v", tc.toolName, mcpResp.Result)
			}
			got := GetMCPResultText(t, mcpResp)
			gotBytes, _ := json.Marshal(got)
			gotStr := string(gotBytes)
			if !strings.Contains(gotStr, tc.wantResult) {
				t.Fatalf(`expected %q to contain %q`, gotStr, tc.wantResult)
			}

		})
	}
}

// RunMCPExecuteSqlToolInvokeTest runs execute-sql tool invoke test cases via MCP.
func RunMCPExecuteSqlToolInvokeTest(t *testing.T, ctx context.Context, createTableStatement, select1Want string, options ...ExecuteSqlOption) {
	configs := &ExecuteSqlTestConfig{
		select1Statement: `"SELECT 1"`,
		createWant:       "[]",
		dropWant:         "[]",
		selectEmptyWant:  "[]",
	}

	for _, option := range options {
		option(configs)
	}

	idToken, err := GetGoogleIdToken(t)
	if err != nil {
		t.Fatalf("error getting Google ID token: %s", err)
	}

	invokeTcs := []struct {
		name       string
		toolName   string
		args       map[string]any
		headers    map[string]string
		want       string
		isErr      bool
		isAgentErr bool
	}{
		{
			name:     "invoke my-exec-sql-tool",
			toolName: "my-exec-sql-tool",
			args:     map[string]any{"sql": strings.Trim(configs.select1Statement, `"`)},
			want:     select1Want,
		},
		{
			name:     "invoke my-exec-sql-tool create table",
			toolName: "my-exec-sql-tool",
			args:     map[string]any{"sql": strings.Trim(createTableStatement, `"`)},
			want:     configs.createWant,
		},
		{
			name:     "invoke my-exec-sql-tool select table",
			toolName: "my-exec-sql-tool",
			args:     map[string]any{"sql": "SELECT * FROM t"},
			want:     configs.selectEmptyWant,
		},
		{
			name:     "invoke my-exec-sql-tool drop table",
			toolName: "my-exec-sql-tool",
			args:     map[string]any{"sql": "DROP TABLE t"},
			want:     configs.dropWant,
		},
		{
			name:       "invoke my-exec-sql-tool without body",
			toolName:   "my-exec-sql-tool",
			args:       map[string]any{},
			isAgentErr: true,
		},
		{
			name:     "Invoke my-auth-exec-sql-tool with auth token",
			toolName: "my-auth-exec-sql-tool",
			args:     map[string]any{"sql": strings.Trim(configs.select1Statement, `"`)},
			headers:  map[string]string{"my-google-auth_token": idToken},
			want:     select1Want,
		},
		{
			name:     "Invoke my-auth-exec-sql-tool with invalid auth token",
			toolName: "my-auth-exec-sql-tool",
			args:     map[string]any{"sql": strings.Trim(configs.select1Statement, `"`)},
			headers:  map[string]string{"my-google-auth_token": "INVALID_TOKEN"},
			isErr:    true,
		},
		{
			name:     "Invoke my-auth-exec-sql-tool without auth token",
			toolName: "my-auth-exec-sql-tool",
			args:     map[string]any{"sql": strings.Trim(configs.select1Statement, `"`)},
			isErr:    true,
		},
		{
			name:       "invoke my-exec-sql-tool with invalid SELECT SQL",
			toolName:   "my-exec-sql-tool",
			args:       map[string]any{"sql": "SELECT * FROM non_existent_table"},
			isAgentErr: true,
		},
		{
			name:       "invoke my-exec-sql-tool with invalid ALTER SQL",
			toolName:   "my-exec-sql-tool",
			args:       map[string]any{"sql": "ALTER TALE t ALTER COLUMN id DROP NOT NULL"},
			isAgentErr: true,
		},
	}

	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			statusCode, mcpResp, err := InvokeMCPTool(t, ctx, tc.toolName, tc.args, tc.headers)

			if tc.isErr {
				if err == nil && mcpResp.Error == nil && !mcpResp.Result.IsError {
					t.Fatalf("expected error but got none")
				}
				return
			}

			if err != nil {
				if tc.isAgentErr {
					return
				}
				t.Fatalf("native error executing %s: %s", tc.toolName, err)
			}

			if mcpResp.Result.IsError {
				if tc.isAgentErr {
					return
				}
				t.Fatalf("%s returned error result: %v", tc.toolName, mcpResp.Result)
			}

			if statusCode != http.StatusOK {
				t.Fatalf("wrong status code: got %d, want %d", statusCode, http.StatusOK)
			}

			if tc.want == "" {
				return
			}

			got := GetMCPResultText(t, mcpResp)
			gotBytes, _ := json.Marshal(got)
			gotStr := string(gotBytes)
			if !strings.Contains(gotStr, tc.want) {
				t.Fatalf(`expected %q to contain %q`, gotStr, tc.want)
			}
		})
	}
}

// RunMCPToolInvokeWithTemplateParameters runs tool invoke test cases with template parameters via MCP.
func RunMCPToolInvokeWithTemplateParameters(t *testing.T, ctx context.Context, tableName string, options ...TemplateParamOption) {
	configs := &TemplateParameterTestConfig{
		ddlWant:         "[]",
		selectAllWant:   "[{\"age\":21,\"id\":1,\"name\":\"Alex\"},{\"age\":100,\"id\":2,\"name\":\"Alice\"}]",
		selectId1Want:   "[{\"age\":21,\"id\":1,\"name\":\"Alex\"}]",
		selectNameWant:  "[{\"age\":21,\"id\":1,\"name\":\"Alex\"}]",
		selectEmptyWant: "[]",
		insert1Want:     "[]",

		nameFieldArray: `["name"]`,
		nameColFilter:  "name",
		createColArray: `["id INT","name VARCHAR(20)","age INT"]`,

		supportDdl:    true,
		supportInsert: true,
	}

	for _, option := range options {
		option(configs)
	}

	selectOnlyNamesWant := "[{\"name\":\"Alex\"},{\"name\":\"Alice\"}]"

	invokeTcs := []struct {
		name     string
		toolName string
		enabled  bool
		ddl      bool
		insert   bool
		args     map[string]any
		want     string
		isErr    bool
	}{
		{
			name:     "invoke create-table-templateParams-tool",
			toolName: "create-table-templateParams-tool",
			ddl:      true,
			enabled:  configs.supportDdl,
			args:     map[string]any{"tableName": tableName, "columns": []string{"id INT", "name VARCHAR(20)", "age INT"}}, // Pass as slice, not string!
			want:     configs.ddlWant,
		},
		{
			name:     "invoke insert-table-templateParams-tool",
			toolName: "insert-table-templateParams-tool",
			insert:   true,
			enabled:  configs.supportInsert,
			args:     map[string]any{"tableName": tableName, "columns": []string{"id", "name", "age"}, "values": "1, 'Alex', 21"},
			want:     configs.insert1Want,
		},
		{
			name:     "invoke insert-table-templateParams-tool 2",
			toolName: "insert-table-templateParams-tool",
			insert:   true,
			enabled:  configs.supportInsert,
			args:     map[string]any{"tableName": tableName, "columns": []string{"id", "name", "age"}, "values": "2, 'Alice', 100"},
			want:     configs.insert1Want,
		},
		{
			name:     "invoke select-templateParams-tool",
			toolName: "select-templateParams-tool",
			enabled:  true,
			args:     map[string]any{"tableName": tableName},
			want:     configs.selectAllWant,
		},
		{
			name:     "invoke select-templateParams-combined-tool",
			toolName: "select-templateParams-combined-tool",
			enabled:  true,
			args:     map[string]any{"id": 1, "tableName": tableName},
			want:     configs.selectId1Want,
		},
		{
			name:     "invoke select-templateParams-combined-tool with no results",
			toolName: "select-templateParams-combined-tool",
			enabled:  true,
			args:     map[string]any{"id": 999, "tableName": tableName},
			want:     configs.selectEmptyWant,
		},
		{
			name:     "invoke select-fields-templateParams-tool",
			toolName: "select-fields-templateParams-tool",
			enabled:  configs.supportSelectFields,
			args:     map[string]any{"tableName": tableName, "fields": []string{"name"}},
			want:     selectOnlyNamesWant,
		},
		{
			name:     "invoke select-filter-templateParams-combined-tool",
			toolName: "select-filter-templateParams-combined-tool",
			enabled:  true,
			args:     map[string]any{"name": "Alex", "tableName": tableName, "columnFilter": configs.nameColFilter},
			want:     configs.selectNameWant,
		},
		{
			name:     "invoke drop-table-templateParams-tool",
			toolName: "drop-table-templateParams-tool",
			ddl:      true,
			enabled:  configs.supportDdl,
			args:     map[string]any{"tableName": tableName},
			want:     configs.ddlWant,
		},
	}

	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			if !tc.enabled {
				return
			}
			statusCode, mcpResp, err := InvokeMCPTool(t, ctx, tc.toolName, tc.args, nil)

			if tc.isErr {
				if err == nil && mcpResp.Error == nil && !mcpResp.Result.IsError {
					t.Fatalf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("native error executing %s: %s", tc.toolName, err)
			}

			if mcpResp.Result.IsError {
				t.Fatalf("%s returned error result: %v", tc.toolName, mcpResp.Result)
			}

			if statusCode != http.StatusOK {
				t.Fatalf("wrong status code: got %d, want %d", statusCode, http.StatusOK)
			}

			if tc.want == "" {
				return
			}

			got := GetMCPResultText(t, mcpResp)
			gotBytes, _ := json.Marshal(got)
			gotStr := string(gotBytes)
			if !strings.Contains(gotStr, tc.want) {
				t.Fatalf(`expected %q to contain %q`, gotStr, tc.want)
			}
		})
	}
}

// setUpPostgresViews creates a test view and returns a cleanup function.
func setUpMCPPostgresViews(t *testing.T, ctx context.Context, pool *pgxpool.Pool, viewName string) func() {
	createView := fmt.Sprintf("CREATE VIEW %s AS SELECT 1 AS col", viewName)
	_, err := pool.Exec(ctx, createView)
	if err != nil {
		t.Fatalf("failed to create view: %v", err)
	}
	return func() {
		dropView := fmt.Sprintf("DROP VIEW %s", viewName)
		_, err := pool.Exec(ctx, dropView)
		if err != nil {
			t.Fatalf("failed to drop view: %v", err)
		}
	}
}

// RunMCPPostgresListViewsTest tests the list_views tool via MCP.
func RunMCPPostgresListViewsTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	viewName := "test_view_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	dropViewfunc1 := setUpMCPPostgresViews(t, ctx, pool, viewName)
	defer dropViewfunc1()

	invokeTcs := []struct {
		name           string
		args           map[string]any
		wantStatusCode int
		want           string
	}{
		{
			name:           "invoke list_views with newly created view",
			args:           map[string]any{"view_name": viewName},
			wantStatusCode: http.StatusOK,
			want:           fmt.Sprintf(`[{"schema_name":"public","view_name":"%s","owner_name":"postgres","definition":" SELECT 1 AS col;"}]`, viewName),
		},
		{
			name:           "invoke list_views with non-existent_view",
			args:           map[string]any{"view_name": "non_existent_view"},
			wantStatusCode: http.StatusOK,
			want:           `[]`,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			statusCode, mcpResp, err := InvokeMCPTool(t, ctx, "list_views", tc.args, nil)
			if err != nil {
				t.Fatalf("native error executing list_views: %s", err)
			}
			if statusCode != tc.wantStatusCode {
				t.Fatalf("wrong status code: got %d, want %d", statusCode, tc.wantStatusCode)
			}
			if tc.wantStatusCode != http.StatusOK {
				return
			}
			if mcpResp.Result.IsError {
				t.Fatalf("list_views returned error result: %v", mcpResp.Result)
			}

			got := GetMCPResultText(t, mcpResp)

			var wantObj []any
			if err := json.Unmarshal([]byte(tc.want), &wantObj); err != nil {
				t.Fatalf("failed to unmarshal want string: %v", err)
			}

			if diff := cmp.Diff(wantObj, got); diff != "" {
				t.Errorf("Unexpected result mismatch (-want +got):\n%s", diff)
			}

		})
	}
}

// RunMCPPostgresListTablesTest tests the list_tables tool via MCP.
func RunMCPPostgresListTablesTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool, user string) {
	uniqueID := strings.ReplaceAll(uuid.New().String(), "-", "")
	tableNameParam := "param_table_" + uniqueID
	tableNameAuth := "auth_table_" + uniqueID

	createParamTableStmt, insertParamTableStmt, _, _, _, _, paramTestParams := GetPostgresSQLParamToolInfo(tableNameParam)
	teardownTable1 := SetupPostgresSQLTable(t, ctx, pool, createParamTableStmt, insertParamTableStmt, tableNameParam, paramTestParams)
	defer teardownTable1(t)

	createAuthTableStmt, insertAuthTableStmt, _, authTestParams := GetPostgresSQLAuthToolInfo(tableNameAuth)
	teardownTable2 := SetupPostgresSQLTable(t, ctx, pool, createAuthTableStmt, insertAuthTableStmt, tableNameAuth, authTestParams)
	defer teardownTable2(t)

	// TableNameParam columns to construct want
	paramTableColumns := fmt.Sprintf(`[
		{"data_type": "integer", "column_name": "id", "column_default": "nextval('%s_id_seq'::regclass)", "is_not_nullable": true, "ordinal_position": 1, "column_comment": null},
		{"data_type": "text", "column_name": "name", "column_default": null, "is_not_nullable": false, "ordinal_position": 2, "column_comment": null}
	]`, tableNameParam)

	// TableNameAuth columns to construct want
	authTableColumns := fmt.Sprintf(`[
		{"data_type": "integer", "column_name": "id", "column_default": "nextval('%s_id_seq'::regclass)", "is_not_nullable": true, "ordinal_position": 1, "column_comment": null},
		{"data_type": "text", "column_name": "name", "column_default": null, "is_not_nullable": false, "ordinal_position": 2, "column_comment": null},
		{"data_type": "text", "column_name": "email", "column_default": null, "is_not_nullable": false, "ordinal_position": 3, "column_comment": null}
	]`, tableNameAuth)

	const (
		// Template to construct detailed output want
		detailedObjectTemplate = `{
            "object_name": "%[1]s", "schema_name": "public",
            "object_details": {
                "owner": "%[3]s", "comment": null,
                "indexes": [{"is_primary": true, "is_unique": true, "index_name": "%[1]s_pkey", "index_method": "btree", "index_columns": ["id"], "index_definition": "CREATE UNIQUE INDEX %[1]s_pkey ON public.%[1]s USING btree (id)"}],
                "triggers": [], "columns": %[2]s, "object_name": "%[1]s", "object_type": "TABLE", "schema_name": "public",
                "constraints": [{"constraint_name": "%[1]s_pkey", "constraint_type": "PRIMARY KEY", "constraint_columns": ["id"], "constraint_definition": "PRIMARY KEY (id)", "foreign_key_referenced_table": null, "foreign_key_referenced_columns": null}]
            }
        }`

		// Template to construct simple output want
		simpleObjectTemplate = `{"object_name":"%s", "schema_name":"public", "object_details":{"name":"%s"}}`
	)

	// Helper to build json for detailed want
	getDetailedWant := func(tableName, columnJSON string) string {
		return fmt.Sprintf(detailedObjectTemplate, tableName, columnJSON, user)
	}

	// Helper to build template for simple want
	getSimpleWant := func(tableName string) string {
		return fmt.Sprintf(simpleObjectTemplate, tableName, tableName)
	}

	invokeTcs := []struct {
		name           string
		args           map[string]any
		wantStatusCode int
		want           string
		isAllTables    bool
		isAgentErr     bool
	}{
		{
			name:           "invoke list_tables all tables detailed output",
			args:           map[string]any{"table_names": ""},
			wantStatusCode: http.StatusOK,
			want:           fmt.Sprintf("[%s,%s]", getDetailedWant(tableNameAuth, authTableColumns), getDetailedWant(tableNameParam, paramTableColumns)),
			isAllTables:    true,
		},
		{
			name:           "invoke list_tables all tables simple output",
			args:           map[string]any{"table_names": "", "output_format": "simple"},
			wantStatusCode: http.StatusOK,
			want:           fmt.Sprintf("[%s,%s]", getSimpleWant(tableNameAuth), getSimpleWant(tableNameParam)),
			isAllTables:    true,
		},
		{
			name:           "invoke list_tables detailed output",
			args:           map[string]any{"table_names": tableNameAuth},
			wantStatusCode: http.StatusOK,
			want:           fmt.Sprintf("[%s]", getDetailedWant(tableNameAuth, authTableColumns)),
		},
		{
			name:           "invoke list_tables simple output",
			args:           map[string]any{"table_names": tableNameAuth, "output_format": "simple"},
			wantStatusCode: http.StatusOK,
			want:           fmt.Sprintf("[%s]", getSimpleWant(tableNameAuth)),
		},
		{
			name:           "invoke list_tables with invalid output format",
			args:           map[string]any{"table_names": "", "output_format": "abcd"},
			wantStatusCode: http.StatusOK,
			isAgentErr:     true,
		},
		{
			name:           "invoke list_tables with malformed table_names parameter",
			args:           map[string]any{"table_names": 12345, "output_format": "detailed"},
			wantStatusCode: http.StatusOK,
			isAgentErr:     true,
		},
		{
			name:           "invoke list_tables with multiple table names",
			args:           map[string]any{"table_names": fmt.Sprintf("%s,%s", tableNameParam, tableNameAuth)},
			wantStatusCode: http.StatusOK,
			want:           fmt.Sprintf("[%s,%s]", getDetailedWant(tableNameAuth, authTableColumns), getDetailedWant(tableNameParam, paramTableColumns)),
		},
		{
			name:           "invoke list_tables with non-existent table",
			args:           map[string]any{"table_names": "non_existent_table"},
			wantStatusCode: http.StatusOK,
			want:           `[]`,
		},
	}

	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			statusCode, mcpResp, err := InvokeMCPTool(t, ctx, "list_tables", tc.args, nil)
			if err != nil {
				t.Fatalf("native error executing list_tables: %s", err)
			}
			if statusCode != tc.wantStatusCode {
				t.Fatalf("wrong status code: got %d, want %d", statusCode, tc.wantStatusCode)
			}
			if tc.wantStatusCode != http.StatusOK {
				return
			}

			if tc.isAgentErr {
				if mcpResp.Error == nil && !mcpResp.Result.IsError {
					t.Fatalf("expected error result or JSON-RPC error, got success")
				}
				return
			}

			if mcpResp.Result.IsError {
				t.Fatalf("list_tables returned error result: %v", mcpResp.Result)
			}

			got := GetMCPResultText(t, mcpResp)

			var wantObj []any
			if err := json.Unmarshal([]byte(tc.want), &wantObj); err != nil {
				t.Fatalf("failed to unmarshal want string: %v", err)
			}

			if tc.isAllTables {
				var filteredGot []any
				for _, item := range got {
					if tableMap, ok := item.(map[string]any); ok {
						name, _ := tableMap["object_name"].(string)
						if name == tableNameParam || name == tableNameAuth {
							filteredGot = append(filteredGot, item)
						}
					}
				}
				got = filteredGot
			}

			// Sort both to ensure comparison works regardless of order
			sort.SliceStable(got, func(i, j int) bool {
				return fmt.Sprintf("%v", got[i]) < fmt.Sprintf("%v", got[j])
			})
			sort.SliceStable(wantObj, func(i, j int) bool {
				return fmt.Sprintf("%v", wantObj[i]) < fmt.Sprintf("%v", wantObj[j])
			})

			if diff := cmp.Diff(wantObj, got); diff != "" {
				t.Errorf("Unexpected result mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// RunMCPPostgresListQueryStatsTest tests the list_query_stats tool via MCP.
func RunMCPPostgresListQueryStatsTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	// Insert a simple query by running a SELECT statement
	// This will record statistics in pg_stat_statements
	selectStmt := "SELECT 1 as test_query"
	if _, err := pool.Exec(ctx, selectStmt); err != nil {
		t.Logf("warning: unable to execute test query: %s", err)
	}

	dropExtensionFunc := createPostgresExtension(t, ctx, pool, "pg_stat_statements")
	defer dropExtensionFunc()

	invokeTcs := []struct {
		name           string
		args           map[string]any
		wantStatusCode int
	}{
		{
			name:           "list query stats with default limit",
			args:           map[string]any{},
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "list query stats with custom limit",
			args:           map[string]any{"limit": 10},
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "list query stats for specific database",
			args:           map[string]any{"database_name": "postgres"},
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "list query stats with non-existent database name",
			args:           map[string]any{"database_name": "non_existent_db_xyz"},
			wantStatusCode: http.StatusOK,
		},
	}

	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			statusCode, mcpResp, err := InvokeMCPTool(t, ctx, "list_query_stats", tc.args, nil)
			if err != nil {
				t.Fatalf("native error executing list_query_stats: %s", err)
			}
			if statusCode != tc.wantStatusCode {
				t.Fatalf("wrong status code: got %d, want %d", statusCode, tc.wantStatusCode)
			}
			if tc.wantStatusCode != http.StatusOK {
				return
			}

			if mcpResp.Result.IsError {
				t.Fatalf("list_query_stats returned error result: %v", mcpResp.Result)
			}

			got := GetMCPResultText(t, mcpResp)

			// Verify that we got a list (even if empty)
			if got == nil {
				t.Fatalf("expected a list result, got nil")
			}

			t.Logf("found %d query stats", len(got))
		})
	}
}

// setupPostgresSchemas creates a test schema and returns a cleanup function.
func setupMCPPostgresSchemas(t *testing.T, ctx context.Context, pool *pgxpool.Pool, schemaName string) func() {
	createSchemaStmt := fmt.Sprintf("CREATE SCHEMA %s", schemaName)
	_, err := pool.Exec(ctx, createSchemaStmt)
	if err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	return func() {
		dropSchemaStmt := fmt.Sprintf("DROP SCHEMA %s CASCADE", schemaName)
		_, err := pool.Exec(ctx, dropSchemaStmt)
		if err != nil {
			t.Fatalf("failed to drop schema: %v", err)
		}
	}
}

// RunMCPPostgresListSchemasTest tests the list_schemas tool via MCP.
func RunMCPPostgresListSchemasTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool, owner string, uniqueID string) {
	schemaName := "test_schema_" + uniqueID
	cleanup := setupMCPPostgresSchemas(t, ctx, pool, schemaName)
	defer cleanup()

	wantSchema := map[string]any{"functions": float64(0), "grants": map[string]any{}, "owner": owner, "schema_name": schemaName, "tables": float64(0), "views": float64(0)}

	invokeTcs := []struct {
		name           string
		args           map[string]any
		wantStatusCode int
		want           []map[string]any
		compareSubset  bool
	}{
		{
			name:           "invoke list_schemas with schema_name",
			args:           map[string]any{"schema_name": schemaName},
			wantStatusCode: http.StatusOK,
			want:           []map[string]any{wantSchema},
		},
		{
			name:           "invoke list_schemas with limit 1",
			args:           map[string]any{"schema_name": schemaName, "limit": 1},
			wantStatusCode: http.StatusOK,
			want:           []map[string]any{wantSchema},
		},
		{
			name:           "invoke list_schemas with non-existent schema",
			args:           map[string]any{"schema_name": "non_existent_schema"},
			wantStatusCode: http.StatusOK,
			want:           []map[string]any{},
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			statusCode, mcpResp, err := InvokeMCPTool(t, ctx, "list_schemas", tc.args, nil)
			if err != nil {
				t.Fatalf("native error executing list_schemas: %s", err)
			}
			if statusCode != tc.wantStatusCode {
				t.Fatalf("wrong status code: got %d, want %d", statusCode, tc.wantStatusCode)
			}
			if tc.wantStatusCode != http.StatusOK {
				return
			}
			if mcpResp.Result.IsError {
				t.Fatalf("list_schemas returned error result: %v", mcpResp.Result)
			}
			gotObj := GetMCPResultText(t, mcpResp)

			if tc.compareSubset {
				found := false
				for _, resultSchemaObj := range gotObj {
					resultSchema, ok := resultSchemaObj.(map[string]any)
					if !ok {
						continue
					}
					if resultSchema["schema_name"] == wantSchema["schema_name"] {
						found = true
						if diff := cmp.Diff(wantSchema, resultSchema); diff != "" {
							t.Errorf("Mismatch in fields for the expected schema (-want +got):\n%s", diff)
						}
						break
					}
				}
				if !found {
					t.Errorf("Expected schema '%+v' not found in the list of all schemas.", wantSchema)
				}
			} else {
				wantObj := []any{}
				for _, item := range tc.want {
					wantObj = append(wantObj, item)
				}
				if diff := cmp.Diff(wantObj, gotObj); diff != "" {
					t.Errorf("Unexpected result mismatch (-want +got):\n%s", diff)
				}
			}

		})
	}
}

// RunMCPPostgresListActiveQueriesTest tests the list_active_queries tool via MCP.
func RunMCPPostgresListActiveQueriesTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	type queryListDetails struct {
		Query string `json:"query"`
	}

	singleQueryWanted := queryListDetails{
		Query: "SELECT pg_sleep(10);",
	}

	invokeTcs := []struct {
		name                string
		args                map[string]any
		clientSleepSecs     int
		waitSecsBeforeCheck int
		wantStatusCode      int
		want                []queryListDetails
	}{
		{
			name:                "invoke list_active_queries when the system is idle",
			args:                map[string]any{"exclude_application_names": "wal_uploader"},
			clientSleepSecs:     0,
			waitSecsBeforeCheck: 0,
			wantStatusCode:      http.StatusOK,
			want:                nil,
		},
		{
			name:                "invoke list_active_queries when there is 1 ongoing but lower than the threshold",
			args:                map[string]any{"min_duration": "100 seconds", "exclude_application_names": "wal_uploader"},
			clientSleepSecs:     1,
			waitSecsBeforeCheck: 1,
			wantStatusCode:      http.StatusOK,
			want:                nil,
		},
		{
			name:                "invoke list_active_queries when 1 ongoing query should show up",
			args:                map[string]any{"min_duration": "1 seconds", "exclude_application_names": "wal_uploader"},
			clientSleepSecs:     10,
			waitSecsBeforeCheck: 5,
			wantStatusCode:      http.StatusOK,
			want:                []queryListDetails{singleQueryWanted},
		},
	}

	var wg sync.WaitGroup
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			if tc.clientSleepSecs > 0 {
				wg.Add(1)

				go func() {
					defer wg.Done()

					err := pool.Ping(ctx)
					if err != nil {
						t.Errorf("unable to connect to test database: %s", err)
						return
					}
					_, err = pool.Exec(ctx, fmt.Sprintf("SELECT pg_sleep(%d);", tc.clientSleepSecs))
					if err != nil {
						t.Errorf("Executing 'SELECT pg_sleep' failed: %s", err)
					}
				}()
			}

			if tc.waitSecsBeforeCheck > 0 {
				time.Sleep(time.Duration(tc.waitSecsBeforeCheck) * time.Second)
			}

			statusCode, mcpResp, err := InvokeMCPTool(t, ctx, "list_active_queries", tc.args, nil)
			if err != nil {
				t.Fatalf("native error executing list_active_queries: %s", err)
			}
			if statusCode != tc.wantStatusCode {
				t.Fatalf("wrong status code: got %d, want %d", statusCode, tc.wantStatusCode)
			}
			if tc.wantStatusCode != http.StatusOK {
				return
			}
			if mcpResp.Result.IsError {
				t.Fatalf("list_active_queries returned error result: %v", mcpResp.Result)
			}
			var details []queryListDetails
			gotObj := GetMCPResultText(t, mcpResp)
			for _, item := range gotObj {
				if m, ok := item.(map[string]any); ok {
					if q, ok := m["query"].(string); ok {
						details = append(details, queryListDetails{Query: q})
					}
				}
			}

			if len(tc.want) == 0 {
				if len(details) != 0 {
					var filtered []queryListDetails
					for _, d := range details {
						if strings.Contains(d.Query, "pg_sleep") {
							filtered = append(filtered, d)
						}
					}
					if len(filtered) != 0 {
						t.Errorf("Unexpected active queries: got %v, want empty", filtered)
					}
				}
			} else {
				found := false
				for _, d := range details {
					if d.Query == tc.want[0].Query {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected query %q not found in active queries: %v", tc.want[0].Query, details)
				}
			}
		})
	}
	wg.Wait()
}

// RunMCPPostgresListAvailableExtensionsTest tests the list_available_extensions tool via MCP.
func RunMCPPostgresListAvailableExtensionsTest(t *testing.T, ctx context.Context) {
	statusCode, mcpResp, err := InvokeMCPTool(t, ctx, "list_available_extensions", map[string]any{}, nil)
	if err != nil {
		t.Fatalf("native error executing list_available_extensions: %s", err)
	}
	if statusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", statusCode)
	}
	if mcpResp.Result.IsError {
		t.Fatalf("list_available_extensions returned error result: %v", mcpResp.Result)
	}
}

// RunMCPPostgresListInstalledExtensionsTest tests the list_installed_extensions tool via MCP.
func RunMCPPostgresListInstalledExtensionsTest(t *testing.T, ctx context.Context) {
	statusCode, mcpResp, err := InvokeMCPTool(t, ctx, "list_installed_extensions", map[string]any{}, nil)
	if err != nil {
		t.Fatalf("native error executing list_installed_extensions: %s", err)
	}
	if statusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", statusCode)
	}
	if mcpResp.Result.IsError {
		t.Fatalf("list_installed_extensions returned error result: %v", mcpResp.Result)
	}
}

// setupPostgresTrigger creates a test trigger and returns a cleanup function.
func setupMCPPostgresTrigger(t *testing.T, ctx context.Context, pool *pgxpool.Pool, schemaName, tableName, functionName, triggerName string) func() {
	t.Helper()

	createSchemaStmt := fmt.Sprintf("CREATE SCHEMA %s", schemaName)
	if _, err := pool.Exec(ctx, createSchemaStmt); err != nil {
		t.Fatalf("failed to create schema %s: %v", schemaName, err)
	}

	createTableStmt := fmt.Sprintf("CREATE TABLE %s.%s (id SERIAL PRIMARY KEY, name TEXT)", schemaName, tableName)
	if _, err := pool.Exec(ctx, createTableStmt); err != nil {
		t.Fatalf("failed to create table %s.%s: %v", schemaName, tableName, err)
	}

	createFunctionStmt := fmt.Sprintf(`
	CREATE OR REPLACE FUNCTION %s.%s() RETURNS TRIGGER AS $$
	BEGIN
		RETURN NEW;
	END;
	$$ LANGUAGE plpgsql;
`, schemaName, functionName)
	if _, err := pool.Exec(ctx, createFunctionStmt); err != nil {
		t.Fatalf("failed to create function %s.%s: %v", schemaName, functionName, err)
	}

	createTriggerStmt := fmt.Sprintf(`
	CREATE TRIGGER %s
	AFTER INSERT ON %s.%s
	FOR EACH ROW
	EXECUTE FUNCTION %s.%s();
`, triggerName, schemaName, tableName, schemaName, functionName)
	if _, err := pool.Exec(ctx, createTriggerStmt); err != nil {
		t.Fatalf("failed to create trigger %s: %v", triggerName, err)
	}

	return func() {
		dropSchemaStmt := fmt.Sprintf("DROP SCHEMA %s CASCADE", schemaName)
		if _, err := pool.Exec(ctx, dropSchemaStmt); err != nil {
			t.Fatalf("failed to drop schema %s: %v", schemaName, err)
		}
	}
}

// RunMCPPostgresListTriggersTest tests the list_triggers tool via MCP.
func RunMCPPostgresListTriggersTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	uniqueID := strings.ReplaceAll(uuid.New().String(), "-", "")
	schemaName := "test_schema_" + uniqueID
	tableName := "test_table_" + uniqueID
	functionName := "test_func_" + uniqueID
	triggerName := "test_trigger_" + uniqueID

	cleanup := setupMCPPostgresTrigger(t, ctx, pool, schemaName, tableName, functionName, triggerName)
	defer cleanup()

	var expectedDef string
	getDefQuery := fmt.Sprintf("SELECT pg_get_triggerdef(oid) FROM pg_trigger WHERE tgname = '%s'", triggerName)
	err := pool.QueryRow(ctx, getDefQuery).Scan(&expectedDef)
	if err != nil {
		t.Fatalf("failed to fetch trigger definition: %v", err)
	}

	wantTrigger := map[string]any{
		"trigger_name":     triggerName,
		"schema_name":      schemaName,
		"table_name":       tableName,
		"status":           "ENABLED",
		"timing":           "AFTER",
		"events":           "INSERT",
		"activation_level": "ROW",
		"function_name":    functionName,
		"definition":       expectedDef,
	}

	invokeTcs := []struct {
		name           string
		args           map[string]any
		wantStatusCode int
		want           []map[string]any
		compareSubset  bool
	}{
		{
			name:           "list all triggers (expecting the one we created)",
			args:           map[string]any{},
			wantStatusCode: http.StatusOK,
			want:           []map[string]any{wantTrigger},
			compareSubset:  true,
		},
		{
			name:           "filter by trigger_name",
			args:           map[string]any{"trigger_name": triggerName},
			wantStatusCode: http.StatusOK,
			want:           []map[string]any{wantTrigger},
		},
		{
			name:           "filter by schema_name",
			args:           map[string]any{"schema_name": schemaName},
			wantStatusCode: http.StatusOK,
			want:           []map[string]any{wantTrigger},
		},
		{
			name:           "filter by table_name",
			args:           map[string]any{"table_name": tableName},
			wantStatusCode: http.StatusOK,
			want:           []map[string]any{wantTrigger},
		},
		{
			name:           "filter by non-existent trigger_name",
			args:           map[string]any{"trigger_name": "non_existent_trigger"},
			wantStatusCode: http.StatusOK,
			want:           []map[string]any{},
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			statusCode, mcpResp, err := InvokeMCPTool(t, ctx, "list_triggers", tc.args, nil)
			if err != nil {
				t.Fatalf("native error executing list_triggers: %s", err)
			}
			if statusCode != tc.wantStatusCode {
				t.Fatalf("wrong status code: got %d, want %d", statusCode, tc.wantStatusCode)
			}
			if tc.wantStatusCode != http.StatusOK {
				return
			}
			if mcpResp.Result.IsError {
				t.Fatalf("list_triggers returned error result: %v", mcpResp.Result)
			}
			gotObj := GetMCPResultText(t, mcpResp)

			if tc.compareSubset {
				found := false
				for _, resultTriggerObj := range gotObj {
					resultTrigger, ok := resultTriggerObj.(map[string]any)
					if !ok {
						continue
					}
					if resultTrigger["trigger_name"] == wantTrigger["trigger_name"] {
						found = true
						if resultTrigger["schema_name"] != wantTrigger["schema_name"] ||
							resultTrigger["table_name"] != wantTrigger["table_name"] ||
							resultTrigger["function_name"] != wantTrigger["function_name"] {
							t.Errorf("Mismatch in fields for the expected trigger: got %v, want %v", resultTrigger, wantTrigger)
						}
						break
					}
				}
				if !found {
					t.Errorf("Expected trigger '%+v' not found in the list. Got: %+v", wantTrigger, gotObj)
				}
			} else {
				wantObj := []any{}
				for _, item := range tc.want {
					wantObj = append(wantObj, item)
				}
				if diff := cmp.Diff(wantObj, gotObj); diff != "" {
					t.Errorf("Unexpected result mismatch (-want +got):\n%s", diff)
				}
			}

		})
	}
}

// setupListSequencesTest creates a test sequence and returns a cleanup function.
func setupMCPListSequencesTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool) (string, func(t *testing.T)) {
	sequenceName := "list_sequences_seq1_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	createSequence1Stmt := fmt.Sprintf("CREATE SEQUENCE %s INCREMENT 1 START 1;", sequenceName)

	_, err := pool.Exec(ctx, createSequence1Stmt)
	if err != nil {
		t.Fatalf("unable to create sequence %s: %s", sequenceName, err)
	}
	return sequenceName, func(t *testing.T) {
		_, err := pool.Exec(ctx, fmt.Sprintf("DROP SEQUENCE IF EXISTS %s;", sequenceName))
		if err != nil {
			t.Errorf("unable to drop sequences: %v", err)
		}
	}
}

// RunMCPPostgresListSequencesTest tests the list_sequences tool via MCP.
func RunMCPPostgresListSequencesTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	sequenceName, teardown := setupMCPListSequencesTest(t, ctx, pool)
	defer teardown(t)

	wantSequence := map[string]any{
		"sequence_name":  sequenceName,
		"schema_name":    "public",
		"sequence_owner": "postgres",
		"data_type":      "bigint",
		"start_value":    float64(1),
		"min_value":      float64(1),
		"max_value":      float64(9223372036854775807),
		"increment_by":   float64(1),
		"last_value":     nil,
	}

	invokeTcs := []struct {
		name           string
		args           map[string]any
		wantStatusCode int
		want           []map[string]any
	}{
		{
			name:           "invoke list_sequences",
			args:           map[string]any{"sequence_name": sequenceName},
			wantStatusCode: http.StatusOK,
			want:           []map[string]any{wantSequence},
		},
		{
			name:           "invoke list_sequences with non-existent sequence",
			args:           map[string]any{"sequence_name": "non_existent_sequence"},
			wantStatusCode: http.StatusOK,
			want:           []map[string]any{},
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			statusCode, mcpResp, err := InvokeMCPTool(t, ctx, "list_sequences", tc.args, nil)
			if err != nil {
				t.Fatalf("native error executing list_sequences: %s", err)
			}
			if statusCode != tc.wantStatusCode {
				t.Fatalf("wrong status code: got %d, want %d", statusCode, tc.wantStatusCode)
			}
			if tc.wantStatusCode != http.StatusOK {
				return
			}
			if mcpResp.Result.IsError {
				t.Fatalf("list_sequences returned error result: %v", mcpResp.Result)
			}
			gotObj := GetMCPResultText(t, mcpResp)
			wantObj := []any{}
			for _, item := range tc.want {
				wantObj = append(wantObj, item)
			}
			if diff := cmp.Diff(wantObj, gotObj); diff != "" {
				t.Errorf("Unexpected result mismatch (-want +got):\n%s", diff)
			}

		})
	}
}

// setupPostgresIndex creates a test schema and table with indexes and returns a cleanup function.
func setupMCPPostgresIndex(t *testing.T, ctx context.Context, pool *pgxpool.Pool, schemaName string, tableName string) func(t *testing.T) {
	t.Helper()
	createSchemaStmt := fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s;", schemaName)
	if _, err := pool.Exec(ctx, createSchemaStmt); err != nil {
		t.Fatalf("unable to create schema %s: %v", schemaName, err)
	}

	fullTableName := fmt.Sprintf("%s.%s", schemaName, tableName)
	createTableStmt := fmt.Sprintf("CREATE TABLE %s (id SERIAL PRIMARY KEY, name TEXT, email TEXT);", fullTableName)
	if _, err := pool.Exec(ctx, createTableStmt); err != nil {
		t.Fatalf("unable to create table %s: %v", fullTableName, err)
	}

	// Create a unique index on email
	index1Stmt := fmt.Sprintf("CREATE UNIQUE INDEX %s_email_idx ON %s (email);", tableName, fullTableName)
	if _, err := pool.Exec(ctx, index1Stmt); err != nil {
		t.Fatalf("unable to create index %s_email_idx: %v", tableName, err)
	}

	// Create a non-unique index on name
	index2Stmt := fmt.Sprintf("CREATE INDEX %s_name_idx ON %s (name);", tableName, fullTableName)
	if _, err := pool.Exec(ctx, index2Stmt); err != nil {
		t.Fatalf("unable to create index %s_name_idx: %v", tableName, err)
	}

	// Force statistics update to ensure indices appear in pg_stat_all_indexes
	if _, err := pool.Exec(ctx, fmt.Sprintf("ANALYZE %s;", fullTableName)); err != nil {
		t.Fatalf("unable to analyze table %s: %v", fullTableName, err)
	}

	return func(t *testing.T) {
		t.Helper()
		if _, err := pool.Exec(ctx, fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE;", schemaName)); err != nil {
			t.Errorf("unable to drop schema: %v", err)
		}
	}
}

// RunMCPPostgresListIndexesTest tests the list_indexes tool via MCP.
func RunMCPPostgresListIndexesTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	schemaName := "testschema_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableName := "table1_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	cleanup := setupMCPPostgresIndex(t, ctx, pool, schemaName, tableName)
	defer cleanup(t)

	wantIndexPK := map[string]any{
		"schema_name":      schemaName,
		"table_name":       tableName,
		"index_name":       tableName + "_pkey",
		"index_type":       "btree",
		"is_unique":        true,
		"is_primary":       true,
		"is_used":          false,
		"index_definition": fmt.Sprintf("CREATE UNIQUE INDEX %s_pkey ON %s.%s USING btree (id)", tableName, schemaName, tableName),
	}
	wantIndexEmail := map[string]any{
		"schema_name":      schemaName,
		"table_name":       tableName,
		"index_name":       tableName + "_email_idx",
		"index_type":       "btree",
		"is_unique":        true,
		"is_primary":       false,
		"is_used":          false,
		"index_definition": fmt.Sprintf("CREATE UNIQUE INDEX %s_email_idx ON %s.%s USING btree (email)", tableName, schemaName, tableName),
	}
	wantIndexName := map[string]any{
		"schema_name":      schemaName,
		"table_name":       tableName,
		"index_name":       tableName + "_name_idx",
		"index_type":       "btree",
		"is_unique":        false,
		"is_primary":       false,
		"is_used":          false,
		"index_definition": fmt.Sprintf("CREATE INDEX %s_name_idx ON %s.%s USING btree (name)", tableName, schemaName, tableName),
	}

	invokeTcs := []struct {
		name           string
		args           map[string]any
		wantStatusCode int
		want           []map[string]any
		compareSubset  bool
	}{
		{
			name:           "invoke list_indexes with schema_name and table_name",
			args:           map[string]any{"schema_name": schemaName, "table_name": tableName},
			wantStatusCode: http.StatusOK,
			want:           []map[string]any{wantIndexEmail, wantIndexName, wantIndexPK},
			compareSubset:  true,
		},
		{
			name:           "invoke list_indexes with table_name",
			args:           map[string]any{"table_name": tableName},
			wantStatusCode: http.StatusOK,
			want:           []map[string]any{wantIndexEmail, wantIndexName, wantIndexPK},
			compareSubset:  true,
		},
		{
			name:           "invoke list_indexes with non-existent table",
			args:           map[string]any{"table_name": "non_existent_table"},
			wantStatusCode: http.StatusOK,
			want:           []map[string]any{},
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			statusCode, mcpResp, err := InvokeMCPTool(t, ctx, "list_indexes", tc.args, nil)
			if err != nil {
				t.Fatalf("native error executing list_indexes: %s", err)
			}
			if statusCode != tc.wantStatusCode {
				t.Fatalf("wrong status code: got %d, want %d", statusCode, tc.wantStatusCode)
			}
			if tc.wantStatusCode != http.StatusOK {
				return
			}
			if mcpResp.Result.IsError {
				t.Fatalf("list_indexes returned error result: %v", mcpResp.Result)
			}
			gotObj := GetMCPResultText(t, mcpResp)

			if tc.compareSubset {
				for _, wantIdx := range tc.want {
					found := false
					for _, gotIdxObj := range gotObj {
						gotIdx, ok := gotIdxObj.(map[string]any)
						if !ok {
							continue
						}
						if gotIdx["index_name"] == wantIdx["index_name"] {
							found = true
							if gotIdx["schema_name"] != wantIdx["schema_name"] ||
								gotIdx["table_name"] != wantIdx["table_name"] ||
								gotIdx["is_unique"] != wantIdx["is_unique"] ||
								gotIdx["is_primary"] != wantIdx["is_primary"] {
								t.Errorf("Mismatch in fields for index %s: got %v, want %v", wantIdx["index_name"], gotIdx, wantIdx)
							}
							break
						}
					}
					if !found {
						t.Errorf("Expected index %s not found in results", wantIdx["index_name"])
					}
				}
			} else {
				wantObj := []any{}
				for _, item := range tc.want {
					wantObj = append(wantObj, item)
				}
				if diff := cmp.Diff(wantObj, gotObj); diff != "" {
					t.Errorf("Unexpected result mismatch (-want +got):\n%s", diff)
				}
			}

		})
	}
}

// cleanupOldSchemas cleans up schemas that were created more than 1 hour ago
func cleanupMCPOldSchemas(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	rows, err := pool.Query(ctx, "SELECT schema_name FROM information_schema.schemata WHERE schema_name LIKE 'test_proc_%'")
	if err != nil {
		return
	}
	defer rows.Close()

	oneHourAgo := time.Now().Add(-1 * time.Hour).Unix()

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}

		parts := strings.Split(name, "_")
		if len(parts) < 3 {
			continue
		}

		timestamp, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			continue
		}

		if timestamp < oneHourAgo {
			_, err := pool.Exec(ctx, fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", name))
			if err == nil {
				t.Logf("Cleaned up schema: %s", name)
			}
		}
	}
}

// RunMCPPostgresListStoredProcedureTest tests the list_stored_procedure tool via MCP.
func RunMCPPostgresListStoredProcedureTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	cleanupMCPOldSchemas(t, ctx, pool)

	type storedProcedureDetails struct {
		SchemaName  string `json:"schema_name"`
		Name        string `json:"name"`
		Owner       string `json:"owner"`
		Language    string `json:"language"`
		Definition  string `json:"definition"`
		Description any    `json:"description"`
	}

	now := time.Now().Unix()
	testSchemaName := fmt.Sprintf("test_proc_%d_%s", now, strings.ReplaceAll(uuid.New().String(), "-", "")[:8])
	createSchemaStmt := fmt.Sprintf("CREATE SCHEMA %s", testSchemaName)
	if _, err := pool.Exec(ctx, createSchemaStmt); err != nil {
		t.Fatalf("unable to create test schema: %v", err)
	}
	defer func() {
		dropSchemaStmt := fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", testSchemaName)
		if _, err := pool.Exec(ctx, dropSchemaStmt); err != nil {
			t.Logf("warning: unable to drop test schema: %v", err)
		}
	}()

	proc1Name := "test_proc_1_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	createProc1Stmt := fmt.Sprintf(`
        CREATE PROCEDURE %s.%s(p_count INT)
        LANGUAGE plpgsql
        AS $$
        BEGIN
            -- We don't need the table to exist to create the procedure
            COMMIT;
        END;
        $$
    `, testSchemaName, proc1Name)

	if _, err := pool.Exec(ctx, createProc1Stmt); err != nil {
		t.Fatalf("unable to create test procedure 1: %v", err)
	}

	commentStmt := fmt.Sprintf("COMMENT ON PROCEDURE %s.%s(INT) IS 'Test procedure that inserts a record'", testSchemaName, proc1Name)
	if _, err := pool.Exec(ctx, commentStmt); err != nil {
		t.Logf("warning: unable to add comment to procedure: %v", err)
	}

	proc2Name := "test_proc_2_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	createProc2Stmt := fmt.Sprintf(`
        CREATE PROCEDURE %s.%s()
        LANGUAGE plpgsql
        AS $$
        DECLARE
            v_count INT;
        BEGIN
            v_count := 0;
        END;
        $$
    `, testSchemaName, proc2Name)

	if _, err := pool.Exec(ctx, createProc2Stmt); err != nil {
		t.Fatalf("unable to create test procedure 2: %v", err)
	}

	invokeTcs := []struct {
		name           string
		args           map[string]any
		wantStatusCode int
		shouldHaveData bool
		expectedCount  int
		filterByRole   string
		filterBySchema string
	}{
		{
			name:           "list stored procedures with no arguments (default limit 20)",
			args:           map[string]any{},
			wantStatusCode: http.StatusOK,
			shouldHaveData: false,
		},
		{
			name:           "list stored procedures filtering by specific schema",
			args:           map[string]any{"schema_name": testSchemaName},
			wantStatusCode: http.StatusOK,
			shouldHaveData: true,
			expectedCount:  2,
			filterBySchema: testSchemaName,
		},
		{
			name:           "list stored procedures filtering by procedure owner (postgres)",
			args:           map[string]any{"role_name": "postgres"},
			wantStatusCode: http.StatusOK,
			shouldHaveData: false,
		},
		{
			name:           "list stored procedures with custom limit",
			args:           map[string]any{"limit": 5},
			wantStatusCode: http.StatusOK,
			shouldHaveData: false,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			statusCode, mcpResp, err := InvokeMCPTool(t, ctx, "list_stored_procedure", tc.args, nil)
			if err != nil {
				t.Fatalf("native error executing list_stored_procedure: %s", err)
			}
			if statusCode != tc.wantStatusCode {
				t.Fatalf("wrong status code: got %d, want %d", statusCode, tc.wantStatusCode)
			}
			if tc.wantStatusCode != http.StatusOK {
				return
			}
			if mcpResp.Result.IsError {
				t.Fatalf("list_stored_procedure returned error result: %v", mcpResp.Result)
			}
			var gotObj []storedProcedureDetails
			got := GetMCPResultText(t, mcpResp)
			for _, item := range got {
				if m, ok := item.(map[string]any); ok {
					proc := storedProcedureDetails{}
					if s, ok := m["schema_name"].(string); ok {
						proc.SchemaName = s
					}
					if p, ok := m["name"].(string); ok {
						proc.Name = p
					}
					if r, ok := m["owner"].(string); ok {
						proc.Owner = r
					}

					gotObj = append(gotObj, proc)
				}
			}

			if tc.shouldHaveData {
				if len(gotObj) != tc.expectedCount {
					t.Errorf("expected %d procedures, got %d", tc.expectedCount, len(gotObj))
				}
				for _, proc := range gotObj {
					if tc.filterBySchema != "" && proc.SchemaName != tc.filterBySchema {
						t.Errorf("expected schema %s, got %s", tc.filterBySchema, proc.SchemaName)
					}
					if tc.filterByRole != "" && proc.Owner != tc.filterByRole {
						t.Errorf("expected owner %s, got %s", tc.filterByRole, proc.Owner)
					}
				}
			}
		})
	}
}

// RunMCPPostgresDatabaseOverviewTest tests the database_overview tool via MCP.
func RunMCPPostgresDatabaseOverviewTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	statusCode, mcpResp, err := InvokeMCPTool(t, ctx, "database_overview", map[string]any{}, nil)
	if err != nil {
		t.Fatalf("native error executing database_overview: %s", err)
	}
	if statusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", statusCode)
	}
	if mcpResp.Result.IsError {
		t.Fatalf("database_overview returned error result: %v", mcpResp.Result)
	}
	gotObj := GetMCPResultText(t, mcpResp)

	if len(gotObj) != 1 {
		t.Fatalf("Expected exactly one row in the result, got %d", len(gotObj))
	}

	resultRow, ok := gotObj[0].(map[string]any)
	if !ok {
		t.Fatalf("Expected result to be a map, got %T", gotObj[0])
	}

	expectedKeys := []string{
		"pg_version",
		"is_replica",
		"uptime",
		"max_connections",
		"current_connections",
		"active_connections",
		"pct_connections_used",
	}

	for _, key := range expectedKeys {
		if _, ok := resultRow[key]; !ok {
			t.Errorf("Missing expected key in result: %s", key)
		}
	}
}

// CreateAndLockPostgresTable creates a table and locks it in a transaction, and returns a cleanup function.
func CreateAndLockMCPPostgresTable(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tableName string) func() {
	_, err := pool.Exec(ctx, fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (id INT PRIMARY KEY)", tableName))
	if err != nil {
		t.Fatalf("unable to create table: %s", err)
	}

	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		t.Fatalf("unable to create transaction: %s", err)
	}
	if _, err := tx.Exec(ctx, fmt.Sprintf("LOCK TABLE %s IN ACCESS EXCLUSIVE MODE", tableName)); err != nil {
		t.Fatalf("unable to acquire lock: %s", err)
	}

	return func() {
		if err := tx.Rollback(ctx); err != nil {
			t.Fatalf("unable to rollback transaction: %s", err)
		}
		if _, err := pool.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", tableName)); err != nil {
			t.Fatalf("unable to drop table: %s", err)
		}
	}
}

// RunMCPPostgresListLocksTest tests the list_locks tool via MCP.
func RunMCPPostgresListLocksTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	cleanup := CreateAndLockMCPPostgresTable(t, ctx, pool, "test_postgres_list_locks_table")
	defer cleanup()

	statusCode, mcpResp, err := InvokeMCPTool(t, ctx, "list_locks", map[string]any{}, nil)
	if err != nil {
		t.Fatalf("native error executing list_locks: %s", err)
	}
	if statusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", statusCode)
	}
	if mcpResp.Result.IsError {
		t.Fatalf("list_locks returned error result: %v", mcpResp.Result)
	}
	gotObj := GetMCPResultText(t, mcpResp)

	if len(gotObj) == 0 {
		t.Errorf("Expected to find locks, got none")
	}

}

// RunMCPPostgresLongRunningTransactionsTest tests the long_running_transactions tool via MCP.
func RunMCPPostgresLongRunningTransactionsTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	statusCode, mcpResp, err := InvokeMCPTool(t, ctx, "long_running_transactions", map[string]any{}, nil)
	if err != nil {
		t.Fatalf("native error executing long_running_transactions: %s", err)
	}
	if statusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", statusCode)
	}
	if mcpResp.Result.IsError {
		t.Fatalf("long_running_transactions returned error result: %v", mcpResp.Result)
	}
}

// RunMCPPostgresReplicationStatsTest tests the replication_stats tool via MCP.
func RunMCPPostgresReplicationStatsTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	statusCode, mcpResp, err := InvokeMCPTool(t, ctx, "replication_stats", map[string]any{}, nil)
	if err != nil {
		t.Fatalf("native error executing replication_stats: %s", err)
	}
	if statusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", statusCode)
	}
	if mcpResp.Result.IsError {
		t.Fatalf("replication_stats returned error result: %v", mcpResp.Result)
	}
}

// RunMCPPostgresGetColumnCardinalityTest tests the get_column_cardinality tool via MCP.
func RunMCPPostgresGetColumnCardinalityTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	schemaName := "testschema_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableName := "table1_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	cleanup := setupMCPPostgresSchemas(t, ctx, pool, schemaName)
	defer cleanup()

	createTableStmt := fmt.Sprintf(`
		CREATE TABLE %s.%s (
			id SERIAL PRIMARY KEY,
			email VARCHAR(100) UNIQUE,
			name VARCHAR(50),
			status VARCHAR(20),
			created_at TIMESTAMP
		)
	`, schemaName, tableName)

	if _, err := pool.Exec(ctx, createTableStmt); err != nil {
		t.Fatalf("unable to create table: %s", err)
	}

	insertStmt := fmt.Sprintf(`
		INSERT INTO %s.%s (email, name, status, created_at) VALUES
		('user1@example.com', 'Alice', 'active', NOW()),
		('user2@example.com', 'Bob', 'inactive', NOW()),
		('user3@example.com', 'Charlie', 'active', NOW()),
		('user4@example.com', 'David', 'active', NOW()),
		('user5@example.com', 'Eve', 'inactive', NOW()),
		('user6@example.com', 'Frank', 'active', NOW()),
		('user7@example.com', 'Grace', 'inactive', NOW()),
		('user8@example.com', 'Henry', 'active', NOW()),
		('user9@example.com', 'Ivy', 'active', NOW()),
		('user10@example.com', 'Jack', 'inactive', NOW())
	`, schemaName, tableName)

	if _, err := pool.Exec(ctx, insertStmt); err != nil {
		t.Fatalf("unable to insert data: %s", err)
	}

	analyzeStmt := fmt.Sprintf(`ANALYZE %s.%s`, schemaName, tableName)
	if _, err := pool.Exec(ctx, analyzeStmt); err != nil {
		t.Fatalf("unable to run ANALYZE: %s", err)
	}

	invokeTcs := []struct {
		name           string
		args           map[string]any
		wantStatusCode int
		shouldHaveData bool
	}{
		{
			name: "get column cardinality",
			args: map[string]any{
				"schema_name": schemaName,
				"table_name":  tableName,
				"column_name": "status",
			},
			wantStatusCode: http.StatusOK,
			shouldHaveData: true,
		},
		{
			name: "get column cardinality for non-existent table",
			args: map[string]any{
				"schema_name": schemaName,
				"table_name":  "non_existent_table",
				"column_name": "status",
			},
			wantStatusCode: http.StatusOK,
			shouldHaveData: false,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			statusCode, mcpResp, err := InvokeMCPTool(t, ctx, "get_column_cardinality", tc.args, nil)
			if err != nil {
				t.Fatalf("native error executing get_column_cardinality: %s", err)
			}
			if statusCode != tc.wantStatusCode {
				t.Fatalf("wrong status code: got %d, want %d", statusCode, tc.wantStatusCode)
			}
			if tc.wantStatusCode != http.StatusOK {
				return
			}
			if mcpResp.Result.IsError {
				t.Fatalf("get_column_cardinality returned error result: %v", mcpResp.Result)
			}
			gotObj := GetMCPResultText(t, mcpResp)
			if len(gotObj) == 0 {
				if tc.shouldHaveData {
					t.Fatalf("get_column_cardinality returned empty content field")
				}
				t.Logf("DEBUG: get_column_cardinality returned empty content as expected for non-existent table")
				return
			}

			if tc.shouldHaveData {
				if len(gotObj) == 0 {
					t.Errorf("Expected to find cardinality data, got none")
				}
			}
		})
	}
}

// RunMCPPostgresListTableStatsTest tests the list_table_stats tool via MCP.
func RunMCPPostgresListTableStatsTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	testTableName := "test_list_table_stats_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	createTableStmt := fmt.Sprintf(`
        CREATE TABLE %s (
            id SERIAL PRIMARY KEY,
            name VARCHAR(100),
            email VARCHAR(100)
        )
    `, testTableName)

	if _, err := pool.Exec(ctx, createTableStmt); err != nil {
		t.Fatalf("unable to create test table: %s", err)
	}
	defer func() {
		dropTableStmt := fmt.Sprintf("DROP TABLE IF EXISTS %s", testTableName)
		if _, err := pool.Exec(ctx, dropTableStmt); err != nil {
			t.Logf("warning: unable to drop test table: %v", err)
		}
	}()

	// Insert some data to generate statistics
	insertStmt := fmt.Sprintf(`
        INSERT INTO %s (name, email) VALUES
        ('Alice', 'alice@example.com'),
        ('Bob', 'bob@example.com'),
        ('Charlie', 'charlie@example.com'),
        ('David', 'david@example.com'),
        ('Eve', 'eve@example.com')
    `, testTableName)

	if _, err := pool.Exec(ctx, insertStmt); err != nil {
		t.Fatalf("unable to insert test data: %s", err)
	}

	// Run some sequential scans to generate statistics
	for i := 0; i < 3; i++ {
		selectStmt := fmt.Sprintf("SELECT * FROM %s WHERE name = 'Alice'", testTableName)
		if _, err := pool.Exec(ctx, selectStmt); err != nil {
			t.Logf("warning: unable to execute select: %v", err)
		}
	}

	// Run ANALYZE to update statistics
	analyzeStmt := fmt.Sprintf("ANALYZE %s", testTableName)
	if _, err := pool.Exec(ctx, analyzeStmt); err != nil {
		t.Fatalf("unable to run ANALYZE: %s", err)
	}

	invokeTcs := []struct {
		name           string
		arguments      map[string]any
		wantStatusCode int
		shouldHaveData bool
		filterTable    bool
	}{
		{
			name:           "list table stats with no arguments (default limit)",
			arguments:      map[string]any{},
			wantStatusCode: http.StatusOK,
			shouldHaveData: false, // may or may not have data depending on what's in the database
		},
		{
			name:           "list table stats with default limit",
			arguments:      map[string]any{"schema_name": "public"},
			wantStatusCode: http.StatusOK,
			shouldHaveData: false,
		},
		{
			name:           "list table stats filtering by specific table",
			arguments:      map[string]any{"table_name": testTableName},
			wantStatusCode: http.StatusOK,
			shouldHaveData: true,
			filterTable:    true,
		},
		{
			name:           "list table stats with custom limit",
			arguments:      map[string]any{"limit": 10},
			wantStatusCode: http.StatusOK,
			shouldHaveData: false,
		},
		{
			name:           "list table stats sorted by size",
			arguments:      map[string]any{"sort_by": "size", "limit": 5},
			wantStatusCode: http.StatusOK,
			shouldHaveData: false,
		},
		{
			name:           "list table stats sorted by seq_scan",
			arguments:      map[string]any{"sort_by": "seq_scan", "limit": 5},
			wantStatusCode: http.StatusOK,
			shouldHaveData: false,
		},
		{
			name:           "list table stats sorted by idx_scan",
			arguments:      map[string]any{"sort_by": "idx_scan", "limit": 5},
			wantStatusCode: http.StatusOK,
			shouldHaveData: false,
		},
		{
			name:           "list table stats sorted by dead_rows",
			arguments:      map[string]any{"sort_by": "dead_rows", "limit": 5},
			wantStatusCode: http.StatusOK,
			shouldHaveData: false,
		},
		{
			name:           "list table stats with non-existent table filter",
			arguments:      map[string]any{"table_name": "non_existent_table_xyz"},
			wantStatusCode: http.StatusOK,
			shouldHaveData: false,
		},
		{
			name:           "list table stats with non-existent schema filter",
			arguments:      map[string]any{"schema_name": "non_existent_schema_xyz"},
			wantStatusCode: http.StatusOK,
			shouldHaveData: false,
		},
		{
			name:           "list table stats with owner filter",
			arguments:      map[string]any{"owner": "postgres"},
			wantStatusCode: http.StatusOK,
			shouldHaveData: false,
		},
	}

	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			statusCode, mcpResp, err := InvokeMCPTool(t, ctx, "list_table_stats", tc.arguments, nil)
			if err != nil {
				t.Fatalf("native error executing list_table_stats: %s", err)
			}
			if statusCode != tc.wantStatusCode {
				t.Fatalf("wrong status code: got %d, want %d", statusCode, tc.wantStatusCode)
			}
			if tc.wantStatusCode != http.StatusOK {
				return
			}
			if mcpResp.Result.IsError {
				t.Fatalf("list_table_stats returned error result: %v", mcpResp.Result)
			}

			gotObj := GetMCPResultText(t, mcpResp)

			// Verify expected data presence
			if tc.shouldHaveData {
				if len(gotObj) == 0 {
					t.Fatalf("expected data but got empty result")
				}

				// Verify the test table is in results
				found := false
				for _, rowObj := range gotObj {
					row, ok := rowObj.(map[string]any)
					if !ok {
						continue
					}
					if row["table_name"] == testTableName {
						found = true
						// Verify expected fields are present
						if row["schema_name"] == nil || row["schema_name"] == "" {
							t.Errorf("schema_name should not be empty")
						}
						if row["owner"] == nil || row["owner"] == "" {
							t.Errorf("owner should not be empty")
						}
						if row["total_size_bytes"] == nil {
							t.Errorf("total_size_bytes should not be null")
						}
						if row["live_rows"] == nil {
							t.Errorf("live_rows should not be null")
						}

						break
					}
				}

				if !found {
					t.Errorf("test table %s not found in results", testTableName)
				}
			} else if tc.filterTable {
				// For filtered queries that shouldn't find anything
				if len(gotObj) != 0 {
					t.Logf("warning: expected no data but got: %v", len(gotObj))
				}
			}
		})
	}
}

// setupPostgresPublicationTable creates a table and a publication for it.
func setupMCPPostgresPublicationTable(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tableName string, pubName string) func(t *testing.T) {
	t.Helper()
	if _, err := pool.Exec(ctx, fmt.Sprintf("DROP PUBLICATION IF EXISTS %s;", pubName)); err != nil {
		t.Errorf("unable to drop publication %s: %v", pubName, err)
	}
	if _, err := pool.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s;", tableName)); err != nil {
		t.Errorf("unable to drop table %s: %v", tableName, err)
	}
	createTableStmt := fmt.Sprintf("CREATE TABLE %s (id SERIAL PRIMARY KEY, name TEXT);", tableName)
	if _, err := pool.Exec(ctx, createTableStmt); err != nil {
		t.Fatalf("unable to create table %s: %v", tableName, err)
	}

	createPubStmt := fmt.Sprintf("CREATE PUBLICATION %s FOR TABLE %s;", pubName, tableName)
	if _, err := pool.Exec(ctx, createPubStmt); err != nil {
		if _, dropErr := pool.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s;", tableName)); dropErr != nil {
			t.Errorf("unable to drop table after failing to create publication: %v", dropErr)
		}
		t.Fatalf("unable to create publication %s: %v", pubName, err)
	}

	return func(t *testing.T) {
		t.Helper()
		if _, err := pool.Exec(ctx, fmt.Sprintf("DROP PUBLICATION IF EXISTS %s;", pubName)); err != nil {
			t.Errorf("unable to drop publication %s: %v", pubName, err)
		}
		if _, err := pool.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s;", tableName)); err != nil {
			t.Errorf("unable to drop table %s: %v", tableName, err)
		}
	}
}

// RunMCPPostgresListPublicationTablesTest tests the list_publication_tables tool via MCP.
func RunMCPPostgresListPublicationTablesTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	table1Name := "pub_table_1"
	pub1Name := "pub_1"

	cleanup := setupMCPPostgresPublicationTable(t, ctx, pool, table1Name, pub1Name)
	defer cleanup(t)

	var currentUser string
	err := pool.QueryRow(ctx, "SELECT current_user;").Scan(&currentUser)
	if err != nil {
		t.Fatalf("unable to fetch current user: %v", err)
	}

	statusCode, mcpResp, err := InvokeMCPTool(t, ctx, "list_publication_tables", map[string]any{}, nil)
	if err != nil {
		t.Fatalf("native error executing list_publication_tables: %s", err)
	}
	if statusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", statusCode)
	}
	if mcpResp.Result.IsError {
		t.Fatalf("list_publication_tables returned error result: %v", mcpResp.Result)
	}
	gotObj := GetMCPResultText(t, mcpResp)

	found := false
	for _, rowObj := range gotObj {
		row, ok := rowObj.(map[string]any)
		if !ok {
			continue
		}
		if row["publication_name"] == pub1Name && row["table_name"] == table1Name {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected to find publication %s for table %s", pub1Name, table1Name)
	}
}

// RunMCPPostgresListTableSpacesTest tests the list_tablespaces tool via MCP.
func RunMCPPostgresListTableSpacesTest(t *testing.T, ctx context.Context) {
	statusCode, mcpResp, err := InvokeMCPTool(t, ctx, "list_tablespaces", map[string]any{}, nil)
	if err != nil {
		t.Fatalf("native error executing list_tablespaces: %s", err)
	}
	if statusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", statusCode)
	}
	if mcpResp.Result.IsError {
		t.Fatalf("list_tablespaces returned error result: %v", mcpResp.Result)
	}
}

// RunMCPPostgresListPgSettingsTest tests the list_pg_settings tool via MCP.
func RunMCPPostgresListPgSettingsTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	targetSetting := "maintenance_work_mem"
	var name, setting, unit, shortDesc, source, contextVal string

	err := pool.QueryRow(ctx, `
		SELECT name, setting, unit, short_desc, source, context 
		FROM pg_settings 
		WHERE name = $1
	`, targetSetting).Scan(&name, &setting, &unit, &shortDesc, &source, &contextVal)

	if err != nil {
		t.Fatalf("Setup failed: could not fetch postgres setting '%s': %v", targetSetting, err)
	}

	requiresRestart := "No"
	switch contextVal {
	case "postmaster":
		requiresRestart = "Yes"
	case "sighup":
		requiresRestart = "No (Reload sufficient)"
	}

	expectedObject := map[string]interface{}{
		"name":             name,
		"current_value":    setting,
		"unit":             unit,
		"short_desc":       shortDesc,
		"source":           source,
		"requires_restart": requiresRestart,
	}

	statusCode, mcpResp, err := InvokeMCPTool(t, ctx, "list_pg_settings", map[string]any{"setting_name": targetSetting}, nil)
	if err != nil {
		t.Fatalf("native error executing list_pg_settings: %s", err)
	}
	if statusCode != http.StatusOK {
		t.Fatalf("wrong status code: got %d, want %d", statusCode, http.StatusOK)
	}
	if mcpResp.Result.IsError {
		t.Fatalf("list_pg_settings returned error result: %v", mcpResp.Result)
	}
	gotObj := GetMCPResultText(t, mcpResp)

	if len(gotObj) != 1 {
		t.Fatalf("Expected exactly one row in the result, got %d", len(gotObj))
	}

	resultRow, ok := gotObj[0].(map[string]any)
	if !ok {
		t.Fatalf("Expected result to be a map, got %T", gotObj[0])
	}

	if diff := cmp.Diff(expectedObject, resultRow); diff != "" {
		t.Errorf("Unexpected result mismatch (-want +got):\n%s", diff)
	}

}

// setUpDatabase creates a database and a role, and returns a cleanup function.
func setUpMCPDatabase(t *testing.T, ctx context.Context, pool *pgxpool.Pool, dbName, dbOwner string) func() {
	_, err := pool.Exec(ctx, fmt.Sprintf("CREATE ROLE %s LOGIN PASSWORD 'password';", dbOwner))
	if err != nil {
		_, _ = pool.Exec(ctx, fmt.Sprintf("DROP ROLE %s;", dbOwner))
		t.Fatalf("failed to create %s: %v", dbOwner, err)
	}
	_, err = pool.Exec(ctx, fmt.Sprintf("GRANT %s TO current_user;", dbOwner))
	if err != nil {
		t.Fatalf("failed to grant %s to current_user: %v", dbOwner, err)
	}
	_, err = pool.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s OWNER %s;", dbName, dbOwner))
	if err != nil {
		t.Fatalf("failed to create %s: %v", dbName, err)
	}
	return func() {
		_, _ = pool.Exec(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s;", dbName))
		_, _ = pool.Exec(ctx, fmt.Sprintf("DROP ROLE IF EXISTS %s;", dbOwner))
	}
}

// RunMCPPostgresListDatabaseStatsTest tests the list_database_stats tool via MCP.
func RunMCPPostgresListDatabaseStatsTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	dbName1 := "test_db_stats_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	dbOwner1 := "test_user_" + strings.ReplaceAll(uuid.NewString(), "-", "")

	cleanup1 := setUpMCPDatabase(t, ctx, pool, dbName1, dbOwner1)
	defer cleanup1()

	statusCode, mcpResp, err := InvokeMCPTool(t, ctx, "list_database_stats", map[string]any{"database_name": dbName1}, nil)
	if err != nil {
		t.Fatalf("native error executing list_database_stats: %s", err)
	}
	if statusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", statusCode)
	}
	if mcpResp.Result.IsError {
		t.Fatalf("list_database_stats returned error result: %v", mcpResp.Result)
	}
	gotObj := GetMCPResultText(t, mcpResp)

	found := false
	for _, rowObj := range gotObj {
		row, ok := rowObj.(map[string]any)
		if !ok {
			continue
		}
		if row["database_name"] == dbName1 {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected to find stats for database %s", dbName1)
	}
}

// setupPostgresRoles creates test roles and returns a cleanup function.
func setupMCPPostgresRoles(t *testing.T, ctx context.Context, pool *pgxpool.Pool) (string, string, string, func(t *testing.T)) {
	t.Helper()
	suffix := strings.ReplaceAll(uuid.New().String(), "-", "")

	adminUser := "test_role_admin_" + suffix
	superUser := "test_role_super_" + suffix
	normalUser := "test_role_normal_" + suffix

	createAdminStmt := fmt.Sprintf("CREATE ROLE %s NOLOGIN;", adminUser)
	if _, err := pool.Exec(ctx, createAdminStmt); err != nil {
		t.Fatalf("unable to create role %s: %v", adminUser, err)
	}

	createSuperUserStmt := fmt.Sprintf("CREATE ROLE %s LOGIN CREATEDB;", superUser)
	if _, err := pool.Exec(ctx, createSuperUserStmt); err != nil {
		t.Fatalf("unable to create role %s: %v", superUser, err)
	}

	createNormalUserStmt := fmt.Sprintf("CREATE ROLE %s LOGIN;", normalUser)
	if _, err := pool.Exec(ctx, createNormalUserStmt); err != nil {
		t.Fatalf("unable to create role %s: %v", normalUser, err)
	}

	// Establish Relationships (Admin -> Superuser -> Normal)
	if _, err := pool.Exec(ctx, fmt.Sprintf("GRANT %s TO %s;", adminUser, superUser)); err != nil {
		t.Fatalf("unable to grant %s to %s: %v", adminUser, superUser, err)
	}
	if _, err := pool.Exec(ctx, fmt.Sprintf("GRANT %s TO %s;", superUser, normalUser)); err != nil {
		t.Fatalf("unable to grant %s to %s: %v", superUser, normalUser, err)
	}

	return adminUser, superUser, normalUser, func(t *testing.T) {
		t.Helper()
		_, _ = pool.Exec(ctx, fmt.Sprintf("DROP ROLE IF EXISTS %s;", normalUser))
		_, _ = pool.Exec(ctx, fmt.Sprintf("DROP ROLE IF EXISTS %s;", superUser))
		_, _ = pool.Exec(ctx, fmt.Sprintf("DROP ROLE IF EXISTS %s;", adminUser))
	}
}

// RunMCPPostgresListRolesTest tests the list_roles tool via MCP.
func RunMCPPostgresListRolesTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	adminUser, _, _, cleanup := setupMCPPostgresRoles(t, ctx, pool)
	defer cleanup(t)

	statusCode, mcpResp, err := InvokeMCPTool(t, ctx, "list_roles", map[string]any{"role_name": "test_role_"}, nil)
	if err != nil {
		t.Fatalf("native error executing list_roles: %s", err)
	}
	if statusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", statusCode)
	}
	if mcpResp.Result.IsError {
		t.Fatalf("list_roles returned error result: %v", mcpResp.Result)
	}
	gotObj := GetMCPResultText(t, mcpResp)

	found := false
	for _, rowObj := range gotObj {
		row, ok := rowObj.(map[string]any)
		if !ok {
			continue
		}
		if row["role_name"] == adminUser {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected to find role %s", adminUser)
	}
}

// RunMCPStatementToolsTest tests statement tools via MCP.
func RunMCPStatementToolsTest(t *testing.T, ctx context.Context, tools map[string]string) {
	for toolName, paramBody := range tools {
		t.Run(toolName, func(t *testing.T) {
			var args map[string]any
			if paramBody != "{}" {
				if err := json.Unmarshal([]byte(paramBody), &args); err != nil {
					t.Fatalf("failed to unmarshal paramBody: %v", err)
				}
			}
			statusCode, mcpResp, err := InvokeMCPTool(t, ctx, toolName, args, nil)
			if err != nil {
				t.Fatalf("native error executing %s: %s", toolName, err)
			}
			if statusCode != http.StatusOK {
				t.Fatalf("response status code is not 200, got %d", statusCode)
			}
			if mcpResp.Result.IsError {
				t.Fatalf("%s returned error result: %v", toolName, mcpResp.Result)
			}
		})
	}
}
