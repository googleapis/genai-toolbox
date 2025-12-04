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
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	"github.com/googleapis/genai-toolbox/internal/server/mcp/jsonrpc"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/jackc/pgx/v5/pgxpool"
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

func RunToolGetTestByName(t *testing.T, name string, want map[string]any) {
	// Test tool get endpoint
	tcs := []struct {
		name string
		api  string
		want map[string]any
	}{
		{
			name: fmt.Sprintf("get %s", name),
			api:  fmt.Sprintf("http://127.0.0.1:5000/api/tool/%s/", name),
			want: want,
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

// RunToolInvokeSimpleTest runs the tool invoke endpoint with no parameters
func RunToolInvokeSimpleTest(t *testing.T, name string, simpleWant string) {
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
			name:          fmt.Sprintf("invoke %s", name),
			api:           fmt.Sprintf("http://127.0.0.1:5000/api/tool/%s/invoke", name),
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),
			want:          simpleWant,
			isErr:         false,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			// Send Tool invocation request
			resp, respBody := RunRequest(t, http.MethodPost, tc.api, tc.requestBody, tc.requestHeader)
			if resp.StatusCode != http.StatusOK {
				if tc.isErr {
					return
				}
				t.Fatalf("response status code is not 200, got %d: %s", resp.StatusCode, string(respBody))
			}

			// Check response body
			var body map[string]interface{}
			err := json.Unmarshal(respBody, &body)
			if err != nil {
				t.Fatalf("error parsing response body")
			}

			got, ok := body["result"].(string)
			if !ok {
				t.Fatalf("unable to find result in response body")
			}

			if !strings.Contains(got, tc.want) {
				t.Fatalf("unexpected value: got %q, want %q", got, tc.want)
			}
		})
	}
}

func RunToolInvokeParametersTest(t *testing.T, name string, params []byte, simpleWant string) {
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
			name:          fmt.Sprintf("invoke %s", name),
			api:           fmt.Sprintf("http://127.0.0.1:5000/api/tool/%s/invoke", name),
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer(params),
			want:          simpleWant,
			isErr:         false,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			// Send Tool invocation request
			resp, respBody := RunRequest(t, http.MethodPost, tc.api, tc.requestBody, tc.requestHeader)
			if resp.StatusCode != http.StatusOK {
				if tc.isErr {
					return
				}
				t.Fatalf("response status code is not 200, got %d: %s", resp.StatusCode, string(respBody))
			}

			// Check response body
			var body map[string]interface{}
			err := json.Unmarshal(respBody, &body)
			if err != nil {
				t.Fatalf("error parsing response body")
			}

			got, ok := body["result"].(string)
			if !ok {
				t.Fatalf("unable to find result in response body")
			}

			if !strings.Contains(got, tc.want) {
				t.Fatalf("unexpected value: got %q, want %q", got, tc.want)
			}
		})
	}
}

// RunToolInvoke runs the tool invoke endpoint
func RunToolInvokeTest(t *testing.T, select1Want string, options ...InvokeTestOption) {
	// Resolve options
	// Default values for InvokeTestConfig
	configs := &InvokeTestConfig{
		myToolId3NameAliceWant:   "[{\"id\":1,\"name\":\"Alice\"},{\"id\":3,\"name\":\"Sid\"}]",
		myToolById4Want:          "[{\"id\":4,\"name\":null}]",
		myArrayToolWant:          "[{\"id\":1,\"name\":\"Alice\"},{\"id\":3,\"name\":\"Sid\"}]",
		nullWant:                 "null",
		supportOptionalNullParam: true,
		supportArrayParam:        true,
		supportClientAuth:        false,
		supportSelect1Want:       true,
		supportSelect1Auth:       true,
	}

	// Apply provided options
	for _, option := range options {
		option(configs)
	}

	// Get ID token
	idToken, err := GetGoogleIdToken(ClientId)
	if err != nil {
		t.Fatalf("error getting Google ID token: %s", err)
	}

	// Get access token
	accessToken, err := sources.GetIAMAccessToken(t.Context())
	if err != nil {
		t.Fatalf("error getting access token from ADC: %s", err)
	}
	accessToken = "Bearer " + accessToken

	// Test tool invoke endpoint
	invokeTcs := []struct {
		name           string
		api            string
		enabled        bool
		requestHeader  map[string]string
		requestBody    io.Reader
		wantStatusCode int
		wantBody       string
	}{
		{
			name:           "invoke my-simple-tool",
			api:            "http://127.0.0.1:5000/api/tool/my-simple-tool/invoke",
			enabled:        configs.supportSelect1Want,
			requestHeader:  map[string]string{},
			requestBody:    bytes.NewBuffer([]byte(`{}`)),
			wantBody:       select1Want,
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "invoke my-tool",
			api:            "http://127.0.0.1:5000/api/tool/my-tool/invoke",
			enabled:        true,
			requestHeader:  map[string]string{},
			requestBody:    bytes.NewBuffer([]byte(`{"id": 3, "name": "Alice"}`)),
			wantBody:       configs.myToolId3NameAliceWant,
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "invoke my-tool-by-id with nil response",
			api:            "http://127.0.0.1:5000/api/tool/my-tool-by-id/invoke",
			enabled:        true,
			requestHeader:  map[string]string{},
			requestBody:    bytes.NewBuffer([]byte(`{"id": 4}`)),
			wantBody:       configs.myToolById4Want,
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "invoke my-tool-by-name with nil response",
			api:            "http://127.0.0.1:5000/api/tool/my-tool-by-name/invoke",
			enabled:        configs.supportOptionalNullParam,
			requestHeader:  map[string]string{},
			requestBody:    bytes.NewBuffer([]byte(`{}`)),
			wantBody:       configs.nullWant,
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "Invoke my-tool without parameters",
			api:            "http://127.0.0.1:5000/api/tool/my-tool/invoke",
			enabled:        true,
			requestHeader:  map[string]string{},
			requestBody:    bytes.NewBuffer([]byte(`{}`)),
			wantBody:       "",
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "Invoke my-tool with insufficient parameters",
			api:            "http://127.0.0.1:5000/api/tool/my-tool/invoke",
			enabled:        true,
			requestHeader:  map[string]string{},
			requestBody:    bytes.NewBuffer([]byte(`{"id": 1}`)),
			wantBody:       "",
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "invoke my-array-tool",
			api:            "http://127.0.0.1:5000/api/tool/my-array-tool/invoke",
			enabled:        configs.supportArrayParam,
			requestHeader:  map[string]string{},
			requestBody:    bytes.NewBuffer([]byte(`{"idArray": [1,2,3], "nameArray": ["Alice", "Sid", "RandomName"], "cmdArray": ["HGETALL", "row3"]}`)),
			wantBody:       configs.myArrayToolWant,
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "Invoke my-auth-tool with auth token",
			api:            "http://127.0.0.1:5000/api/tool/my-auth-tool/invoke",
			enabled:        configs.supportSelect1Auth,
			requestHeader:  map[string]string{"my-google-auth_token": idToken},
			requestBody:    bytes.NewBuffer([]byte(`{}`)),
			wantBody:       configs.myAuthToolWant,
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "Invoke my-auth-tool with invalid auth token",
			api:            "http://127.0.0.1:5000/api/tool/my-auth-tool/invoke",
			enabled:        configs.supportSelect1Auth,
			requestHeader:  map[string]string{"my-google-auth_token": "INVALID_TOKEN"},
			requestBody:    bytes.NewBuffer([]byte(`{}`)),
			wantBody:       "",
			wantStatusCode: http.StatusUnauthorized,
		},
		{
			name:           "Invoke my-auth-tool without auth token",
			api:            "http://127.0.0.1:5000/api/tool/my-auth-tool/invoke",
			enabled:        true,
			requestHeader:  map[string]string{},
			requestBody:    bytes.NewBuffer([]byte(`{}`)),
			wantBody:       "",
			wantStatusCode: http.StatusUnauthorized,
		},
		{
			name:           "Invoke my-auth-required-tool with auth token",
			api:            "http://127.0.0.1:5000/api/tool/my-auth-required-tool/invoke",
			enabled:        configs.supportSelect1Auth,
			requestHeader:  map[string]string{"my-google-auth_token": idToken},
			requestBody:    bytes.NewBuffer([]byte(`{}`)),
			wantBody:       select1Want,
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "Invoke my-auth-required-tool with invalid auth token",
			api:            "http://127.0.0.1:5000/api/tool/my-auth-required-tool/invoke",
			enabled:        true,
			requestHeader:  map[string]string{"my-google-auth_token": "INVALID_TOKEN"},
			requestBody:    bytes.NewBuffer([]byte(`{}`)),
			wantBody:       "",
			wantStatusCode: http.StatusUnauthorized,
		},
		{
			name:           "Invoke my-auth-required-tool without auth token",
			api:            "http://127.0.0.1:5000/api/tool/my-auth-tool/invoke",
			enabled:        true,
			requestHeader:  map[string]string{},
			requestBody:    bytes.NewBuffer([]byte(`{}`)),
			wantBody:       "",
			wantStatusCode: http.StatusUnauthorized,
		},
		{
			name:           "Invoke my-client-auth-tool with auth token",
			api:            "http://127.0.0.1:5000/api/tool/my-client-auth-tool/invoke",
			enabled:        configs.supportClientAuth,
			requestHeader:  map[string]string{"Authorization": accessToken},
			requestBody:    bytes.NewBuffer([]byte(`{}`)),
			wantBody:       select1Want,
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "Invoke my-client-auth-tool without auth token",
			api:            "http://127.0.0.1:5000/api/tool/my-client-auth-tool/invoke",
			enabled:        configs.supportClientAuth,
			requestHeader:  map[string]string{},
			requestBody:    bytes.NewBuffer([]byte(`{}`)),
			wantStatusCode: http.StatusUnauthorized,
		},
		{

			name:           "Invoke my-client-auth-tool with invalid auth token",
			api:            "http://127.0.0.1:5000/api/tool/my-client-auth-tool/invoke",
			enabled:        configs.supportClientAuth,
			requestHeader:  map[string]string{"Authorization": "Bearer invalid-token"},
			requestBody:    bytes.NewBuffer([]byte(`{}`)),
			wantStatusCode: http.StatusUnauthorized,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			if !tc.enabled {
				return
			}
			// Send Tool invocation request
			resp, respBody := RunRequest(t, http.MethodPost, tc.api, tc.requestBody, tc.requestHeader)

			// Check status code
			if resp.StatusCode != tc.wantStatusCode {
				t.Errorf("StatusCode mismatch: got %d, want %d. Response body: %s", resp.StatusCode, tc.wantStatusCode, string(respBody))
			}

			// skip response body check
			if tc.wantBody == "" {
				return
			}

			// Check response body
			var body map[string]interface{}
			err = json.Unmarshal(respBody, &body)
			if err != nil {
				t.Fatalf("error parsing response body: %s", err)
			}

			got, ok := body["result"].(string)
			if !ok {
				t.Fatalf("unable to find result in response body")
			}

			if got != tc.wantBody {
				t.Fatalf("unexpected value: got %q, want %q", got, tc.wantBody)
			}
		})
	}
}

