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
	"io"
	"net/http"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/tests"
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
		"tools": map[string]any{
			"list-batches": map[string]any{
				"kind":   "serverless-spark-list-batches",
				"source": "my-spark",
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

	runListBatchesTest(t)
}

// runListBatchesTest invokes the running list-batches tool and ensures it returns the correct
// number of results. It can run successfully against any GCP project that has at least 2 succeeded
// or failed Serverless Spark batches, of any age.
func runListBatchesTest(t *testing.T) {
	requestBody := bytes.NewBuffer([]byte(`{"pageSize": 2, "filter": "state = SUCCEEDED OR state = FAILED"}`))
	req, err := http.NewRequest(http.MethodPost, "http://127.0.0.1:5000/api/tool/list-batches/invoke", requestBody)
	if err != nil {
		t.Fatalf("unable to create request: %s", err)
	}
	req.Header.Add("Content-type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("unable to send request: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("response status code is not 200, got %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var body map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&body)
	if err != nil {
		t.Fatalf("error parsing response body")
	}

	result, ok := body["result"].(string)
	if !ok {
		t.Fatalf("unable to find result in response body")
	}

	var listResponse struct {
		Batches []struct {
			Name  string `json:"name"`
			State string `json:"state"`
		} `json:"batches"`
	}

	if err := json.Unmarshal([]byte(result), &listResponse); err != nil {
		t.Fatalf("error unmarshalling result: %s", err)
	}

	if len(listResponse.Batches) != 2 {
		t.Fatalf("expected exactly 2 batches, got %d", len(listResponse.Batches))
	}

	for _, batch := range listResponse.Batches {
		t.Logf("Returned batch: %+v\n", batch)
		if batch.State != "SUCCEEDED" && batch.State != "FAILED" {
			t.Fatalf("unexpected batch state: %s", batch.State)
		}
	}
}
