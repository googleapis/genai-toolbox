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

package cloudmonitoring

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/sources"
	cloudmonitoringsrc "github.com/googleapis/genai-toolbox/internal/sources/cloudmonitoring"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/tools/cloudmonitoring"
	"github.com/googleapis/genai-toolbox/internal/util"
	"go.opentelemetry.io/otel/trace"
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
		AllParams:   tools.Parameters{},
		BaseURL:     server.URL,
		Client:      &http.Client{},
	}

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
		t.Errorf("Invoke() result mismatch (-want +got): %s", diff)
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
		Kind:        "cloud-monitoring-query-prometheus",
		Description: "Test Cloudmonitoring Tool",
		AllParams:   tools.Parameters{},
		BaseURL:     server.URL,
		Client:      &http.Client{},
	}

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

func TestTool_Invoke_MalformedJSON(t *testing.T) {
	t.Parallel()

	// Mock the monitoring server to return malformed JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"status":"success","data":`) // Malformed JSON
	}))
	defer server.Close()

	// Create a new observability tool
	tool := &cloudmonitoring.Tool{
		Name:        "test-cloudmonitoring",
		Kind:        "cloud-monitoring-query-prometheus",
		Description: "Test Cloudmonitoring Tool",
		AllParams:   tools.Parameters{},
		BaseURL:     server.URL,
		Client:      &http.Client{},
	}

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

func TestTool_Invoke_MissingProjectID(t *testing.T) {
	t.Parallel()

	tool := &cloudmonitoring.Tool{
		Name:        "test-cloudmonitoring",
		Kind:        "cloud-monitoring-query-prometheus",
		Description: "Test Cloudmonitoring Tool",
		AllParams:   tools.Parameters{},
	}

	params := tools.ParamValues{
		{Name: "query", Value: "up"},
	}

	_, err := tool.Invoke(context.Background(), params, "")
	if err == nil {
		t.Fatal("Invoke() error = nil, want error")
	}
	expected := `projectId parameter not found or not a string`
	if err.Error() != expected {
		t.Errorf("Invoke() error = %q, want %q", err.Error(), expected)
	}
}

func TestTool_Invoke_MissingQuery(t *testing.T) {
	t.Parallel()

	tool := &cloudmonitoring.Tool{
		Name:        "test-cloudmonitoring",
		Kind:        "cloud-monitoring-query-prometheus",
		Description: "Test Cloudmonitoring Tool",
		AllParams:   tools.Parameters{},
	}

	params := tools.ParamValues{
		{Name: "projectId", Value: "test-project"},
	}

	_, err := tool.Invoke(context.Background(), params, "")
	if err == nil {
		t.Fatal("Invoke() error = nil, want error")
	}
	expected := `query parameter not found or not a string`
	if err.Error() != expected {
		t.Errorf("Invoke() error = %q, want %q", err.Error(), expected)
	}
}

// transport is a custom http.RoundTripper that always returns an error.

type errorTransport struct{}

func (t *errorTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, fmt.Errorf("client error")
}

func TestTool_Invoke_ClientError(t *testing.T) {
	t.Parallel()

	tool := &cloudmonitoring.Tool{
		Name:        "test-cloudmonitoring",
		Kind:        "cloud-monitoring-query-prometheus",
		Description: "Test Cloudmonitoring Tool",
		AllParams:   tools.Parameters{},
		BaseURL:     "http://localhost",
		Client:      &http.Client{Transport: &errorTransport{}},
	}

	params := tools.ParamValues{
		{Name: "projectId", Value: "test-project"},
		{Name: "query", Value: "up"},
	}

	_, err := tool.Invoke(context.Background(), params, "")
	if err == nil {
		t.Fatal("Invoke() error = nil, want error")
	}
}

func TestTool_Invoke_NonEmptyResult(t *testing.T) {
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
		fmt.Fprintln(w, `{
			"status":"success",
			"data":{
				"resultType":"vector",
				"result":[
					{
						"metric":{"__name__":"up","instance":"localhost:9090","job":"prometheus"},
						"value":[1617916800,"1"]
					}
				]
			}
		}`)
	}))
	defer server.Close()

	// Create a new observability tool
	tool := &cloudmonitoring.Tool{
		Name:        "test-cloudmonitoring",
		Kind:        "cloud-monitoring-query-prometheus",
		Description: "Test Cloudmonitoring Tool",
		AllParams:   tools.Parameters{},
		BaseURL:     server.URL,
		Client:      &http.Client{},
	}

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
			"result": []any{
				map[string]any{
					"metric": map[string]any{
						"__name__": "up",
						"instance": "localhost:9090",
						"job":      "prometheus",
					},
					"value": []any{
						float64(1617916800), "1",
					},
				},
			},
		},
	}
	if diff := cmp.Diff(expected, result); diff != "" {
		t.Errorf("Invoke() result mismatch (-want +got): %s", diff)
	}
}

func TestTool_Invoke_InvalidKind(t *testing.T) {
	t.Parallel()

	tool := &cloudmonitoring.Tool{
		Name:        "test-cloudmonitoring",
		Kind:        "invalid-kind",
		Description: "Test Cloudmonitoring Tool",
		AllParams:   tools.Parameters{},
		Client:      &http.Client{},
	}

	params := tools.ParamValues{
		{Name: "projectId", Value: "test-project"},
		{Name: "query", Value: "up"},
	}

	_, err := tool.Invoke(context.Background(), params, "")
	if err == nil {
		t.Fatal("Invoke() error = nil, want error")
	}
}

func TestTool_Invoke_BadRequest(t *testing.T) {
	t.Parallel()

	// Mock the monitoring server to return a bad request error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer server.Close()

	// Create a new observability tool
	tool := &cloudmonitoring.Tool{
		Name:        "test-cloudmonitoring",
		Kind:        "cloud-monitoring-query-prometheus",
		Description: "Test Cloudmonitoring Tool",
		AllParams:   tools.Parameters{},
		BaseURL:     server.URL,
		Client:      &http.Client{},
	}

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

func TestInitialization(t *testing.T) {
	t.Parallel()

	sourceCfg := cloudmonitoringsrc.Config{
		Name: "test-cm-source",
		Kind: "cloud-monitoring",
	}

	ctx := util.WithUserAgent(context.Background(), "test-agent")
	tracer := trace.NewNoopTracerProvider().Tracer("")

	src, err := sourceCfg.Initialize(ctx, tracer)
	if err != nil {
		t.Fatalf("sourceCfg.Initialize() error = %v", err)
	}

	srcs := map[string]sources.Source{
		"test-cm-source": src,
	}

	toolCfg := cloudmonitoring.Config{
		Name:        "test-cm-tool",
		Kind:        "cloud-monitoring-query-prometheus",
		Source:      "test-cm-source",
		Description: "a test tool",
	}

	_, err = toolCfg.Initialize(srcs)
	if err != nil {
		t.Fatalf("toolCfg.Initialize() error = %v", err)
	}
}