// RunToolInvokeWithTemplateParameters runs tool invoke test cases with template parameters.
func RunToolInvokeWithTemplateParameters(t *testing.T, tableName string, options ...TemplateParamOption) {
	// Resolve options
	// Default values for TemplateParameterTestConfig
	configs := &TemplateParameterTestConfig{
		ddlWant:         "null",
		selectAllWant:   "[{\"age\":21,\"id\":1,\"name\":\"Alex\"},{\"age\":100,\"id\":2,\"name\":\"Alice\"}]",
		selectId1Want:   "[{\"age\":21,\"id\":1,\"name\":\"Alex\"}]",
		selectNameWant:  "[{\"age\":21,\"id\":1,\"name\":\"Alex\"}]",
		selectEmptyWant: "null",
		insert1Want:     "null",

		nameFieldArray: `["name"]`,
		nameColFilter:  "name",
		createColArray: `["id INT","name VARCHAR(20)","age INT"]`,

		supportDdl:    true,
		supportInsert: true,
	}

	// Apply provided options
	for _, option := range options {
		option(configs)
	}

	selectOnlyNamesWant := "[{\"name\":\"Alex\"},{\"name\":\"Alice\"}]"

	// Test tool invoke endpoint
	invokeTcs := []struct {
		name          string
		enabled       bool
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
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf(`{"tableName": "%s", "columns":%s}`, tableName, configs.createColArray))),
			want:          configs.ddlWant,
			isErr:         false,
		},
		{
			name:          "invoke insert-table-templateParams-tool",
			insert:        true,
			api:           "http://127.0.0.1:5000/api/tool/insert-table-templateParams-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf(`{"tableName": "%s", "columns":["id","name","age"], "values":"1, 'Alex', 21"}`, tableName))),
			want:          configs.insert1Want,
			isErr:         false,
		},
		{
			name:          "invoke insert-table-templateParams-tool",
			insert:        true,
			api:           "http://127.0.0.1:5000/api/tool/insert-table-templateParams-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf(`{"tableName": "%s", "columns":["id","name","age"], "values":"2, 'Alice', 100"}`, tableName))),
			want:          configs.insert1Want,
			isErr:         false,
		},
		{
			name:          "invoke select-templateParams-tool",
			api:           "http://127.0.0.1:5000/api/tool/select-templateParams-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf(`{"tableName": "%s"}`, tableName))),
			want:          configs.selectAllWant,
			isErr:         false,
		},
		{
			name:          "invoke select-templateParams-combined-tool",
			api:           "http://127.0.0.1:5000/api/tool/select-templateParams-combined-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf(`{"id": 1, "tableName": "%s"}`, tableName))),
			want:          configs.selectId1Want,
			isErr:         false,
		},
		{
			name:          "invoke select-templateParams-combined-tool with no results",
			api:           "http://127.0.0.1:5000/api/tool/select-templateParams-combined-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf(`{"id": 999, "tableName": "%s"}`, tableName))),
			want:          configs.selectEmptyWant,
			isErr:         false,
		},
		{
			name:          "invoke select-fields-templateParams-tool",
			enabled:       configs.supportSelectFields,
			api:           "http://127.0.0.1:5000/api/tool/select-fields-templateParams-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf(`{"tableName": "%s", "fields":%s}`, tableName, configs.nameFieldArray))),
			want:          selectOnlyNamesWant,
			isErr:         false,
		},
		{
			name:          "invoke select-filter-templateParams-combined-tool",
			api:           "http://127.0.0.1:5000/api/tool/select-filter-templateParams-combined-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf(`{"name": "Alex", "tableName": "%s", "columnFilter": "%s"}`, tableName, configs.nameColFilter))),
			want:          configs.selectNameWant,
			isErr:         false,
		},
		{
			name:          "invoke drop-table-templateParams-tool",
			ddl:           true,
			api:           "http://127.0.0.1:5000/api/tool/drop-table-templateParams-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf(`{"tableName": "%s"}`, tableName))),
			want:          configs.ddlWant,
			isErr:         false,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			if !tc.enabled {
				return
			}
			// if test case is DDL and source support ddl test cases
			ddlAllow := !tc.ddl || (tc.ddl && configs.supportDdl)
			// if test case is insert statement and source support insert test cases
			insertAllow := !tc.insert || (tc.insert && configs.supportInsert)
			if ddlAllow && insertAllow {
				// Send Tool invocation request
				resp, respBody := RunRequest(t, http.MethodPost, tc.api, tc.requestBody, tc.requestHeader)
				if resp.StatusCode != http.StatusOK {
					if tc.isErr {
						return
					}
					t.Fatalf("response status code is not 200, got %d: %s", resp.StatusCode, string(respBody))
				}

				// Check response body
				var body map[string]interface{}
				err := json.Unmarshal(respBody, &body)
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

func RunExecuteSqlToolInvokeTest(t *testing.T, createTableStatement, select1Want string, options ...ExecuteSqlOption) {
	// Resolve options
	// Default values for ExecuteSqlTestConfig
	configs := &ExecuteSqlTestConfig{
		select1Statement: `"SELECT 1"`,
	}

	// Apply provided options
	for _, option := range options {
		option(configs)
	}

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
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf(`{"sql": %s}`, configs.select1Statement))),
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
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf(`{"sql": %s}`, configs.select1Statement))),
			isErr:         false,
			want:          select1Want,
		},
		{
			name:          "Invoke my-auth-exec-sql-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-exec-sql-tool/invoke",
			requestHeader: map[string]string{"my-google-auth_token": "INVALID_TOKEN"},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf(`{"sql": %s}`, configs.select1Statement))),
			isErr:         true,
		},
		{
			name:          "Invoke my-auth-exec-sql-tool without auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-exec-sql-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(fmt.Sprintf(`{"sql": %s}`, configs.select1Statement))),
			isErr:         true,
		},
		{
			name:          "invoke my-exec-sql-tool with invalid SELECT SQL",
			api:           "http://127.0.0.1:5000/api/tool/my-exec-sql-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"sql":"SELECT * FROM non_existent_table"}`)),
			isErr:         true,
		},
		{
			name:          "invoke my-exec-sql-tool with invalid ALTER SQL",
			api:           "http://127.0.0.1:5000/api/tool/my-exec-sql-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"sql":"ALTER TALE t ALTER COLUMN id DROP NOT NULL"}`)),
			isErr:         true,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			// Send Tool invocation request
			resp, respBody := RunRequest(t, http.MethodPost, tc.api, tc.requestBody, tc.requestHeader)
			if resp.StatusCode != http.StatusOK {
				if tc.isErr {
					return
				}
				t.Fatalf("response status code is not 200, got %d: %s", resp.StatusCode, string(respBody))
			}

			// Check response body
			var body map[string]interface{}
			err = json.Unmarshal(respBody, &body)
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

// RunMCPToolCallMethod runs the tool/call for mcp endpoint
func RunMCPToolCallMethod(t *testing.T, myFailToolWant, select1Want string, options ...McpTestOption) {
	// Resolve options
	// Default values for MCPTestConfig
	configs := &MCPTestConfig{
		myToolId3NameAliceWant: `{"jsonrpc":"2.0","id":"my-tool","result":{"content":[{"type":"text","text":"{\"id\":1,\"name\":\"Alice\"}"},{"type":"text","text":"{\"id\":3,\"name\":\"Sid\"}"}]}}`,
		supportClientAuth:      false,
		supportSelect1Auth:     true,
	}

	// Apply provided options
	for _, option := range options {
		option(configs)
	}

	sessionId := RunInitialize(t, "2024-11-05")

	// Get access token
	accessToken, err := sources.GetIAMAccessToken(t.Context())
	if err != nil {
		t.Fatalf("error getting access token from ADC: %s", err)
	}
	accessToken = "Bearer " + accessToken

	idToken, err := GetGoogleIdToken(ClientId)
	if err != nil {
		t.Fatalf("error getting Google ID token: %s", err)
	}

	// Test tool invoke endpoint
	invokeTcs := []struct {
		name           string
		api            string
		enabled        bool // switch to turn on/off the test case
		requestBody    jsonrpc.JSONRPCRequest
		requestHeader  map[string]string
		wantStatusCode int
		wantBody       string
	}{
		{
			name:          "MCP Invoke my-tool",
			api:           "http://127.0.0.1:5000/mcp",
			enabled:       true,
			requestHeader: map[string]string{},
			requestBody: jsonrpc.JSONRPCRequest{
				Jsonrpc: "2.0",
				Id:      "my-tool",
				Request: jsonrpc.Request{
					Method: "tools/call",
				},
				Params: map[string]any{
					"name": "my-tool",
					"arguments": map[string]any{
						"id":   int(3),
						"name": "Alice",
					},
				},
			},
			wantStatusCode: http.StatusOK,
			wantBody:       configs.myToolId3NameAliceWant,
		},
		{
			name:          "MCP Invoke invalid tool",
			api:           "http://127.0.0.1:5000/mcp",
			enabled:       true,
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
			wantStatusCode: http.StatusOK,
			wantBody:       `{"jsonrpc":"2.0","id":"invalid-tool","error":{"code":-32602,"message":"invalid tool name: tool with name \"foo\" does not exist"}}`,
		},
		{
			name:          "MCP Invoke my-tool without parameters",
			api:           "http://127.0.0.1:5000/mcp",
			enabled:       true,
			requestHeader: map[string]string{},
			requestBody: jsonrpc.JSONRPCRequest{
				Jsonrpc: "2.0",
				Id:      "invoke-without-parameter",
				Request: jsonrpc.Request{
					Method: "tools/call",
				},
				Params: map[string]any{
					"name":      "my-tool",
					"arguments": map[string]any{},
				},
			},
			wantStatusCode: http.StatusOK,
			wantBody:       `{"jsonrpc":"2.0","id":"invoke-without-parameter","error":{"code":-32602,"message":"provided parameters were invalid: parameter \"id\" is required"}}`,
		},
		{
			name:          "MCP Invoke my-tool with insufficient parameters",
			api:           "http://127.0.0.1:5000/mcp",
			enabled:       true,
			requestHeader: map[string]string{},
			requestBody: jsonrpc.JSONRPCRequest{
				Jsonrpc: "2.0",
				Id:      "invoke-insufficient-parameter",
				Request: jsonrpc.Request{
					Method: "tools/call",
				},
				Params: map[string]any{
					"name":      "my-tool",
					"arguments": map[string]any{"id": 1},
				},
			},
			wantStatusCode: http.StatusOK,
			wantBody:       `{"jsonrpc":"2.0","id":"invoke-insufficient-parameter","error":{"code":-32602,"message":"provided parameters were invalid: parameter \"name\" is required"}}`,
		},
		{
			name:          "MCP Invoke my-auth-required-tool",
			api:           "http://127.0.0.1:5000/mcp",
			enabled:       configs.supportSelect1Auth,
			requestHeader: map[string]string{"my-google-auth_token": idToken},
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
			wantStatusCode: http.StatusOK,
			wantBody:       select1Want,
		},
		{
			name:          "MCP Invoke my-auth-required-tool with invalid auth token",
			api:           "http://127.0.0.1:5000/mcp",
			requestHeader: map[string]string{"my-google-auth_token": "INVALID_TOKEN"},
			requestBody: jsonrpc.JSONRPCRequest{
				Jsonrpc: "2.0",
				Id:      "invoke my-auth-required-tool with invalid token",
				Request: jsonrpc.Request{
					Method: "tools/call",
				},
				Params: map[string]any{
					"name":      "my-auth-required-tool",
					"arguments": map[string]any{},
				},
			},
			wantStatusCode: http.StatusUnauthorized,
			wantBody:       "{\"jsonrpc\":\"2.0\",\"id\":\"invoke my-auth-required-tool with invalid token\",\"error\":{\"code\":-32600,\"message\":\"unauthorized Tool call: Please make sure your specify correct auth headers: unauthorized\"}}",
		},
		{
			name:          "MCP Invoke my-auth-required-tool without auth token",
			api:           "http://127.0.0.1:5000/mcp",
			requestHeader: map[string]string{},
			requestBody: jsonrpc.JSONRPCRequest{
				Jsonrpc: "2.0",
				Id:      "invoke my-auth-required-tool without token",
				Request: jsonrpc.Request{
					Method: "tools/call",
				},
				Params: map[string]any{
					"name":      "my-auth-required-tool",
					"arguments": map[string]any{},
				},
			},
			wantStatusCode: http.StatusUnauthorized,
			wantBody:       "{\"jsonrpc\":\"2.0\",\"id\":\"invoke my-auth-required-tool without token\",\"error\":{\"code\":-32600,\"message\":\"unauthorized Tool call: Please make sure your specify correct auth headers: unauthorized\"}}",
		},

		{
			name:          "MCP Invoke my-client-auth-tool",
			enabled:       configs.supportClientAuth,
			api:           "http://127.0.0.1:5000/mcp",
			requestHeader: map[string]string{"Authorization": accessToken},
			requestBody: jsonrpc.JSONRPCRequest{
				Jsonrpc: "2.0",
				Id:      "invoke my-client-auth-tool",
				Request: jsonrpc.Request{
					Method: "tools/call",
				},
				Params: map[string]any{
					"name":      "my-client-auth-tool",
					"arguments": map[string]any{},
				},
			},
			wantStatusCode: http.StatusOK,
			wantBody:       "{\"jsonrpc\":\"2.0\",\"id\":\"invoke my-client-auth-tool\",\"result\":{\"content\":[{\"type\":\"text\",\"text\":\"{\\\"f0_\\\":1}\"}]}}",
		},
		{
			name:          "MCP Invoke my-client-auth-tool without access token",
			enabled:       configs.supportClientAuth,
			api:           "http://127.0.0.1:5000/mcp",
			requestHeader: map[string]string{},
			requestBody: jsonrpc.JSONRPCRequest{
				Jsonrpc: "2.0",
				Id:      "invoke my-client-auth-tool",
				Request: jsonrpc.Request{
					Method: "tools/call",
				},
				Params: map[string]any{
					"name":      "my-client-auth-tool",
					"arguments": map[string]any{},
				},
			},
			wantStatusCode: http.StatusUnauthorized,
			wantBody:       "{\"jsonrpc\":\"2.0\",\"id\":\"invoke my-client-auth-tool\",\"error\":{\"code\":-32600,\"message\":\"missing access token in the 'Authorization' header\"}",
		},
		{
			name:          "MCP Invoke my-client-auth-tool with invalid access token",
			enabled:       configs.supportClientAuth,
			api:           "http://127.0.0.1:5000/mcp",
			requestHeader: map[string]string{"Authorization": "Bearer invalid-token"},
			requestBody: jsonrpc.JSONRPCRequest{
				Jsonrpc: "2.0",
				Id:      "invoke my-client-auth-tool",
				Request: jsonrpc.Request{
					Method: "tools/call",
				},
				Params: map[string]any{
					"name":      "my-client-auth-tool",
					"arguments": map[string]any{},
				},
			},
			wantStatusCode: http.StatusUnauthorized,
		},
		{
			name:          "MCP Invoke my-fail-tool",
			api:           "http://127.0.0.1:5000/mcp",
			enabled:       true,
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
			wantStatusCode: http.StatusOK,
			wantBody:       myFailToolWant,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			if !tc.enabled {
				return
			}
			reqMarshal, err := json.Marshal(tc.requestBody)
			if err != nil {
				t.Fatalf("unexpected error during marshaling of request body")
			}

			// add headers
			headers := map[string]string{}
			if sessionId != "" {
				headers["Mcp-Session-Id"] = sessionId
			}
			for key, value := range tc.requestHeader {
				headers[key] = value
			}

			httpResponse, respBody := RunRequest(t, http.MethodPost, tc.api, bytes.NewBuffer(reqMarshal), headers)

			// Check status code
			if httpResponse.StatusCode != tc.wantStatusCode {
				t.Errorf("StatusCode mismatch: got %d, want %d", httpResponse.StatusCode, tc.wantStatusCode)
			}

			// Check response body
			got := string(bytes.TrimSpace(respBody))
			if !strings.Contains(got, tc.wantBody) {
				t.Fatalf("Expected substring not found:\ngot:  %q\nwant: %q (to be contained within got)", got, tc.wantBody)
			}
		})
	}
}

