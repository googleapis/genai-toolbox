// Copyright 2026 Google LLC
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

package dataproc

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
	"strings"
	"testing"
	"time"

	dataproc "cloud.google.com/go/dataproc/v2/apiv1"
	"cloud.google.com/go/dataproc/v2/apiv1/dataprocpb"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools/dataproc/dataproclistclusters"
	"github.com/googleapis/genai-toolbox/tests"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

var (
	dataprocRegion  = os.Getenv("DATAPROC_REGION")
	dataprocProject = os.Getenv("DATAPROC_PROJECT")

	// dataprocListJobsCluster is the name of a cluster in the project that has jobs.
	//
	// This is necessary to work around a performance issue in the Dataproc API where listing all
	// jobs in a project is very slow.
	dataprocListJobsCluster = os.Getenv("DATAPROC_LIST_JOBS_CLUSTER")
)

const (
	clusterURLPrefix = "https://console.cloud.google.com/dataproc/clusters/"
	logsURLPrefix    = "https://console.cloud.google.com/logs/viewer?"
)

func getDataprocVars(t *testing.T) map[string]any {
	switch "" {
	case dataprocRegion:
		t.Fatal("'DATAPROC_REGION' not set")
	case dataprocProject:
		t.Fatal("'DATAPROC_PROJECT' not set")
	case dataprocListJobsCluster:
		t.Fatal("'DATAPROC_LIST_JOBS_CLUSTER' not set")
	}

	return map[string]any{
		"type":    "dataproc",
		"project": dataprocProject,
		"region":  dataprocRegion,
	}
}

func TestDataprocClustersToolEndpoints(t *testing.T) {
	sourceConfig := getDataprocVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute) // Clusters take time
	defer cancel()

	toolsFile := map[string]any{
		"sources": map[string]any{
			"my-dataproc": sourceConfig,
		},
		"authServices": map[string]any{
			"my-google-auth": map[string]any{
				"type":     "google",
				"clientId": tests.ClientId,
			},
		},
		"tools": map[string]any{
			"list-clusters": map[string]any{
				"type":   "dataproc-list-clusters",
				"source": "my-dataproc",
			},
			"list-clusters-with-auth": map[string]any{
				"type":         "dataproc-list-clusters",
				"source":       "my-dataproc",
				"authRequired": []string{"my-google-auth"},
			},
		},
	}

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

	endpoint := fmt.Sprintf("%s-dataproc.googleapis.com:443", dataprocRegion)
	clusterClient, err := dataproc.NewClusterControllerClient(ctx, option.WithEndpoint(endpoint))
	if err != nil {
		t.Fatalf("failed to create dataproc client: %v", err)
	}
	defer clusterClient.Close()

	t.Run("list-clusters", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			runListClustersTest(t, clusterClient, ctx)
		})
		t.Run("errors", func(t *testing.T) {
			t.Parallel()
			tcs := []struct {
				name     string
				toolName string
				request  map[string]any
				wantCode int
				wantMsg  string
			}{
				{
					name:     "zero page size",
					toolName: "list-clusters",
					request:  map[string]any{"pageSize": 0},
					wantCode: http.StatusBadRequest,
					wantMsg:  "pageSize must be positive: 0",
				},
				{
					name:     "negative page size",
					toolName: "list-clusters",
					request:  map[string]any{"pageSize": -1},
					wantCode: http.StatusBadRequest,
					wantMsg:  "pageSize must be positive: -1",
				},
			}
			for _, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {
					t.Parallel()
					testError(t, tc.toolName, tc.request, tc.wantCode, tc.wantMsg)
				})
			}
		})
		t.Run("auth", func(t *testing.T) {
			t.Parallel()
			runAuthTest(t, "list-clusters-with-auth", map[string]any{"pageSize": 1}, http.StatusOK)
		})
	})
}

func invokeTool(toolName string, request map[string]any, headers map[string]string) (*http.Response, error) {
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("http://127.0.0.1:5000/api/tool/%s/invoke", toolName)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(requestBytes))
	if err != nil {
		return nil, fmt.Errorf("unable to create request: %w", err)
	}
	req.Header.Add("Content-type", "application/json")
	for k, v := range headers {
		req.Header.Add(k, v)
	}

	return http.DefaultClient.Do(req)
}

