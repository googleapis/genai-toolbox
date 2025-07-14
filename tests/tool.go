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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/googleapis/genai-toolbox/internal/server/mcp/jsonrpc"
)

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
func RunToolInvokeTest(t *testing.T, select1Want, invokeParamWant, invokeIdParamWant, nullString string, supportNullParam, supportsArray bool) {
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
			want:          select1Want,
			isErr:         false,
		},
		{
			name:          "invoke my-param-tool",
			api:           "http://127.0.0.1:5000/api/tool/my-param-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"id": 3, "name": "Alice"}`)),
			want:          invokeParamWant,
			isErr:         false,
		},
		{
			name:          "invoke my-param-tool-id with nil response",
			api:           "http://127.0.0.1:5000/api/tool/my-param-tool-id/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"id": 4}`)),
			want:          invokeIdParamWant,
			isErr:         false,
		},
		{
			name:          "invoke my-param-tool-name with nil response",
			api:           "http://127.0.0.1:5000/api/tool/my-param-tool-name/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			want:          nullString,
			isErr:         !supportNullParam,
		},
		{
			name:          "Invoke my-param-tool without parameters",
			api:           "http://127.0.0.1:5000/api/tool/my-param-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			isErr:         true,
		},
		{
			name:          "Invoke my-param-tool with insufficient parameters",
			api:           "http://127.0.0.1:5000/api/tool/my-param-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"id": 1}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-array-tool",
			api:           "http://127.0.0.1:5000/api/tool/my-array-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"idArray": [1,2,3], "nameArray": ["Alice", "Sid", "RandomName"], "cmdArray": ["HGETALL", "row3"]}`)),
			want:          invokeParamWant,
			isErr:         !supportsArray,
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
			want:          select1Want,
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

			if got != tc.want {
				t.Fatalf("unexpected value: got %q, want %q", got, tc.want)
			}
		})
	}
}

// TemplateParameterTestConfig represents the various configuration options for template parameter tests.
type TemplateParameterTestConfig struct {
	ignoreDdl      bool
	ignoreInsert   bool
	selectAllWant  string
	select1Want    string
	nameFieldArray string
	nameColFilter  string
	createColArray string
}

type Option func(*TemplateParameterTestConfig)

// WithIgnoreDdl is the option function to configure ignoreDdl.
func WithIgnoreDdl() Option {
	return func(c *TemplateParameterTestConfig) {
		c.ignoreDdl = true
	}
}

// WithIgnoreInsert is the option function to configure ignoreInsert.
func WithIgnoreInsert() Option {
	return func(c *TemplateParameterTestConfig) {
		c.ignoreInsert = true
	}
}

// WithSelectAllWant is the option function to configure selectAllWant.
func WithSelectAllWant(s string) Option {
	return func(c *TemplateParameterTestConfig) {
		c.selectAllWant = s
	}
}

// WithSelect1Want is the option function to configure select1Want.
func WithSelect1Want(s string) Option {
	return func(c *TemplateParameterTestConfig) {
		c.select1Want = s
	}
}

// WithReplaceNameFieldArray is the option function to configure replaceNameFieldArray.
func WithReplaceNameFieldArray(s string) Option {
	return func(c *TemplateParameterTestConfig) {
		c.nameFieldArray = s
	}
}

// WithReplaceNameColFilter is the option function to configure replaceNameColFilter.
func WithReplaceNameColFilter(s string) Option {
	return func(c *TemplateParameterTestConfig) {
		c.nameColFilter = s
	}
}

// WithCreateColArray is the option function to configure replaceNameColFilter.
func WithCreateColArray(s string) Option {
	return func(c *TemplateParameterTestConfig) {
		c.createColArray = s
	}
}

// NewTemplateParameterTestConfig creates a new TemplateParameterTestConfig instances with options.
func NewTemplateParameterTestConfig(options ...Option) *TemplateParameterTestConfig {
	templateParamTestOption := &TemplateParameterTestConfig{
		ignoreDdl:      false,
		ignoreInsert:   false,
		selectAllWant:  "[{\"age\":21,\"id\":1,\"name\":\"Alex\"},{\"age\":100,\"id\":2,\"name\":\"Alice\"}]",
		select1Want:    "[{\"age\":21,\"id\":1,\"name\":\"Alex\"}]",
		nameFieldArray: `["name"]`,
		nameColFilter:  "name",
		createColArray: `["id INT","name VARCHAR(20)","age INT"]`,
	}

	// Apply provided options
	for _, option := range options {
		option(templateParamTestOption)
	}

	return templateParamTestOption
}