func setupPostgresSchemas(t *testing.T, ctx context.Context, pool *pgxpool.Pool, schemaName string) func() {
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

func RunPostgresListTablesTest(t *testing.T, tableNameParam, tableNameAuth, user string) {
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
		api            string
		requestBody    io.Reader
		wantStatusCode int
		want           string
		isAllTables    bool
	}{
		{
			name:           "invoke list_tables all tables detailed output",
			api:            "http://127.0.0.1:5000/api/tool/list_tables/invoke",
			requestBody:    bytes.NewBuffer([]byte(`{"table_names": ""}`)),
			wantStatusCode: http.StatusOK,
			want:           fmt.Sprintf("[%s,%s]", getDetailedWant(tableNameAuth, authTableColumns), getDetailedWant(tableNameParam, paramTableColumns)),
			isAllTables:    true,
		},
		{
			name:           "invoke list_tables all tables simple output",
			api:            "http://127.0.0.1:5000/api/tool/list_tables/invoke",
			requestBody:    bytes.NewBuffer([]byte(`{"table_names": "", "output_format": "simple"}`)),
			wantStatusCode: http.StatusOK,
			want:           fmt.Sprintf("[%s,%s]", getSimpleWant(tableNameAuth), getSimpleWant(tableNameParam)),
			isAllTables:    true,
		},
		{
			name:           "invoke list_tables detailed output",
			api:            "http://127.0.0.1:5000/api/tool/list_tables/invoke",
			requestBody:    bytes.NewBuffer([]byte(fmt.Sprintf(`{"table_names": "%s"}`, tableNameAuth))),
			wantStatusCode: http.StatusOK,
			want:           fmt.Sprintf("[%s]", getDetailedWant(tableNameAuth, authTableColumns)),
		},
		{
			name:           "invoke list_tables simple output",
			api:            "http://127.0.0.1:5000/api/tool/list_tables/invoke",
			requestBody:    bytes.NewBuffer([]byte(fmt.Sprintf(`{"table_names": "%s", "output_format": "simple"}`, tableNameAuth))),
			wantStatusCode: http.StatusOK,
			want:           fmt.Sprintf("[%s]", getSimpleWant(tableNameAuth)),
		},
		{
			name:           "invoke list_tables with invalid output format",
			api:            "http://127.0.0.1:5000/api/tool/list_tables/invoke",
			requestBody:    bytes.NewBuffer([]byte(`{"table_names": "", "output_format": "abcd"}`)),
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "invoke list_tables with malformed table_names parameter",
			api:            "http://127.0.0.1:5000/api/tool/list_tables/invoke",
			requestBody:    bytes.NewBuffer([]byte(`{"table_names": 12345, "output_format": "detailed"}`)),
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "invoke list_tables with multiple table names",
			api:            "http://127.0.0.1:5000/api/tool/list_tables/invoke",
			requestBody:    bytes.NewBuffer([]byte(fmt.Sprintf(`{"table_names": "%s,%s"}`, tableNameParam, tableNameAuth))),
			wantStatusCode: http.StatusOK,
			want:           fmt.Sprintf("[%s,%s]", getDetailedWant(tableNameAuth, authTableColumns), getDetailedWant(tableNameParam, paramTableColumns)),
		},
		{
			name:           "invoke list_tables with non-existent table",
			api:            "http://127.0.0.1:5000/api/tool/list_tables/invoke",
			requestBody:    bytes.NewBuffer([]byte(`{"table_names": "non_existent_table"}`)),
			wantStatusCode: http.StatusOK,
			want:           `null`,
		},
		{
			name:           "invoke list_tables with one existing and one non-existent table",
			api:            "http://127.0.0.1:5000/api/tool/list_tables/invoke",
			requestBody:    bytes.NewBuffer([]byte(fmt.Sprintf(`{"table_names": "%s,non_existent_table"}`, tableNameParam))),
			wantStatusCode: http.StatusOK,
			want:           fmt.Sprintf("[%s]", getDetailedWant(tableNameParam, paramTableColumns)),
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			resp, respBytes := RunRequest(t, http.MethodPost, tc.api, tc.requestBody, nil)
			if resp.StatusCode != tc.wantStatusCode {
				t.Fatalf("response status code is not 200, got %d: %s", resp.StatusCode, string(respBytes))
			}

			if tc.wantStatusCode == http.StatusOK {
				var bodyWrapper map[string]json.RawMessage

				if err := json.Unmarshal(respBytes, &bodyWrapper); err != nil {
					t.Fatalf("error parsing response wrapper: %s, body: %s", err, string(respBytes))
				}

				resultJSON, ok := bodyWrapper["result"]
				if !ok {
					t.Fatal("unable to find 'result' in response body")
				}

				var resultString string
				if err := json.Unmarshal(resultJSON, &resultString); err != nil {
					t.Fatalf("'result' is not a JSON-encoded string: %s", err)
				}

				var got, want []any

				if err := json.Unmarshal([]byte(resultString), &got); err != nil {
					t.Fatalf("failed to unmarshal actual result string: %v", err)
				}
				if err := json.Unmarshal([]byte(tc.want), &want); err != nil {
					t.Fatalf("failed to unmarshal expected want string: %v", err)
				}

				// Checking only the default public schema where the test tables are created to avoid brittle tests.
				if tc.isAllTables {
					var filteredGot []any
					for _, item := range got {
						if tableMap, ok := item.(map[string]interface{}); ok {
							if schema, ok := tableMap["schema_name"]; ok && schema == "public" {
								filteredGot = append(filteredGot, item)
							}
						}
					}
					got = filteredGot
				}

				sort.SliceStable(got, func(i, j int) bool {
					return fmt.Sprintf("%v", got[i]) < fmt.Sprintf("%v", got[j])
				})
				sort.SliceStable(want, func(i, j int) bool {
					return fmt.Sprintf("%v", want[i]) < fmt.Sprintf("%v", want[j])
				})

				if !reflect.DeepEqual(got, want) {
					t.Errorf("Unexpected result: got  %#v, want: %#v", got, want)
				}
			}
		})
	}
}

func setUpPostgresViews(t *testing.T, ctx context.Context, pool *pgxpool.Pool, viewName, tableName string) func() {
	createView := fmt.Sprintf("CREATE VIEW %s AS SELECT name FROM %s", viewName, tableName)
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

func RunPostgresListViewsTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tableName string) {
	viewName1 := "test_view_1" + strings.ReplaceAll(uuid.New().String(), "-", "")
	dropViewfunc1 := setUpPostgresViews(t, ctx, pool, viewName1, tableName)
	defer dropViewfunc1()

	invokeTcs := []struct {
		name           string
		requestBody    io.Reader
		wantStatusCode int
		want           string
	}{
		{
			name:           "invoke list_views with newly created view",
			requestBody:    bytes.NewBuffer([]byte(fmt.Sprintf(`{"viewname": "%s"}`, viewName1))),
			wantStatusCode: http.StatusOK,
			want:           fmt.Sprintf(`[{"schemaname":"public","viewname":"%s","viewowner":"postgres"}]`, viewName1),
		},
		{
			name:           "invoke list_views with non-existent_view",
			requestBody:    bytes.NewBuffer([]byte(`{"viewname": "non_existent_view"}`)),
			wantStatusCode: http.StatusOK,
			want:           `null`,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			const api = "http://127.0.0.1:5000/api/tool/list_views/invoke"
			resp, body := RunRequest(t, http.MethodPost, api, tc.requestBody, nil)

			if resp.StatusCode != tc.wantStatusCode {
				t.Fatalf("wrong status code: got %d, want %d, body: %s", resp.StatusCode, tc.wantStatusCode, string(body))
			}
			if tc.wantStatusCode != http.StatusOK {
				return
			}

			var bodyWrapper struct {
				Result json.RawMessage `json:"result"`
			}
			if err := json.Unmarshal(body, &bodyWrapper); err != nil {
				t.Fatalf("error decoding response wrapper: %v", err)
			}

			var resultString string
			if err := json.Unmarshal(bodyWrapper.Result, &resultString); err != nil {
				resultString = string(bodyWrapper.Result)
			}

			var got, want any
			if err := json.Unmarshal([]byte(resultString), &got); err != nil {
				t.Fatalf("failed to unmarshal nested result string: %v", err)
			}
			if err := json.Unmarshal([]byte(tc.want), &want); err != nil {
				t.Fatalf("failed to unmarshal want string: %v", err)
			}

			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("Unexpected result (-want +got):\n%s", diff)
			}
		})
	}
}

func RunPostgresListSchemasTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	schemaName := "test_schema_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	cleanup := setupPostgresSchemas(t, ctx, pool, schemaName)
	defer cleanup()

	wantSchema := map[string]any{"functions": float64(0), "grants": map[string]any{}, "owner": "postgres", "schema_name": schemaName, "tables": float64(0), "views": float64(0)}

	invokeTcs := []struct {
		name           string
		requestBody    io.Reader
		wantStatusCode int
		want           []map[string]any
	}{
		{
			name:           "invoke list_schemas with schema_name",
			requestBody:    bytes.NewBuffer([]byte(fmt.Sprintf(`{"schema_name": "%s"}`, schemaName))),
			wantStatusCode: http.StatusOK,
			want:           []map[string]any{wantSchema},
		},
		{
			name:           "invoke list_schemas with non-existent schema",
			requestBody:    bytes.NewBuffer([]byte(`{"schema_name": "non_existent_schema"}`)),
			wantStatusCode: http.StatusOK,
			want:           nil,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			const api = "http://127.0.0.1:5000/api/tool/list_schemas/invoke"
			resp, respBody := RunRequest(t, http.MethodPost, api, tc.requestBody, nil)
			if resp.StatusCode != tc.wantStatusCode {
				t.Fatalf("wrong status code: got %d, want %d, body: %s", resp.StatusCode, tc.wantStatusCode, string(respBody))
			}
			if tc.wantStatusCode != http.StatusOK {
				return
			}

			var bodyWrapper struct {
				Result json.RawMessage `json:"result"`
			}
			if err := json.Unmarshal(respBody, &bodyWrapper); err != nil {
				t.Fatalf("error decoding response wrapper: %v", err)
			}

			var resultString string
			if err := json.Unmarshal(bodyWrapper.Result, &resultString); err != nil {
				resultString = string(bodyWrapper.Result)
			}

			var got []map[string]any
			if err := json.Unmarshal([]byte(resultString), &got); err != nil {
				t.Fatalf("failed to unmarshal nested result string: %v", err)
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Unexpected result (-want +got):\n%s", diff)
			}
		})
	}
}

func RunPostgresDatabaseOverviewTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	const api = "http://127.0.0.1:5000/api/tool/database_overview/invoke"
	requestBody := bytes.NewBuffer([]byte(`{}`))

	resp, respBody := RunRequest(t, http.MethodPost, api, requestBody, nil)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("wrong status code: got %d, want %d, body: %s", resp.StatusCode, http.StatusOK, string(respBody))
	}

	var bodyWrapper struct {
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(respBody, &bodyWrapper); err != nil {
		t.Fatalf("error decoding response wrapper: %v, body: %s", err, string(respBody))
	}

	var resultString string
	if err := json.Unmarshal(bodyWrapper.Result, &resultString); err != nil {
		resultString = string(bodyWrapper.Result)
	}

	var got []map[string]any
	if err := json.Unmarshal([]byte(resultString), &got); err != nil {
		t.Fatalf("failed to unmarshal nested result string: %v, result string: %s", err, resultString)
	}

	if len(got) != 1 {
		t.Fatalf("Expected exactly one row in the result, got %d", len(got))
	}

	resultRow := got[0]

	// Define expected keys based on the SELECT statement
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

	// Check types of the fields. JSON numbers are unmarshalled into float64.
	if _, ok := resultRow["pg_version"].(string); !ok {
		t.Errorf("Expected 'pg_version' to be a string, got %T", resultRow["pg_version"])
	}
	if _, ok := resultRow["is_replica"].(bool); !ok {
		t.Errorf("Expected 'is_replica' to be a bool, got %T", resultRow["is_replica"])
	}
	if _, ok := resultRow["uptime"].(string); !ok {
		t.Errorf("Expected 'uptime' to be a string, got %T", resultRow["uptime"])
	}
	if _, ok := resultRow["max_connections"].(float64); !ok {
		t.Errorf("Expected 'max_connections' to be a number (float64), got %T", resultRow["max_connections"])
	}
	if _, ok := resultRow["current_connections"].(float64); !ok {
		t.Errorf("Expected 'current_connections' to be a number (float64), got %T", resultRow["current_connections"])
	}
	if _, ok := resultRow["active_connections"].(float64); !ok {
		t.Errorf("Expected 'active_connections' to be a number (float64), got %T", resultRow["active_connections"])
	}
	if _, ok := resultRow["pct_connections_used"].(float64); !ok {
		t.Errorf("Expected 'pct_connections_used' to be a number (float64), got %T", resultRow["pct_connections_used"])
	}

	// Basic sanity checks on values
	if maxConn, ok := resultRow["max_connections"].(float64); ok {
		if maxConn <= 0 {
			t.Errorf("Expected 'max_connections' to be positive, got %f", maxConn)
		}
	}

	if pctUsed, ok := resultRow["pct_connections_used"].(float64); ok {
		if pctUsed < 0 || pctUsed > 100 {
			t.Errorf("Expected 'pct_connections_used' to be between 0 and 100, got %f", pctUsed)
		}
	}
}

