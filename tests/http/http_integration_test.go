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

package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/googleapis/genai-toolbox/tests"
)

var (
	HTTP_SOURCE_KIND = "http"
	HTTP_TOOL_KIND   = "http"
)

func getHTTPSourceConfig(t *testing.T) map[string]any {
	idToken, err := tests.GetGoogleIdToken(tests.ClientId)
	if err != nil {
		t.Fatalf("error getting ID token: %s", err)
	}
	idToken = "Bearer " + idToken
	return map[string]any{
		"kind":    HTTP_SOURCE_KIND,
		"headers": map[string]string{"Authorization": idToken},
	}
}

// handler function for the test server
func multiTool(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	path = strings.TrimPrefix(path, "/") // Remove leading slash

	switch path {
	case "tool0":
		handleTool0(w, r)
	case "tool1":
		handleTool1(w, r)
	case "tool2":
		handleTool2(w, r)
	case "tool3":
		handleTool3(w, r)
	default:
		http.NotFound(w, r) // Return 404 for unknown paths
	}
}

// handler function for the test server
func handleTool0(w http.ResponseWriter, r *http.Request) {
	// expect POST method
	if r.Method != http.MethodPost {
		errorMessage := fmt.Sprintf("expected POST method but got: %s", string(r.Method))
		http.Error(w, errorMessage, http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	response := []string{
		"Hello",
		"World",
	}
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
		return
	}
}

// handler function for the test server
func handleTool1(w http.ResponseWriter, r *http.Request) {
	// expect GET method
	if r.Method != http.MethodGet {
		errorMessage := fmt.Sprintf("expected GET method but got: %s", string(r.Method))
		http.Error(w, errorMessage, http.StatusBadRequest)
		return
	}
	// Parse request body
	var requestBody map[string]interface{}
	bodyBytes, readErr := io.ReadAll(r.Body)
	if readErr != nil {
		http.Error(w, "Bad Request: Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	err := json.Unmarshal(bodyBytes, &requestBody)
	if err != nil {
		errorMessage := fmt.Sprintf("Bad Request: Error unmarshalling request body: %s, Raw body: %s", err, string(bodyBytes))
		http.Error(w, errorMessage, http.StatusBadRequest)
		return
	}

	// Extract name
	name, ok := requestBody["name"].(string)
	if !ok || name == "" {
		http.Error(w, "Bad Request: Missing or invalid name", http.StatusBadRequest)
		return
	}

	if name == "Alice" {
		response := `[{"id":1,"name":"Alice"},{"id":3,"name":"Sid"}]`
		_, err := w.Write([]byte(response))
		if err != nil {
			http.Error(w, "Failed to write response", http.StatusInternalServerError)
		}
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// handler function for the test server
func handleTool2(w http.ResponseWriter, r *http.Request) {
	// expect GET method
	if r.Method != http.MethodGet {
		errorMessage := fmt.Sprintf("expected GET method but got: %s", string(r.Method))
		http.Error(w, errorMessage, http.StatusBadRequest)
		return
	}
	email := r.URL.Query().Get("email")
	if email != "" {
		response := `{"name":"Alice"}`
		_, err := w.Write([]byte(response))
		if err != nil {
			http.Error(w, "Failed to write response", http.StatusInternalServerError)
		}
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// handler function for the test server
func handleTool3(w http.ResponseWriter, r *http.Request) {
	// expect GET method
	if r.Method != http.MethodGet {
		errorMessage := fmt.Sprintf("expected GET method but got: %s", string(r.Method))
		http.Error(w, errorMessage, http.StatusBadRequest)
		return
	}

	// Check request headers
	expectedHeaders := map[string]string{
		"Content-Type":    "application/json",
		"X-Custom-Header": "example",
		"X-Other-Header":  "test",
	}
	for header, expectedValue := range expectedHeaders {
		if r.Header.Get(header) != expectedValue {
			errorMessage := fmt.Sprintf("Bad Request: Missing or incorrect header: %s", header)
			http.Error(w, errorMessage, http.StatusBadRequest)
			return
		}
	}

	// Check query parameters
	expectedQueryParams := map[string][]string{
		"id":      []string{"2", "1", "3"},
		"country": []string{"US"},
	}
	query := r.URL.Query()
	for param, expectedValueSlice := range expectedQueryParams {
		values, ok := query[param]
		if ok {
			if !reflect.DeepEqual(expectedValueSlice, values) {
				errorMessage := fmt.Sprintf("Bad Request: Incorrect query parameter: %s, actual: %s", param, query[param])
				http.Error(w, errorMessage, http.StatusBadRequest)
				return
			}
		} else {
			errorMessage := fmt.Sprintf("Bad Request: Missing query parameter: %s, actual: %s", param, query[param])
			http.Error(w, errorMessage, http.StatusBadRequest)
			return
		}
	}

	// Parse request body
	var requestBody map[string]interface{}
	bodyBytes, readErr := io.ReadAll(r.Body)
	if readErr != nil {
		http.Error(w, "Bad Request: Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	err := json.Unmarshal(bodyBytes, &requestBody)
	if err != nil {
		errorMessage := fmt.Sprintf("Bad Request: Error unmarshalling request body: %s, Raw body: %s", err, string(bodyBytes))
		http.Error(w, errorMessage, http.StatusBadRequest)
		return
	}

	// Check request body
	expectedBody := map[string]interface{}{
		"place":   "zoo",
		"animals": []any{"rabbit", "ostrich", "whale"},
	}

	if !reflect.DeepEqual(requestBody, expectedBody) {
		errorMessage := fmt.Sprintf("Bad Request: Incorrect request body. Expected: %v, Got: %v", expectedBody, requestBody)
		http.Error(w, errorMessage, http.StatusBadRequest)
		return
	}

	// Return a JSON array as the response
	response := []any{
		"Hello", "World",
	}
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
		return
	}
}

func TestHttpToolEndpoints(t *testing.T) {
	// start a test server
	server := httptest.NewServer(http.HandlerFunc(multiTool))
	defer server.Close()

	sourceConfig := getHTTPSourceConfig(t)
	sourceConfig["baseUrl"] = server.URL
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	toolsFile := tests.GetHTTPToolsConfig(sourceConfig, HTTP_TOOL_KIND)
	cmd, cleanup, err := tests.StartCmd(ctx, toolsFile, args...)
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

	select1Want := `["Hello","World"]`
	invokeParamWant, _ := tests.GetNonSpannerInvokeParamWant()
	tests.RunToolGetTest(t)
	tests.RunToolInvokeTest(t, select1Want, invokeParamWant)
	runAdvancedHTTPInvokeTest(t)
}

// runToolInvoke runs the tool invoke endpoint
func runAdvancedHTTPInvokeTest(t *testing.T) {
	// Test HTTP tool invoke endpoint
	invokeTcs := []struct {
		name          string
		api           string
		requestHeader map[string]string
		requestBody   io.Reader
		want          string
		isErr         bool
	}{
		{
			name:          "invoke my-advanced-tool",
			api:           "http://127.0.0.1:5000/api/tool/my-advanced-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"animalArray": ["rabbit", "ostrich", "whale"], "id": 3, "country": "US", "X-Other-Header": "test"}`)),
			want:          `["Hello","World"]`,
			isErr:         false,
		},
		{
			name:          "invoke my-advanced-tool with wrong params",
			api:           "http://127.0.0.1:5000/api/tool/my-advanced-tool/invoke",
			requestHeader: map[string]string{},
			requestBody:   bytes.NewBuffer([]byte(`{"animalArray": ["rabbit", "ostrich", "whale"], "id": 4, "country": "US", "X-Other-Header": "test"}`)),
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
				if tc.isErr == true {
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
