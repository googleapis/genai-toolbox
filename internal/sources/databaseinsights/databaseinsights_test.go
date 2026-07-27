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

package databaseinsights_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/mcp-toolbox/internal/server"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/sources/databaseinsights"
	"github.com/googleapis/mcp-toolbox/internal/testutils"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestParseFromYamlDatabaseInsights(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		want server.SourceConfigs
	}{
		{
			desc: "basic example",
			in: `
			kind: source
			name: my-insights
			type: databaseinsights
			project: my-project
			`,
			want: map[string]sources.SourceConfig{
				"my-insights": databaseinsights.Config{
					Name:    "my-insights",
					Type:    databaseinsights.SourceKind,
					Project: "my-project",
				},
			},
		},
		{
			desc: "with endpoint override",
			in: `
			kind: source
			name: my-insights
			type: databaseinsights
			project: my-project
			endpoint: staging-databaseinsights.sandbox.googleapis.com
			`,
			want: map[string]sources.SourceConfig{
				"my-insights": databaseinsights.Config{
					Name:     "my-insights",
					Type:     databaseinsights.SourceKind,
					Project:  "my-project",
					Endpoint: "staging-databaseinsights.sandbox.googleapis.com",
				},
			},
		},
		{
			desc: "without project",
			in: `
			kind: source
			name: my-insights
			type: databaseinsights
			`,
			want: map[string]sources.SourceConfig{
				"my-insights": databaseinsights.Config{
					Name: "my-insights",
					Type: databaseinsights.SourceKind,
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got, _, _, _, _, _, err := server.UnmarshalPrimitiveConfig(context.Background(), testutils.FormatYaml(tc.in))
			if err != nil {
				t.Fatalf("unable to unmarshal: %s", err)
			}
			if !cmp.Equal(tc.want, got) {
				t.Fatalf("incorrect parse: want %v, got %v", tc.want, got)
			}
		})
	}
}

func TestFailParseFromYaml(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		err  string
	}{
		{
			desc: "extra field",
			in: `
			kind: source
			name: my-insights
			type: databaseinsights
			project: my-project
			foo: bar
			`,
			err: "error unmarshaling source: unable to parse source \"my-insights\" as \"databaseinsights\": [1:1] unknown field \"foo\"\n>  1 | foo: bar\n       ^\n   2 | name: my-insights\n   3 | project: my-project\n   4 | type: databaseinsights",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			_, _, _, _, _, _, err := server.UnmarshalPrimitiveConfig(context.Background(), testutils.FormatYaml(tc.in))
			if err == nil {
				t.Fatalf("expect parsing to fail")
			}
			errStr := err.Error()
			if errStr != tc.err {
				t.Fatalf("unexpected error: got %q, want %q", errStr, tc.err)
			}
		})
	}
}

type mockTransport struct {
	roundTrip func(req *http.Request) (*http.Response, error)
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.roundTrip(req)
}

func TestFetchQueryStats(t *testing.T) {
	ctx := context.Background()
	ctx = util.WithUserAgent(ctx, "test")
	tracer := noop.NewTracerProvider().Tracer("noop")

	cfg := databaseinsights.Config{
		Name: "test-insights",
		Type: "databaseinsights",
	}

	source, err := cfg.Initialize(ctx, tracer)
	if err != nil {
		t.Fatalf("failed to initialize source: %v", err)
	}

	diSource := source.(*databaseinsights.Source)

	mockResp := `{
		"results": [
			["query_123", "postgres", 150.5]
		],
		"metadata": {
			"fields": [
				{"name": "query_id", "type": "STRING"},
				{"name": "database", "type": "STRING"},
				{"name": "sum(execution_time)", "type": "DOUBLE"}
			]
		}
	}`

	diSource.HTTPClient().Transport = &mockTransport{
		roundTrip: func(req *http.Request) (*http.Response, error) {
			if req.Method != "POST" {
				t.Errorf("expected POST request, got %s", req.Method)
			}
			if !strings.HasSuffix(req.URL.Path, "/queryStats:fetch") {
				t.Errorf("unexpected URL path: %s", req.URL.Path)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(mockResp)),
				Header:     make(http.Header),
			}, nil
		},
	}

	req := &databaseinsights.FetchQueryStatsRequest{
		Parent:           "projects/p1/locations/l1",
		FullResourceName: "//alloydb.googleapis.com/clusters/c1/instances/i1",
	}

	resp, err := diSource.FetchQueryStats(ctx, req)
	if err != nil {
		t.Fatalf("FetchQueryStats failed: %v", err)
	}

	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Results))
	}
}

