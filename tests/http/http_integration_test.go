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

	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
	"github.com/googleapis/genai-toolbox/tests"
)

var (
	HttpSourceType = "http"
	HttpToolType   = "http"
)

func getHTTPSourceConfig(t *testing.T) map[string]any {
	idToken, err := tests.GetGoogleIdToken(tests.ClientId)
	if err != nil {
		t.Fatalf("error getting ID token: %s", err)
	}
	idToken = "Bearer " + idToken

	return map[string]any{
		"type":    HttpSourceType,
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
	case "tool1id":
		handleTool1Id(w, r)
	case "tool1name":
		handleTool1Name(w, r)
	case "tool2":
		handleTool2(w, r)
	case "tool3":
		handleTool3(w, r)
	case "toolQueryTest":
		handleQueryTest(w, r)
	default:
		http.NotFound(w, r) // Return 404 for unknown paths
	}
}

// handleQueryTest simply returns the raw query string it received so the test
// can verify it's formatted correctly.
func handleQueryTest(w http.ResponseWriter, r *http.Request) {
	// expect GET method
	if r.Method != http.MethodGet {
		errorMessage := fmt.Sprintf("expected GET method but got: %s", string(r.Method))
		http.Error(w, errorMessage, http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)

	err := enc.Encode(r.URL.RawQuery)
	if err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
		return
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
	response := "hello world"
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
func handleTool1Id(w http.ResponseWriter, r *http.Request) {
	// expect GET method
	if r.Method != http.MethodGet {
		errorMessage := fmt.Sprintf("expected GET method but got: %s", string(r.Method))
		http.Error(w, errorMessage, http.StatusBadRequest)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "4" {
		response := `[{"id":4,"name":null}]`
		_, err := w.Write([]byte(response))
		if err != nil {
			http.Error(w, "Failed to write response", http.StatusInternalServerError)
		}
		return
	}
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// handler function for the test server
func handleTool1Name(w http.ResponseWriter, r *http.Request) {
	// expect GET method
	if r.Method != http.MethodGet {
		errorMessage := fmt.Sprintf("expected GET method but got: %s", string(r.Method))
		http.Error(w, errorMessage, http.StatusBadRequest)
		return
	}

	if !r.URL.Query().Has("name") {
		response := "null"
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
		response := `[{"name":"Alice"}]`
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

	response := "hello world"
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

	toolsFile := getHTTPToolsConfig(sourceConfig, HttpToolType)
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

	// Native tools/list invocation test
	statusCodeList, toolsResp, errList := tests.GetMCPToolsList(t, nil)
	if errList != nil {
		t.Fatalf("native error executing tools/list: %s", errList)
	}
	if statusCodeList != http.StatusOK {
		t.Fatalf("expected status 200 for tools/list, got %d", statusCodeList)
	}

	// Ensure tools/list returned valid tools mapping
	resultMap, ok := toolsResp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("tools/list result is not a map: %v", toolsResp.Result)
	}

	toolsList, ok := resultMap["tools"].([]interface{})
	if !ok || len(toolsList) == 0 {
		t.Fatalf("tools/list did not contain tools array: %v", resultMap)
	}

	// Verify "my-simple-tool" is explicitly registered natively
	foundTool := false
	for _, toolItem := range toolsList {
		if toolMap, ok := toolItem.(map[string]interface{}); ok {
			if toolMap["name"] == "my-simple-tool" {
				foundTool = true
				break
			}
		}
	}
	if !foundTool {
		t.Fatalf("my-simple-tool was not found in the tools/list native registry")
	}
	statusCode, mcpResp, err := tests.InvokeMCPTool(t, "my-simple-tool", map[string]any{}, nil)
	if err != nil {
		t.Fatalf("native error executing my-simple-tool: %s", err)
	}
	if statusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", statusCode)
	}
	if mcpResp.Result.IsError {
		t.Fatalf("my-simple-tool returned error result: %v", mcpResp.Result)
	}
	if len(mcpResp.Result.Content) == 0 {
		t.Fatalf("my-simple-tool returned empty content field")
	}
	if got := mcpResp.Result.Content[0].Text; got != `"hello world"` {
		t.Fatalf(`expected '"hello world"', got %q`, got)
	}

	runAdvancedHTTPInvokeTest(t)
	runQueryParamInvokeTest(t)
}

func runQueryParamInvokeTest(t *testing.T) {
	invokeTcs := []struct {
		name       string
		toolName   string
		arguments  map[string]any
		want       string
		wantErrMsg string
	}{
		{
			name:      "invoke query-param-tool (optional omitted)",
			toolName:  "my-query-param-tool",
			arguments: map[string]any{"reqId": "test1"},
			want:      `"reqId=test1"`,
		},
		{
			name:      "invoke query-param-tool (some optional nil)",
			toolName:  "my-query-param-tool",
			arguments: map[string]any{"reqId": "test2", "page": "5", "filter": nil},
			want:      `"page=5\u0026reqId=test2"`,
		},
		{
			name:      "invoke query-param-tool (some optional absent)",
			toolName:  "my-query-param-tool",
			arguments: map[string]any{"reqId": "test2", "page": "5"},
			want:      `"page=5\u0026reqId=test2"`,
		},
		{
			name:       "invoke query-param-tool (required param nil)",
			toolName:   "my-query-param-tool",
			arguments:  map[string]any{"reqId": nil, "page": "1"},
			wantErrMsg: `parameter "reqId" is required`,
		},
	}
	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			statusCode, mcpResp, err := tests.InvokeMCPTool(t, tc.toolName, tc.arguments, nil)
			if err != nil {
				t.Fatalf("error calling MCP tool: %v", err)
			}

			if statusCode != http.StatusOK {
				t.Fatalf("response status code is not 200, got %d", statusCode)
			}

			if tc.wantErrMsg != "" {
				if !mcpResp.Result.IsError {
					t.Fatalf("expected application error containing %q, but got success result: %v", tc.wantErrMsg, mcpResp.Result)
				}
				var errText string
				for _, content := range mcpResp.Result.Content {
					if content.Type == "text" {
						errText += content.Text
					}
				}
				if !strings.Contains(errText, tc.wantErrMsg) {
					t.Fatalf("expected error text containing %q, got %q", tc.wantErrMsg, errText)
				}
				return
			}

			if mcpResp.Result.IsError {
				t.Fatalf("unexpected application error: %v", mcpResp.Result)
			}
			if len(mcpResp.Result.Content) == 0 {
				t.Fatalf("expected result content but got none")
			}

			// Extract the raw text and assert
			got := mcpResp.Result.Content[0].Text

			if got != tc.want {
				t.Fatalf("unexpected value: got %q, want %q", got, tc.want)
			}
		})
	}
}

