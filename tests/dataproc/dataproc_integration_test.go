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
	"github.com/google/go-cmp/cmp"
	dataprocsrc "github.com/googleapis/genai-toolbox/internal/sources/dataproc"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/testing/protocmp"
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
	jobURLPrefix     = "https://console.cloud.google.com/dataproc/jobs/"
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
			"get-cluster": map[string]any{
				"type":   "dataproc-get-cluster",
				"source": "my-dataproc",
			},
			"get-cluster-with-auth": map[string]any{
				"type":         "dataproc-get-cluster",
				"source":       "my-dataproc",
				"authRequired": []string{"my-google-auth"},
			},
			"get-job": map[string]any{
				"type":   "dataproc-get-job",
				"source": "my-dataproc",
			},
			"get-job-with-auth": map[string]any{
				"type":         "dataproc-get-job",
				"source":       "my-dataproc",
				"authRequired": []string{"my-google-auth"},
			},
			"list-clusters": map[string]any{
				"type":   "dataproc-list-clusters",
				"source": "my-dataproc",
			},
			"list-clusters-with-auth": map[string]any{
				"type":         "dataproc-list-clusters",
				"source":       "my-dataproc",
				"authRequired": []string{"my-google-auth"},
			},
			"list-jobs": map[string]any{
				"type":   "dataproc-list-jobs",
				"source": "my-dataproc",
			},
			"list-jobs-with-auth": map[string]any{
				"type":         "dataproc-list-jobs",
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

	jobClient, err := dataproc.NewJobControllerClient(ctx, option.WithEndpoint(endpoint))
	if err != nil {
		t.Fatalf("failed to create dataproc client: %v", err)
	}
	defer jobClient.Close()

	t.Run("get-cluster", func(t *testing.T) {
		clusterName := listClustersRpc(t, clusterClient, ctx, "", 1)[0].Name
		t.Run("success", func(t *testing.T) {
			t.Parallel()
			runGetClusterTest(t, clusterClient, ctx, clusterName)
		})
		t.Run("errors", func(t *testing.T) {
			t.Parallel()
			missingClusterFullName := fmt.Sprintf("projects/%s/regions/%s/clusters/INVALID_CLUSTER", dataprocProject, dataprocRegion)
			tcs := []struct {
				name     string
				toolName string
				request  map[string]any
				wantCode int
				wantMsg  string
			}{
				{
					name:     "missing cluster",
					toolName: "get-cluster",
					request:  map[string]any{"clusterName": "INVALID_CLUSTER"},
					wantCode: http.StatusOK,
					wantMsg:  fmt.Sprintf("Not found: Cluster projects/%s/regions/%s/clusters/INVALID_CLUSTER", dataprocProject, dataprocRegion),
				},
				{
					name:     "full cluster name",
					toolName: "get-cluster",
					request:  map[string]any{"clusterName": missingClusterFullName},
					wantCode: http.StatusOK,
					wantMsg:  fmt.Sprintf("clusterName must be a short name without '/': %s", missingClusterFullName),
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
			runAuthTest(t, "get-cluster-with-auth", map[string]any{"clusterName": shortName(clusterName)}, http.StatusOK)
		})
	})

	t.Run("get-job", func(t *testing.T) {
		jobId := listJobsRpc(t, jobClient, ctx, "", 1)[0].ID
		t.Run("success", func(t *testing.T) {
			t.Parallel()
			runGetJobTest(t, jobClient, ctx, jobId)
		})
		t.Run("errors", func(t *testing.T) {
			t.Parallel()
			missingJobFullName := fmt.Sprintf("projects/%s/regions/%s/jobs/INVALID_JOB", dataprocProject, dataprocRegion)
			tcs := []struct {
				name     string
				toolName string
				request  map[string]any
				wantCode int
				wantMsg  string
			}{
				{
					name:     "missing job",
					toolName: "get-job",
					request:  map[string]any{"jobId": "INVALID_JOB"},
					wantCode: http.StatusOK,
					wantMsg:  fmt.Sprintf("Not found: Job projects/%s/regions/%s/jobs/INVALID_JOB", dataprocProject, dataprocRegion),
				},
				{
					name:     "full job name",
					toolName: "get-job",
					request:  map[string]any{"jobId": missingJobFullName},
					wantCode: http.StatusOK,
					wantMsg:  fmt.Sprintf("jobId must be a short name without '/': %s", missingJobFullName),
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
			runAuthTest(t, "get-job-with-auth", map[string]any{"jobId": jobId}, http.StatusOK)
		})
	})
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
					wantCode: http.StatusOK,
					wantMsg:  "pageSize must be positive: 0",
				},
				{
					name:     "negative page size",
					toolName: "list-clusters",
					request:  map[string]any{"pageSize": -1},
					wantCode: http.StatusOK,
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

	t.Run("list-jobs", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			runListJobsTest(t, jobClient, ctx)
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
					toolName: "list-jobs",
					request:  map[string]any{"pageSize": 0},
					wantCode: http.StatusOK,
					wantMsg:  "pageSize must be positive: 0",
				},
				{
					name:     "negative page size",
					toolName: "list-jobs",
					request:  map[string]any{"pageSize": -1},
					wantCode: http.StatusOK,
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
			runAuthTest(t, "list-jobs-with-auth", map[string]any{
				"pageSize": 1,
				"filter":   "clusterName = " + dataprocListJobsCluster,
			}, http.StatusOK)
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

			var want []dataprocsrc.Cluster
			if tc.wantN > 0 {
				want = listClustersRpc(t, client, ctx, tc.filter, tc.wantN)
			}

			var actual []dataprocsrc.Cluster
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

				var listResponse dataprocsrc.ListClustersResponse
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

func runGetClusterTest(t *testing.T, client *dataproc.ClusterControllerClient, ctx context.Context, fullName string) {
	// First get the cluster details directly from the Go proto API.
	req := &dataprocpb.GetClusterRequest{
		ProjectId:   dataprocProject,
		Region:      dataprocRegion,
		ClusterName: fullName[strings.LastIndex(fullName, "/")+1:],
	}
	rawWantClusterPb, err := client.GetCluster(ctx, req)
	if err != nil {
		t.Fatalf("failed to get cluster: %s", err)
	}

	// Trim unknown fields from the proto by marshalling and unmarshalling.
	jsonBytes, err := protojson.Marshal(rawWantClusterPb)
	if err != nil {
		t.Fatalf("failed to marshal cluster to JSON: %s", err)
	}
	var wantClusterPb dataprocpb.Cluster
	if err := protojson.Unmarshal(jsonBytes, &wantClusterPb); err != nil {
		t.Fatalf("error unmarshalling result: %s", err)
	}

	shortName := fullName[strings.LastIndex(fullName, "/")+1:]

	tcs := []struct {
		name        string
		clusterName string
		want        *dataprocpb.Cluster
	}{
		{
			name:        "found cluster",
			clusterName: shortName,
			want:        &wantClusterPb,
		},
	}

	t.Run("success", func(t *testing.T) {
		for _, tc := range tcs {
			t.Run(tc.name, func(t *testing.T) {
				request := map[string]any{"clusterName": tc.clusterName}
				resp, err := invokeTool("get-cluster", request, nil)
				if err != nil {
					t.Fatalf("invokeTool failed: %v", err)
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					body, _ := io.ReadAll(resp.Body)
					t.Fatalf("status code got %d, want 200. Body: %s", resp.StatusCode, body)
				}

				var body map[string]any
				if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
					t.Fatalf("error parsing response body: %v", err)
				}
				resultStr, ok := body["result"].(string)
				if !ok {
					t.Fatalf("result is not a string, got %T", body["result"])
				}
				var wrappedResult map[string]any
				if err := json.Unmarshal([]byte(resultStr), &wrappedResult); err != nil {
					t.Fatalf("error unmarshalling result: %s", err)
				}

				consoleURL, ok := wrappedResult["consoleUrl"].(string)
				if !ok || !strings.HasPrefix(consoleURL, clusterURLPrefix) {
					t.Errorf("unexpected consoleUrl: %v", consoleURL)
				}
				logsURL, ok := wrappedResult["logsUrl"].(string)
				if !ok || !strings.HasPrefix(logsURL, logsURLPrefix) {
					t.Errorf("unexpected logsUrl: %v", logsURL)
				}

				clusterJSON, err := json.Marshal(wrappedResult["cluster"])
				if err != nil {
					t.Fatalf("failed to marshal cluster: %v", err)
				}

				// Unmarshal JSON to proto for proto-aware deep comparison.
				var cluster dataprocpb.Cluster
				if err := protojson.Unmarshal(clusterJSON, &cluster); err != nil {
					t.Fatalf("error unmarshalling cluster from wrapped result: %s", err)
				}

				if !cmp.Equal(&cluster, tc.want, protocmp.Transform()) {
					diff := cmp.Diff(&cluster, tc.want, protocmp.Transform())
					t.Errorf("GetCluster() returned diff (-got +want):\n%s", diff)
				}
			})
		}
	})

	t.Run("errors", func(t *testing.T) {
		tcs := []struct {
			name     string
			request  map[string]any
			wantCode int
			wantMsg  string
		}{
			{
				name:     "missing clusterName",
				request:  map[string]any{},
				wantCode: http.StatusOK,
				wantMsg:  "missing required parameter: clusterName",
			},
			{
				name:     "invalid name with slash",
				request:  map[string]any{"clusterName": "projects/foo/regions/bar/clusters/baz"}, // Full name requires matching project/region
				wantCode: http.StatusOK,
				wantMsg:  "clusterName must be a short name without '/'",
			},
		}
		for _, tc := range tcs {
			t.Run(tc.name, func(t *testing.T) {
				testError(t, "get-cluster", tc.request, tc.wantCode, tc.wantMsg)
			})
		}
	})
}

func listClustersRpc(t *testing.T, client *dataproc.ClusterControllerClient, ctx context.Context, filter string, n int) []dataprocsrc.Cluster {
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

	clusters, err := dataprocsrc.ToClusters(clusterPbs, dataprocRegion)
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

func runListJobsTest(t *testing.T, client *dataproc.JobControllerClient, ctx context.Context) {
	tcs := []struct {
		name     string
		filter   string
		pageSize int
		numPages int
		wantN    int
	}{
		{name: "one page", pageSize: 2, numPages: 1, wantN: 2},
		{name: "two pages", pageSize: 1, numPages: 2, wantN: 2},
		{name: "10 batches", pageSize: 10, numPages: 1, wantN: 10},
		{name: "omit page size", numPages: 1, wantN: 20},
		{
			name:     "filtered",
			filter:   "status.state = NON_ACTIVE",
			pageSize: 20,
			numPages: 1,
			wantN:    20,
		},
		{
			name:     "empty",
			filter:   "status.state = NON_ACTIVE AND status.state = ACTIVE",
			pageSize: 1,
			numPages: 1,
			wantN:    0,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var want []dataprocsrc.Job
			if tc.wantN > 0 {
				want = listJobsRpc(t, client, ctx, tc.filter, tc.wantN)
			}

			var actual []dataprocsrc.Job
			var pageToken string
			for i := 0; i < tc.numPages; i++ {
				filter := tc.filter
				if filter != "" {
					filter += " AND "
				}
				filter += "clusterName = " + dataprocListJobsCluster
				request := map[string]any{
					"filter":    filter,
					"pageToken": pageToken,
				}
				if tc.pageSize > 0 {
					request["pageSize"] = tc.pageSize
				}

				resp, err := invokeTool("list-jobs", request, nil)
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

				var listResponse dataprocsrc.ListJobsResponse
				if err := json.Unmarshal([]byte(result), &listResponse); err != nil {
					t.Fatalf("error unmarshalling result: %s", err)
				}
				actual = append(actual, listResponse.Jobs...)
				pageToken = listResponse.NextPageToken
			}

			if !reflect.DeepEqual(actual, want) {
				t.Fatalf("unexpected jobs: got %+v, want %+v", actual, want)
			}

			// want has URLs because it's created from Job instances by the same utility function
			// used by the tool internals. Double-check that the URLs are reasonable.
			for _, job := range want {
				if !strings.HasPrefix(job.ConsoleURL, jobURLPrefix) {
					t.Errorf("unexpected consoleUrl in job: %#v", job)
				}
				if !strings.HasPrefix(job.LogsURL, logsURLPrefix) {
					t.Errorf("unexpected logsUrl in job: %#v", job)
				}
			}
		})
	}
}

func runGetJobTest(t *testing.T, client *dataproc.JobControllerClient, ctx context.Context, jobId string) {
	// First get the job details directly from the Go proto API.
	req := &dataprocpb.GetJobRequest{
		ProjectId: dataprocProject,
		Region:    dataprocRegion,
		JobId:     jobId,
	}
	rawWantJobPb, err := client.GetJob(ctx, req)
	if err != nil {
		t.Fatalf("failed to get job: %s", err)
	}

	// Trim unknown fields from the proto by marshalling and unmarshalling.
	jsonBytes, err := protojson.Marshal(rawWantJobPb)
	if err != nil {
		t.Fatalf("failed to marshal job to JSON: %s", err)
	}
	var wantJobPb dataprocpb.Job
	if err := protojson.Unmarshal(jsonBytes, &wantJobPb); err != nil {
		t.Fatalf("error unmarshalling result: %s", err)
	}

	tcs := []struct {
		name  string
		jobId string
		want  *dataprocpb.Job
	}{
		{
			name:  "found job",
			jobId: jobId,
			want:  &wantJobPb,
		},
	}

	t.Run("success", func(t *testing.T) {
		for _, tc := range tcs {
			t.Run(tc.name, func(t *testing.T) {
				request := map[string]any{"jobId": tc.jobId}
				resp, err := invokeTool("get-job", request, nil)
				if err != nil {
					t.Fatalf("invokeTool failed: %v", err)
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					body, _ := io.ReadAll(resp.Body)
					t.Fatalf("status code got %d, want 200. Body: %s", resp.StatusCode, body)
				}

				var body map[string]any
				if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
					t.Fatalf("error parsing response body: %v", err)
				}
				resultStr, ok := body["result"].(string)
				if !ok {
					t.Fatalf("result is not a string, got %T", body["result"])
				}
				var wrappedResult map[string]any
				if err := json.Unmarshal([]byte(resultStr), &wrappedResult); err != nil {
					t.Fatalf("error unmarshalling result: %s", err)
				}

				consoleURL, ok := wrappedResult["consoleUrl"].(string)
				if !ok || !strings.HasPrefix(consoleURL, jobURLPrefix) {
					t.Errorf("unexpected consoleUrl: %v", consoleURL)
				}
				logsURL, ok := wrappedResult["logsUrl"].(string)
				if !ok || !strings.HasPrefix(logsURL, logsURLPrefix) {
					t.Errorf("unexpected logsUrl: %v", logsURL)
				}

				jobJSON, err := json.Marshal(wrappedResult["job"])
				if err != nil {
					t.Fatalf("failed to marshal job: %v", err)
				}

				// Unmarshal JSON to proto for proto-aware deep comparison.
				var job dataprocpb.Job
				if err := protojson.Unmarshal(jobJSON, &job); err != nil {
					t.Fatalf("error unmarshalling job from wrapped result: %s", err)
				}

				if !cmp.Equal(&job, tc.want, protocmp.Transform()) {
					diff := cmp.Diff(&job, tc.want, protocmp.Transform())
					t.Errorf("GetJob() returned diff (-got +want):\n%s", diff)
				}
			})
		}
	})

	t.Run("errors", func(t *testing.T) {
		tcs := []struct {
			name     string
			request  map[string]any
			wantCode int
			wantMsg  string
		}{
			{
				name:     "missing jobId",
				request:  map[string]any{},
				wantCode: http.StatusOK,
				wantMsg:  "missing required parameter: jobId",
			},
			{
				name:     "invalid name with slash",
				request:  map[string]any{"jobId": "projects/foo/regions/bar/jobs/baz"},
				wantCode: http.StatusOK,
				wantMsg:  "jobId must be a short name without '/'",
			},
		}
		for _, tc := range tcs {
			t.Run(tc.name, func(t *testing.T) {
				testError(t, "get-job", tc.request, tc.wantCode, tc.wantMsg)
			})
		}
	})
}

func listJobsRpc(t *testing.T, client *dataproc.JobControllerClient, ctx context.Context, filter string, n int) []dataprocsrc.Job {
	req := &dataprocpb.ListJobsRequest{
		ProjectId:   dataprocProject,
		Region:      dataprocRegion,
		ClusterName: dataprocListJobsCluster,
		PageSize:    int32(n),
	}
	if filter != "" {
		req.Filter = filter
	}

	it := client.ListJobs(ctx, req)
	pager := iterator.NewPager(it, n, "")
	var jobPbs []*dataprocpb.Job
	_, err := pager.NextPage(&jobPbs)
	if err != nil {
		t.Fatalf("failed to list jobs: %s", err)
	}

	jobs, err := dataprocsrc.ToJobs(jobPbs, dataprocRegion)
	if err != nil {
		t.Fatalf("failed to convert jobs: %v", err)
	}
	return jobs
}

func shortName(fullName string) string {
	parts := strings.Split(fullName, "/")
	return parts[len(parts)-1]
}