func TestFetchWaitEventStats(t *testing.T) {
	ctx := context.Background()
	ctx = util.WithUserAgent(ctx, "test")
	tracer := noop.NewTracerProvider().Tracer("noop")

	cfg := databaseinsights.Config{
		Name: "test-insights",
		Type: "databaseinsights",
	}

	source, err := cfg.Initialize(ctx, tracer)
	if err != nil {
		t.Fatalf("failed to initialize source: %v", err)
	}

	diSource := source.(*databaseinsights.Source)

	mockResp := `{
		"results": [
			["Lock", 12.5, 100]
		],
		"metadata": {
			"fields": [
				{"name": "wait_class", "type": "STRING"},
				{"name": "sum(wait_time)", "type": "DOUBLE"},
				{"name": "sum(count)", "type": "INT64"}
			]
		}
	}`

	diSource.HTTPClient().Transport = &mockTransport{
		roundTrip: func(req *http.Request) (*http.Response, error) {
			if !strings.HasSuffix(req.URL.Path, "/waitEventStats:fetch") {
				t.Errorf("unexpected URL path: %s", req.URL.Path)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(mockResp)),
				Header:     make(http.Header),
			}, nil
		},
	}

	req := &databaseinsights.FetchWaitEventStatsRequest{
		Parent:           "projects/p1/locations/l1",
		FullResourceName: "//alloydb.googleapis.com/clusters/c1/instances/i1",
	}

	resp, err := diSource.FetchWaitEventStats(ctx, req)
	if err != nil {
		t.Fatalf("FetchWaitEventStats failed: %v", err)
	}

	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Results))
	}
}

func TestFetchQueryTimeSeries(t *testing.T) {
	ctx := context.Background()
	ctx = util.WithUserAgent(ctx, "test")
	tracer := noop.NewTracerProvider().Tracer("noop")

	cfg := databaseinsights.Config{
		Name: "test-insights",
		Type: "databaseinsights",
	}

	source, err := cfg.Initialize(ctx, tracer)
	if err != nil {
		t.Fatalf("failed to initialize source: %v", err)
	}

	diSource := source.(*databaseinsights.Source)

	mockResp := `{
		"timeseries": [
			{
				"groupbyFieldValues": ["query_123"],
				"values": [
					{
						"interval": {"startTime": "2026-06-15T10:00:00Z", "endTime": "2026-06-15T10:05:00Z"},
						"value": 150.5
					}
				]
			}
		]
	}`

	diSource.HTTPClient().Transport = &mockTransport{
		roundTrip: func(req *http.Request) (*http.Response, error) {
			if !strings.HasSuffix(req.URL.Path, "/queryTimeSeries:fetch") {
				t.Errorf("unexpected URL path: %s", req.URL.Path)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(mockResp)),
				Header:     make(http.Header),
			}, nil
		},
	}

	req := &databaseinsights.FetchQueryTimeSeriesRequest{
		Parent:           "projects/p1/locations/l1",
		FullResourceName: "//alloydb.googleapis.com/clusters/c1/instances/i1",
	}

	resp, err := diSource.FetchQueryTimeSeries(ctx, req)
	if err != nil {
		t.Fatalf("FetchQueryTimeSeries failed: %v", err)
	}

	if len(resp.TimeSeries) != 1 {
		t.Fatalf("expected 1 series, got %d", len(resp.TimeSeries))
	}
}

