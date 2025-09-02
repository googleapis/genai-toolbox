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

package cloudsql

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

	_ "github.com/googleapis/genai-toolbox/internal/tools/cloudsql/cloudsqlwaitforoperation"
)

var (
	cloudsqlWaitToolKind = "cloudsql-wait-for-operation"
)

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
}

func (h *cloudsqlHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()

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

func TestCloudSQLWaitToolEndpoints(t *testing.T) {
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
	}
	server := httptest.NewServer(h)
	defer server.Close()

	h.operations["op1"].TargetLink = fmt.Sprintf("%s/v1/projects/p1/instances/i1/databases/d1", server.URL)
	h.operations["op2"].TargetLink = fmt.Sprintf("%s/v1/projects/p1/instances/i2/databases/d2", server.URL)
	h.operations["op3"].TargetLink = fmt.Sprintf("%s/v1/projects/p1/instances/i1", server.URL)

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	toolsFile := getCloudSQLWaitToolsConfig(server.URL)
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
			name:          "successful operation",
			toolName:      "wait-for-op1",
			body:          `{"project": "p1", "operation": "op1"}`,
			want:          "Your CloudSQL resource is ready",
			wantSubstring: true,
		},
		{
			name:        "failed operation",
			toolName:    "wait-for-op2",
			body:        `{"project": "p1", "operation": "op2"}`,
			expectError: true,
		},
		{
			name:     "non-database create operation",
			toolName: "wait-for-op3",
			body:     `{"project": "p1", "operation": "op3"}`,
			want:     `{"name":"op3","status":"DONE","targetLink":"` + h.operations["op3"].TargetLink + `","operationType":"CREATE"}`,
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

			if tc.wantSubstring {
				var result struct {
					Result string `json:"result"`
				}
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}

				if !bytes.Contains([]byte(result.Result), []byte(tc.want)) {
					t.Fatalf("unexpected result: got %q, want substring %q", result.Result, tc.want)
				}
				return
			}

			var result struct {
				Result string `json:"result"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			var tempString string
			if err := json.Unmarshal([]byte(result.Result), &tempString); err != nil {
				t.Fatalf("failed to unmarshal outer JSON string: %v", err)
			}

			var got, want map[string]any
			if err := json.Unmarshal([]byte(tempString), &got); err != nil {
				t.Fatalf("failed to unmarshal inner JSON object: %v", err)
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

func getCloudSQLWaitToolsConfig(baseURL string) map[string]any {
	return map[string]any{
		"tools": map[string]any{
			"wait-for-op1": map[string]any{
				"kind":         cloudsqlWaitToolKind,
				"description":  "wait for op1",
				"baseURL":      baseURL,
				"authRequired": []string{},
			},
			"wait-for-op2": map[string]any{
				"kind":         cloudsqlWaitToolKind,
				"description":  "wait for op2",
				"baseURL":      baseURL,
				"authRequired": []string{},
			},
			"wait-for-op3": map[string]any{
				"kind":         cloudsqlWaitToolKind,
				"description":  "wait for op3",
				"baseURL":      baseURL,
				"authRequired": []string{},
			},
		},
	}
}
