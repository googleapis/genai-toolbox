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

package cloudsql

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/googleapis/mcp-toolbox/internal/testutils"
	"github.com/googleapis/mcp-toolbox/tests"

	_ "github.com/googleapis/mcp-toolbox/internal/tools/cloudsql/cloudsqlwaitforoperation"
)

var (
	cloudsqlWaitToolType = "cloud-sql-wait-for-operation"
)

type waitForOperationTransport struct {
	transport http.RoundTripper
	url       *url.URL
}

func (t *waitForOperationTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if strings.HasPrefix(req.URL.String(), "https://sqladmin.googleapis.com") {
		req.URL.Scheme = t.url.Scheme
		req.URL.Host = t.url.Host
	}
	return t.transport.RoundTrip(req)
}

type cloudsqlOperation struct {
	Name          string `json:"name"`
	Status        string `json:"status"`
	TargetLink    string `json:"targetLink"`
	OperationType string `json:"operationType"`
	Error         *struct {
		Errors []struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"errors"`
	} `json:"error,omitempty"`
}

type cloudsqlInstance struct {
	Region          string `json:"region"`
	DatabaseVersion string `json:"databaseVersion"`
}

type cloudsqlHandler struct {
	mu         sync.Mutex
	operations map[string]*cloudsqlOperation
	instances  map[string]*cloudsqlInstance
	t          *testing.T
}

func (h *cloudsqlHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !strings.Contains(r.UserAgent(), "genai-toolbox/") {
		h.t.Errorf("User-Agent header not found")
	}

	if match, _ := regexp.MatchString("/v1/projects/p1/operations/.*", r.URL.Path); match {
		parts := regexp.MustCompile("/").Split(r.URL.Path, -1)
		opName := parts[len(parts)-1]

		op, ok := h.operations[opName]
		if !ok {
			http.NotFound(w, r)
			return
		}

		if op.Status != "DONE" {
			op.Status = "DONE"
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(op); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	} else if match, _ := regexp.MatchString("/v1/projects/p1/instances/.*", r.URL.Path); match {
		parts := regexp.MustCompile("/").Split(r.URL.Path, -1)
		instanceName := parts[len(parts)-1]

		instance, ok := h.instances[instanceName]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(instance); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	} else {
		http.NotFound(w, r)
	}
}

func getCloudSQLWaitToolsConfig() map[string]any {
	return map[string]any{
		"sources": map[string]any{
			"my-cloud-sql-source": map[string]any{
				"type": "cloud-sql-admin",
			},
		},
		"tools": map[string]any{
			"wait-for-op1": map[string]any{
				"type":        cloudsqlWaitToolType,
				"source":      "my-cloud-sql-source",
				"description": "wait for op1",
			},
			"wait-for-op2": map[string]any{
				"type":        cloudsqlWaitToolType,
				"source":      "my-cloud-sql-source",
				"description": "wait for op2",
			},
			"wait-for-op3": map[string]any{
				"type":        cloudsqlWaitToolType,
				"source":      "my-cloud-sql-source",
				"description": "wait for op3",
			},
		},
	}
}

func TestCloudSQLWaitToolMCP(t *testing.T) {
	h := &cloudsqlHandler{
		operations: map[string]*cloudsqlOperation{
			"op1": {Name: "op1", Status: "PENDING", OperationType: "CREATE_DATABASE"},
			"op2": {Name: "op2", Status: "PENDING", OperationType: "CREATE_DATABASE", Error: &struct {
				Errors []struct {
					Code    string `json:"code"`
					Message string `json:"message"`
				} `json:"errors"`
			}{
				Errors: []struct {
					Code    string `json:"code"`
					Message string `json:"message"`
				}{
					{Code: "ERROR_CODE", Message: "failed"},
				},
			}},
			"op3": {Name: "op3", Status: "PENDING", OperationType: "CREATE"},
		},
		instances: map[string]*cloudsqlInstance{
			"i1": {Region: "r1", DatabaseVersion: "POSTGRES_13"},
		},
		t: t,
	}
	server := httptest.NewServer(h)
	defer server.Close()

	h.operations["op1"].TargetLink = "https://sqladmin.googleapis.com/v1/projects/p1/instances/i1/databases/d1"
	h.operations["op2"].TargetLink = "https://sqladmin.googleapis.com/v1/projects/p1/instances/i2/databases/d2"
	h.operations["op3"].TargetLink = "https://sqladmin.googleapis.com/v1/projects/p1/instances/i1"

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

	toolsFile := getCloudSQLWaitToolsConfig()
	cmd, cleanup, err := tests.StartCmd(ctx, toolsFile)
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
		args          map[string]any
		want          string
		expectError   bool
		wantError     string
		wantSubstring bool
	}{
		{
			name:          "successful operation",
			toolName:      "wait-for-op1",
			args:          map[string]any{"project": "p1", "operation": "op1"},
			want:          "Your Cloud SQL resource is ready",
			wantSubstring: true,
		},
		{
			name:        "failed operation - agent error",
			toolName:    "wait-for-op2",
			args:        map[string]any{"project": "p1", "operation": "op2"},
			expectError: true,
			wantError:   "failed",
		},
		{
			name:     "non-database create operation",
			toolName: "wait-for-op3",
			args:     map[string]any{"project": "p1", "operation": "op3"},
			want:     `{"name":"op3","status":"DONE","targetLink":"` + h.operations["op3"].TargetLink + `","operationType":"CREATE"}`,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			statusCode, mcpResp, err := tests.InvokeMCPTool(t, tc.toolName, tc.args, nil)
			if err != nil {
				t.Fatalf("native error executing %s: %s", tc.toolName, err)
			}
			if statusCode != http.StatusOK {
				t.Fatalf("expected status 200, got %d", statusCode)
			}

			if tc.expectError {
				tests.AssertMCPError(t, mcpResp, tc.wantError)
				return
			}

			if mcpResp.Result.IsError {
				t.Fatalf("%s returned error result: %v", tc.toolName, mcpResp.Result)
			}

			if len(mcpResp.Result.Content) == 0 {
				t.Fatalf("%s returned empty content field", tc.toolName)
			}

			// Gather all the text blocks
			var blocks []string
			for _, content := range mcpResp.Result.Content {
				if content.Type == "text" {
					blocks = append(blocks, strings.TrimSpace(content.Text))
				}
			}

			got := strings.Join(blocks, "")

			if tc.wantSubstring {
				if !strings.Contains(got, tc.want) {
					t.Fatalf("unexpected result: got %q, want substring %q", got, tc.want)
				}
				return
			}

			var tempString string
			if err := json.Unmarshal([]byte(got), &tempString); err != nil {
				t.Fatalf("failed to unmarshal outer JSON string: %v\nraw: %s", err, got)
			}

			var gotMap, wantMap map[string]any
			if err := json.Unmarshal([]byte(tempString), &gotMap); err != nil {
				t.Fatalf("failed to unmarshal inner JSON object: %v\ntemp: %s", err, tempString)
			}
			if err := json.Unmarshal([]byte(tc.want), &wantMap); err != nil {
				t.Fatalf("failed to unmarshal want: %v", err)
			}

			if !reflect.DeepEqual(gotMap, wantMap) {
				t.Fatalf("unexpected result: got %+v, want %+v", gotMap, wantMap)
			}
		})
	}
}