func TestFetchWaitEventTimeSeries(t *testing.T) {
	ctx := context.Background()
	ctx = util.WithUserAgent(ctx, "test")
	tracer := noop.NewTracerProvider().Tracer("noop")

	cfg := databaseinsights.Config{
		Name: "test-insights",
		Type: "databaseinsights",
	}

	source, err := cfg.Initialize(ctx, tracer)
	if err != nil {
		t.Fatalf("failed to initialize source: %v", err)
	}

	diSource := source.(*databaseinsights.Source)

	mockResp := `{
		"timeseries": [
			{
				"groupbyFieldValues": ["Lock"],
				"values": [
					{
						"interval": {"startTime": "2026-06-15T10:00:00Z", "endTime": "2026-06-15T10:05:00Z"},
						"value": 1.2
					}
				]
			}
		]
	}`

	diSource.HTTPClient().Transport = &mockTransport{
		roundTrip: func(req *http.Request) (*http.Response, error) {
			if !strings.HasSuffix(req.URL.Path, "/waitEventTimeSeries:fetch") {
				t.Errorf("unexpected URL path: %s", req.URL.Path)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(mockResp)),
				Header:     make(http.Header),
			}, nil
		},
	}

	req := &databaseinsights.FetchWaitEventTimeSeriesRequest{
		Parent:           "projects/p1/locations/l1",
		FullResourceName: "//alloydb.googleapis.com/clusters/c1/instances/i1",
	}

	resp, err := diSource.FetchWaitEventTimeSeries(ctx, req)
	if err != nil {
		t.Fatalf("FetchWaitEventTimeSeries failed: %v", err)
	}

	if len(resp.TimeSeries) != 1 {
		t.Fatalf("expected 1 series, got %d", len(resp.TimeSeries))
	}
	if resp.TimeSeries[0].Values[0].Value != 1.2 {
		t.Errorf("unexpected metric value: %f", resp.TimeSeries[0].Values[0].Value)
	}
}

func TestBatchQueryIndexRecommendations(t *testing.T) {
	ctx := context.Background()
	ctx = util.WithUserAgent(ctx, "test")
	tracer := noop.NewTracerProvider().Tracer("noop")

	cfg := databaseinsights.Config{
		Name: "test-insights",
		Type: "databaseinsights",
	}

	source, err := cfg.Initialize(ctx, tracer)
	if err != nil {
		t.Fatalf("failed to initialize source: %v", err)
	}

	diSource := source.(*databaseinsights.Source)

	mockResp := `{
		"databaseIndexRecommendations": [
			{
				"database": "postgres",
				"indexRecommendations": [
					{
						"sqlCommand": "CREATE INDEX ON t (c)"
					}
				]
			}
		]
	}`

	diSource.HTTPClient().Transport = &mockTransport{
		roundTrip: func(req *http.Request) (*http.Response, error) {
			if !strings.HasSuffix(req.URL.Path, "/indexRecommendations:batchQuery") {
				t.Errorf("unexpected URL path: %s", req.URL.Path)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString(mockResp)),
				Header:     make(http.Header),
			}, nil
		},
	}

	req := &databaseinsights.BatchQueryIndexRecommendationsRequest{
		Parent:           "projects/p1/locations/l1",
		FullResourceName: "//alloydb.googleapis.com/clusters/c1/instances/i1",
	}

	resp, err := diSource.BatchQueryIndexRecommendations(ctx, req)
	if err != nil {
		t.Fatalf("BatchQueryIndexRecommendations failed: %v", err)
	}

	if len(resp.DatabaseIndexRecommendations) != 1 {
		t.Fatalf("expected 1 recommendation, got %d", len(resp.DatabaseIndexRecommendations))
	}
	if resp.DatabaseIndexRecommendations[0].IndexRecommendations[0].SQLCommand != "CREATE INDEX ON t (c)" {
		t.Errorf("unexpected command: %s", resp.DatabaseIndexRecommendations[0].IndexRecommendations[0].SQLCommand)
	}
}
