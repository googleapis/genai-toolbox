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

package bigquerycommon

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"google.golang.org/api/option"
	bigqueryrestapi "google.golang.org/api/bigquery/v2"
)

func TestGetLabels(t *testing.T) {
	toolName := "test-tool"
	expected := map[string]string{"genai-toolbox-tool": toolName}
	actual := getLabels(toolName)
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("getLabels() = %v, want %v", actual, expected)
	}
}

func TestDryRunQuery(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	location := "us"
	sql := "SELECT 1"
	toolName := "test-tool"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/projects/test-project/jobs" {
			t.Errorf("expected to request '/projects/test-project/jobs', got: %s", r.URL.Path)
		}
		var job bigqueryrestapi.Job
		if err := json.NewDecoder(r.Body).Decode(&job); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		expectedLabels := map[string]string{"genai-toolbox-tool": toolName}
		if !reflect.DeepEqual(job.Configuration.Labels, expectedLabels) {
			t.Errorf("expected labels %v, got %v", expectedLabels, job.Configuration.Labels)
		}

		if !job.Configuration.DryRun {
			t.Errorf("expected DryRun to be true")
		}

		// Send back a dummy response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&bigqueryrestapi.Job{
			JobReference: &bigqueryrestapi.JobReference{
				ProjectId: projectID,
				JobId:     "job_123",
				Location:  location,
			},
			Configuration: &bigqueryrestapi.JobConfiguration{
				DryRun: true,
				Query: &bigqueryrestapi.JobConfigurationQuery{
					Query: sql,
				},
			},
		})
	}))
	defer server.Close()

	restService, err := bigqueryrestapi.NewService(ctx, option.WithEndpoint(server.URL), option.WithHTTPClient(http.DefaultClient))
	if err != nil {
		t.Fatalf("failed to create test service: %v", err)
	}

	_, err = DryRunQuery(ctx, restService, projectID, location, sql, nil, nil, toolName)
	if err != nil {
		t.Fatalf("DryRunQuery failed: %v", err)
	}
}
