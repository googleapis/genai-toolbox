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
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server/mcp/jsonrpc"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"
)

var (
	AlloyDBCreateClusterToolKind = "alloydb-create-cluster"
	AlloyDBProject               = os.Getenv("ALLOYDB_PROJECT")
	AlloyDBLocation              = os.Getenv("ALLOYDB_REGION")
	AlloyDBCluster               = os.Getenv("ALLOYDB_CLUSTER")
)

func getAlloyDBVars(t *testing.T) map[string]string {
	if AlloyDBProject == "" {
		t.Fatal("'ALLOYDB_PROJECT' not set")
	}
	if AlloyDBLocation == "" {
		t.Fatal("'ALLOYDB_REGION' not set")
	}
	if AlloyDBCluster == "" {
		t.Fatal("'ALLOYDB_CLUSTER' not set")
	}
	return map[string]string{
		"projectId":  AlloyDBProject,
		"locationId": AlloyDBLocation,
		"clusterId":  AlloyDBCluster,
	}
}

func getAlloyDBToolsConfig() map[string]any {
	return map[string]any{
		"sources": map[string]any{
			"alloydb-admin-source": map[string]any{
				"kind":    "alloydb-admin",
			},
		},
		"tools" : map[string]any{
			// Tool for RunAlloyDBToolGetTest
			"my-simple-tool": map[string]any{
				"kind":        "alloydb-list-clusters",
				"source":      "alloydb-admin-source",
				"description": "Simple tool to test end to end functionality.",
			},
			// Tool for MCP test
			"my-param-tool": map[string]any{
				"kind":        "alloydb-list-clusters",
				"source":      "alloydb-admin-source",
				"description": "Tool to list clusters",
			},
			// Tool for MCP test that fails
			"my-fail-tool": map[string]any{
				"kind":        "alloydb-list-clusters",
				"source":      "alloydb-admin-source",
				"description": "Tool that will fail",
			},
			// AlloyDB specific tools
			"alloydb-list-clusters": map[string]any{
				"kind":        "alloydb-list-clusters",
				"source":      "alloydb-admin-source",
				"description": "Lists all AlloyDB clusters in a given project and location.",
			},
		},
	}
}

func TestAlloyDBToolEndpoints(t *testing.T) {
	vars := getAlloyDBVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	var args []string
	toolsFile := getAlloyDBToolsConfig()

	cmd, cleanup, err := tests.StartCmd(ctx, toolsFile, args...)
	if err != nil {
		t.Fatalf("command initialization returned an error: %v", err)
	}
	defer cleanup()

	waitCtx, cancelWait := context.WithTimeout(ctx, 20*time.Second)
	defer cancelWait()
	out, err := testutils.WaitForString(waitCtx, regexp.MustCompile(`Server ready to serve`), cmd.Out)
	if err != nil {
		t.Logf("toolbox command logs: \n%s", out)
		t.Fatalf("toolbox didn't start successfully: %v", err)
	}

	runAlloyDBToolGetTest(t)
	runAlloyDBMCPToolCallMethod(t, vars)

	// Run tool-specific invoke tests
	runAlloyDBListClustersTest(t, vars)
}

