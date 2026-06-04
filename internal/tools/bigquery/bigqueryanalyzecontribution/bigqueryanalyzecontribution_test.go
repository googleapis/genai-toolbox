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

package bigqueryanalyzecontribution_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	bigqueryapi "cloud.google.com/go/bigquery"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/mcp-toolbox/internal/server"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	bigqueryds "github.com/googleapis/mcp-toolbox/internal/sources/bigquery"
	"github.com/googleapis/mcp-toolbox/internal/testutils"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/tools/bigquery/bigqueryanalyzecontribution"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
	bigqueryrestapi "google.golang.org/api/bigquery/v2"
	"google.golang.org/api/option"
)

func TestParseFromYamlBigQueryAnalyzeContribution(t *testing.T) {
	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	tcs := []struct {
		desc string
		in   string
		want server.ToolConfigs
	}{
		{
			desc: "basic example",
			in: `
            kind: tool
            name: example_tool
            type: bigquery-analyze-contribution
            source: my-instance
            description: some description
            `,
			want: server.ToolConfigs{
				"example_tool": bigqueryanalyzecontribution.Config{
					ConfigBase: tools.ConfigBase{
						Name:         "example_tool",
						Description:  "some description",
						AuthRequired: []string{},
					},
					Type:   "bigquery-analyze-contribution",
					Source: "my-instance",
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			// Parse contents
			_, _, _, got, _, _, err := server.UnmarshalResourceConfig(ctx, testutils.FormatYaml(tc.in))
			if err != nil {
				t.Fatalf("unable to unmarshal: %s", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("incorrect parse: diff %v", diff)
			}
		})
	}
}

type mockTransport struct {
	roundTrip func(*http.Request) (*http.Response, error)
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.roundTrip(req)
}

type mockSource struct {
	sources.Source
	calledSQL string
	client    *bigqueryapi.Client
}

func (m *mockSource) BigQueryClient() *bigqueryapi.Client {
	return m.client
}

func (m *mockSource) UseClientAuthorization() bool {
	return false
}

func (m *mockSource) GetAuthTokenHeaderName() string {
	return ""
}

func (m *mockSource) GetMaximumBytesBilled() int64 {
	return 0
}

func (m *mockSource) IsDatasetAllowed(projectID, datasetID string) bool {
	return true
}

func (m *mockSource) BigQueryAllowedDatasets() []string {
	return nil
}

func (m *mockSource) BigQuerySession() bigqueryds.BigQuerySessionProvider {
	return func(ctx context.Context) (*bigqueryds.Session, error) {
		return &bigqueryds.Session{ID: "mock-session-id"}, nil
	}
}

func (m *mockSource) RetrieveClientAndService(tools.AccessToken) (*bigqueryapi.Client, *bigqueryrestapi.Service, error) {
	return m.client, nil, nil
}

func (m *mockSource) RunSQL(ctx context.Context, client *bigqueryapi.Client, sql string, queryType string, params []bigqueryapi.QueryParameter, connProps []*bigqueryapi.ConnectionProperty) (any, error) {
	m.calledSQL = sql
	return "mocked_analyze_contribution_result", nil
}

type mockSourceProvider struct {
	tools.SourceProvider
	source *mockSource
}

func (m *mockSourceProvider) GetSource(name string) (sources.Source, bool) {
	return m.source, true
}

func TestInvoke(t *testing.T) {
	mockClient := &http.Client{
		Transport: &mockTransport{
			roundTrip: func(req *http.Request) (*http.Response, error) {
				// We expect requests to the jobs API (either POST to create, or GET to check status)
				respBody := `{
					"kind": "bigquery#job",
					"jobReference": {
						"projectId": "my-project",
						"jobId": "mock-job-id",
						"location": "US"
					},
					"status": {
						"state": "DONE"
					}
				}`
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(respBody)),
				}, nil
			},
		},
	}
	bqClient, err := bigqueryapi.NewClient(context.Background(), "my-project", option.WithHTTPClient(mockClient))
	if err != nil {
		t.Fatalf("failed to create bigquery client: %v", err)
	}

	cfg := bigqueryanalyzecontribution.Config{
		Name:        "analyze_contribution_tool",
		Type:        "bigquery-analyze-contribution",
		Source:      "my-bq-source",
		Description: "Analyze Contribution",
	}
	src := &mockSource{client: bqClient}
	sourcesMap := map[string]sources.Source{
		"my-bq-source": src,
	}
	tool, err := cfg.Initialize(sourcesMap)
	if err != nil {
		t.Fatalf("failed to initialize tool: %v", err)
	}

	analyzeContributionTool, ok := tool.(bigqueryanalyzecontribution.Tool)
	if !ok {
		t.Fatalf("expected bigqueryanalyzecontribution.Tool, got %T", tool)
	}

	tcs := []struct {
		desc               string
		inputData          any
		contributionMetric any
		isTestCol          any
		dimensionIdCols    any
		wantErr            bool
		wantSubstr         string
		wantSQLSub         string
	}{
		{
			desc:               "happy path",
			inputData:          "my_dataset.my_table",
			contributionMetric: "SUM(metric)",
			isTestCol:          "is_test",
			dimensionIdCols:    []any{"dim1", "dim2"},
			wantSQLSub:         "SELECT * FROM ML.GET_INSIGHTS(MODEL contribution_analysis_model_",
		},
		{
			desc:               "SQL injection attempt in dimension_id_cols",
			inputData:          "my_dataset.my_table",
			contributionMetric: "SUM(metric)",
			isTestCol:          "is_test",
			dimensionIdCols:    []any{"dim1", "dim2; drop table x"},
			wantErr:            true,
			wantSubstr:         "invalid column name in 'dimension_id_cols'",
		},
		{
			desc:               "SQL injection attempt in is_test_col",
			inputData:          "my_dataset.my_table",
			contributionMetric: "SUM(metric)",
			isTestCol:          "is_test; drop table x",
			dimensionIdCols:    []any{"dim1"},
			wantErr:            true,
			wantSubstr:         "invalid column name for 'is_test_col'",
		},
		{
			desc:               "single quote in contribution_metric",
			inputData:          "my_dataset.my_table",
			contributionMetric: "SUM('metric')",
			isTestCol:          "is_test",
			dimensionIdCols:    []any{"dim1"},
			wantErr:            true,
			wantSubstr:         "invalid 'contribution_metric': must not contain single quotes",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			provider := &mockSourceProvider{source: src}

			data := map[string]any{}
			if tc.inputData != nil {
				data["input_data"] = tc.inputData
			}
			if tc.contributionMetric != nil {
				data["contribution_metric"] = tc.contributionMetric
			}
			if tc.isTestCol != nil {
				data["is_test_col"] = tc.isTestCol
			}
			if tc.dimensionIdCols != nil {
				data["dimension_id_cols"] = tc.dimensionIdCols
			}

			paramVals, err := parameters.ParseParams(analyzeContributionTool.Parameters, data, nil)
			if err != nil {
				if tc.wantErr {
					if !strings.Contains(err.Error(), tc.wantSubstr) {
						t.Errorf("expected parse error to contain %q, got %v", tc.wantSubstr, err)
					}
					return
				}
				t.Fatalf("unexpected error parsing parameters: %v", err)
			}

			ctx, err := testutils.ContextWithNewLogger()
			if err != nil {
				t.Fatalf("failed to create context with logger: %v", err)
			}

			resp, err := tool.Invoke(ctx, provider, paramVals, "")
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tc.wantSubstr) {
					t.Errorf("expected error to contain %q, got %v", tc.wantSubstr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if resp != "mocked_analyze_contribution_result" {
				t.Errorf("unexpected response: got %v", resp)
			}

			if tc.wantSQLSub != "" && !strings.Contains(src.calledSQL, tc.wantSQLSub) {
				t.Errorf("expected SQL to contain %q, but got:\n%s", tc.wantSQLSub, src.calledSQL)
			}
		})
	}
}