func runAdvancedHTTPInvokeTest(t *testing.T) {
	// Test HTTP tool invoke endpoint
	invokeTcs := []struct {
		name          string
		toolName      string
		arguments     map[string]any
		requestHeader map[string]string
		want          string
		wantErrMsg    string
	}{
		{
			name:          "invoke my-advanced-tool",
			toolName:      "my-advanced-tool",
			requestHeader: map[string]string{},
			arguments:     map[string]any{"animalArray": []string{"rabbit", "ostrich", "whale"}, "id": 3, "path": "tool3", "country": "US", "X-Other-Header": "test"},
			want:          `"hello world"`,
			wantErrMsg:    "",
		},
		{
			name:          "invoke my-advanced-tool with wrong params",
			toolName:      "my-advanced-tool",
			requestHeader: map[string]string{},
			arguments:     map[string]any{"animalArray": []string{"rabbit", "ostrich", "whale"}, "id": 4, "path": "tool3", "country": "US", "X-Other-Header": "test"},
			want:          "",
			wantErrMsg:    "error processing request: unexpected status code: 400, response body: Bad Request: Incorrect query parameter: id, actual: [2 1 4]",
		},
	}

	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			statusCode, mcpResp, err := tests.InvokeMCPTool(t, tc.toolName, tc.arguments, tc.requestHeader)
			if err != nil {
				t.Fatalf("error calling MCP tool: %v", err)
			}

			if statusCode != http.StatusOK {
				t.Fatalf("expected status 200 from toolbox, got %d", statusCode)
			}

			if tc.wantErrMsg != "" {
				if !mcpResp.Result.IsError {
					t.Fatalf("expected application error containing %q, but got success result: %v", tc.wantErrMsg, mcpResp.Result)
				}
				var errText string
				for _, content := range mcpResp.Result.Content {
					if content.Type == "text" {
						errText += content.Text
					}
				}
				if !strings.Contains(errText, tc.wantErrMsg) {
					t.Fatalf("unexpected error message: got %q, want it to contain %q", errText, tc.wantErrMsg)
				}
				return
			}

			if mcpResp.Result.IsError {
				t.Fatalf("unexpected application error: %v", mcpResp.Result)
			}
			if len(mcpResp.Result.Content) == 0 {
				t.Fatalf("expected result content but got none")
			}

			got := mcpResp.Result.Content[0].Text
			if got != tc.want {
				t.Fatalf("unexpected result: got %q, want %q", got, tc.want)
			}
		})
	}
}

