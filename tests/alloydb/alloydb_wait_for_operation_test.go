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

package alloydb

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"

	_ "github.com/googleapis/genai-toolbox/internal/tools/alloydb/alloydbwaitforoperation"
)

var (
	waitToolType = "alloydb-wait-for-operation"
)

type waitForOperationTransport struct {
	transport http.RoundTripper
	url       *url.URL
}

func (t *waitForOperationTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if strings.HasPrefix(req.URL.String(), "https://alloydb.googleapis.com") {
		req.URL.Scheme = t.url.Scheme
		req.URL.Host = t.url.Host
	}
	return t.transport.RoundTrip(req)
}

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
	t          *testing.T
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !strings.Contains(r.UserAgent(), "genai-toolbox/") {
		h.t.Errorf("User-Agent header not found")
	}

	// The format is projects/{project}/locations/{location}/operations/{operation}
	// The tool will call something like /v1/projects/p1/locations/l1/operations/op1
	if match, _ := regexp.MatchString("/v1/projects/.*/locations/.*/operations/.*", r.URL.Path); match {
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
	} else {
		http.NotFound(w, r)
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
	http.DefaultClient.Transport = &waitForOperationTransport{
		transport: originalTransport,
		url:       serverURL,
	}
	t.Cleanup(func() {
		http.DefaultClient.Transport = originalTransport
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	toolsFile := getWaitToolsConfig()
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
			body:     `{"project": "p1", "location": "l1", "operation": "op1"}`,
			want:     `{"done":true,"name":"op1","response":"success"}`,
		},
		{
			name:     "failed operation",
			toolName: "wait-for-op2",
			body:     `{"project": "p1", "location": "l1", "operation": "op2"}`,
			want:     `{"error":"error processing request: operation finished with error: {\"code\":1,\"message\":\"failed\"}"}`,
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

			if resp.StatusCode != http.StatusOK {
				bodyBytes, _ := io.ReadAll(resp.Body)
				t.Fatalf("response status code is not 200, got %d: %s", resp.StatusCode, string(bodyBytes))
			}
			var response struct {
				Result any `json:"result"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			var got string
			// Check if the result is a string (which contains JSON)
			if s, ok := response.Result.(string); ok {
				got = s
			} else {
				b, err := json.Marshal(response.Result)
				if err != nil {
					t.Fatalf("failed to marshal result object: %v", err)
				}
				got = string(b)
			}

			// Clean up both strings to ignore whitespace differences
			got = strings.ReplaceAll(strings.ReplaceAll(got, " ", ""), "\n", "")
			want := strings.ReplaceAll(strings.ReplaceAll(tc.want, " ", ""), "\n", "")

			if tc.wantSubstring {
				if !strings.Contains(got, want) {
					t.Fatalf("unexpected result: got %q, want substring %q", got, want)
				}
				return
			}

			if got != want {
				t.Fatalf("unexpected result: \ngot:  %s\nwant: %s", got, want)
			}
		})
	}
}

func getWaitToolsConfig() map[string]any {
	return map[string]any{
		"sources": map[string]any{
			"my-alloydb-source": map[string]any{
				"type": "alloydb-admin",
			},
		},
		"tools": map[string]any{
			"wait-for-op1": map[string]any{
				"type":        waitToolType,
				"source":      "my-alloydb-source",
				"description": "wait for op1",
			},
			"wait-for-op2": map[string]any{
				"type":        waitToolType,
				"source":      "my-alloydb-source",
				"description": "wait for op2",
			},
		},
	}
}
