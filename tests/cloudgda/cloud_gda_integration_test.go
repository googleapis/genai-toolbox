// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cloudgda_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/googleapis/genai-toolbox/internal/server/mcp/jsonrpc"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools/cloudgda"
	"github.com/googleapis/genai-toolbox/tests"
)

var (
	cloudGdaToolKind = "cloud-gemini-data-analytics-query"
)

type cloudGdaTransport struct {
	transport http.RoundTripper
	url       *url.URL
}

func (t *cloudGdaTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if strings.HasPrefix(req.URL.String(), "https://geminidataanalytics.googleapis.com") {
		req.URL.Scheme = t.url.Scheme
		req.URL.Host = t.url.Host
	}
	return t.transport.RoundTrip(req)
}

type masterHandler struct {
	t *testing.T
}

func (h *masterHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.Contains(r.UserAgent(), "genai-toolbox/") {
		h.t.Errorf("User-Agent header not found")
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Verify URL structure
	// Expected: /v1beta/projects/{project}/locations/global:queryData
	if !strings.Contains(r.URL.Path, ":queryData") || !strings.Contains(r.URL.Path, "locations/global") {
		h.t.Errorf("unexpected URL path: %s", r.URL.Path)
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	var reqBody cloudgda.QueryDataRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		h.t.Fatalf("failed to decode request body: %v", err)
	}

	if reqBody.Prompt == "" {
		http.Error(w, "missing prompt", http.StatusBadRequest)
		return
	}

	response := map[string]any{
		"queryResult":           "SELECT * FROM table;",
		"naturalLanguageAnswer": "Here is the answer.",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func TestCloudGdaToolEndpoints(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	handler := &masterHandler{t: t}
	server := httptest.NewServer(handler)
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("failed to parse server URL: %v", err)
	}

	originalTransport := http.DefaultClient.Transport
	if originalTransport == nil {
		originalTransport = http.DefaultTransport
	}
	http.DefaultClient.Transport = &cloudGdaTransport{
		transport: originalTransport,
		url:       serverURL,
	}
	t.Cleanup(func() {
		http.DefaultClient.Transport = originalTransport
	})

	var args []string
	toolsFile := getCloudGdaToolsConfig()
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

	toolName := "cloud-gda-query"

	// 1. RunToolGetTestByName
	expectedManifest := map[string]any{
		toolName: map[string]any{
			"description": "Test GDA Tool",
			"parameters": []any{
				map[string]any{
					"name":        "prompt",
					"type":        "string",
					"description": "The natural language question to ask.",
					"required":    true,
					"authSources": []any{},
				},
			},
			"authRequired": []any{},
		},
	}
	tests.RunToolGetTestByName(t, toolName, expectedManifest)

	// 2. RunToolInvokeParametersTest
	params := []byte(`{"prompt": "test question"}`)
	tests.RunToolInvokeParametersTest(t, toolName, params, "\"queryResult\":\"SELECT * FROM table;\"")

	// 3. Manual MCP Tool Call Test
	// Initialize MCP session
	sessionId := tests.RunInitialize(t, "2025-12-12")

	// Construct MCP Request
	mcpReq := jsonrpc.JSONRPCRequest{
		Jsonrpc: "2.0",
		Id:      "test-mcp-call",
		Request: jsonrpc.Request{
			Method: "tools/call",
		},
		Params: map[string]any{
			"name": toolName,
			"arguments": map[string]any{
				"prompt": "test question",
			},
		},
	}
	reqBytes, _ := json.Marshal(mcpReq)

	headers := map[string]string{}
	if sessionId != "" {
		headers["Mcp-Session-Id"] = sessionId
	}

	// Send Request
	resp, respBody := tests.RunRequest(t, http.MethodPost, "http://127.0.0.1:5000/mcp", bytes.NewBuffer(reqBytes), headers)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("MCP request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	// Check Response
	respStr := string(respBody)
	if !strings.Contains(respStr, "SELECT * FROM table;") {
		t.Errorf("MCP response does not contain expected query result: %s", respStr)
	}
}

func getCloudGdaToolsConfig() map[string]any {
	return map[string]any{
		"sources": map[string]any{
			"my-gda-source": map[string]any{
				"kind":      "cloud-gemini-data-analytics",
				"projectId": "test-project",
			},
		},
		"tools": map[string]any{
			"cloud-gda-query": map[string]any{
				"kind":        cloudGdaToolKind,
				"source":      "my-gda-source",
				"description": "Test GDA Tool",
				"location":    "us-central1",
				"context": map[string]any{
					"datasourceReferences": map[string]any{
						"spannerReference": map[string]any{
							"databaseReference": map[string]any{
								"projectId":  "test-project",
								"instanceId": "test-instance",
								"databaseId": "test-db",
								"engine":     "GOOGLE_SQL",
							},
						},
					},
				},
			},
		},
	}
}
