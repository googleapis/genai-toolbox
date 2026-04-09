// Copyright 2026 Google LLC
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

package alloydb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/googleapis/mcp-toolbox/internal/testutils"
	"github.com/googleapis/mcp-toolbox/tests"
)

const waitToolTypeMCP = "alloydb-wait-for-operation"

type operationMCP struct {
	Name     string `json:"name"`
	Done     bool   `json:"done"`
	Response string `json:"response,omitempty"`
	Error    *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type handlerMCP struct {
	mu         sync.Mutex
	operations map[string]*operationMCP
	t          *testing.T
}

func (h *handlerMCP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.Contains(r.UserAgent(), "genai-toolbox/") {
		h.t.Errorf("User-Agent header not found")
	}

	path := r.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		http.NotFound(w, r)
		return
	}
	opName := parts[len(parts)-1]

	h.mu.Lock()
	defer h.mu.Unlock()

	op, ok := h.operations[opName]
	if ok {
		if !op.Done {
			op.Done = true
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(op); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	} else {
		http.NotFound(w, r)
	}
}

type waitForOperationTransportMCP struct {
	transport http.RoundTripper
	url       *url.URL
}

func (t *waitForOperationTransportMCP) RoundTrip(req *http.Request) (*http.Response, error) {
	if strings.Contains(req.URL.String(), "alloydb.googleapis.com") {
		req.URL.Scheme = t.url.Scheme
		req.URL.Host = t.url.Host
	}
	return t.transport.RoundTrip(req)
}

func TestWaitToolEndpointsMCP(t *testing.T) {
	h := &handlerMCP{
		operations: map[string]*operationMCP{
			"op1": {Name: "op1", Done: false, Response: "success"},
			"op2": {Name: "op2", Done: false, Error: &struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			}{Code: 1, Message: "failed"}},
		},
		t: t,
	}
	server := httptest.NewServer(h)
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("failed to parse server URL: %v", err)
	}

	originalTransport := http.DefaultClient.Transport
	if originalTransport == nil {
		originalTransport = http.DefaultTransport
	}
	http.DefaultClient.Transport = &waitForOperationTransportMCP{
		transport: originalTransport,
		url:       serverURL,
	}
	t.Cleanup(func() {
		http.DefaultClient.Transport = originalTransport
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	args := []string{"--enable-api"}

	toolsFile := getWaitToolsConfigMCP()
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

	tcs := []struct {
		name        string
		toolName    string
		body        string
		want        string
		expectError bool
	}{
		{
			name:     "successful operation",
			toolName: "wait-for-op1",
			body:     `{"project": "p1", "location": "l1", "operation": "op1"}`,
			want:     `{"done":true,"name":"op1","response":"success"}`,
		},
		{
			name:        "failed operation",
			toolName:    "wait-for-op2",
			body:        `{"project": "p1", "location": "l1", "operation": "op2"}`,
			want:        `{"error":"error processing request: operation finished with error: {\"code\":1,\"message\":\"failed\"}"}`,
			expectError: true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			var args map[string]any
			if err := json.Unmarshal([]byte(tc.body), &args); err != nil {
				t.Fatalf("failed to unmarshal body: %v", err)
			}

			statusCode, mcpResp, err := tests.InvokeMCPTool(t, tc.toolName, args, nil)
			if err != nil {
				t.Fatalf("native error executing %s: %s", tc.toolName, err)
			}

			if statusCode != http.StatusOK {
				t.Fatalf("expected status 200, got %d", statusCode)
			}

			if len(mcpResp.Result.Content) == 0 {
				t.Fatalf("expected at least one content item, got none")
			}
			got := mcpResp.Result.Content[0].Text

			if tc.expectError {
				if !mcpResp.Result.IsError {
					t.Fatalf("expected error result, got success")
				}
				var wantMap map[string]string
				if err := json.Unmarshal([]byte(tc.want), &wantMap); err != nil {
					t.Fatalf("failed to unmarshal want: %v", err)
				}
				assertContains(t, got, wantMap["error"])
			} else {
				if mcpResp.Result.IsError {
					t.Fatalf("expected success result, got error: %v", mcpResp.Result)
				}
				// Clean up both strings to ignore whitespace differences
				got = strings.ReplaceAll(strings.ReplaceAll(got, " ", ""), "\n", "")
				want := strings.ReplaceAll(strings.ReplaceAll(tc.want, " ", ""), "\n", "")
				assertContains(t, got, want)
			}
		})
	}
}

func getWaitToolsConfigMCP() map[string]any {
	return map[string]any{
		"sources": map[string]any{
			"my-alloydb-source": map[string]any{
				"type": "alloydb-admin",
			},
		},
		"tools": map[string]any{
			"wait-for-op1": map[string]any{
				"type":        waitToolTypeMCP,
				"source":      "my-alloydb-source",
				"description": "wait for op1",
			},
			"wait-for-op2": map[string]any{
				"type":        waitToolTypeMCP,
				"source":      "my-alloydb-source",
				"description": "wait for op2",
			},
		},
	}
}

func assertContains(t *testing.T, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("unexpected result: got %q, want substring %q", got, want)
	}
}
