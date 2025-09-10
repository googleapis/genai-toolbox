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

package cloud_monitoring_integration_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/tools/cloudmonitoring"
)

func TestTool_Invoke(t *testing.T) {
	t.Parallel()

	// Mock the monitoring server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/projects/test-project/location/global/prometheus/api/v1/query" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		query := r.URL.Query().Get("query")
		if query != "up" {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"status":"success","data":{"resultType":"vector","result":[]}}`)
	}))
	defer server.Close()

	// Create a new observability tool
	tool := &cloudmonitoring.Tool{
		Name:        "test-cloudmonitoring",
		Kind:        "cloud-monitoring-query-prometheus",
		Description: "Test Cloudmonitoring Tool",
		AllParams:      tools.Parameters{},
		Client:     &http.Client{},
	}

	// Set the monitoring endpoint to the mock server
	cloudmonitoring.SetMonitoringEndpoint(server.URL)

	// Define the test parameters
	params := tools.ParamValues{
		{Name: "projectId", Value: "test-project"},
		{Name: "query", Value: "up"},
	}

	// Invoke the tool
	result, err := tool.Invoke(context.Background(), params, "")
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}

	// Check the result
	expected := map[string]any{
		"status": "success",
		"data": map[string]any{
			"resultType": "vector",
			"result":     []any{},
		},
	}
	if diff := cmp.Diff(expected, result); diff != "" {
		t.Errorf("Invoke() result mismatch (-want +got):\n%s", diff)
	}
}

func TestTool_Invoke_Error(t *testing.T) {
	t.Parallel()

	// Mock the monitoring server to return an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer server.Close()

	// Create a new observability tool
	tool := &cloudmonitoring.Tool{
		Name:        "test-cloudmonitoring",
		Kind:        "clou-monitoring-query-prometheus",
		Description: "Test Cloudmonitoring Tool",
		AllParams:      tools.Parameters{},
		Client:     &http.Client{},
	}

	// Set the monitoring endpoint to the mock server
	cloudmonitoring.SetMonitoringEndpoint(server.URL)

	// Define the test parameters
	params := tools.ParamValues{
		{Name: "projectId", Value: "test-project"},
		{Name: "query", Value: "up"},
	}

	// Invoke the tool
	_, err := tool.Invoke(context.Background(), params, "")
	if err == nil {
		t.Fatal("Invoke() error = nil, want error")
	}
}
