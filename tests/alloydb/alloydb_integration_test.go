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
	"testing"
	"time"

	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"
)

var (
	AlloyDBProject  = os.Getenv("ALLOYDB_PROJECT")
	AlloyDBLocation = os.Getenv("ALLOYDB_REGION")
	AlloyDBCluster  = os.Getenv("ALLOYDB_CLUSTER")
	AlloyDBInstance = os.Getenv("ALLOYDB_INSTANCE")
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
	return map[string]string{
		"projectId":  AlloyDBProject,
		"locationId": AlloyDBLocation,
		"clusterId":  AlloyDBCluster,
		"instanceId": AlloyDBInstance,
	}
}

func getAlloyDBToolsConfig() map[string]any {
	return map[string]any{
		"sources": map[string]any{
			"alloydb-admin-source": map[string]any{
				"kind":    "http",
				"baseUrl": "https://alloydb.googleapis.com",
			},
		},
		"tools" : map[string]any{
			// AlloyDB specific tools
			"alloydb-list-instances": map[string]any{
				"kind":        "alloydb-list-instances",
				"source":      "alloydb-admin-source",
				"description": "Lists all AlloyDB instances within a specific cluster.",
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
	runAlloyDBListInstancesTest(t, vars)
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
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name:           "list instances non-existent location",
			requestBody:    bytes.NewBufferString(fmt.Sprintf(`{"projectId": "%s", "locationId": "non-existent-location", "clusterId": "%s"}`, vars["projectId"], vars["clusterId"])),
			wantStatusCode: http.StatusBadRequest,
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