func runAlloyDBToolGetTest(t *testing.T) {
	tcs := []struct {
		name string
		api  string
		want map[string]any
	}{
		{
			name: "get my-simple-tool",
			api:  "http://127.0.0.1:5000/api/tool/my-simple-tool/",
			want: map[string]any{
				"my-simple-tool": map[string]any{
					"description": "Simple tool to test end to end functionality.",
					"parameters": []any{
						map[string]any{"name": "projectId", "type": "string", "description": "The GCP project ID to list clusters for.", "required": true, "authSources": []any{}},
						map[string]any{"name": "locationId", "type": "string", "description": "Optional: The location to list clusters in (e.g., 'us-central1'). Use '-' to list clusters across all locations.(Default: '-')", "required": false, "authSources": []any{}},
					},
					"authRequired": nil,
				},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := http.Get(tc.api)
			if err != nil {
				t.Fatalf("error when sending a request: %s", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("response status code is not 200")
			}

			var body map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
				t.Fatalf("error parsing response body: %v", err)
			}

			got, ok := body["tools"]
			if !ok {
				t.Fatalf("unable to find 'tools' in response body")
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("response mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func runAlloyDBMCPToolCallMethod(t *testing.T, vars map[string]string) {
	sessionId := tests.RunInitialize(t, "2024-11-05")
	header := map[string]string{}
	if sessionId != "" {
		header["Mcp-Session-Id"] = sessionId
	}

	invokeTcs := []struct {
		name        string
		requestBody jsonrpc.JSONRPCRequest
		wantContains        string
		isErr       bool
	}{
		{
			name: "MCP Invoke my-param-tool",
			requestBody: jsonrpc.JSONRPCRequest{
				Jsonrpc: "2.0",
				Id:      "my-param-tool-mcp",
				Request: jsonrpc.Request{Method: "tools/call"},
				Params: map[string]any{
					"name": "my-param-tool",
					"arguments": map[string]any{
						"projectId":  vars["projectId"],
						"locationId": vars["locationId"],
					},
				},
			},
			wantContains:  fmt.Sprintf(`"name\":\"projects/%s/locations/%s/clusters/%s\"`, vars["projectId"], vars["locationId"], vars["clusterId"]),
			isErr: false,
		},
		{
			name: "MCP Invoke my-fail-tool",
			requestBody: jsonrpc.JSONRPCRequest{
				Jsonrpc: "2.0",
				Id:      "invoke-fail-tool",
				Request: jsonrpc.Request{Method: "tools/call"},
				Params: map[string]any{
					"name": "my-fail-tool",
					"arguments": map[string]any{
						"locationId": vars["locationId"],
					},
				},
			},
			wantContains:  `parameter \"projectId\" is required`,
			isErr: true,
		},
		{
			name: "MCP Invoke invalid tool",
			requestBody: jsonrpc.JSONRPCRequest{
				Jsonrpc: "2.0",
				Id:      "invalid-tool-mcp",
				Request: jsonrpc.Request{Method: "tools/call"},
				Params: map[string]any{
					"name":      "non-existent-tool",
					"arguments": map[string]any{},
				},
			},
			wantContains:  `tool with name \"non-existent-tool\" does not exist`,
			isErr: true,
		},
		{
			name: "MCP Invoke tool without required parameters",
			requestBody: jsonrpc.JSONRPCRequest{
				Jsonrpc: "2.0",
				Id:      "invoke-without-params-mcp",
				Request: jsonrpc.Request{Method: "tools/call"},
				Params: map[string]any{
					"name":      "my-param-tool",
					"arguments": map[string]any{"locationId": vars["locationId"]},
				},
			},
			wantContains:  `parameter \"projectId\" is required`,
			isErr: true,
		},
		{
			name: "MCP Invoke my-auth-required-tool",
			requestBody: jsonrpc.JSONRPCRequest{
				Jsonrpc: "2.0",
				Id:      "invoke my-auth-required-tool",
				Request: jsonrpc.Request{Method: "tools/call"},
				Params: map[string]any{
					"name":      "my-auth-required-tool",
					"arguments": map[string]any{},
				},
			},
			wantContains:  `tool with name \"my-auth-required-tool\" does not exist`,
			isErr: true,
		},
	}

	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			api := "http://127.0.0.1:5000/mcp"
			reqMarshal, err := json.Marshal(tc.requestBody)
			if err != nil {
				t.Fatalf("unexpected error during marshaling of request body: %v", err)
			}

			req, err := http.NewRequest(http.MethodPost, api, bytes.NewBuffer(reqMarshal))
			if err != nil {
				t.Fatalf("unable to create request: %s", err)
			}
			req.Header.Add("Content-type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("unable to send request: %s", err)
			}
			defer resp.Body.Close()

			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("unable to read request body: %s", err)
			}

			got := string(bytes.TrimSpace(respBody))
			if !strings.Contains(got, tc.wantContains) {
				t.Fatalf("Expected substring not found:\ngot:  %q\nwant: %q (to be contained within got)", got, tc.wantContains)
			}
		})
	}
}

func runAlloyDBListClustersTest(t *testing.T, vars map[string]string) {

	type ListClustersResponse struct {
        Clusters []struct {
            Name string `json:"name"`
        } `json:"clusters"`
    }

	type ToolResponse struct {
		Result string `json:"result"`
	}

	// NOTE: If clusters are added, removed or changed in the test project,
    // this list must be updated for the "list clusters specific locations" test to pass
	wantForSpecificLocation := []string{
        fmt.Sprintf("projects/%s/locations/us-central1/clusters/alloydb-ai-nl-testing", vars["projectId"]),
        fmt.Sprintf("projects/%s/locations/us-central1/clusters/alloydb-pg-testing", vars["projectId"]),
    }

	// NOTE: If clusters are added, removed, or changed in the test project,
    // this list must be updated for the "list clusters all locations" test to pass
	wantForAllLocations := []string{
        fmt.Sprintf("projects/%s/locations/us-central1/clusters/alloydb-ai-nl-testing", vars["projectId"]),
        fmt.Sprintf("projects/%s/locations/us-central1/clusters/alloydb-pg-testing", vars["projectId"]),
        fmt.Sprintf("projects/%s/locations/us-east4/clusters/alloydb-private-pg-testing", vars["projectId"]),
        fmt.Sprintf("projects/%s/locations/us-east4/clusters/colab-testing", vars["projectId"]),
    }

	invokeTcs := []struct {
		name           string
		requestBody    io.Reader
		want           []string
		wantStatusCode int
	}{
		{
			name:        "list clusters for all locations",
			requestBody: bytes.NewBufferString(fmt.Sprintf(`{"projectId": "%s", "locationId": "-"}`, vars["projectId"])),
			want:        wantForAllLocations,
			wantStatusCode: http.StatusOK,
		},
		{
			name:        "list clusters specific location",
			requestBody: bytes.NewBufferString(fmt.Sprintf(`{"projectId": "%s", "locationId": "us-central1"}`, vars["projectId"])),
			want:        wantForSpecificLocation,
			wantStatusCode: http.StatusOK,
		},
		{
			name:        "list clusters missing project",
			requestBody: bytes.NewBufferString(fmt.Sprintf(`{"locationId": "%s"}`, vars["locationId"])),
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:        "list clusters non-existent location",
			requestBody: bytes.NewBufferString(fmt.Sprintf(`{"projectId": "%s", "locationId": "abcd"}`, vars["projectId"])),
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name:        "list clusters non-existent project",
			requestBody: bytes.NewBufferString(fmt.Sprintf(`{"projectId": "non-existent-project", "locationId": "%s"}`, vars["locationId"])),
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name:        "list clusters empty project",
			requestBody: bytes.NewBufferString(fmt.Sprintf(`{"projectId": "", "locationId": "%s"}`, vars["locationId"])),
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:        "list clusters empty location",
			requestBody: bytes.NewBufferString(fmt.Sprintf(`{"projectId": "%s", "locationId": ""}`, vars["projectId"])),
			wantStatusCode: http.StatusBadRequest,
		},
	}

	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			api := "http://127.0.0.1:5000/api/tool/alloydb-list-clusters/invoke"
			req, err := http.NewRequest(http.MethodPost, api, tc.requestBody)
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
				t.Fatalf("response status code is not %d, got %d: %s", tc.wantStatusCode, resp.StatusCode, string(bodyBytes))
			}

			if tc.wantStatusCode == http.StatusOK {
				var body ToolResponse
                if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
                    t.Fatalf("error parsing outer response body: %v", err)
                }

                var clustersData ListClustersResponse
                if err := json.Unmarshal([]byte(body.Result), &clustersData); err != nil {
                    t.Fatalf("error parsing nested result JSON: %v", err)
                }

                var got []string
                for _, cluster := range clustersData.Clusters {
                    got = append(got, cluster.Name)
                }

                sort.Strings(got)
                sort.Strings(tc.want)

                if !reflect.DeepEqual(got, tc.want) {
                    t.Errorf("cluster list mismatch:\n got: %v\nwant: %v", got, tc.want)
                }
			}
		})
	}
}

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

