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

package serverlessspark

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
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools/serverlessspark/serverlesssparklistbatches"
	"github.com/googleapis/genai-toolbox/tests"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/testing/protocmp"
)

var (
	serverlessSparkProject  = os.Getenv("SERVERLESS_SPARK_PROJECT")
	serverlessSparkLocation = os.Getenv("SERVERLESS_SPARK_LOCATION")
)

func getServerlessSparkVars(t *testing.T) map[string]any {
	switch "" {
	case serverlessSparkProject:
		t.Fatal("'SERVERLESS_SPARK_PROJECT' not set")
	case serverlessSparkLocation:
		t.Fatal("'SERVERLESS_SPARK_LOCATION' not set")
	}

	return map[string]any{
		"kind":     "serverless-spark",
		"project":  serverlessSparkProject,
		"location": serverlessSparkLocation,
	}
}

func TestServerlessSparkToolEndpoints(t *testing.T) {
	sourceConfig := getServerlessSparkVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	toolsFile := map[string]any{
		"sources": map[string]any{
			"my-spark": sourceConfig,
		},
		"authServices": map[string]any{
			"my-google-auth": map[string]any{
				"kind":     "google",
				"clientId": tests.ClientId,
			},
		},
		"tools": map[string]any{
			"list-batches": map[string]any{
				"kind":   "serverless-spark-list-batches",
				"source": "my-spark",
			},
			"list-batches-with-auth": map[string]any{
				"kind":         "serverless-spark-list-batches",
				"source":       "my-spark",
				"authRequired": []string{"my-google-auth"},
			},
			"get-batch": map[string]any{
				"kind":   "serverless-spark-get-batch",
				"source": "my-spark",
			},
			"get-batch-with-auth": map[string]any{
				"kind":         "serverless-spark-get-batch",
				"source":       "my-spark",
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

	endpoint := fmt.Sprintf("%s-dataproc.googleapis.com:443", serverlessSparkLocation)
	client, err := dataproc.NewBatchControllerClient(ctx, option.WithEndpoint(endpoint))
	if err != nil {
		t.Fatalf("failed to create dataproc client: %v", err)
	}
	defer client.Close()

	t.Run("list-batches", func(t *testing.T) {
		// list-batches is sensitive to state changes, so this test must run sequentially.
		t.Run("success", func(t *testing.T) {
			runListBatchesTest(t, client, ctx)
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
					toolName: "list-batches",
					request:  map[string]any{"pageSize": 0},
					wantCode: http.StatusBadRequest,
					wantMsg:  "pageSize must be positive: 0",
				},
				{
					name:     "negative page size",
					toolName: "list-batches",
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
			runAuthTest(t, "list-batches-with-auth", map[string]any{"pageSize": 1}, http.StatusOK)
		})
	})

	// The following tool tests are independent and can run in parallel with each other.
	t.Run("parallel-tool-tests", func(t *testing.T) {
		t.Run("get-batch", func(t *testing.T) {
			t.Parallel()
			fullName := listBatchesRpc(t, client, ctx, "", 1, true)[0].Name
			t.Run("success", func(t *testing.T) {
				t.Parallel()
				runGetBatchTest(t, client, ctx, fullName)
			})
			t.Run("errors", func(t *testing.T) {
				t.Parallel()
				missingBatchFullName := fmt.Sprintf("projects/%s/locations/%s/batches/INVALID_BATCH", serverlessSparkProject, serverlessSparkLocation)
				tcs := []struct {
					name     string
					toolName string
					request  map[string]any
					wantCode int
					wantMsg  string
				}{
					{
						name:     "missing batch",
						toolName: "get-batch",
						request:  map[string]any{"name": "INVALID_BATCH"},
						wantCode: http.StatusBadRequest,
						wantMsg:  fmt.Sprintf("Not found: Batch projects/%s/locations/%s/batches/INVALID_BATCH", serverlessSparkProject, serverlessSparkLocation),
					},
					{
						name:     "full batch name",
						toolName: "get-batch",
						request:  map[string]any{"name": missingBatchFullName},
						wantCode: http.StatusBadRequest,
						wantMsg:  fmt.Sprintf("name must be a short batch name without '/': %s", missingBatchFullName),
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
				runAuthTest(t, "get-batch-with-auth", map[string]any{"name": shortName(fullName)}, http.StatusOK)
			})
		})
	})
}

// runListBatchesTest invokes the running list-batches tool and ensures it returns the correct
// number of results. It can run successfully against any GCP project that contains at least 2 total
// Serverless Spark batches.
func runListBatchesTest(t *testing.T, client *dataproc.BatchControllerClient, ctx context.Context) {
	batch2 := listBatchesRpc(t, client, ctx, "", 2, true)
	batch20 := listBatchesRpc(t, client, ctx, "", 20, false)

	tcs := []struct {
		name     string
		filter   string
		pageSize int
		numPages int
		want     []serverlesssparklistbatches.Batch
	}{
		{name: "one page", pageSize: 2, numPages: 1, want: batch2},
		{name: "two pages", pageSize: 1, numPages: 2, want: batch2},
		{name: "20 batches", pageSize: 20, numPages: 1, want: batch20},
		{name: "omit page size", numPages: 1, want: batch20},
		{
			name:     "filtered",
			filter:   "state = SUCCEEDED",
			pageSize: 2,
			numPages: 1,
			want:     listBatchesRpc(t, client, ctx, "state = SUCCEEDED", 2, true),
		},
		{
			name:     "empty",
			filter:   "state = SUCCEEDED AND state = FAILED",
			pageSize: 1,
			numPages: 1,
			want:     nil,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var actual []serverlesssparklistbatches.Batch
			var pageToken string
			for i := 0; i < tc.numPages; i++ {
				request := map[string]any{
					"filter":    tc.filter,
					"pageToken": pageToken,
				}
				if tc.pageSize > 0 {
					request["pageSize"] = tc.pageSize
				}

				resp, err := invokeTool("list-batches", request, nil)
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

				var listResponse serverlesssparklistbatches.ListBatchesResponse
				if err := json.Unmarshal([]byte(result), &listResponse); err != nil {
					t.Fatalf("error unmarshalling result: %s", err)
				}
				actual = append(actual, listResponse.Batches...)
				pageToken = listResponse.NextPageToken
			}

			if !reflect.DeepEqual(actual, tc.want) {
				t.Fatalf("unexpected batches: got %+v, want %+v", actual, tc.want)
			}
		})
	}
}

func listBatchesRpc(t *testing.T, client *dataproc.BatchControllerClient, ctx context.Context, filter string, n int, exact bool) []serverlesssparklistbatches.Batch {
	parent := fmt.Sprintf("projects/%s/locations/%s", serverlessSparkProject, serverlessSparkLocation)
	req := &dataprocpb.ListBatchesRequest{
		Parent:   parent,
		PageSize: 2,
		OrderBy:  "create_time desc",
	}
	if filter != "" {
		req.Filter = filter
	}

	it := client.ListBatches(ctx, req)
	pager := iterator.NewPager(it, n, "")
	var batchPbs []*dataprocpb.Batch
	_, err := pager.NextPage(&batchPbs)
	if err != nil {
		t.Fatalf("failed to list batches: %s", err)
	}
	if exact && len(batchPbs) != n {
		t.Fatalf("expected exactly %d batches, got %d", n, len(batchPbs))
	}
	if !exact && (len(batchPbs) == 0 || len(batchPbs) > n) {
		t.Fatalf("expected between 1 and %d batches, got %d", n, len(batchPbs))
	}

	return serverlesssparklistbatches.ToBatches(batchPbs)
}

func runAuthTest(t *testing.T, toolName string, request map[string]any, wantStatus int) {
	idToken, err := tests.GetGoogleIdToken(tests.ClientId)
	if err != nil {
		t.Fatalf("error getting Google ID token: %s", err)
	}
	tcs := []struct {
		name       string
		headers    map[string]string
		wantStatus int
	}{
		{
			name:       "valid auth token",
			headers:    map[string]string{"my-google-auth_token": idToken},
			wantStatus: wantStatus,
		},
		{
			name:       "invalid auth token",
			headers:    map[string]string{"my-google-auth_token": "INVALID_TOKEN"},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "no auth token",
			headers:    nil,
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			resp, err := invokeTool(toolName, request, tc.headers)
			if err != nil {
				t.Fatalf("invokeTool failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.wantStatus {
				bodyBytes, _ := io.ReadAll(resp.Body)
				t.Fatalf("response status code is not %d, got %d: %s", tc.wantStatus, resp.StatusCode, string(bodyBytes))
			}
		})
	}
}

func runGetBatchTest(t *testing.T, client *dataproc.BatchControllerClient, ctx context.Context, fullName string) {
	// First get the batch details directly from the Go proto API.
	req := &dataprocpb.GetBatchRequest{
		Name: fullName,
	}
	rawWantBatchPb, err := client.GetBatch(ctx, req)
	if err != nil {
		t.Fatalf("failed to get batch: %s", err)
	}

	// Trim unknown fields from the proto by marshalling and unmarshalling.
	jsonBytes, err := protojson.Marshal(rawWantBatchPb)
	if err != nil {
		t.Fatalf("failed to marshal batch to JSON: %s", err)
	}
	var wantBatchPb dataprocpb.Batch
	if err := protojson.Unmarshal(jsonBytes, &wantBatchPb); err != nil {
		t.Fatalf("error unmarshalling result: %s", err)
	}

	tcs := []struct {
		name      string
		batchName string
		want      *dataprocpb.Batch
	}{
		{
			name:      "found batch",
			batchName: shortName(fullName),
			want:      &wantBatchPb,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			request := map[string]any{"name": tc.batchName}
			resp, err := invokeTool("get-batch", request, nil)
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

			// Unmarshal JSON to proto for proto-aware deep comparison.
			var batch dataprocpb.Batch
			if err := protojson.Unmarshal([]byte(result), &batch); err != nil {
				t.Fatalf("error unmarshalling result: %s", err)
			}

			if !cmp.Equal(&batch, tc.want, protocmp.Transform()) {
				diff := cmp.Diff(&batch, tc.want, protocmp.Transform())
				t.Errorf("GetBatch() returned diff (-got +want):\n%s", diff)
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

func shortName(fullName string) string {
	parts := strings.Split(fullName, "/")
	return parts[len(parts)-1]
}