func setupPostgresTrigger(t *testing.T, ctx context.Context, pool *pgxpool.Pool, schemaName, tableName, functionName, triggerName string) func() {
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

func RunPostgresListTriggersTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	uniqueID := strings.ReplaceAll(uuid.New().String(), "-", "")
	schemaName := "test_schema_" + uniqueID
	tableName := "test_table_" + uniqueID
	functionName := "test_func_" + uniqueID
	triggerName := "test_trigger_" + uniqueID

	cleanup := setupPostgresTrigger(t, ctx, pool, schemaName, tableName, functionName, triggerName)
	defer cleanup()

	// Definition can vary slightly based on server version/settings, so we fetch it to compare.
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
		requestBody    io.Reader
		wantStatusCode int
		want           []map[string]any
	}{
		{
			name:           "list all triggers (expecting the one we created)",
			requestBody:    bytes.NewBuffer([]byte(`{}`)),
			wantStatusCode: http.StatusOK,
			want:           []map[string]any{wantTrigger},
		},
		{
			name:           "filter by trigger_name",
			requestBody:    bytes.NewBuffer([]byte(fmt.Sprintf(`{"trigger_name": "%s"}`, triggerName))),
			wantStatusCode: http.StatusOK,
			want:           []map[string]any{wantTrigger},
		},
		{
			name:           "filter by schema_name",
			requestBody:    bytes.NewBuffer([]byte(fmt.Sprintf(`{"schema_name": "%s"}`, schemaName))),
			wantStatusCode: http.StatusOK,
			want:           []map[string]any{wantTrigger},
		},
		{
			name:           "filter by table_name",
			requestBody:    bytes.NewBuffer([]byte(fmt.Sprintf(`{"table_name": "%s"}`, tableName))),
			wantStatusCode: http.StatusOK,
			want:           []map[string]any{wantTrigger},
		},
		{
			name:           "filter by non-existent trigger_name",
			requestBody:    bytes.NewBuffer([]byte(`{"trigger_name": "non_existent_trigger"}`)),
			wantStatusCode: http.StatusOK,
			want:           nil,
		},
		{
			name:           "filter by non-existent schema_name",
			requestBody:    bytes.NewBuffer([]byte(`{"schema_name": "non_existent_schema"}`)),
			wantStatusCode: http.StatusOK,
			want:           nil,
		},
		{
			name:           "filter by non-existent table_name",
			requestBody:    bytes.NewBuffer([]byte(`{"table_name": "non_existent_table"}`)),
			wantStatusCode: http.StatusOK,
			want:           nil,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			const api = "http://127.0.0.1:5000/api/tool/list_triggers/invoke"
			resp, respBody := RunRequest(t, http.MethodPost, api, tc.requestBody, nil)
			if resp.StatusCode != tc.wantStatusCode {
				t.Fatalf("wrong status code: got %d, want %d, body: %s", resp.StatusCode, tc.wantStatusCode, string(respBody))
			}
			if tc.wantStatusCode != http.StatusOK {
				return
			}

			var bodyWrapper struct {
				Result json.RawMessage `json:"result"`
			}
			if err := json.Unmarshal(respBody, &bodyWrapper); err != nil {
				t.Fatalf("error decoding response wrapper: %v", err)
			}

			var resultString string
			if err := json.Unmarshal(bodyWrapper.Result, &resultString); err != nil {
				resultString = string(bodyWrapper.Result)
			}

			var got []map[string]any
			if err := json.Unmarshal([]byte(resultString), &got); err != nil {
				t.Fatalf("failed to unmarshal nested result string: %v, content: %s", err, resultString)
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Unexpected result (-want +got):\n%s", diff)
			}
		})
	}
}

func RunPostgresListActiveQueriesTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	type queryListDetails struct {
		ProcessId        any    `json:"pid"`
		User             string `json:"user"`
		Datname          string `json:"datname"`
		ApplicationName  string `json:"application_name"`
		ClientAddress    string `json:"client_addr"`
		State            string `json:"state"`
		WaitEventType    string `json:"wait_event_type"`
		WaitEvent        string `json:"wait_event"`
		BackendStart     any    `json:"backend_start"`
		TransactionStart any    `json:"xact_start"`
		QueryStart       any    `json:"query_start"`
		QueryDuration    any    `json:"query_duration"`
		Query            string `json:"query"`
	}

	singleQueryWanted := queryListDetails{
		ProcessId:        any(nil),
		User:             "",
		Datname:          "",
		ApplicationName:  "",
		ClientAddress:    "",
		State:            "",
		WaitEventType:    "",
		WaitEvent:        "",
		BackendStart:     any(nil),
		TransactionStart: any(nil),
		QueryStart:       any(nil),
		QueryDuration:    any(nil),
		Query:            "SELECT pg_sleep(10);",
	}

	invokeTcs := []struct {
		name                string
		requestBody         io.Reader
		clientSleepSecs     int
		waitSecsBeforeCheck int
		wantStatusCode      int
		want                any
	}{
		// exclude background monitoring apps such as "wal_uploader"
		{
			name:                "invoke list_active_queries when the system is idle",
			requestBody:         bytes.NewBufferString(`{"exclude_application_names": "wal_uploader"}`),
			clientSleepSecs:     0,
			waitSecsBeforeCheck: 0,
			wantStatusCode:      http.StatusOK,
			want:                []queryListDetails(nil),
		},
		{
			name:                "invoke list_active_queries when there is 1 ongoing but lower than the threshold",
			requestBody:         bytes.NewBufferString(`{"min_duration": "100 seconds", "exclude_application_names": "wal_uploader"}`),
			clientSleepSecs:     1,
			waitSecsBeforeCheck: 1,
			wantStatusCode:      http.StatusOK,
			want:                []queryListDetails(nil),
		},
		{
			name:                "invoke list_active_queries when 1 ongoing query should show up",
			requestBody:         bytes.NewBufferString(`{"min_duration": "1 seconds", "exclude_application_names": "wal_uploader"}`),
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

			const api = "http://127.0.0.1:5000/api/tool/list_active_queries/invoke"
			resp, respBody := RunRequest(t, http.MethodPost, api, tc.requestBody, nil)
			if resp.StatusCode != tc.wantStatusCode {
				t.Fatalf("wrong status code: got %d, want %d, body: %s", resp.StatusCode, tc.wantStatusCode, string(respBody))
			}
			if tc.wantStatusCode != http.StatusOK {
				return
			}

			var bodyWrapper struct {
				Result json.RawMessage `json:"result"`
			}
			if err := json.Unmarshal(respBody, &bodyWrapper); err != nil {
				t.Fatalf("error decoding response wrapper: %v", err)
			}

			var resultString string
			if err := json.Unmarshal(bodyWrapper.Result, &resultString); err != nil {
				resultString = string(bodyWrapper.Result)
			}

			var got any
			var details []queryListDetails
			if err := json.Unmarshal([]byte(resultString), &details); err != nil {
				t.Fatalf("failed to unmarshal nested ObjectDetails string: %v", err)
			}
			got = details

			if diff := cmp.Diff(tc.want, got, cmp.Comparer(func(a, b queryListDetails) bool {
				return a.Query == b.Query
			})); diff != "" {
				t.Errorf("Unexpected result: got %#v, want: %#v", got, tc.want)
			}
		})
	}
	wg.Wait()
}

func RunPostgresListAvailableExtensionsTest(t *testing.T) {
	invokeTcs := []struct {
		name           string
		api            string
		requestBody    io.Reader
		wantStatusCode int
	}{
		{
			name:           "invoke list_available_extensions output",
			api:            "http://127.0.0.1:5000/api/tool/list_available_extensions/invoke",
			wantStatusCode: http.StatusOK,
			requestBody:    bytes.NewBuffer([]byte(`{}`)),
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			resp, respBody := RunRequest(t, http.MethodPost, tc.api, tc.requestBody, nil)
			if resp.StatusCode != tc.wantStatusCode {
				t.Fatalf("response status code is not 200, got %d: %s", resp.StatusCode, string(respBody))
			}

			// Intentionally not adding the output check as output depends on the postgres instance used where the the functional test runs.
			// Adding the check will make the test flaky.
		})
	}
}

func RunPostgresListInstalledExtensionsTest(t *testing.T) {
	invokeTcs := []struct {
		name           string
		api            string
		requestBody    io.Reader
		wantStatusCode int
	}{
		{
			name:           "invoke list_installed_extensions output",
			api:            "http://127.0.0.1:5000/api/tool/list_installed_extensions/invoke",
			wantStatusCode: http.StatusOK,
			requestBody:    bytes.NewBuffer([]byte(`{}`)),
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			resp, bodyBytes := RunRequest(t, http.MethodPost, tc.api, tc.requestBody, nil)
			if resp.StatusCode != tc.wantStatusCode {
				t.Fatalf("response status code is not 200, got %d: %s", resp.StatusCode, string(bodyBytes))
			}

			// Intentionally not adding the output check as output depends on the postgres instance used where the the functional test runs.
			// Adding the check will make the test flaky.
		})
	}
}

func setupPostgresIndex(t *testing.T, ctx context.Context, pool *pgxpool.Pool, schemaName string, tableName string) func(t *testing.T) {
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

	return func(t *testing.T) {
		t.Helper()
		if _, err := pool.Exec(ctx, fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE;", schemaName)); err != nil {
			t.Errorf("unable to drop schema: %v", err)
		}
	}
}

func RunPostgresListIndexesTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	schemaName := "testschema_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableName := "table1_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	cleanup := setupPostgresIndex(t, ctx, pool, schemaName, tableName)
	defer cleanup(t)

	// Primary key index
	wantIndexPK := map[string]any{
		"schema_name":      schemaName,
		"table_name":       tableName,
		"index_name":       tableName + "_pkey",
		"index_type":       "btree",
		"is_unique":        true,
		"is_primary":       true,
		"is_used":          false,
		"index_definition": fmt.Sprintf("CREATE UNIQUE INDEX %s_pkey ON %s.%s USING btree (id)", tableName, schemaName, tableName),
		// Size and scan counts can vary, so omitting them from strict checks or using ranges might be better in real tests.
	}
	// Email unique index
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
	// Name non-unique index
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

	allWantIndexes := []map[string]any{wantIndexEmail, wantIndexName, wantIndexPK}

	invokeTcs := []struct {
		name           string
		requestBody    io.Reader
		wantStatusCode int
		want           []map[string]any
	}{
		// List all indexes is skipped because the output might include indexes for other database tables
		// defined outside of this test, which could make the test flaky.
		{
			name:           "list_indexes for a specific schema and table",
			requestBody:    bytes.NewBufferString(fmt.Sprintf(`{"schema_name": "%s", "table_name": "%s"}`, schemaName, tableName)),
			wantStatusCode: http.StatusOK,
			want:           allWantIndexes,
		},
		{
			name:           "list_indexes for a specific schema",
			requestBody:    bytes.NewBufferString(fmt.Sprintf(`{"schema_name": "%s"}`, schemaName)),
			wantStatusCode: http.StatusOK,
			want:           allWantIndexes,
		},
		{
			name:           "list_indexes with non-existent schema",
			requestBody:    bytes.NewBufferString(`{"schema_name": "non_existent_schema"}`),
			wantStatusCode: http.StatusOK,
			want:           nil,
		},
		{
			name:           "list_indexes with non-existent table in existing schema",
			requestBody:    bytes.NewBufferString(fmt.Sprintf(`{"schema_name": "%s", "table_name": "non_existent_table"}`, schemaName)),
			wantStatusCode: http.StatusOK,
			want:           nil,
		},
		{
			name:           "list_indexes filter by index name",
			requestBody:    bytes.NewBufferString(fmt.Sprintf(`{"schema_name": "%s", "table_name": "%s", "index_name": "%s"}`, schemaName, tableName, tableName+"_email_idx")),
			wantStatusCode: http.StatusOK,
			want:           []map[string]any{wantIndexEmail},
		},
		{
			name:           "list_indexes filter by non-existent index name",
			requestBody:    bytes.NewBufferString(fmt.Sprintf(`{"schema_name": "%s", "table_name": "%s", "index_name": "non_existent_idx"}`, schemaName, tableName)),
			wantStatusCode: http.StatusOK,
			want:           nil,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			const api = "http://127.0.0.1:5000/api/tool/list_indexes/invoke"

			resp, respBody := RunRequest(t, http.MethodPost, api, tc.requestBody, nil)
			if resp.StatusCode != tc.wantStatusCode {
				t.Fatalf("wrong status code: got %d, want %d, body: %s", resp.StatusCode, tc.wantStatusCode, string(respBody))
			}
			if tc.wantStatusCode != http.StatusOK {
				return
			}

			var bodyWrapper struct {
				Result json.RawMessage `json:"result"`
			}
			if err := json.Unmarshal(respBody, &bodyWrapper); err != nil {
				t.Fatalf("error decoding response wrapper: %v", err)
			}

			var resultString string
			if err := json.Unmarshal(bodyWrapper.Result, &resultString); err != nil {
				resultString = string(bodyWrapper.Result)
			}

			var got []map[string]any
			if err := json.Unmarshal([]byte(resultString), &got); err != nil {
				t.Fatalf("failed to unmarshal nested result string: %v, resultString: %s", err, resultString)
			}
			// Normalize got by removing fields that are hard to predict (like size)
			for _, index := range got {
				delete(index, "index_size_bytes")
				delete(index, "index_scans")
				delete(index, "tuples_read")
				delete(index, "tuples_fetched")
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Unexpected result (-want +got):\n%s", diff)
			}
		})
	}
}

func setupListSequencesTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool) (string, func(t *testing.T)) {
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

func RunPostgresListSequencesTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	sequenceName, teardown := setupListSequencesTest(t, ctx, pool)
	defer teardown(t)

	wantSequence := map[string]any{
		"sequencename":  sequenceName,
		"schemaname":    "public",
		"sequenceowner": "postgres",
		"data_type":     "bigint",
		"start_value":   float64(1),
		"min_value":     float64(1),
		"max_value":     float64(9223372036854775807),
		"increment_by":  float64(1),
		"last_value":    nil,
	}

	invokeTcs := []struct {
		name           string
		api            string
		requestBody    io.Reader
		wantStatusCode int
		want           []map[string]any
	}{
		{
			name:           "invoke list_sequences",
			requestBody:    bytes.NewBufferString(fmt.Sprintf(`{"sequencename": "%s"}`, sequenceName)),
			wantStatusCode: http.StatusOK,
			want:           []map[string]any{wantSequence},
		},
		{
			name:           "invoke list_sequences with non-existent sequence",
			requestBody:    bytes.NewBufferString(`{"sequencename": "non_existent_sequence"}`),
			wantStatusCode: http.StatusOK,
			want:           nil,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			const api = "http://127.0.0.1:5000/api/tool/list_sequences/invoke"
			resp, respBody := RunRequest(t, http.MethodPost, api, tc.requestBody, nil)
			if resp.StatusCode != tc.wantStatusCode {
				t.Fatalf("wrong status code: got %d, want %d, body: %s", resp.StatusCode, tc.wantStatusCode, string(respBody))
			}
			if tc.wantStatusCode != http.StatusOK {
				return
			}

			var bodyWrapper struct {
				Result json.RawMessage `json:"result"`
			}
			if err := json.Unmarshal(respBody, &bodyWrapper); err != nil {
				t.Fatalf("error decoding response wrapper: %v", err)
			}

			var resultString string
			if err := json.Unmarshal(bodyWrapper.Result, &resultString); err != nil {
				resultString = string(bodyWrapper.Result)
			}

			var got []map[string]any
			if err := json.Unmarshal([]byte(resultString), &got); err != nil {
				t.Fatalf("failed to unmarshal nested result string: %v", err)
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Unexpected result (-want +got):\n%s", diff)
			}
		})
	}
}

// RunMySQLListTablesTest run tests against the mysql-list-tables tool
func RunMySQLListTablesTest(t *testing.T, databaseName, tableNameParam, tableNameAuth string) {
	type tableInfo struct {
		ObjectName    string `json:"object_name"`
		SchemaName    string `json:"schema_name"`
		ObjectDetails string `json:"object_details"`
	}

	type column struct {
		DataType        string `json:"data_type"`
		ColumnName      string `json:"column_name"`
		ColumnComment   string `json:"column_comment"`
		ColumnDefault   any    `json:"column_default"`
		IsNotNullable   int    `json:"is_not_nullable"`
		OrdinalPosition int    `json:"ordinal_position"`
	}

	type objectDetails struct {
		Owner       any      `json:"owner"`
		Columns     []column `json:"columns"`
		Comment     string   `json:"comment"`
		Indexes     []any    `json:"indexes"`
		Triggers    []any    `json:"triggers"`
		Constraints []any    `json:"constraints"`
		ObjectName  string   `json:"object_name"`
		ObjectType  string   `json:"object_type"`
		SchemaName  string   `json:"schema_name"`
	}

	paramTableWant := objectDetails{
		ObjectName: tableNameParam,
		SchemaName: databaseName,
		ObjectType: "TABLE",
		Columns: []column{
			{DataType: "int", ColumnName: "id", IsNotNullable: 1, OrdinalPosition: 1},
			{DataType: "varchar(255)", ColumnName: "name", OrdinalPosition: 2},
		},
		Indexes:     []any{map[string]any{"index_columns": []any{"id"}, "index_name": "PRIMARY", "is_primary": float64(1), "is_unique": float64(1)}},
		Triggers:    []any{},
		Constraints: []any{map[string]any{"constraint_columns": []any{"id"}, "constraint_name": "PRIMARY", "constraint_type": "PRIMARY KEY", "foreign_key_referenced_columns": any(nil), "foreign_key_referenced_table": any(nil), "constraint_definition": ""}},
	}

	authTableWant := objectDetails{
		ObjectName: tableNameAuth,
		SchemaName: databaseName,
		ObjectType: "TABLE",
		Columns: []column{
			{DataType: "int", ColumnName: "id", IsNotNullable: 1, OrdinalPosition: 1},
			{DataType: "varchar(255)", ColumnName: "name", OrdinalPosition: 2},
			{DataType: "varchar(255)", ColumnName: "email", OrdinalPosition: 3},
		},
		Indexes:     []any{map[string]any{"index_columns": []any{"id"}, "index_name": "PRIMARY", "is_primary": float64(1), "is_unique": float64(1)}},
		Triggers:    []any{},
		Constraints: []any{map[string]any{"constraint_columns": []any{"id"}, "constraint_name": "PRIMARY", "constraint_type": "PRIMARY KEY", "foreign_key_referenced_columns": any(nil), "foreign_key_referenced_table": any(nil), "constraint_definition": ""}},
	}

	invokeTcs := []struct {
		name           string
		requestBody    io.Reader
		wantStatusCode int
		want           any
		isSimple       bool
		isAllTables    bool
	}{
		{
			name:           "invoke list_tables for all tables detailed output",
			requestBody:    bytes.NewBufferString(`{"table_names":""}`),
			wantStatusCode: http.StatusOK,
			want:           []objectDetails{authTableWant, paramTableWant},
			isAllTables:    true,
		},
		{
			name:           "invoke list_tables detailed output",
			requestBody:    bytes.NewBufferString(fmt.Sprintf(`{"table_names": "%s"}`, tableNameAuth)),
			wantStatusCode: http.StatusOK,
			want:           []objectDetails{authTableWant},
		},
		{
			name:           "invoke list_tables simple output",
			requestBody:    bytes.NewBufferString(fmt.Sprintf(`{"table_names": "%s", "output_format": "simple"}`, tableNameAuth)),
			wantStatusCode: http.StatusOK,
			want:           []map[string]any{{"name": tableNameAuth}},
			isSimple:       true,
		},
		{
			name:           "invoke list_tables with multiple table names",
			requestBody:    bytes.NewBufferString(fmt.Sprintf(`{"table_names": "%s,%s"}`, tableNameParam, tableNameAuth)),
			wantStatusCode: http.StatusOK,
			want:           []objectDetails{authTableWant, paramTableWant},
		},
		{
			name:           "invoke list_tables with one existing and one non-existent table",
			requestBody:    bytes.NewBufferString(fmt.Sprintf(`{"table_names": "%s,non_existent_table"}`, tableNameAuth)),
			wantStatusCode: http.StatusOK,
			want:           []objectDetails{authTableWant},
		},
		{
			name:           "invoke list_tables with non-existent table",
			requestBody:    bytes.NewBufferString(`{"table_names": "non_existent_table"}`),
			wantStatusCode: http.StatusOK,
			want:           nil,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			const api = "http://127.0.0.1:5000/api/tool/list_tables/invoke"
			resp, body := RunRequest(t, http.MethodPost, api, tc.requestBody, nil)
			if resp.StatusCode != tc.wantStatusCode {
				t.Fatalf("wrong status code: got %d, want %d, body: %s", resp.StatusCode, tc.wantStatusCode, string(body))
			}
			if tc.wantStatusCode != http.StatusOK {
				return
			}

			var bodyWrapper struct {
				Result json.RawMessage `json:"result"`
			}
			if err := json.Unmarshal(body, &bodyWrapper); err != nil {
				t.Fatalf("error decoding response wrapper: %v", err)
			}

			var resultString string
			if err := json.Unmarshal(bodyWrapper.Result, &resultString); err != nil {
				resultString = string(bodyWrapper.Result)
			}

			var got any
			if tc.isSimple {
				var tables []tableInfo
				if err := json.Unmarshal([]byte(resultString), &tables); err != nil {
					t.Fatalf("failed to unmarshal outer JSON array into []tableInfo: %v", err)
				}
				var details []map[string]any
				for _, table := range tables {
					var d map[string]any
					if err := json.Unmarshal([]byte(table.ObjectDetails), &d); err != nil {
						t.Fatalf("failed to unmarshal nested ObjectDetails string: %v", err)
					}
					details = append(details, d)
				}
				got = details
			} else {
				if resultString == "null" {
					got = nil
				} else {
					var tables []tableInfo
					if err := json.Unmarshal([]byte(resultString), &tables); err != nil {
						t.Fatalf("failed to unmarshal outer JSON array into []tableInfo: %v", err)
					}
					var details []objectDetails
					for _, table := range tables {
						var d objectDetails
						if err := json.Unmarshal([]byte(table.ObjectDetails), &d); err != nil {
							t.Fatalf("failed to unmarshal nested ObjectDetails string: %v", err)
						}
						details = append(details, d)
					}
					got = details
				}
			}

			opts := []cmp.Option{
				cmpopts.SortSlices(func(a, b objectDetails) bool { return a.ObjectName < b.ObjectName }),
				cmpopts.SortSlices(func(a, b column) bool { return a.ColumnName < b.ColumnName }),
				cmpopts.SortSlices(func(a, b map[string]any) bool { return a["name"].(string) < b["name"].(string) }),
			}

			// Checking only the current database where the test tables are created to avoid brittle tests.
			if tc.isAllTables {
				var filteredGot []objectDetails
				if got != nil {
					for _, item := range got.([]objectDetails) {
						if item.SchemaName == databaseName {
							filteredGot = append(filteredGot, item)
						}
					}
				}
				if len(filteredGot) == 0 {
					got = nil
				} else {
					got = filteredGot
				}
			}

			if diff := cmp.Diff(tc.want, got, opts...); diff != "" {
				t.Errorf("Unexpected result: got %#v, want: %#v", got, tc.want)
			}
		})
	}
}

// RunMySQLListActiveQueriesTest run tests against the mysql-list-active-queries tests
func RunMySQLListActiveQueriesTest(t *testing.T, ctx context.Context, pool *sql.DB) {
	type queryListDetails struct {
		ProcessId       any    `json:"process_id"`
		Query           string `json:"query"`
		TrxStarted      any    `json:"trx_started"`
		TrxDuration     any    `json:"trx_duration_seconds"`
		TrxWaitDuration any    `json:"trx_wait_duration_seconds"`
		QueryTime       any    `json:"query_time"`
		TrxState        string `json:"trx_state"`
		ProcessState    string `json:"process_state"`
		User            string `json:"user"`
		TrxRowsLocked   any    `json:"trx_rows_locked"`
		TrxRowsModified any    `json:"trx_rows_modified"`
		Db              string `json:"db"`
	}

	singleQueryWanted := queryListDetails{
		ProcessId:       any(nil),
		Query:           "SELECT sleep(10)",
		TrxStarted:      any(nil),
		TrxDuration:     any(nil),
		TrxWaitDuration: any(nil),
		QueryTime:       any(nil),
		TrxState:        "",
		ProcessState:    "User sleep",
		User:            "",
		TrxRowsLocked:   any(nil),
		TrxRowsModified: any(nil),
		Db:              "",
	}

	invokeTcs := []struct {
		name                string
		requestBody         io.Reader
		clientSleepSecs     int
		waitSecsBeforeCheck int
		wantStatusCode      int
		want                any
	}{
		{
			name:                "invoke list_active_queries when the system is idle",
			requestBody:         bytes.NewBufferString(`{}`),
			clientSleepSecs:     0,
			waitSecsBeforeCheck: 0,
			wantStatusCode:      http.StatusOK,
			want:                []queryListDetails(nil),
		},
		{
			name:                "invoke list_active_queries when there is 1 ongoing but lower than the threshold",
			requestBody:         bytes.NewBufferString(`{"min_duration_secs": 100}`),
			clientSleepSecs:     10,
			waitSecsBeforeCheck: 1,
			wantStatusCode:      http.StatusOK,
			want:                []queryListDetails(nil),
		},
		{
			name:                "invoke list_active_queries when 1 ongoing query should show up",
			requestBody:         bytes.NewBufferString(`{"min_duration_secs": 5}`),
			clientSleepSecs:     0,
			waitSecsBeforeCheck: 5,
			wantStatusCode:      http.StatusOK,
			want:                []queryListDetails{singleQueryWanted},
		},
		{
			name:                "invoke list_active_queries when 2 ongoing query should show up",
			requestBody:         bytes.NewBufferString(`{"min_duration_secs": 2}`),
			clientSleepSecs:     10,
			waitSecsBeforeCheck: 3,
			wantStatusCode:      http.StatusOK,
			want:                []queryListDetails{singleQueryWanted, singleQueryWanted},
		},
	}

	var wg sync.WaitGroup
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			if tc.clientSleepSecs > 0 {
				wg.Add(1)

				go func() {
					defer wg.Done()

					err := pool.PingContext(ctx)
					if err != nil {
						t.Errorf("unable to connect to test database: %s", err)
						return
					}
					_, err = pool.ExecContext(ctx, fmt.Sprintf("SELECT sleep(%d);", tc.clientSleepSecs))
					if err != nil {
						t.Errorf("Executing 'SELECT sleep' failed: %s", err)
					}
				}()
			}

			if tc.waitSecsBeforeCheck > 0 {
				time.Sleep(time.Duration(tc.waitSecsBeforeCheck) * time.Second)
			}

			const api = "http://127.0.0.1:5000/api/tool/list_active_queries/invoke"
			resp, respBody := RunRequest(t, http.MethodPost, api, tc.requestBody, nil)
			if resp.StatusCode != tc.wantStatusCode {
				t.Fatalf("wrong status code: got %d, want %d, body: %s", resp.StatusCode, tc.wantStatusCode, string(respBody))
			}
			if tc.wantStatusCode != http.StatusOK {
				return
			}

			var bodyWrapper struct {
				Result json.RawMessage `json:"result"`
			}
			if err := json.Unmarshal(respBody, &bodyWrapper); err != nil {
				t.Fatalf("error decoding response wrapper: %v", err)
			}

			var resultString string
			if err := json.Unmarshal(bodyWrapper.Result, &resultString); err != nil {
				resultString = string(bodyWrapper.Result)
			}

			var got any
			var details []queryListDetails
			if err := json.Unmarshal([]byte(resultString), &details); err != nil {
				t.Fatalf("failed to unmarshal nested ObjectDetails string: %v", err)
			}
			got = details

			if diff := cmp.Diff(tc.want, got, cmp.Comparer(func(a, b queryListDetails) bool {
				return a.Query == b.Query && a.ProcessState == b.ProcessState
			})); diff != "" {
				t.Errorf("Unexpected result: got %#v, want: %#v", got, tc.want)
			}
		})
	}
	wg.Wait()
}