// RunToolInvokeWithTemplateParameters runs tool invoke test cases with template parameters.
func RunToolInvokeWithTemplateParameters(t *testing.T, tableName string, config *TemplateParameterTestConfig) {
	selectOnlyNamesWant := "[{\"name\":\"Alex\"},{\"name\":\"Alice\"}]"

	// Test tool invoke endpoint
	invokeTcs := []struct {
		name          string
		ddl           bool
		insert        bool
		api           string
		requestHeader map[string]string
		requestBody   io.Reader
		want          string
		isErr         bool
	}{
		{
			name:          "invoke create-table-templateParams-tool",
			ddl:           true,
			api:           "http://127.0.0.1:5000/api/tool/create-table-templateParams-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf(`{"tableName": "%s", "columns":%s}`, tableName, config.createColArray))),
			want:          "null",
			isErr:         false,
		},
		{
			name:          "invoke insert-table-templateParams-tool",
			insert:        true,
			api:           "http://127.0.0.1:5000/api/tool/insert-table-templateParams-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf(`{"tableName": "%s", "columns":["id","name","age"], "values":"1, 'Alex', 21"}`, tableName))),
			want:          "null",
			isErr:         false,
		},
		{
			name:          "invoke insert-table-templateParams-tool",
			insert:        true,
			api:           "http://127.0.0.1:5000/api/tool/insert-table-templateParams-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf(`{"tableName": "%s", "columns":["id","name","age"], "values":"2, 'Alice', 100"}`, tableName))),
			want:          "null",
			isErr:         false,
		},
		{
			name:          "invoke select-templateParams-tool",
			api:           "http://127.0.0.1:5000/api/tool/select-templateParams-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf(`{"tableName": "%s"}`, tableName))),
			want:          config.selectAllWant,
			isErr:         false,
		},
		{
			name:          "invoke select-templateParams-combined-tool",
			api:           "http://127.0.0.1:5000/api/tool/select-templateParams-combined-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf(`{"id": 1, "tableName": "%s"}`, tableName))),
			want:          config.select1Want,
			isErr:         false,
		},
		{
			name:          "invoke select-fields-templateParams-tool",
			api:           "http://127.0.0.1:5000/api/tool/select-fields-templateParams-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf(`{"tableName": "%s", "fields":%s}`, tableName, config.nameFieldArray))),
			want:          selectOnlyNamesWant,
			isErr:         false,
		},
		{
			name:          "invoke select-filter-templateParams-combined-tool",
			api:           "http://127.0.0.1:5000/api/tool/select-filter-templateParams-combined-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf(`{"name": "Alex", "tableName": "%s", "columnFilter": "%s"}`, tableName, config.nameColFilter))),
			want:          config.select1Want,
			isErr:         false,
		},
		{
			name:          "invoke drop-table-templateParams-tool",
			ddl:           true,
			api:           "http://127.0.0.1:5000/api/tool/drop-table-templateParams-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf(`{"tableName": "%s"}`, tableName))),
			want:          "null",
			isErr:         false,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			// if test case is DDL and source does not ignore ddl test cases
			ddlAllow := !tc.ddl || (tc.ddl && !config.ignoreDdl)
			// if test case is insert statement and source does not ignore insert test cases
			insertAllow := !tc.insert || (tc.insert && !config.ignoreInsert)
			if ddlAllow && insertAllow {
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

				if got != tc.want {
					t.Fatalf("unexpected value: got %q, want %q", got, tc.want)
				}
			}
		})
	}
}

func RunExecuteSqlToolInvokeTest(t *testing.T, createTableStatement string, select1Want string) {
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
			name:          "invoke my-exec-sql-tool",
			api:           "http://127.0.0.1:5000/api/tool/my-exec-sql-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"sql":"SELECT 1"}`)),
			want:          select1Want,
			isErr:         false,
		},
		{
			name:          "invoke my-exec-sql-tool create table",
			api:           "http://127.0.0.1:5000/api/tool/my-exec-sql-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf(`{"sql": %s}`, createTableStatement))),
			want:          "null",
			isErr:         false,
		},
		{
			name:          "invoke my-exec-sql-tool select table",
			api:           "http://127.0.0.1:5000/api/tool/my-exec-sql-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"sql":"SELECT * FROM t"}`)),
			want:          "null",
			isErr:         false,
		},
		{
			name:          "invoke my-exec-sql-tool drop table",
			api:           "http://127.0.0.1:5000/api/tool/my-exec-sql-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"sql":"DROP TABLE t"}`)),
			want:          "null",
			isErr:         false,
		},
		{
			name:          "invoke my-exec-sql-tool without body",
			api:           "http://127.0.0.1:5000/api/tool/my-exec-sql-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			isErr:         true,
		},
		{
			name:          "Invoke my-auth-exec-sql-tool with auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-exec-sql-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(`{"sql":"SELECT 1"}`)),
			isErr:         false,
			want:          select1Want,
		},
		{
			name:          "Invoke my-auth-exec-sql-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-exec-sql-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": "INVALID_TOKEN"},
			requestBody:   bytes.NewBuffer([]byte(`{"sql":"SELECT 1"}`)),
			isErr:         true,
		},
		{
			name:          "Invoke my-auth-exec-sql-tool without auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-exec-sql-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"sql":"SELECT 1"}`)),
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

			if got != tc.want {
				t.Fatalf("unexpected value: got %q, want %q", got, tc.want)
			}
		})
	}
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

	resp, _ := runRequest(t, http.MethodPost, url, bytes.NewBuffer(reqMarshal), nil)
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

	_, _ = runRequest(t, http.MethodPost, url, bytes.NewBuffer(notiMarshal), header)
	return sessionId
}

