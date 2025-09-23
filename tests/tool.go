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
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/genai-toolbox/internal/server/mcp/jsonrpc"
	"github.com/googleapis/genai-toolbox/internal/sources"
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
		nullWant:                 "null",
		supportOptionalNullParam: true,
		supportArrayParam:        true,
		supportClientAuth:        false,
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
			enabled:        true,
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
			wantBody:       configs.myToolId3NameAliceWant,
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "Invoke my-auth-tool with auth token",
			api:            "http://127.0.0.1:5000/api/tool/my-auth-tool/invoke",
			enabled:        true,
			requestHeader:  map[string]string{"my-google-auth_token": idToken},
			requestBody:    bytes.NewBuffer([]byte(`{}`)),
			wantBody:       "[{\"name\":\"Alice\"}]",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "Invoke my-auth-tool with invalid auth token",
			api:            "http://127.0.0.1:5000/api/tool/my-auth-tool/invoke",
			enabled:        true,
			requestHeader:  map[string]string{"my-google-auth_token": "INVALID_TOKEN"},
			requestBody:    bytes.NewBuffer([]byte(`{}`)),
			wantStatusCode: http.StatusUnauthorized,
		},
		{
			name:           "Invoke my-auth-tool without auth token",
			api:            "http://127.0.0.1:5000/api/tool/my-auth-tool/invoke",
			enabled:        true,
			requestHeader:  map[string]string{},
			requestBody:    bytes.NewBuffer([]byte(`{}`)),
			wantStatusCode: http.StatusUnauthorized,
		},
		{
			name:          "Invoke my-auth-required-tool with auth token",
			api:           "http://127.0.0.1:5000/api/tool/my-auth-required-tool/invoke",
			enabled:       true,
			requestHeader: map[string]string{"my-google-auth_token": idToken},
			requestBody:   bytes.NewBuffer([]byte(`{}`)),

			wantBody:       select1Want,
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "Invoke my-auth-required-tool with invalid auth token",
			api:            "http://127.0.0.1:5000/api/tool/my-auth-required-tool/invoke",
			enabled:        true,
			requestHeader:  map[string]string{"my-google-auth_token": "INVALID_TOKEN"},
			requestBody:    bytes.NewBuffer([]byte(`{}`)),
			wantStatusCode: http.StatusUnauthorized,
		},
		{
			name:           "Invoke my-auth-required-tool without auth token",
			api:            "http://127.0.0.1:5000/api/tool/my-auth-tool/invoke",
			enabled:        true,
			requestHeader:  map[string]string{},
			requestBody:    bytes.NewBuffer([]byte(`{}`)),
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
			req, err := http.NewRequest(http.MethodPost, tc.api, tc.requestBody)
			if err != nil {
				t.Fatalf("unable to create request: %s", err)
			}
			req.Header.Add("Content-type", "application/json")
			// Add headers
			for k, v := range tc.requestHeader {
				req.Header.Add(k, v)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("unable to send request: %s", err)
			}
			defer resp.Body.Close()

			// Check status code
			if resp.StatusCode != tc.wantStatusCode {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("StatusCode mismatch: got %d, want %d. Response body: %s", resp.StatusCode, tc.wantStatusCode, string(body))
			}

			// skip response body check
			if tc.wantBody == "" {
				return
			}

			// Check response body
			var body map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&body)
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
			want:          configs.selectId1Want,
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
			// if test case is DDL and source support ddl test cases
			ddlAllow := !tc.ddl || (tc.ddl && configs.supportDdl)
			// if test case is insert statement and source support insert test cases
			insertAllow := !tc.insert || (tc.insert && configs.supportInsert)
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
			enabled:       true,
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
	}{
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
			req, err := http.NewRequest(http.MethodPost, api, tc.requestBody)
			if err != nil {
				t.Fatalf("unable to create request: %v", err)
			}
			req.Header.Add("Content-type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("unable to send request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.wantStatusCode {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("wrong status code: got %d, want %d, body: %s", resp.StatusCode, tc.wantStatusCode, string(body))
			}
			if tc.wantStatusCode != http.StatusOK {
				return
			}

			var bodyWrapper struct {
				Result json.RawMessage `json:"result"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&bodyWrapper); err != nil {
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
			req, err := http.NewRequest(http.MethodPost, api, tc.requestBody)
			if err != nil {
				t.Fatalf("unable to create request: %v", err)
			}
			req.Header.Add("Content-type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("unable to send request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.wantStatusCode {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("wrong status code: got %d, want %d, body: %s", resp.StatusCode, tc.wantStatusCode, string(body))
			}
			if tc.wantStatusCode != http.StatusOK {
				return
			}

			var bodyWrapper struct {
				Result json.RawMessage `json:"result"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&bodyWrapper); err != nil {
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
			req, err := http.NewRequest(http.MethodPost, api, tc.requestBody)
			if err != nil {
				t.Fatalf("unable to create request: %v", err)
			}
			req.Header.Add("Content-type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("unable to send request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.wantStatusCode {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("wrong status code: got %d, want %d, body: %s", resp.StatusCode, tc.wantStatusCode, string(body))
			}
			if tc.wantStatusCode != http.StatusOK {
				return
			}

			var bodyWrapper struct {
				Result json.RawMessage `json:"result"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&bodyWrapper); err != nil {
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
			req, err := http.NewRequest(http.MethodPost, api, tc.requestBody)
			if err != nil {
				t.Fatalf("unable to create request: %v", err)
			}
			req.Header.Add("Content-type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("unable to send request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.wantStatusCode {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("wrong status code: got %d, want %d, body: %s", resp.StatusCode, tc.wantStatusCode, string(body))
			}
			if tc.wantStatusCode != http.StatusOK {
				return
			}

			var bodyWrapper struct {
				Result json.RawMessage `json:"result"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&bodyWrapper); err != nil {
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