func RunMySQLListTablesMissingUniqueIndexes(t *testing.T, ctx context.Context, pool *sql.DB, databaseName string) {
	type listDetails struct {
		TableSchema string `json:"table_schema"`
		TableName   string `json:"table_name"`
	}

	// bunch of wanted
	nonUniqueKeyTableName := "t03_non_unqiue_key_table"
	noKeyTableName := "t04_no_key_table"
	nonUniqueKeyTableWant := listDetails{
		TableSchema: databaseName,
		TableName:   nonUniqueKeyTableName,
	}
	noKeyTableWant := listDetails{
		TableSchema: databaseName,
		TableName:   noKeyTableName,
	}

	invokeTcs := []struct {
		name                 string
		requestBody          io.Reader
		newTableName         string
		newTablePrimaryKey   bool
		newTableUniqueKey    bool
		newTableNonUniqueKey bool
		wantStatusCode       int
		want                 any
	}{
		{
			name:                 "invoke list_tables_missing_unique_indexes when nothing to be found",
			requestBody:          bytes.NewBufferString(`{}`),
			newTableName:         "",
			newTablePrimaryKey:   false,
			newTableUniqueKey:    false,
			newTableNonUniqueKey: false,
			wantStatusCode:       http.StatusOK,
			want:                 []listDetails(nil),
		},
		{
			name:                 "invoke list_tables_missing_unique_indexes pk table will not show",
			requestBody:          bytes.NewBufferString(`{}`),
			newTableName:         "t01",
			newTablePrimaryKey:   true,
			newTableUniqueKey:    false,
			newTableNonUniqueKey: false,
			wantStatusCode:       http.StatusOK,
			want:                 []listDetails(nil),
		},
		{
			name:                 "invoke list_tables_missing_unique_indexes uk table will not show",
			requestBody:          bytes.NewBufferString(`{}`),
			newTableName:         "t02",
			newTablePrimaryKey:   false,
			newTableUniqueKey:    true,
			newTableNonUniqueKey: false,
			wantStatusCode:       http.StatusOK,
			want:                 []listDetails(nil),
		},
		{
			name:                 "invoke list_tables_missing_unique_indexes non-unique key only table will show",
			requestBody:          bytes.NewBufferString(`{}`),
			newTableName:         nonUniqueKeyTableName,
			newTablePrimaryKey:   false,
			newTableUniqueKey:    false,
			newTableNonUniqueKey: true,
			wantStatusCode:       http.StatusOK,
			want:                 []listDetails{nonUniqueKeyTableWant},
		},
		{
			name:                 "invoke list_tables_missing_unique_indexes table with no key at all will show",
			requestBody:          bytes.NewBufferString(`{}`),
			newTableName:         noKeyTableName,
			newTablePrimaryKey:   false,
			newTableUniqueKey:    false,
			newTableNonUniqueKey: false,
			wantStatusCode:       http.StatusOK,
			want:                 []listDetails{nonUniqueKeyTableWant, noKeyTableWant},
		},
		{
			name:                 "invoke list_tables_missing_unique_indexes table w/ both pk & uk will not show",
			requestBody:          bytes.NewBufferString(`{}`),
			newTableName:         "t05",
			newTablePrimaryKey:   true,
			newTableUniqueKey:    true,
			newTableNonUniqueKey: false,
			wantStatusCode:       http.StatusOK,
			want:                 []listDetails{nonUniqueKeyTableWant, noKeyTableWant},
		},
		{
			name:                 "invoke list_tables_missing_unique_indexes table w/ uk & nk will not show",
			requestBody:          bytes.NewBufferString(`{}`),
			newTableName:         "t06",
			newTablePrimaryKey:   false,
			newTableUniqueKey:    true,
			newTableNonUniqueKey: true,
			wantStatusCode:       http.StatusOK,
			want:                 []listDetails{nonUniqueKeyTableWant, noKeyTableWant},
		},
		{
			name:                 "invoke list_tables_missing_unique_indexes table w/ pk & nk will not show",
			requestBody:          bytes.NewBufferString(`{}`),
			newTableName:         "t07",
			newTablePrimaryKey:   true,
			newTableUniqueKey:    false,
			newTableNonUniqueKey: true,
			wantStatusCode:       http.StatusOK,
			want:                 []listDetails{nonUniqueKeyTableWant, noKeyTableWant},
		},
		{
			name:                 "invoke list_tables_missing_unique_indexes with a non-exist database, nothing to show",
			requestBody:          bytes.NewBufferString(`{"table_schema": "non-exist-database"}`),
			newTableName:         "",
			newTablePrimaryKey:   false,
			newTableUniqueKey:    false,
			newTableNonUniqueKey: false,
			wantStatusCode:       http.StatusOK,
			want:                 []listDetails(nil),
		},
		{
			name:                 "invoke list_tables_missing_unique_indexes with the right database, show everything",
			requestBody:          bytes.NewBufferString(fmt.Sprintf(`{"table_schema": "%s"}`, databaseName)),
			newTableName:         "",
			newTablePrimaryKey:   false,
			newTableUniqueKey:    false,
			newTableNonUniqueKey: false,
			wantStatusCode:       http.StatusOK,
			want:                 []listDetails{nonUniqueKeyTableWant, noKeyTableWant},
		},
		{
			name:                 "invoke list_tables_missing_unique_indexes with limited output",
			requestBody:          bytes.NewBufferString(`{"limit": 1}`),
			newTableName:         "",
			newTablePrimaryKey:   false,
			newTableUniqueKey:    false,
			newTableNonUniqueKey: false,
			wantStatusCode:       http.StatusOK,
			want:                 []listDetails{nonUniqueKeyTableWant},
		},
	}

	createTableHelper := func(t *testing.T, tableName, databaseName string, primaryKey, uniqueKey, nonUniqueKey bool, ctx context.Context, pool *sql.DB) func() {
		var stmt strings.Builder
		stmt.WriteString(fmt.Sprintf("CREATE TABLE %s (", tableName))
		stmt.WriteString("c1 INT")
		if primaryKey {
			stmt.WriteString(" PRIMARY KEY")
		}
		stmt.WriteString(", c2 INT, c3 CHAR(8)")
		if uniqueKey {
			stmt.WriteString(", UNIQUE(c2)")
		}
		if nonUniqueKey {
			stmt.WriteString(", INDEX(c3)")
		}
		stmt.WriteString(")")

		t.Logf("Creating table: %s", stmt.String())
		if _, err := pool.ExecContext(ctx, stmt.String()); err != nil {
			t.Fatalf("failed executing %s: %v", stmt.String(), err)
		}

		return func() {
			t.Logf("Dropping table: %s", tableName)
			if _, err := pool.ExecContext(ctx, fmt.Sprintf("DROP TABLE %s", tableName)); err != nil {
				t.Errorf("failed to drop table %s: %v", tableName, err)
			}
		}
	}

	var cleanups []func()
	defer func() {
		for i := len(cleanups) - 1; i >= 0; i-- {
			cleanups[i]()
		}
	}()

	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			if tc.newTableName != "" {
				cleanup := createTableHelper(t, tc.newTableName, databaseName, tc.newTablePrimaryKey, tc.newTableUniqueKey, tc.newTableNonUniqueKey, ctx, pool)
				cleanups = append(cleanups, cleanup)
			}

			const api = "http://127.0.0.1:5000/api/tool/list_tables_missing_unique_indexes/invoke"
			resp, respBody := RunRequest(t, http.MethodPost, api, tc.requestBody, nil)
			if resp.StatusCode != tc.wantStatusCode {
				t.Fatalf("wrong status code: got %d, want %d, body: %s", resp.StatusCode, tc.wantStatusCode, string(respBody))
			}
			if tc.wantStatusCode != http.StatusOK {
				return
			}

			var bodyWrapper struct {
				Result json.RawMessage `json:"result"`
			}
			if err := json.Unmarshal(respBody, &bodyWrapper); err != nil {
				t.Fatalf("error decoding response wrapper: %v", err)
			}

			var resultString string
			if err := json.Unmarshal(bodyWrapper.Result, &resultString); err != nil {
				resultString = string(bodyWrapper.Result)
			}

			var got any
			var details []listDetails
			if err := json.Unmarshal([]byte(resultString), &details); err != nil {
				t.Fatalf("failed to unmarshal nested listDetails string: %v", err)
			}
			got = details

			if diff := cmp.Diff(tc.want, got, cmp.Comparer(func(a, b listDetails) bool {
				return a.TableSchema == b.TableSchema && a.TableName == b.TableName
			})); diff != "" {
				t.Errorf("Unexpected result: got %#v, want: %#v", got, tc.want)
			}
		})
	}
}

func RunMySQLListTableFragmentationTest(t *testing.T, databaseName, tableNameParam, tableNameAuth string) {
	type tableFragmentationDetails struct {
		TableSchema             string `json:"table_schema"`
		TableName               string `json:"table_name"`
		DataSize                any    `json:"data_size"`
		IndexSize               any    `json:"index_size"`
		DataFree                any    `json:"data_free"`
		FragmentationPercentage any    `json:"fragmentation_percentage"`
	}

	paramTableEntryWanted := tableFragmentationDetails{
		TableSchema:             databaseName,
		TableName:               tableNameParam,
		DataSize:                any(nil),
		IndexSize:               any(nil),
		DataFree:                any(nil),
		FragmentationPercentage: any(nil),
	}
	authTableEntryWanted := tableFragmentationDetails{
		TableSchema:             databaseName,
		TableName:               tableNameAuth,
		DataSize:                any(nil),
		IndexSize:               any(nil),
		DataFree:                any(nil),
		FragmentationPercentage: any(nil),
	}

	invokeTcs := []struct {
		name           string
		requestBody    io.Reader
		wantStatusCode int
		want           any
	}{
		{
			name:           "invoke list_table_fragmentation on all, no data_free threshold, expected to have 2 results",
			requestBody:    bytes.NewBufferString(`{"data_free_threshold_bytes": 0}`),
			wantStatusCode: http.StatusOK,
			want:           []tableFragmentationDetails{authTableEntryWanted, paramTableEntryWanted},
		},
		{
			name:           "invoke list_table_fragmentation on all, no data_free threshold, limit to 1, expected to have 1 results",
			requestBody:    bytes.NewBufferString(`{"data_free_threshold_bytes": 0, "limit": 1}`),
			wantStatusCode: http.StatusOK,
			want:           []tableFragmentationDetails{authTableEntryWanted},
		},
		{
			name:           "invoke list_table_fragmentation on all databases and 1 specific table name, no data_free threshold, expected to have 1 result",
			requestBody:    bytes.NewBufferString(fmt.Sprintf(`{"table_name": "%s","data_free_threshold_bytes": 0}`, tableNameAuth)),
			wantStatusCode: http.StatusOK,
			want:           []tableFragmentationDetails{authTableEntryWanted},
		},
		{
			name:           "invoke list_table_fragmentation on 1 database and 1 specific table name, no data_free threshold, expected to have 1 result",
			requestBody:    bytes.NewBufferString(fmt.Sprintf(`{"table_schema": "%s", "table_name": "%s", "data_free_threshold_bytes": 0}`, databaseName, tableNameParam)),
			wantStatusCode: http.StatusOK,
			want:           []tableFragmentationDetails{paramTableEntryWanted},
		},
		{
			name:           "invoke list_table_fragmentation on 1 database and 1 specific table name, high data_free threshold, expected to have 0 result",
			requestBody:    bytes.NewBufferString(fmt.Sprintf(`{"table_schema": "%s", "table_name": "%s", "data_free_threshold_bytes": 1000000000}`, databaseName, tableNameParam)),
			wantStatusCode: http.StatusOK,
			want:           []tableFragmentationDetails(nil),
		},
		{
			name:           "invoke list_table_fragmentation on 1 non-exist database, no data_free threshold, expected to have 0 result",
			requestBody:    bytes.NewBufferString(`{"table_schema": "non_existent_database", "data_free_threshold_bytes": 0}`),
			wantStatusCode: http.StatusOK,
			want:           []tableFragmentationDetails(nil),
		},
		{
			name:           "invoke list_table_fragmentation on 1 non-exist table, no data_free threshold, expected to have 0 result",
			requestBody:    bytes.NewBufferString(`{"table_name": "non_existent_table", "data_free_threshold_bytes": 0}`),
			wantStatusCode: http.StatusOK,
			want:           []tableFragmentationDetails(nil),
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			const api = "http://127.0.0.1:5000/api/tool/list_table_fragmentation/invoke"
			resp, respBody := RunRequest(t, http.MethodPost, api, tc.requestBody, nil)
			if resp.StatusCode != tc.wantStatusCode {
				t.Fatalf("wrong status code: got %d, want %d, body: %s", resp.StatusCode, tc.wantStatusCode, string(respBody))
			}
			if tc.wantStatusCode != http.StatusOK {
				return
			}

			var bodyWrapper struct {
				Result json.RawMessage `json:"result"`
			}
			if err := json.Unmarshal(respBody, &bodyWrapper); err != nil {
				t.Fatalf("error decoding response wrapper: %v", err)
			}

			var resultString string
			if err := json.Unmarshal(bodyWrapper.Result, &resultString); err != nil {
				resultString = string(bodyWrapper.Result)
			}

			var got any
			var details []tableFragmentationDetails
			if err := json.Unmarshal([]byte(resultString), &details); err != nil {
				t.Fatalf("failed to unmarshal outer JSON array into []tableInfo: %v", err)
			}
			got = details

			if diff := cmp.Diff(tc.want, got, cmp.Comparer(func(a, b tableFragmentationDetails) bool {
				return a.TableSchema == b.TableSchema && a.TableName == b.TableName
			})); diff != "" {
				t.Errorf("Unexpected result: got %#v, want: %#v", got, tc.want)
			}
		})
	}
}

