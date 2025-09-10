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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

func TestTool_Invoke(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/projects/test-project/location/global/prometheus/api/v1/query" {
			t.Errorf("unexpected path: got %q", r.URL.Path)
		}
		if r.URL.Query().Get("query") != "up" {
			t.Errorf("unexpected query: got %q", r.URL.Query().Get("query"))
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[]}}`)); err != nil {
			t.Fatalf("w.Write() err = %v", err)
		}
	}))
	defer server.Close()

	tool := Tool{
		Name:        "cloud-monitoring-query-prometheus",
		Kind:        "cloud-monitoring-query-prometheus",
		Description: "a test tool",
		Client:     &http.Client{},
		AllParams:      tools.Parameters{},
	}

	params := tools.ParamValues{
		{Name: "projectId", Value: "test-project"},
		{Name: "query", Value: "up"},
	}

	ctx := context.Background()

	// Another hack to inject the mock server url.
	monitoringEndpoint = server.URL

	got, err := tool.Invoke(ctx, params, "")
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}

	want := map[string]any{
		"status": "success",
		"data": map[string]any{
			"resultType": "vector",
			"result":     []any{},
		},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Invoke() mismatch (-want +got):\n%s", diff)
	}
}