// getHTTPToolsConfig returns a mock HTTP tool's config file
func getHTTPToolsConfig(sourceConfig map[string]any, toolType string) map[string]any {
	// Write config into a file and pass it to command
	otherSourceConfig := make(map[string]any)
	for k, v := range sourceConfig {
		otherSourceConfig[k] = v
	}
	otherSourceConfig["headers"] = map[string]string{"X-Custom-Header": "unexpected", "Content-Type": "application/json"}
	otherSourceConfig["queryParams"] = map[string]any{"id": 1, "name": "Sid"}

	toolsFile := map[string]any{
		"sources": map[string]any{
			"my-instance":    sourceConfig,
			"other-instance": otherSourceConfig,
		},
		"authServices": map[string]any{
			"my-google-auth": map[string]any{
				"type":     "google",
				"clientId": tests.ClientId,
			},
		},
		"tools": map[string]any{
			"my-simple-tool": map[string]any{
				"type":        toolType,
				"path":        "/tool0",
				"method":      "POST",
				"source":      "my-instance",
				"requestBody": "{}",
				"description": "Simple tool to test end to end functionality.",
			},
			"my-tool": map[string]any{
				"type":        toolType,
				"source":      "my-instance",
				"method":      "GET",
				"path":        "/tool1",
				"description": "some description",
				"queryParams": []parameters.Parameter{
					parameters.NewIntParameter("id", "user ID")},
				"bodyParams": []parameters.Parameter{parameters.NewStringParameter("name", "user name")},
				"requestBody": `{
"age": 36,
"name": "{{.name}}"
}
`,
				"headers": map[string]string{"Content-Type": "application/json"},
			},
			"my-tool-by-id": map[string]any{
				"type":        toolType,
				"source":      "my-instance",
				"method":      "GET",
				"path":        "/tool1id",
				"description": "some description",
				"queryParams": []parameters.Parameter{
					parameters.NewIntParameter("id", "user ID")},
				"headers": map[string]string{"Content-Type": "application/json"},
			},
			"my-tool-by-name": map[string]any{
				"type":        toolType,
				"source":      "my-instance",
				"method":      "GET",
				"path":        "/tool1name",
				"description": "some description",
				"queryParams": []parameters.Parameter{
					parameters.NewStringParameterWithRequired("name", "user name", false)},
				"headers": map[string]string{"Content-Type": "application/json"},
			},
			"my-query-param-tool": map[string]any{
				"type":        toolType,
				"source":      "my-instance",
				"method":      "GET",
				"path":        "/toolQueryTest",
				"description": "Tool to test optional query parameters.",
				"queryParams": []parameters.Parameter{
					parameters.NewStringParameterWithRequired("reqId", "required ID", true),
					parameters.NewStringParameterWithRequired("page", "optional page number", false),
					parameters.NewStringParameterWithRequired("filter", "optional filter string", false),
				},
			},
			"my-auth-tool": map[string]any{
				"type":        toolType,
				"source":      "my-instance",
				"method":      "GET",
				"path":        "/tool2",
				"description": "some description",
				"requestBody": "{}",
				"queryParams": []parameters.Parameter{
					parameters.NewStringParameterWithAuth("email", "some description",
						[]parameters.ParamAuthService{{Name: "my-google-auth", Field: "email"}}),
				},
			},
			"my-auth-required-tool": map[string]any{
				"type":         toolType,
				"source":       "my-instance",
				"method":       "POST",
				"path":         "/tool0",
				"description":  "some description",
				"requestBody":  "{}",
				"authRequired": []string{"my-google-auth"},
			},
			"my-advanced-tool": map[string]any{
				"type":        toolType,
				"source":      "other-instance",
				"method":      "get",
				"path":        "/{{.path}}?id=2",
				"description": "some description",
				"headers": map[string]string{
					"X-Custom-Header": "example",
				},
				"pathParams": []parameters.Parameter{
					&parameters.StringParameter{
						CommonParameter: parameters.CommonParameter{Name: "path", Type: "string", Desc: "path param"},
					},
				},
				"queryParams": []parameters.Parameter{
					parameters.NewIntParameter("id", "user ID"), parameters.NewStringParameter("country", "country"),
				},
				"requestBody": `{
					"place": "zoo",
					"animals": {{json .animalArray }}
					}
					`,
				"bodyParams":   []parameters.Parameter{parameters.NewArrayParameter("animalArray", "animals in the zoo", parameters.NewStringParameter("animals", "desc"))},
				"headerParams": []parameters.Parameter{parameters.NewStringParameter("X-Other-Header", "custom header")},
			},
		},
	}
	return toolsFile
}
