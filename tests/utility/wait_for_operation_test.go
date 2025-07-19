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

package utility

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/tests"
)

var (
	httpSourceKind = "http"
	waitToolKind   = "wait-for-operation"
)

type operation struct {
	Name   string `json:"name"`
	Done   bool   `json:"done"`
	Result string `json:"result,omitempty"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type handler struct {
	mu         sync.Mutex
	operations map[string]*operation
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()

	opName := r.URL.Path[len("/operations/"):]
	op, ok := h.operations[opName]
	if !ok {
		http.NotFound(w, r)
		return
	}

	if !op.Done {
		op.Done = true
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(op); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func TestWaitToolEndpoints(t *testing.T) {
	h := &handler{
		operations: map[string]*operation{
			"op1": {Name: "op1", Done: false, Result: "success"},
			"op2": {Name: "op2", Done: false, Error: &struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			}{Code: 1, Message: "failed"}},
		},
	}
	server := httptest.NewServer(h)
	defer server.Close()

	sourceConfig := map[string]any{
		"kind":    httpSourceKind,
		"baseUrl": server.URL,
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	toolsFile := getWaitToolsConfig(sourceConfig)
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
			body:     `{"opId": "op1"}`,
			want:     `{"name":"op1","done":true,"result":"success"}`,
		},
		{
			name:        "failed operation",
			toolName:    "wait-for-op2",
			body:        `{"opId": "op2"}`,
			expectError: true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			api := fmt.Sprintf("http://127.0.0.1:5000/api/tool/%s/invoke", tc.toolName)
			req, err := http.NewRequest(http.MethodPost, api, bytes.NewBufferString(tc.body))
			if err != nil {
				t.Fatalf("unable to create request: %s", err)
			}
			req.Header.Add("Content-type", "application/json")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("unable to send request: %s", err)
			}
			defer resp.Body.Close()

			if tc.expectError {
				if resp.StatusCode == http.StatusOK {
					t.Fatal("expected error but got status 200")
				}
				return
			}

			if resp.StatusCode != http.StatusOK {
				bodyBytes, _ := io.ReadAll(resp.Body)
				t.Fatalf("response status code is not 200, got %d: %s", resp.StatusCode, string(bodyBytes))
			}

			var result struct {
				Result string `json:"result"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			var innerResult []string
			if err := json.Unmarshal([]byte(result.Result), &innerResult); err != nil {
				t.Fatalf("failed to decode inner result: %v", err)
			}
			var finalResult struct {
				Result string `json:"result"`
			}
			if err := json.Unmarshal([]byte(innerResult[0]), &finalResult); err != nil {
				t.Fatalf("failed to decode final result: %v", err)
			}

			var wantResult struct {
				Result string `json:"result"`
			}
			if err := json.Unmarshal([]byte(tc.want), &wantResult); err != nil {
				t.Fatalf("failed to decode want: %v", err)
			}

			if finalResult.Result != wantResult.Result {
				t.Fatalf("unexpected result: got %q, want %q", finalResult.Result, wantResult.Result)
			}
		})
	}
}

func getWaitToolsConfig(sourceConfig map[string]any) map[string]any {
	return map[string]any{
		"sources": map[string]any{
			"my-instance": sourceConfig,
		},
		"tools": map[string]any{
			"wait-for-op1": map[string]any{
				"kind":        waitToolKind,
				"description": "wait for op1",
				"source":      "my-instance",
				"method":      "GET",
				"path":        "/operations/{{.opId}}",
				"pathParams": []tools.Parameter{
					&tools.StringParameter{
						CommonParameter: tools.CommonParameter{Name: "opId", Type: "string", Desc: "The operation ID"},
					},
				},
			},
			"wait-for-op2": map[string]any{
				"kind":        waitToolKind,
				"description": "wait for op2",
				"source":      "my-instance",
				"method":      "GET",
				"path":        "/operations/{{.opId}}",
				"pathParams": []tools.Parameter{
					&tools.StringParameter{
						CommonParameter: tools.CommonParameter{Name: "opId", Type: "string", Desc: "The operation ID"},
					},
				},
			},
		},
	}
}