// RunMSSQLListTablesTest run tests againsts the mssql-list-tables tools.
func RunMSSQLListTablesTest(t *testing.T, tableNameParam, tableNameAuth string) {
	// TableNameParam columns to construct want.
	const paramTableColumns = `[
        {"column_name": "id", "data_type": "INT", "column_ordinal_position": 1, "is_not_nullable": true},
        {"column_name": "name", "data_type": "VARCHAR(255)", "column_ordinal_position": 2, "is_not_nullable": false}
    ]`

	// TableNameAuth columns to construct want
	const authTableColumns = `[
		{"column_name": "id", "data_type": "INT", "column_ordinal_position": 1, "is_not_nullable": true},
		{"column_name": "name", "data_type": "VARCHAR(255)", "column_ordinal_position": 2, "is_not_nullable": false},
		{"column_name": "email", "data_type": "VARCHAR(255)", "column_ordinal_position": 3, "is_not_nullable": false}
    ]`

	const (
		// Template to construct detailed output want.
		detailedObjectTemplate = `{
            "schema_name": "dbo",
            "object_name": "%[1]s",
            "object_details": {
                "owner": "dbo",
                "triggers": [],
                "columns": %[2]s,
                "object_name": "%[1]s",
                "object_type": "TABLE",
                "schema_name": "dbo"
            }
        }`

		// Template to construct simple output want
		simpleObjectTemplate = `{"object_name":"%s", "schema_name":"dbo", "object_details":{"name":"%s"}}`
	)

	// Helper to build json for detailed want
	getDetailedWant := func(tableName, columnJSON string) string {
		return fmt.Sprintf(detailedObjectTemplate, tableName, columnJSON)
	}

	// Helper to build template for simple want
	getSimpleWant := func(tableName string) string {
		return fmt.Sprintf(simpleObjectTemplate, tableName, tableName)
	}

	invokeTcs := []struct {
		name           string
		api            string
		requestBody    string
		wantStatusCode int
		want           string
		isAllTables    bool
	}{
		{
			name:           "invoke list_tables for all tables detailed output",
			api:            "http://127.0.0.1:5000/api/tool/list_tables/invoke",
			requestBody:    `{"table_names": ""}`,
			wantStatusCode: http.StatusOK,
			want:           fmt.Sprintf("[%s,%s]", getDetailedWant(tableNameAuth, authTableColumns), getDetailedWant(tableNameParam, paramTableColumns)),
			isAllTables:    true,
		},
		{
			name:           "invoke list_tables for all tables simple output",
			api:            "http://127.0.0.1:5000/api/tool/list_tables/invoke",
			requestBody:    `{"table_names": "", "output_format": "simple"}`,
			wantStatusCode: http.StatusOK,
			want:           fmt.Sprintf("[%s,%s]", getSimpleWant(tableNameAuth), getSimpleWant(tableNameParam)),
			isAllTables:    true,
		},
		{
			name:           "invoke list_tables detailed output",
			api:            "http://127.0.0.1:5000/api/tool/list_tables/invoke",
			requestBody:    fmt.Sprintf(`{"table_names": "%s"}`, tableNameAuth),
			wantStatusCode: http.StatusOK,
			want:           fmt.Sprintf("[%s]", getDetailedWant(tableNameAuth, authTableColumns)),
		},
		{
			name:           "invoke list_tables simple output",
			api:            "http://127.0.0.1:5000/api/tool/list_tables/invoke",
			requestBody:    fmt.Sprintf(`{"table_names": "%s", "output_format": "simple"}`, tableNameAuth),
			wantStatusCode: http.StatusOK,
			want:           fmt.Sprintf("[%s]", getSimpleWant(tableNameAuth)),
		},
		{
			name:           "invoke list_tables with invalid output format",
			api:            "http://127.0.0.1:5000/api/tool/list_tables/invoke",
			requestBody:    `{"table_names": "", "output_format": "abcd"}`,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "invoke list_tables with malformed table_names parameter",
			api:            "http://127.0.0.1:5000/api/tool/list_tables/invoke",
			requestBody:    `{"table_names": 12345, "output_format": "detailed"}`,
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "invoke list_tables with multiple table names",
			api:            "http://127.0.0.1:5000/api/tool/list_tables/invoke",
			requestBody:    fmt.Sprintf(`{"table_names": "%s,%s"}`, tableNameParam, tableNameAuth),
			wantStatusCode: http.StatusOK,
			want:           fmt.Sprintf("[%s,%s]", getDetailedWant(tableNameAuth, authTableColumns), getDetailedWant(tableNameParam, paramTableColumns)),
		},
		{
			name:           "invoke list_tables with non-existent table",
			api:            "http://127.0.0.1:5000/api/tool/list_tables/invoke",
			requestBody:    `{"table_names": "non_existent_table"}`,
			wantStatusCode: http.StatusOK,
			want:           `null`,
		},
		{
			name:           "invoke list_tables with one existing and one non-existent table",
			api:            "http://127.0.0.1:5000/api/tool/list_tables/invoke",
			requestBody:    fmt.Sprintf(`{"table_names": "%s,non_existent_table"}`, tableNameParam),
			wantStatusCode: http.StatusOK,
			want:           fmt.Sprintf("[%s]", getDetailedWant(tableNameParam, paramTableColumns)),
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			resp, respBytes := RunRequest(t, http.MethodPost, tc.api, bytes.NewBuffer([]byte(tc.requestBody)), nil)

			if resp.StatusCode != tc.wantStatusCode {
				t.Fatalf("response status code is not %d, got %d: %s", tc.wantStatusCode, resp.StatusCode, string(respBytes))
			}

			if tc.wantStatusCode == http.StatusOK {
				var bodyWrapper map[string]json.RawMessage

				if err := json.Unmarshal(respBytes, &bodyWrapper); err != nil {
					t.Fatalf("error parsing response wrapper: %s, body: %s", err, string(respBytes))
				}

				resultJSON, ok := bodyWrapper["result"]
				if !ok {
					t.Fatal("unable to find 'result' in response body")
				}

				var resultString string
				if err := json.Unmarshal(resultJSON, &resultString); err != nil {
					if string(resultJSON) == "null" {
						resultString = "null"
					} else {
						t.Fatalf("'result' is not a JSON-encoded string: %s", err)
					}
				}

				var got, want []any

				if err := json.Unmarshal([]byte(resultString), &got); err != nil {
					t.Fatalf("failed to unmarshal actual result string: %v", err)
				}
				if err := json.Unmarshal([]byte(tc.want), &want); err != nil {
					t.Fatalf("failed to unmarshal expected want string: %v", err)
				}

				for _, item := range got {
					itemMap, ok := item.(map[string]any)
					if !ok {
						continue
					}

					detailsStr, ok := itemMap["object_details"].(string)
					if !ok {
						continue
					}

					var detailsMap map[string]any
					if err := json.Unmarshal([]byte(detailsStr), &detailsMap); err != nil {
						t.Fatalf("failed to unmarshal nested object_details string: %v", err)
					}

					// clean unpredictable fields
					delete(detailsMap, "constraints")
					delete(detailsMap, "indexes")

					itemMap["object_details"] = detailsMap
				}

				// Checking only the default dbo schema where the test tables are created to avoid brittle tests.
				if tc.isAllTables {
					var filteredGot []any
					for _, item := range got {
						if tableMap, ok := item.(map[string]interface{}); ok {
							if schema, ok := tableMap["schema_name"]; ok && schema == "dbo" {
								filteredGot = append(filteredGot, item)
							}
						}
					}
					got = filteredGot
				}

				sort.SliceStable(got, func(i, j int) bool {
					return fmt.Sprintf("%v", got[i]) < fmt.Sprintf("%v", got[j])
				})
				sort.SliceStable(want, func(i, j int) bool {
					return fmt.Sprintf("%v", want[i]) < fmt.Sprintf("%v", want[j])
				})

				if !reflect.DeepEqual(got, want) {
					gotJSON, _ := json.MarshalIndent(got, "", "  ")
					wantJSON, _ := json.MarshalIndent(want, "", "  ")
					t.Errorf("Unexpected result:\ngot:\n%s\n\nwant:\n%s", string(gotJSON), string(wantJSON))
				}
			}
		})
	}
}

// RunPostgresListLocksTest runs tests for the postgres list-locks tool
func RunPostgresListLocksTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	type lockDetails struct {
		Pid           any    `json:"pid"`
		Usename       string `json:"usename"`
		Database      string `json:"database"`
		RelName       string `json:"relname"`
		LockType      string `json:"locktype"`
		Mode          string `json:"mode"`
		Granted       bool   `json:"granted"`
		FastPath      bool   `json:"fastpath"`
		VirtualXid    any    `json:"virtualxid"`
		TransactionId any    `json:"transactionid"`
		ClassId       any    `json:"classid"`
		ObjId         any    `json:"objid"`
		ObjSubId      any    `json:"objsubid"`
		PageNumber    any    `json:"page"`
		TupleNumber   any    `json:"tuple"`
		VirtualBlock  any    `json:"virtualblock"`
		BlockNumber   any    `json:"blockno"`
	}

	invokeTcs := []struct {
		name           string
		requestBody    io.Reader
		wantStatusCode int
		expectResults  bool
	}{
		{
			name:           "invoke list_locks with no arguments",
			requestBody:    bytes.NewBuffer([]byte(`{}`)),
			wantStatusCode: http.StatusOK,
			expectResults:  false, // locks may or may not exist
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			const api = "http://127.0.0.1:5000/api/tool/list_locks/invoke"
			resp, respBody := RunRequest(t, http.MethodPost, api, tc.requestBody, nil)
			if resp.StatusCode != tc.wantStatusCode {
				t.Fatalf("wrong status code: got %d, want %d, body: %s", resp.StatusCode, tc.wantStatusCode, string(respBody))
			}
			if tc.wantStatusCode != http.StatusOK {
				return
			}

			var bodyWrapper struct {
				Result json.RawMessage `json:"result"`
			}
			if err := json.Unmarshal(respBody, &bodyWrapper); err != nil {
				t.Fatalf("error decoding response wrapper: %v", err)
			}

			var resultString string
			if err := json.Unmarshal(bodyWrapper.Result, &resultString); err != nil {
				resultString = string(bodyWrapper.Result)
			}

			var got []lockDetails
			if resultString != "null" {
				if err := json.Unmarshal([]byte(resultString), &got); err != nil {
					t.Fatalf("failed to unmarshal result: %v, result string: %s", err, resultString)
				}
			}

			// Verify that if results exist, they have the expected structure
			for _, lock := range got {
				if lock.LockType == "" {
					t.Errorf("lock type should not be empty")
				}
			}
		})
	}
}

