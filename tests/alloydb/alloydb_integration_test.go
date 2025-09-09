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

package alloydb

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"
)

var (
	AlloyDBCreateInstanceToolKind = "alloydb-create-instance"
)

// HTTP handler for mock server
type handler struct {
	mu       sync.Mutex
	response mockResponse
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.response.body == "" {
		return
	}

	w.WriteHeader(h.response.statusCode)
	if _, err := w.Write([]byte(h.response.body)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *handler) setResponse(res mockResponse) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.response = res
}

type mockResponse struct {
	statusCode int
	body       string
}

func TestAlloyDBCreateInstance(t *testing.T) {
	h := &handler{}
	server := httptest.NewServer(h)
	defer server.Close()

	toolsFile := getAlloyDBCreateToolsConfig(server.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd, cleanup, err := tests.StartCmd(ctx, toolsFile)
	if err != nil {
		t.Fatalf("command initialization failed: %v", err)
	}
	defer cleanup()

	waitCtx, waitCancel := context.WithTimeout(ctx, 10*time.Second)
	defer waitCancel()
	_, err = testutils.WaitForString(waitCtx, regexp.MustCompile(`Server ready to serve`), cmd.Out)
	if err != nil {
		t.Fatalf("toolbox server didn't start successfully: %s", err)
	}

	tcs := []struct {
		name           string
		requestBody    string
		wantStatusCode int
		mockResponse   mockResponse
		want           map[string]any
	}{
		{
			name:           "create primary instance success",
			requestBody:    `{"projectId": "test-p", "locationId": "test-l", "clusterId": "test-c", "instanceId": "inst-1", "instanceType": "PRIMARY", "displayName": "My Primary"}`,
			wantStatusCode: http.StatusOK,
			mockResponse:   mockResponse{statusCode: http.StatusOK, body: `{"done":false,"metadata":{"@type":"type.googleapis.com/google.cloud.alloydb.v1.OperationMetadata","apiVersion":"v1","target":"projects/test-p/locations/test-l/clusters/test-c/instances/inst-1","verb":"create"},"name":"projects/test-p/locations/test-l/operations/op-123"}`},
			want:           map[string]any{"done": false, "metadata": map[string]any{"@type": "type.googleapis.com/google.cloud.alloydb.v1.OperationMetadata", "apiVersion": "v1", "target": "projects/test-p/locations/test-l/clusters/test-c/instances/inst-1", "verb": "create"}, "name": "projects/test-p/locations/test-l/operations/op-123"},
		},
		{
			name:           "create read pool instance success",
			requestBody:    `{"projectId": "test-p", "locationId": "test-l", "clusterId": "test-c", "instanceId": "read-1", "instanceType": "READ_POOL", "displayName": "read-instance", "nodeCount": 3}`,
			wantStatusCode: http.StatusOK,
			mockResponse:   mockResponse{statusCode: http.StatusOK, body: `{"done":false,"metadata":{"@type":"type.googleapis.com/google.cloud.alloydb.v1.OperationMetadata","apiVersion":"v1","target":"projects/test-p/locations/test-l/clusters/test-c/instances/read-1","verb":"create"},"name":"projects/test-p/locations/test-l/operations/op-456"}`},
			want:           map[string]any{"done": false, "metadata": map[string]any{"@type": "type.googleapis.com/google.cloud.alloydb.v1.OperationMetadata", "apiVersion": "v1", "target": "projects/test-p/locations/test-l/clusters/test-c/instances/read-1", "verb": "create"}, "name": "projects/test-p/locations/test-l/operations/op-456"},
		},
		{
			name:           "create instance api failure",
			requestBody:    `{"projectId": "test-p", "locationId": "test-l", "clusterId": "test-c", "instanceId": "inst-fail", "instanceType": "PRIMARY"}`,
			wantStatusCode: http.StatusBadRequest,
			mockResponse:   mockResponse{statusCode: http.StatusInternalServerError, body: `{"error": "some api error"}`},
		},
		{
			name:           "create instance missing projectId",
			requestBody:    `{"locationId": "l1", "clusterId": "c1", "instanceId": "i1", "instanceType": "PRIMARY"}`,
			wantStatusCode: http.StatusBadRequest,
			mockResponse:   mockResponse{body: ""},
		},
		{
			name:           "create instance missing instanceType",
			requestBody:    `{"projectId": "p1", "locationId": "l1", "clusterId": "c1", "instanceId": "i1"}`,
			wantStatusCode: http.StatusBadRequest,
			mockResponse:   mockResponse{body: ""},
		},
		{
			name:           "create instance missing clusterId",
			requestBody:    `{"projectId": "p1", "locationId": "l1", "instanceId": "i1", "instanceType": "PRIMARY"}`,
			wantStatusCode: http.StatusBadRequest,
			mockResponse:   mockResponse{body: ""},
		},
		{
			name:           "create instance missing instanceId",
			requestBody:    `{"projectId": "p1", "locationId": "l1", "clusterId": "c1", "instanceType": "PRIMARY"}`,
			wantStatusCode: http.StatusBadRequest,
			mockResponse:   mockResponse{body: ""},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			h.setResponse(tc.mockResponse)

			api := "http://127.0.0.1:5000/api/tool/alloydb-create-instance/invoke"
			req, err := http.NewRequest(http.MethodPost, api, bytes.NewBufferString(tc.requestBody))
			if err != nil {
				t.Fatalf("unable to create request: %s", err)
			}
			req.Header.Add("Content-type", "application/json")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("unable to send request: %s", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.wantStatusCode {
				bodyBytes, _ := io.ReadAll(resp.Body)
				t.Fatalf("expected status %d, got %d: %s", tc.wantStatusCode, resp.StatusCode, string(bodyBytes))
			}

			if tc.want != nil {
				var result struct {
					Result string `json:"result"`
				}
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}

				var got map[string]any
				if err := json.Unmarshal([]byte(result.Result), &got); err != nil {
					t.Fatalf("failed to unmarshal nested result: %v", err)
				}

				if diff := cmp.Diff(got, tc.want); diff != "" {
					t.Fatalf("got %v, want %v", got, tc.want)
				}
			}
		})
	}
}

func getAlloyDBCreateToolsConfig(baseURL string) map[string]any {
	return map[string]any{
		"sources": map[string]any{
			"alloydb-admin-source": map[string]any{
				"kind":    "http",
				"baseUrl": baseURL,
			},
		},
		"tools": map[string]any{
			"alloydb-create-instance": map[string]any{
				"kind":        AlloyDBCreateInstanceToolKind,
				"source":      "alloydb-admin-source",
				"description": "Create a new AlloyDB instance (PRIMARY or READ_POOL) within a cluster. This is a long-running operation.",
				"baseURL":     baseURL,
			},
		},
	}
}