func TestAlloyDBCreateCluster(t *testing.T) {
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
		want     map[string]any
	}{
		{
			name:           "create cluster success",
			requestBody:    fmt.Sprintf(`{"projectId": "%s", "locationId": "%s", "clusterId": "test", "password": "p"}`, "test-project", "test-location"),
			wantStatusCode: http.StatusOK,
			mockResponse:   mockResponse{statusCode: http.StatusOK, body: `{"done":false,"metadata":{"@type":"type.googleapis.com/google.cloud.alloydb.v1.OperationMetadata","apiVersion":"v1","createTime":"2025-09-04T05:38:38.055667814Z","requestedCancellation":false,"target":"projects/test-project/locations/test-location/clusters/test-create-cluster","verb":"create"},"name":"projects/test-project/locations/test-location/operations/test-operation"}`},
			want:     map[string]any{"done": false, "metadata": map[string]any{"@type": "type.googleapis.com/google.cloud.alloydb.v1.OperationMetadata", "apiVersion": "v1", "createTime": "2025-09-04T05:38:38.055667814Z", "requestedCancellation": false, "target": "projects/test-project/locations/test-location/clusters/test-create-cluster", "verb": "create"}, "name": "projects/test-project/locations/test-location/operations/test-operation"},
		},
		{
			name:           "create cluster failure",
			requestBody:    fmt.Sprintf(`{"projectId": "%s", "locationId": "%s", "clusterId": "test", "password": "p"}`, AlloyDBProject, AlloyDBLocation),
			wantStatusCode: http.StatusBadRequest,
			mockResponse:   mockResponse{statusCode: http.StatusInternalServerError, body: `{"error": "failed"}`},
		},
		{
			name:           "create cluster with missing projectId",
			requestBody:    `{"locationId": "l1", "clusterId": "c1", "password": "p1"}`,
			wantStatusCode: http.StatusBadRequest,
			mockResponse:   mockResponse{statusCode: http.StatusBadRequest, body: ""},
		},
		{
			name:           "create cluster with missing locationId",
			requestBody:    `{"projectId": "p1", "clusterId": "c1", "password": "p1"}`,
			wantStatusCode: http.StatusBadRequest,
			mockResponse:   mockResponse{statusCode: http.StatusBadRequest, body: ""},
		},
		{
			name:           "create cluster with missing clusterId",
			requestBody:    `{"projectId": "p1", "locationId": "l1", "password": "p1"}`,
			wantStatusCode: http.StatusBadRequest,
			mockResponse:   mockResponse{statusCode: http.StatusBadRequest, body: ""},
		},
		{
			name:           "create cluster with missing password",
			requestBody:    `{"projectId": "p1", "locationId": "l1", "clusterId": "c1"}`,
			wantStatusCode: http.StatusBadRequest,
			mockResponse:   mockResponse{statusCode: http.StatusBadRequest, body: ""},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			h.setResponse(tc.mockResponse)

			api := "http://127.0.0.1:5000/api/tool/alloydb-create-cluster/invoke"
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
			"alloydb-create-cluster": map[string]any{
				"kind":        AlloyDBCreateClusterToolKind,
				"source": "alloydb-admin-source",
				"description": "Create a new AlloyDB cluster. This is a long-running operation, but the API call returns quickly. This will return operation id to be used by get operations tool. Take all parameters from user in one go.",
				"baseURL":     baseURL,
			},
		},
	}
}