// RunPostgresLongRunningTransactionsTest runs tests for the postgres long-running-transactions tool
func RunPostgresLongRunningTransactionsTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	type transactionDetails struct {
		Pid               any    `json:"pid"`
		Usename           string `json:"usename"`
		Database          string `json:"database"`
		ApplicationName   string `json:"application_name"`
		XactStart         any    `json:"xact_start"`
		XactDurationSecs  any    `json:"xact_duration_secs"`
		IdleInTransaction string `json:"idle_in_transaction"`
		Query             string `json:"query"`
	}

	invokeTcs := []struct {
		name           string
		requestBody    io.Reader
		wantStatusCode int
	}{
		{
			name:           "invoke long_running_transactions with default threshold",
			requestBody:    bytes.NewBuffer([]byte(`{}`)),
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "invoke long_running_transactions with custom threshold",
			requestBody:    bytes.NewBuffer([]byte(`{"min_transaction_duration_secs": 3600}`)),
			wantStatusCode: http.StatusOK,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			const api = "http://127.0.0.1:5000/api/tool/long_running_transactions/invoke"
			resp, respBody := RunRequest(t, http.MethodPost, api, tc.requestBody, nil)
			if resp.StatusCode != tc.wantStatusCode {
				t.Fatalf("wrong status code: got %d, want %d, body: %s", resp.StatusCode, tc.wantStatusCode, string(respBody))
			}
			if tc.wantStatusCode != http.StatusOK {
				return
			}

			var bodyWrapper struct {
				Result json.RawMessage `json:"result"`
			}
			if err := json.Unmarshal(respBody, &bodyWrapper); err != nil {
				t.Fatalf("error decoding response wrapper: %v", err)
			}

			var resultString string
			if err := json.Unmarshal(bodyWrapper.Result, &resultString); err != nil {
				resultString = string(bodyWrapper.Result)
			}

			var got []transactionDetails
			if resultString != "null" {
				if err := json.Unmarshal([]byte(resultString), &got); err != nil {
					t.Fatalf("failed to unmarshal result: %v, result string: %s", err, resultString)
				}
			}

			// Verify that if results exist, they have the expected structure
			for _, tx := range got {
				if tx.XactDurationSecs == nil {
					t.Errorf("transaction duration should not be null for long-running transactions")
				}
			}
		})
	}
}

// RunPostgresReplicationStatsTest runs tests for the postgres replication-stats tool
func RunPostgresReplicationStatsTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	type replicationStats struct {
		ClientAddr          string `json:"client_addr"`
		Username            string `json:"usename"`
		ApplicationName     string `json:"application_name"`
		ClientHostname      string `json:"client_hostname"`
		BackendStart        any    `json:"backend_start"`
		State               string `json:"state"`
		SyncState           string `json:"sync_state"`
		ReplyTime           any    `json:"reply_time"`
		FlushLsn            string `json:"flush_lsn"`
		ReplayLsn           string `json:"replay_lsn"`
		WriteLag            any    `json:"write_lag"`
		FlushLag            any    `json:"flush_lag"`
		ReplayLag           any    `json:"replay_lag"`
		SyncPriority        any    `json:"sync_priority"`
		ReplicationSlotName any    `json:"slot_name"`
		IsStreaming         bool   `json:"is_streaming"`
	}

	invokeTcs := []struct {
		name           string
		requestBody    io.Reader
		wantStatusCode int
	}{
		{
			name:           "invoke replication_stats with no arguments",
			requestBody:    bytes.NewBuffer([]byte(`{}`)),
			wantStatusCode: http.StatusOK,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			const api = "http://127.0.0.1:5000/api/tool/replication_stats/invoke"
			resp, respBody := RunRequest(t, http.MethodPost, api, tc.requestBody, nil)
			if resp.StatusCode != tc.wantStatusCode {
				t.Fatalf("wrong status code: got %d, want %d, body: %s", resp.StatusCode, tc.wantStatusCode, string(respBody))
			}
			if tc.wantStatusCode != http.StatusOK {
				return
			}

			var bodyWrapper struct {
				Result json.RawMessage `json:"result"`
			}
			if err := json.Unmarshal(respBody, &bodyWrapper); err != nil {
				t.Fatalf("error decoding response wrapper: %v", err)
			}

			var resultString string
			if err := json.Unmarshal(bodyWrapper.Result, &resultString); err != nil {
				resultString = string(bodyWrapper.Result)
			}

			var got []replicationStats
			if resultString != "null" {
				if err := json.Unmarshal([]byte(resultString), &got); err != nil {
					t.Fatalf("failed to unmarshal result: %v, result string: %s", err, resultString)
				}
			}

			// Verify that if results exist, they have the expected structure
			for _, stat := range got {
				if stat.State == "" {
					t.Errorf("replication state should not be empty")
				}
			}
		})
	}
}

func RunPostgresGetColumnCardinalityTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	schemaName := "testschema_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	tableName := "table1_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	cleanup := setupPostgresSchemas(t, ctx, pool, schemaName)
	defer cleanup()

	// Create table with multiple columns
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

	// Insert larger sample data to ensure statistics are collected
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

	// Run ANALYZE to update statistics
	analyzeStmt := fmt.Sprintf(`ANALYZE %s.%s`, schemaName, tableName)
	if _, err := pool.Exec(ctx, analyzeStmt); err != nil {
		t.Fatalf("unable to run ANALYZE: %s", err)
	}

	invokeTcs := []struct {
		name           string
		requestBody    io.Reader
		wantStatusCode int
		shouldHaveData bool // Whether we expect data in the response
	}{
		{
			name:           "get cardinality for a specific column",
			requestBody:    bytes.NewBufferString(fmt.Sprintf(`{"schema_name": "%s", "table_name": "%s", "column_name": "email"}`, schemaName, tableName)),
			wantStatusCode: http.StatusOK,
			shouldHaveData: true,
		},
		{
			name:           "get cardinality for all columns",
			requestBody:    bytes.NewBufferString(fmt.Sprintf(`{"schema_name": "%s", "table_name": "%s"}`, schemaName, tableName)),
			wantStatusCode: http.StatusOK,
			shouldHaveData: true,
		},
		{
			name:           "get cardinality with non-existent column",
			requestBody:    bytes.NewBufferString(fmt.Sprintf(`{"schema_name": "%s", "table_name": "%s", "column_name": "non_existent"}`, schemaName, tableName)),
			wantStatusCode: http.StatusOK,
			shouldHaveData: false,
		},
		{
			name:           "get cardinality with non-existent schema",
			requestBody:    bytes.NewBufferString(fmt.Sprintf(`{"schema_name": "non_existent_schema", "table_name": "%s"}`, tableName)),
			wantStatusCode: http.StatusOK,
			shouldHaveData: false,
		},
	}

	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			const api = "http://127.0.0.1:5000/api/tool/get_column_cardinality/invoke"
			resp, respBody := RunRequest(t, http.MethodPost, api, tc.requestBody, nil)
			if resp.StatusCode != tc.wantStatusCode {
				t.Fatalf("wrong status code: got %d, want %d, body: %s", resp.StatusCode, tc.wantStatusCode, string(respBody))
			}
			if tc.wantStatusCode != http.StatusOK {
				return
			}

			var bodyWrapper struct {
				Result json.RawMessage `json:"result"`
			}
			if err := json.Unmarshal(respBody, &bodyWrapper); err != nil {
				t.Fatalf("error decoding response wrapper: %v", err)
			}

			var resultString string
			if err := json.Unmarshal(bodyWrapper.Result, &resultString); err != nil {
				resultString = string(bodyWrapper.Result)
			}

			var got []map[string]any
			if err := json.Unmarshal([]byte(resultString), &got); err != nil {
				t.Fatalf("failed to unmarshal nested result string: %v", err)
			}

			// Verify that we got the expected data presence
			if tc.shouldHaveData {
				if len(got) == 0 {
					t.Logf("warning: expected data but got empty result. This can happen if pg_stats is not populated yet.")
					return
				}

				// Verify column names and cardinality values
				for _, row := range got {
					columnName, ok := row["column_name"].(string)
					if !ok {
						t.Fatalf("column_name is not a string: %v", row["column_name"])
					}

					// Check that estimated_cardinality is present and is a number
					cardinality, ok := row["estimated_cardinality"]
					if !ok {
						t.Fatalf("estimated_cardinality is missing for column %s", columnName)
					}

					// Convert to float64 for numeric checks
					cardinalityFloat, ok := cardinality.(float64)
					if !ok {
						t.Fatalf("estimated_cardinality is not a number: %v", cardinality)
					}

					// Cardinality should be >= 0
					if cardinalityFloat < 0 {
						t.Errorf("cardinality for column %s is negative: %v", columnName, cardinalityFloat)
					}
				}
			} else {
				if len(got) != 0 {
					t.Errorf("expected no data but got: %v", got)
				}
			}
		})
	}
}

func createPostgresExtension(t *testing.T, ctx context.Context, pool *pgxpool.Pool, extensionName string) func() {
	createExtensionCmd := fmt.Sprintf("CREATE EXTENSION IF NOT EXISTS %s", extensionName)
	_, err := pool.Exec(ctx, createExtensionCmd)
	if err != nil {
		t.Fatalf("failed to create extension: %v", err)
	}
	return func() {
		dropExtensionCmd := fmt.Sprintf("DROP EXTENSION IF EXISTS %s", extensionName)
		_, err := pool.Exec(ctx, dropExtensionCmd)
		if err != nil {
			t.Fatalf("failed to drop extension: %v", err)
		}
	}
}

func RunPostgresListQueryStatsTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	// Insert a simple query by running a SELECT statement
	// This will record statistics in pg_stat_statements
	selectStmt := "SELECT 1 as test_query"
	if _, err := pool.Exec(ctx, selectStmt); err != nil {
		t.Logf("warning: unable to execute test query: %s", err)
	}

	dropExtensionFunc := createPostgresExtension(t, ctx, pool, "pg_stat_statements")
	defer dropExtensionFunc()

	type queryStatDetails struct {
		Datname        string `json:"datname"`
		Query          string `json:"query"`
		Calls          any    `json:"calls"`
		TotalExecTime  any    `json:"total_exec_time"`
		MinExecTime    any    `json:"min_exec_time"`
		MaxExecTime    any    `json:"max_exec_time"`
		MeanExecTime   any    `json:"mean_exec_time"`
		Rows           any    `json:"rows"`
		SharedBlksHit  any    `json:"shared_blks_hit"`
		SharedBlksRead any    `json:"shared_blks_read"`
	}

	invokeTcs := []struct {
		name           string
		requestBody    io.Reader
		wantStatusCode int
	}{
		{
			name:           "list query stats with default limit",
			requestBody:    bytes.NewBufferString(`{}`),
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "list query stats with custom limit",
			requestBody:    bytes.NewBufferString(`{"limit": 10}`),
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "list query stats for specific database",
			requestBody:    bytes.NewBufferString(`{"database_name": "postgres"}`),
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "list query stats with non-existent database name",
			requestBody:    bytes.NewBufferString(`{"database_name": "non_existent_db_xyz"}`),
			wantStatusCode: http.StatusOK,
		},
	}

	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			const api = "http://127.0.0.1:5000/api/tool/list_query_stats/invoke"
			resp, respBody := RunRequest(t, http.MethodPost, api, tc.requestBody, nil)
			if resp.StatusCode != tc.wantStatusCode {
				t.Fatalf("wrong status code: got %d, want %d, body: %s", resp.StatusCode, tc.wantStatusCode, string(respBody))
			}
			if tc.wantStatusCode != http.StatusOK {
				return
			}

			var bodyWrapper struct {
				Result json.RawMessage `json:"result"`
			}
			if err := json.Unmarshal(respBody, &bodyWrapper); err != nil {
				t.Fatalf("error decoding response wrapper: %v", err)
			}

			var resultString string
			if err := json.Unmarshal(bodyWrapper.Result, &resultString); err != nil {
				resultString = string(bodyWrapper.Result)
			}

			var got []map[string]any
			if err := json.Unmarshal([]byte(resultString), &got); err != nil {
				t.Fatalf("failed to unmarshal nested result string: %v, resultString: %s", err, resultString)
			}

			// For databases with pg_stat_statements available, verify response structure
			if len(got) > 0 {
				// Verify the response has the expected fields
				requiredFields := []string{"datname", "query", "calls", "total_exec_time", "min_exec_time", "max_exec_time", "mean_exec_time", "rows", "shared_blks_hit", "shared_blks_read"}
				for _, field := range requiredFields {
					if _, ok := got[0][field]; !ok {
						t.Errorf("missing expected field: %s in result: %v", field, got[0])
					}
				}

				// Verify data types
				var stat queryStatDetails
				statData, _ := json.Marshal(got[0])
				if err := json.Unmarshal(statData, &stat); err != nil {
					t.Logf("warning: unable to unmarshal query stat: %v", err)
				}

				// Verify that results are ordered by total_exec_time (descending)
				if len(got) > 1 {
					for i := 0; i < len(got)-1; i++ {
						currentTime, ok1 := got[i]["total_exec_time"].(float64)
						nextTime, ok2 := got[i+1]["total_exec_time"].(float64)
						if ok1 && ok2 && currentTime < nextTime {
							t.Logf("warning: results may not be ordered by total_exec_time descending: %f vs %f", currentTime, nextTime)
						}
					}
				}
			}
		})
	}
}

// RunRequest is a helper function to send HTTP requests and return the response
func RunRequest(t *testing.T, method, url string, body io.Reader, headers map[string]string) (*http.Response, []byte) {
	// Send request
	req, err := http.NewRequest(method, url, body)
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
