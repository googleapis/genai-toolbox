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
	AlloyDBInstance = os.Getenv("ALLOYDB_INSTANCE")
	AlloyDBUser     = os.Getenv("ALLOYDB_POSTGRES_USER")
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
	if AlloyDBInstance == "" {
		t.Fatal("'ALLOYDB_INSTANCE' not set")
	}
	if AlloyDBUser == "" {
		t.Fatal("'ALLOYDB_USER' not set")
	}
	return map[string]string{
		"projectId":  AlloyDBProject,
		"locationId": AlloyDBLocation,
		"clusterId":  AlloyDBCluster,
		"instanceId": AlloyDBInstance,
		"user": AlloyDBUser,
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
			"alloydb-list-users": map[string]any{
				"kind":        "alloydb-list-users",
				"source":      "alloydb-admin-source",
				"description": "Lists all AlloyDB users within a specific cluster.",
			},
			"alloydb-list-instances": map[string]any{
				"kind":        "alloydb-list-instances",
				"source":      "alloydb-admin-source",
				"description": "Lists all AlloyDB instances within a specific cluster.",
			},
			"alloydb-get-cluster": map[string]any{
				"kind":        "alloydb-get-cluster",
				"source":      "alloydb-admin-source",
				"description": "Retrieves details of a specific AlloyDB cluster.",
			},
			"alloydb-get-user": map[string]any{
				"kind":        "alloydb-get-user",
				"source":      "alloydb-admin-source",
				"description": "Retrieves details of a specific AlloyDB user.",
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
	runAlloyDBListUsersTest(t, vars)
	runAlloyDBListInstancesTest(t, vars)
	runAlloyDBGetClusterTest(t, vars)
	runAlloyDBGetUserTest(t, vars)
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

func runAlloyDBListUsersTest(t *testing.T, vars map[string]string) {
	type UsersResponse struct {
		Users []struct {
			Name string `json:"name"`
		} `json:"users"`
	}

	type ToolResponse struct {
		Result string `json:"result"`
	}

	invokeTcs := []struct {
		name           string
		requestBody    io.Reader
		wantContains   string
		wantCount      int
		wantStatusCode int
	}{
		{
			name:           "list users success",
			requestBody:    bytes.NewBufferString(fmt.Sprintf(`{"projectId": "%s", "locationId": "%s", "clusterId": "%s"}`, vars["projectId"], vars["locationId"], vars["clusterId"])),
			wantContains:   fmt.Sprintf("projects/%s/locations/%s/clusters/%s/users/%s", vars["projectId"], vars["locationId"], vars["clusterId"], AlloyDBUser),
			wantCount:      3,   // NOTE: If users are added or removed in the test project, update the number of users here must be updated for this test to pass 
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "list users missing project",
			requestBody:    bytes.NewBufferString(fmt.Sprintf(`{"locationId": "%s", "clusterId": "%s"}`, vars["locationId"], vars["clusterId"])),
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "list users missing location",
			requestBody:    bytes.NewBufferString(fmt.Sprintf(`{"projectId": "%s", "clusterId": "%s"}`, vars["projectId"], vars["clusterId"])),
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "list users missing cluster",
			requestBody:    bytes.NewBufferString(fmt.Sprintf(`{"projectId": "%s", "locationId": "%s"}`, vars["projectId"], vars["clusterId"])),
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "list users non-existent project",
			requestBody:    bytes.NewBufferString(fmt.Sprintf(`{"projectId": "non-existent-project", "locationId": "%s", "clusterId": "%s"}`, vars["locationId"], vars["clusterId"])),
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name:           "list users non-existent location",
			requestBody:    bytes.NewBufferString(fmt.Sprintf(`{"projectId": "%s", "locationId": "non-existent-location", "clusterId": "%s"}`, vars["projectId"], vars["clusterId"])),
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name:           "list users non-existent cluster",
			requestBody:    bytes.NewBufferString(fmt.Sprintf(`{"projectId": "%s", "locationId": "%s", "clusterId": "non-existent-cluster"}`, vars["projectId"], vars["locationId"])),
			wantStatusCode: http.StatusBadRequest,
		},
	}

	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			api := "http://127.0.0.1:5000/api/tool/alloydb-list-users/invoke"
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

				var usersData UsersResponse
				if err := json.Unmarshal([]byte(body.Result), &usersData); err != nil {
					t.Fatalf("error parsing nested result JSON: %v", err)
				}

				var got []string
				for _, user := range usersData.Users {
					got = append(got, user.Name)
				}

				sort.Strings(got)

				if len(got) != tc.wantCount {
					t.Errorf("user count mismatch:\n got: %v\nwant: %v", len(got), tc.wantCount)
				}

				found := false
				for _, g := range got {
					if g == tc.wantContains {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("wantContains not found in response:\n got: %v\nwant: %v", got, tc.wantContains)
				}
			}
		})
	}
}

