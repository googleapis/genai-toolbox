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
	"regexp"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
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
			// AlloyDB specific tools
			"alloydb-get-cluster": map[string]any{
				"kind":        "alloydb-get-cluster",
				"source":      "alloydb-admin-source",
				"description": "Retrieves details of a specific AlloyDB cluster.",
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

	// Run tool-specific invoke tests
	runAlloyDBGetClusterTest(t, vars)
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