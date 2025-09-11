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
	if AlloyDBUser == "" {
		t.Fatal("'ALLOYDB_USER' not set")
	}
	return map[string]string{
		"projectId":  AlloyDBProject,
		"locationId": AlloyDBLocation,
		"clusterId":  AlloyDBCluster,
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
		"tools": map[string]any{
			"alloydb-list-users": map[string]any{
				"kind":        "alloydb-list-users",
				"source":      "alloydb-admin-source",
				"description": "Lists all AlloyDB users within a specific cluster.",
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
	runAlloyDBListUsersTest(t, vars)
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