func runAlloyDBListInstancesTest(t *testing.T, vars map[string]string) {
	type ListInstancesResponse struct {
		Instances []struct {
			Name string `json:"name"`
		} `json:"instances"`
	}

	type ToolResponse struct {
		Result string `json:"result"`
	}

	wantForSpecificClusterAndLocation := []string{
		fmt.Sprintf("projects/%s/locations/%s/clusters/%s/instances/%s", vars["projectId"], vars["locationId"], vars["clusterId"], vars["instanceId"]),
	}

	// NOTE: If clusters or instances are added, removed or changed in the test project,
	// the below lists must be updated for the tests to pass.
	wantForAllClustersSpecificLocation := []string{
		fmt.Sprintf("projects/%s/locations/%s/clusters/alloydb-ai-nl-testing/instances/alloydb-ai-nl-testing-instance", vars["projectId"], vars["locationId"]),
		fmt.Sprintf("projects/%s/locations/%s/clusters/alloydb-pg-testing/instances/alloydb-pg-testing-instance", vars["projectId"], vars["locationId"]),
	}

	wantForAllClustersAllLocations := []string{
		fmt.Sprintf("projects/%s/locations/us-central1/clusters/alloydb-ai-nl-testing/instances/alloydb-ai-nl-testing-instance", vars["projectId"]),
		fmt.Sprintf("projects/%s/locations/us-central1/clusters/alloydb-pg-testing/instances/alloydb-pg-testing-instance", vars["projectId"]),
		fmt.Sprintf("projects/%s/locations/us-east4/clusters/alloydb-private-pg-testing/instances/alloydb-private-pg-testing-instance", vars["projectId"]),
        fmt.Sprintf("projects/%s/locations/us-east4/clusters/colab-testing/instances/colab-testing-primary", vars["projectId"]),
	}

	invokeTcs := []struct {
		name           string
		requestBody    io.Reader
		want           []string
		wantStatusCode int
	}{
		{
			name:           "list instances for a specific cluster and location",
			requestBody:    bytes.NewBufferString(fmt.Sprintf(`{"projectId": "%s", "locationId": "%s", "clusterId": "%s"}`, vars["projectId"], vars["locationId"], vars["clusterId"])),
			want:           wantForSpecificClusterAndLocation,
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "list instances for all clusters and specific location",
			requestBody:    bytes.NewBufferString(fmt.Sprintf(`{"projectId": "%s", "locationId": "%s", "clusterId": "-"}`, vars["projectId"], vars["locationId"])),
			want:           wantForAllClustersSpecificLocation,
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "list instances for all clusters and all locations",
			requestBody:    bytes.NewBufferString(fmt.Sprintf(`{"projectId": "%s", "locationId": "-", "clusterId": "-"}`, vars["projectId"])),
			want:           wantForAllClustersAllLocations,
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "list instances missing project",
			requestBody:    bytes.NewBufferString(fmt.Sprintf(`{"locationId": "%s", "clusterId": "%s"}`, vars["locationId"], vars["clusterId"])),
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "list instances non-existent project",
			requestBody:    bytes.NewBufferString(fmt.Sprintf(`{"projectId": "non-existent-project", "locationId": "%s", "clusterId": "%s"}`, vars["locationId"], vars["clusterId"])),
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name:           "list instances non-existent location",
			requestBody:    bytes.NewBufferString(fmt.Sprintf(`{"projectId": "%s", "locationId": "non-existent-location", "clusterId": "%s"}`, vars["projectId"], vars["clusterId"])),
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name:           "list instances non-existent cluster",
			requestBody:    bytes.NewBufferString(fmt.Sprintf(`{"projectId": "%s", "locationId": "%s", "clusterId": "non-existent-cluster"}`, vars["projectId"], vars["locationId"])),
			wantStatusCode: http.StatusBadRequest,
		},
	}

	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			api := "http://127.0.0.1:5000/api/tool/alloydb-list-instances/invoke"
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

				var instancesData ListInstancesResponse
				if err := json.Unmarshal([]byte(body.Result), &instancesData); err != nil {
					t.Fatalf("error parsing nested result JSON: %v", err)
				}

				var got []string
				for _, instance := range instancesData.Instances {
					got = append(got, instance.Name)
				}

				sort.Strings(got)
				sort.Strings(tc.want)

				if !reflect.DeepEqual(got, tc.want) {
					t.Errorf("instance list mismatch:\n got: %v\nwant: %v", got, tc.want)
				}
			}
		})
	}
}