// RunMCPToolCallMethod runs the tool/call for mcp endpoint
func RunMCPToolCallMethod(t *testing.T, invokeParamWant, failInvocationWant string) {
	sessionId := RunInitialize(t, "2024-11-05")
	header := map[string]string{}
	if sessionId != "" {
		header["Mcp-Session-Id"] = sessionId
	}

	// Test tool invoke endpoint
	invokeTcs := []struct {
		name          string
		api           string
		requestBody   jsonrpc.JSONRPCRequest
		requestHeader map[string]string
		want          string
	}{
		{
			name:          "MCP Invoke my-param-tool",
			api:           "http://127.0.0.1:5000/mcp",
			requestHeader: map[string]string{},
			requestBody: jsonrpc.JSONRPCRequest{
				Jsonrpc: "2.0",
				Id:      "my-param-tool",
				Request: jsonrpc.Request{
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
			want: invokeParamWant,
		},
		{
			name:          "MCP Invoke invalid tool",
			api:           "http://127.0.0.1:5000/mcp",
			requestHeader: map[string]string{},
			requestBody: jsonrpc.JSONRPCRequest{
				Jsonrpc: "2.0",
				Id:      "invalid-tool",
				Request: jsonrpc.Request{
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
			requestBody: jsonrpc.JSONRPCRequest{
				Jsonrpc: "2.0",
				Id:      "invoke-without-parameter",
				Request: jsonrpc.Request{
					Method: "tools/call",
				},
				Params: map[string]any{
					"name":      "my-param-tool",
					"arguments": map[string]any{},
				},
			},
			want: `{"jsonrpc":"2.0","id":"invoke-without-parameter","error":{"code":-32602,"message":"provided parameters were invalid: parameter \"id\" is required"}}`,
		},
		{
			name:          "MCP Invoke my-param-tool with insufficient parameters",
			api:           "http://127.0.0.1:5000/mcp",
			requestHeader: map[string]string{},
			requestBody: jsonrpc.JSONRPCRequest{
				Jsonrpc: "2.0",
				Id:      "invoke-insufficient-parameter",
				Request: jsonrpc.Request{
					Method: "tools/call",
				},
				Params: map[string]any{
					"name":      "my-param-tool",
					"arguments": map[string]any{"id": 1},
				},
			},
			want: `{"jsonrpc":"2.0","id":"invoke-insufficient-parameter","error":{"code":-32602,"message":"provided parameters were invalid: parameter \"name\" is required"}}`,
		},
		{
			name:          "MCP Invoke my-auth-required-tool",
			api:           "http://127.0.0.1:5000/mcp",
			requestHeader: map[string]string{},
			requestBody: jsonrpc.JSONRPCRequest{
				Jsonrpc: "2.0",
				Id:      "invoke my-auth-required-tool",
				Request: jsonrpc.Request{
					Method: "tools/call",
				},
				Params: map[string]any{
					"name":      "my-auth-required-tool",
					"arguments": map[string]any{},
				},
			},
			want: "{\"jsonrpc\":\"2.0\",\"id\":\"invoke my-auth-required-tool\",\"error\":{\"code\":-32600,\"message\":\"unauthorized Tool call: `authRequired` is set for the target Tool\"}}",
		},
		{
			name:          "MCP Invoke my-fail-tool",
			api:           "http://127.0.0.1:5000/mcp",
			requestHeader: map[string]string{},
			requestBody: jsonrpc.JSONRPCRequest{
				Jsonrpc: "2.0",
				Id:      "invoke-fail-tool",
				Request: jsonrpc.Request{
					Method: "tools/call",
				},
				Params: map[string]any{
					"name":      "my-fail-tool",
					"arguments": map[string]any{"id": 1},
				},
			},
			want: failInvocationWant,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			reqMarshal, err := json.Marshal(tc.requestBody)
			if err != nil {
				t.Fatalf("unexpected error during marshaling of request body")
			}

			_, respBody := runRequest(t, http.MethodPost, tc.api, bytes.NewBuffer(reqMarshal), header)
			got := string(bytes.TrimSpace(respBody))

			if !strings.Contains(got, tc.want) {
				t.Fatalf("Expected substring not found:\ngot:  %q\nwant: %q (to be contained within got)", got, tc.want)
			}
		})
	}
}

func runRequest(t *testing.T, method, url string, body io.Reader, header map[string]string) (*http.Response, []byte) {
	// Send request
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatalf("unable to create request: %s", err)
	}

	req.Header.Add("Content-type", "application/json")
	for k, v := range header {
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
	return resp, respBody
}