func runListClustersTest(t *testing.T, client *dataproc.ClusterControllerClient, ctx context.Context) {
	tcs := []struct {
		name     string
		filter   string
		pageSize int
		numPages int
		wantN    int
	}{
		{name: "one page", pageSize: 2, numPages: 1, wantN: 2},
		{name: "two pages", pageSize: 1, numPages: 2, wantN: 2},
		{name: "5 clusters", pageSize: 5, numPages: 1, wantN: 5},
		{name: "omit page size", numPages: 1, wantN: 20},
		{
			name:     "filtered",
			filter:   "status.state = STOPPED",
			pageSize: 2,
			numPages: 1,
			wantN:    2,
		},
		{
			name:     "empty",
			filter:   "status.state = STOPPED AND status.state = RUNNING",
			pageSize: 1,
			numPages: 1,
			wantN:    0,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var want []dataproclistclusters.Cluster
			if tc.wantN > 0 {
				want = listClustersRpc(t, client, ctx, tc.filter, tc.wantN)
			}

			var actual []dataproclistclusters.Cluster
			var pageToken string

			for i := 0; i < tc.numPages; i++ {
				request := map[string]any{
					"filter":    tc.filter,
					"pageToken": pageToken,
				}
				if tc.pageSize > 0 {
					request["pageSize"] = tc.pageSize
				}

				resp, err := invokeTool("list-clusters", request, nil)
				if err != nil {
					t.Fatalf("invokeTool failed: %v", err)
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					bodyBytes, _ := io.ReadAll(resp.Body)
					t.Fatalf("response status code is not 200, got %d: %s", resp.StatusCode, string(bodyBytes))
				}

				var body map[string]any
				if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
					t.Fatalf("error parsing response body: %v", err)
				}

				result, ok := body["result"].(string)
				if !ok {
					t.Fatalf("unable to find result in response body")
				}

				var listResponse dataproclistclusters.ListClustersResponse
				if err := json.Unmarshal([]byte(result), &listResponse); err != nil {
					t.Fatalf("error unmarshalling result: %s", err)
				}
				actual = append(actual, listResponse.Clusters...)
				pageToken = listResponse.NextPageToken
			}

			if !reflect.DeepEqual(actual, want) {
				t.Fatalf("unexpected clusters: got %+v, want %+v", actual, want)
			}

			// want has URLs because it's created from Batch instances by the same utility function
			// used by the tool internals. Double-check that the URLs are reasonable.
			for _, cluster := range want {
				if !strings.HasPrefix(cluster.ConsoleURL, clusterURLPrefix) {
					t.Errorf("unexpected consoleUrl in cluster: %#v", cluster)
				}
				if !strings.HasPrefix(cluster.LogsURL, logsURLPrefix) {
					t.Errorf("unexpected logsUrl in cluster: %#v", cluster)
				}
			}
		})
	}
}

func listClustersRpc(t *testing.T, client *dataproc.ClusterControllerClient, ctx context.Context, filter string, n int) []dataproclistclusters.Cluster {
	req := &dataprocpb.ListClustersRequest{
		ProjectId: dataprocProject,
		Region:    dataprocRegion,
		PageSize:  int32(n),
	}
	if filter != "" {
		req.Filter = filter
	}

	it := client.ListClusters(ctx, req)
	pager := iterator.NewPager(it, n, "")
	var clusterPbs []*dataprocpb.Cluster
	_, err := pager.NextPage(&clusterPbs)
	if err != nil {
		t.Fatalf("failed to list clusters: %s", err)
	}

	clusters, err := dataproclistclusters.ToClusters(clusterPbs, dataprocRegion)
	if err != nil {
		t.Fatalf("failed to convert clusters to JSON: %v", err)
	}

	return clusters
}

func runAuthTest(t *testing.T, toolName string, request map[string]any, wantStatus int) {
	idToken, err := tests.GetGoogleIdToken(tests.ClientId)
	if err != nil {
		t.Fatalf("error getting Google ID token: %s", err)
	}

	tcs := []struct {
		name     string
		headers  map[string]string
		wantCode int
	}{
		{
			name:     "valid token",
			headers:  map[string]string{"my-google-auth_token": idToken},
			wantCode: wantStatus,
		},
		{
			name:     "invalid token",
			headers:  map[string]string{"my-google-auth_token": "INVALID"},
			wantCode: http.StatusUnauthorized,
		},
		{
			name:     "missing header",
			headers:  nil,
			wantCode: http.StatusUnauthorized,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := invokeTool(toolName, request, tc.headers)
			if err != nil {
				t.Fatalf("invokeTool failed: %s", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != tc.wantCode {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("status code got %d, want %d. Body: %s", resp.StatusCode, tc.wantCode, body)
			}
		})
	}
}

func testError(t *testing.T, toolName string, request map[string]any, wantCode int, wantMsg string) {
	resp, err := invokeTool(toolName, request, nil)
	if err != nil {
		t.Fatalf("invokeTool failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != wantCode {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("response status code is not %d, got %d: %s", wantCode, resp.StatusCode, string(bodyBytes))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	if !bytes.Contains(bodyBytes, []byte(wantMsg)) {
		t.Fatalf("response body does not contain %q: %s", wantMsg, string(bodyBytes))
	}
}
