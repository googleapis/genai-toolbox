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
	AlloyDBCreateUserToolKind = "alloydb-create-user"
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

func TestAlloyDBCreateUser(t *testing.T) {
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
			name:           "create built-in user with roles success",
			requestBody:    `{"projectId": "p1", "locationId": "l1", "clusterId": "c1", "userId": "test-user", "userType": "ALLOYDB_BUILT_IN", "password": "SafePassword!123", "databaseRoles": ["alloydbsuperuser"]}`,
			wantStatusCode: http.StatusOK,
			mockResponse: mockResponse{statusCode: http.StatusOK, body: `{"name":"projects/p1/locations/l1/clusters/c1/users/test-user", "userType":"ALLOYDB_BUILT_IN", "databaseRoles":["alloydbsuperuser"]}`},
			want: map[string]any{"name": "projects/p1/locations/l1/clusters/c1/users/test-user", "userType": "ALLOYDB_BUILT_IN", "databaseRoles": []any{"alloydbsuperuser"}},
		},
		{
			name:           "create iam user success",
			requestBody:    `{"projectId": "p1", "locationId": "l1", "clusterId": "c1", "userId": "iam-user@example.com", "userType": "ALLOYDB_IAM_USER"}`,
			wantStatusCode: http.StatusOK,
			mockResponse: mockResponse{statusCode: http.StatusOK, body: `{"name":"projects/p1/locations/l1/clusters/c1/users/iam-user@example.com", "userType":"ALLOYDB_IAM_USER", "databaseRoles":["alloydbiamuser"]}`},
			want: map[string]any{"name": "projects/p1/locations/l1/clusters/c1/users/iam-user@example.com", "userType": "ALLOYDB_IAM_USER", "databaseRoles": []any{"alloydbiamuser"}},
		},
		{
			name:           "create user failure",
			requestBody:    `{"projectId": "p1", "locationId": "l1", "clusterId": "c1", "userId": "api-fail-user", "userType": "ALLOYDB_BUILT_IN", "password": "SafePassword!123"}`,
			wantStatusCode: http.StatusBadRequest,
			mockResponse:   mockResponse{statusCode: http.StatusInternalServerError, body: `{"error": "some api error"}`},
		},
		{
			name:           "create built-in user with missing password",
			requestBody:    `{"projectId": "p1", "locationId": "l1", "clusterId": "c1", "userId": "bad-user", "userType": "ALLOYDB_BUILT_IN"}`,
			wantStatusCode: http.StatusBadRequest,
			mockResponse:   mockResponse{body: ""},
		},
		{
			name:           "create user missing userId",
			requestBody:    `{"projectId": "p1", "locationId": "l1", "clusterId": "c1", "userType": "ALLOYDB_BUILT_IN", "password": "pw"}`,
			wantStatusCode: http.StatusBadRequest,
			mockResponse:   mockResponse{body: ""},
		},
		{
			name:           "create user missing clusterId",
			requestBody:    `{"projectId": "p1", "locationId": "l1", "userId": "u1", "userType": "ALLOYDB_BUILT_IN", "password": "pw"}`,
			wantStatusCode: http.StatusBadRequest,
			mockResponse:   mockResponse{body: ""},
		},
		{
			name:           "create user missing projectId",
			requestBody:    `{"locationId": "l1", "clusterId": "c1", "userId": "u1", "userType": "ALLOYDB_BUILT_IN", "password": "pw"}`,
			wantStatusCode: http.StatusBadRequest,
			mockResponse:   mockResponse{body: ""},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			h.setResponse(tc.mockResponse)

			api := "http://127.0.0.1:5000/api/tool/alloydb-create-user/invoke"
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
			"alloydb-create-user": map[string]any{
				"kind":        AlloyDBCreateUserToolKind,
				"source":      "alloydb-admin-source",
				"description": "Creates a new database user in an AlloyDB cluster. Takes the new user's name and a secure password. Optionally, a list of database roles can be assigned.",
				"baseURL":     baseURL,
			},
		},
	}
}