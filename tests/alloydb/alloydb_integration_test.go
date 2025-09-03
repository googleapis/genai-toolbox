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
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server/mcp/jsonrpc"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"
)

var (
	AlloyDBProject  = os.Getenv("ALLOYDB_PROJECT")
	AlloyDBLocation = os.Getenv("ALLOYDB_REGION")
	AlloyDBCluster  = os.Getenv("ALLOYDB_CLUSTER")
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
		"project":  AlloyDBProject,
		"location": AlloyDBLocation,
		"cluster":  AlloyDBCluster,
	}
}

func getAlloyDBToolsConfig() map[string]any {
	return map[string]any{
		"tools" : map[string]any{
			// Tool for RunAlloyDBToolGetTest
			"my-simple-tool": map[string]any{
				"kind":        "alloydb-list-clusters",
				"description": "Simple tool to test end to end functionality.",
			},
			// Tool for MCP test
			"my-param-tool": map[string]any{
				"kind":        "alloydb-list-clusters",
				"description": "Tool to list clusters",
			},
			// Tool for MCP test that fails
			"my-fail-tool": map[string]any{
				"kind":        "alloydb-list-clusters",
				"description": "Tool that will fail",
			},
			// AlloyDB specific tools
			"alloydb-list-clusters": map[string]any{
				"kind":        "alloydb-list-clusters",
				"description": "Lists all AlloyDB clusters in a given project and location.",
			},
		},
	}
}

func TestAlloyDBAdminToolEndpoints(t *testing.T) {
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
						map[string]any{"name": "project", "type": "string", "description": "The GCP project ID to list clusters for.", "required": true, "authSources": []any{}},
						map[string]any{"name": "location", "type": "string", "description": "Optional: The location to list clusters in (e.g., 'us-central1'). Use '-' to list clusters across all locations.(Default: '-')", "required": false, "authSources": []any{}},
					},
					"authRequired": []any{},
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
						"project":  vars["project"],
						"location": vars["location"],
					},
				},
			},
			wantContains:  fmt.Sprintf(`"name\":\"projects/%s/locations/%s/clusters/%s\"`, vars["project"], vars["location"], vars["cluster"]),
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
						"location": vars["location"], 
					},
				},
			},
			wantContains:  `parameter \"project\" is required`,
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
					"arguments": map[string]any{"location": vars["location"]},
				},
			},
			wantContains:  `parameter \"project\" is required`,
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
        fmt.Sprintf("projects/%s/locations/us-central1/clusters/alloydb-ai-nl-testing", vars["project"]),
        fmt.Sprintf("projects/%s/locations/us-central1/clusters/alloydb-pg-testing", vars["project"]),
    }

	// NOTE: If clusters are added, removed, or changed in the test project,
    // this list must be updated for the "list clusters all locations" test to pass
	wantForAllLocations := []string{
        fmt.Sprintf("projects/%s/locations/us-central1/clusters/alloydb-ai-nl-testing", vars["project"]),
        fmt.Sprintf("projects/%s/locations/us-central1/clusters/alloydb-pg-testing", vars["project"]),
        fmt.Sprintf("projects/%s/locations/us-east4/clusters/alloydb-private-pg-testing", vars["project"]),
        fmt.Sprintf("projects/%s/locations/us-east4/clusters/colab-testing", vars["project"]),
    }

	invokeTcs := []struct {
		name           string
		requestBody    io.Reader
		want           []string
		wantStatusCode int
	}{
		{
			name:        "list clusters for all locations",
			requestBody: bytes.NewBufferString(fmt.Sprintf(`{"project": "%s", "location": "-"}`, vars["project"])),
			want:        wantForAllLocations,
			wantStatusCode: http.StatusOK,
		},
		{
			name:        "list clusters specific location",
			requestBody: bytes.NewBufferString(fmt.Sprintf(`{"project": "%s", "location": "us-central1"}`, vars["project"])),
			want:        wantForSpecificLocation,
			wantStatusCode: http.StatusOK,
		},
		{
			name:        "list clusters missing project",
			requestBody: bytes.NewBufferString(fmt.Sprintf(`{"location": "%s"}`, vars["location"])),
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:        "list clusters non-existent location",
			requestBody: bytes.NewBufferString(fmt.Sprintf(`{"project": "%s", "location": "abcd"}`, vars["project"])),
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:        "list clusters non-existent project",
			requestBody: bytes.NewBufferString(fmt.Sprintf(`{"project": "non-existent-project", "location": "%s"}`, vars["location"])),
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:        "list clusters empty project",
			requestBody: bytes.NewBufferString(fmt.Sprintf(`{"project": "", "location": "%s"}`, vars["location"])),
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:        "list clusters empty location",
			requestBody: bytes.NewBufferString(fmt.Sprintf(`{"project": "%s", "location": ""}`, vars["project"])),
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