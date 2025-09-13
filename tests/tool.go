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
			wantBody:       "[{\"name\":\"Alice\"}]",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "Invoke my-auth-tool with invalid auth token",
			api:            "http://127.0.0.1:5000/api/tool/my-auth-tool/invoke",
			enabled:        configs.supportSelect1Auth,
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

			httpResponse, respBody := runRequest(t, http.MethodPost, tc.api, bytes.NewBuffer(reqMarshal), headers)

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

func runRequest(t *testing.T, method, url string, body io.Reader, headers map[string]string) (*http.Response, []byte) {
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
