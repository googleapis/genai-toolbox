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
	"reflect"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"
)

var (
	waitToolKind = "alloydb-wait-for-operation"
)

type operation struct {
	Name     string `json:"name"`
	Done     bool   `json:"done"`
	Response any    `json:"response,omitempty"`
	Error    *struct {
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

	// The format is projects/{project}/locations/{location}/operations/{operation_id}
	// We only care about the operation_id for the test
	parts := regexp.MustCompile("/").Split(r.URL.Path, -1)
	opName := parts[len(parts)-1]

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
			"op1": {Name: "op1", Done: false, Response: "success"},
			"op2": {Name: "op2", Done: false, Error: &struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			}{Code: 1, Message: "failed"}},
		},
	}
	server := httptest.NewServer(h)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	toolsFile := getWaitToolsConfig(server.URL)
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
		name          string
		toolName      string
		body          string
		want          string
		expectError   bool
		wantSubstring bool
	}{
		{
			name:     "successful operation",
			toolName: "wait-for-op1",
			body:     `{"project": "p1", "location": "l1", "operation_id": "op1"}`,
			want:     `{"name":"op1","done":true,"response":"success"}`,
		},
		{
			name:        "failed operation",
			toolName:    "wait-for-op2",
			body:        `{"project": "p1", "location": "l1", "operation_id": "op2"}`,
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

			if tc.wantSubstring {
				if !bytes.Contains([]byte(result.Result), []byte(tc.want)) {
					t.Fatalf("unexpected result: got %q, want substring %q", result.Result, tc.want)
				}
				return
			}

			// The result is a JSON-encoded string, so we need to unmarshal it twice.
			var unmarshaledResult string
			if err := json.Unmarshal([]byte(result.Result), &unmarshaledResult); err != nil {
				t.Fatalf("failed to unmarshal result string: %v", err)
			}

			var got, want map[string]any
			if err := json.Unmarshal([]byte(unmarshaledResult), &got); err != nil {
				t.Fatalf("failed to unmarshal result: %v", err)
			}
			if err := json.Unmarshal([]byte(tc.want), &want); err != nil {
				t.Fatalf("failed to unmarshal want: %v", err)
			}

			if !reflect.DeepEqual(got, want) {
				t.Fatalf("unexpected result: got %+v, want %+v", got, want)
			}
		})
	}
}

func getWaitToolsConfig(baseURL string) map[string]any {
	return map[string]any{
		"tools": map[string]any{
			"wait-for-op1": map[string]any{
				"kind":         waitToolKind,
				"description":  "wait for op1",
				"baseURL":      baseURL,
				"authRequired": []string{},
			},
			"wait-for-op2": map[string]any{
				"kind":         waitToolKind,
				"description":  "wait for op2",
				"baseURL":      baseURL,
				"authRequired": []string{},
			},
		},
	}
}