func runAlloyDBGetClusterTest(t *testing.T, vars map[string]string) {
	type ToolResponse struct {
		Result string `json:"result"`
	}

	invokeTcs := []struct {
		name           string
		requestBody    io.Reader
		want           map[string]any
		wantStatusCode int
	}{
		{
			name:        "get cluster success",
			requestBody: bytes.NewBufferString(fmt.Sprintf(`{"projectId": "%s", "locationId": "%s", "clusterId": "%s"}`, vars["projectId"], vars["locationId"], vars["clusterId"])),
			want: map[string]any{
				"clusterType": "PRIMARY",
				"name":        fmt.Sprintf("projects/%s/locations/%s/clusters/%s", vars["projectId"], vars["locationId"], vars["clusterId"]),
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "get cluster missing project",
			requestBody:    bytes.NewBufferString(fmt.Sprintf(`{"locationId": "%s", "clusterId": "%s"}`, vars["locationId"], vars["clusterId"])),
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "get cluster missing location",
			requestBody:    bytes.NewBufferString(fmt.Sprintf(`{"projectId": "%s", "clusterId": "%s"}`, vars["projectId"], vars["clusterId"])),
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "get cluster missing clusterId",
			requestBody:    bytes.NewBufferString(fmt.Sprintf(`{"projectId": "%s", "locationId": "%s"}`, vars["projectId"], vars["locationId"])),
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "get cluster non-existent cluster",
			requestBody:    bytes.NewBufferString(fmt.Sprintf(`{"projectId": "%s", "locationId": "%s", "clusterId": "non-existent-cluster"}`, vars["projectId"], vars["locationId"])),
			wantStatusCode: http.StatusBadRequest,
		},
	}

	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			api := "http://127.0.0.1:5000/api/tool/alloydb-get-cluster/invoke"
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
					t.Fatalf("error parsing response body: %v", err)
				}

				if tc.want != nil {
					var gotMap map[string]any
					if err := json.Unmarshal([]byte(body.Result), &gotMap); err != nil {
						t.Fatalf("failed to unmarshal JSON result into map: %v", err)
					}

					got := make(map[string]any)
					for key := range tc.want {
						if value, ok := gotMap[key]; ok {
							got[key] = value
						}
					}

					if diff := cmp.Diff(tc.want, got); diff != "" {
						t.Errorf("Unexpected result: got %#v, want: %#v", got, tc.want)
					}
				}
			}
		})
	}
}

func runAlloyDBGetUserTest(t *testing.T, vars map[string]string) {
	type ToolResponse struct {
		Result string `json:"result"`
	}

	invokeTcs := []struct {
		name           string
		requestBody    io.Reader
		want           map[string]any
		wantStatusCode int
	}{
		{
			name:        "get user success",
			requestBody: bytes.NewBufferString(fmt.Sprintf(`{"projectId": "%s", "locationId": "%s", "clusterId": "%s", "userId": "%s"}`, vars["projectId"], vars["locationId"], vars["clusterId"], vars["user"])),
			want: map[string]any{
				"name": fmt.Sprintf("projects/%s/locations/%s/clusters/%s/users/%s", vars["projectId"], vars["locationId"], vars["clusterId"], vars["user"]),
				"userType": "ALLOYDB_BUILT_IN",
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "get user missing project",
			requestBody:    bytes.NewBufferString(fmt.Sprintf(`{"locationId": "%s", "clusterId": "%s", "userId": "%s"}`, vars["locationId"], vars["clusterId"], vars["user"])),
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "get user missing location",
			requestBody:    bytes.NewBufferString(fmt.Sprintf(`{"projectId": "%s", "clusterId": "%s", "userId": "%s"}`, vars["projectId"], vars["clusterId"], vars["user"])),
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "get user missing cluster",
			requestBody:    bytes.NewBufferString(fmt.Sprintf(`{"projectId": "%s", "locationId": "%s", "userId": "%s"}`, vars["projectId"], vars["locationId"], vars["user"])),
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "get user missing user",
			requestBody:    bytes.NewBufferString(fmt.Sprintf(`{"projectId": "%s", "locationId": "%s", "clusterId": "%s"}`, vars["projectId"], vars["locationId"], vars["clusterId"])),
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "get non-existent user",
			requestBody:    bytes.NewBufferString(fmt.Sprintf(`{"projectId": "%s", "locationId": "%s", "clusterId": "%s", "userId": "non-existent-user"}`, vars["projectId"], vars["locationId"], vars["clusterId"])),
			wantStatusCode: http.StatusBadRequest,
		},
	}

	for _, tc := range invokeTcs {
		t.Run(tc.name, func(t *testing.T) {
			api := "http://127.0.0.1:5000/api/tool/alloydb-get-user/invoke"
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
					t.Fatalf("error parsing response body: %v", err)
				}

				if tc.want != nil {
					var gotMap map[string]any
					if err := json.Unmarshal([]byte(body.Result), &gotMap); err != nil {
						t.Fatalf("failed to unmarshal JSON result into map: %v", err)
					}

					got := make(map[string]any)
					for key := range tc.want {
						if value, ok := gotMap[key]; ok {
							got[key] = value
						}
					}

					if diff := cmp.Diff(tc.want, got); diff != "" {
						t.Errorf("Unexpected result: got %#v, want: %#v", got, tc.want)
					}
				}
			}
		})
	}
}
