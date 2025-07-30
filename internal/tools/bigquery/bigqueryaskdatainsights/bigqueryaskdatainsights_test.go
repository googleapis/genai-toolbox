// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package bigqueryaskdatainsights

import (
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
)

func TestParseFromYamlBigQueryAskDataInsights(t *testing.T) {
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
			tools:
				example_tool:
					kind: bigquery-ask-data-insights
					source: my-instance
					description: some description
			`,
			want: server.ToolConfigs{
				"example_tool": Config{
					Name:         "example_tool",
					Kind:         "bigquery-ask-data-insights",
					Source:       "my-instance",
					Description:  "some description",
					AuthRequired: []string{},
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := struct {
				Tools server.ToolConfigs `yaml:"tools"`
			}{}
			// Parse contents
			err := yaml.UnmarshalContext(ctx, testutils.FormatYaml(tc.in), &got)
			if err != nil {
				t.Fatalf("unable to unmarshal: %s", err)
			}
			if diff := cmp.Diff(tc.want, got.Tools); diff != "" {
				t.Fatalf("incorrect parse: diff %v", diff)
			}
		})
	}
}
func TestHandleDataResponse(t *testing.T) {
	tcs := []struct {
		desc         string
		responseDict map[string]any
		maxRows      int
		want         map[string]any
	}{
		{
			desc: "format_generated_sql",
			responseDict: map[string]any{
				"generatedSql": "SELECT * FROM my_table;",
			},
			maxRows: 100,
			want:    map[string]any{"SQL Generated": "SELECT * FROM my_table;"},
		},
		{
			desc: "format_data_result_table",
			responseDict: map[string]any{
				"result": map[string]any{
					"schema": map[string]any{
						"fields": []any{
							map[string]any{"name": "id"},
							map[string]any{"name": "name"},
						},
					},
					"data": []any{
						map[string]any{"id": 1, "name": "A"},
						map[string]any{"id": 2, "name": "B"},
					},
				},
			},
			maxRows: 100,
			want: map[string]any{
				"Data Retrieved": map[string]any{
					"headers": []string{"id", "name"},
					"rows":    [][]any{{1, "A"}, {2, "B"}},
					"summary": "Showing all 2 rows.",
				},
			},
		},
		{
			desc: "check_data_truncation_with_two_rows_and_max_one",
			responseDict: map[string]any{
				"result": map[string]any{
					"schema": map[string]any{
						"fields": []any{
							map[string]any{"name": "id"},
							map[string]any{"name": "name"},
						},
					},
					// <-- Total 2 rows of data
					"data": []any{
						map[string]any{"id": 1, "name": "A"},
						map[string]any{"id": 2, "name": "B"},
					},
				},
			},
			maxRows: 1,
			want: map[string]any{
				"Data Retrieved": map[string]any{
					"headers": []string{"id", "name"},
					"rows":    [][]any{{1, "A"}},
					"summary": "Showing the first 1 of 2 total rows.",
				},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := handleDataResponse(tc.responseDict, tc.maxRows)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("handleDataResponse() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestHandleSchemaResponse(t *testing.T) {
	tcs := []struct {
		desc         string
		responseDict map[string]any
		want         map[string]any
	}{
		{
			desc: "schema_query_path",
			responseDict: map[string]any{
				"query": map[string]any{
					"question": "What is the schema?",
				},
			},
			want: map[string]any{"Question": "What is the schema?"},
		},
		{
			desc: "schema_result_path",
			responseDict: map[string]any{
				"result": map[string]any{
					"datasources": []any{
						map[string]any{
							"bigqueryTableReference": map[string]any{
								"projectId": "p",
								"datasetId": "d",
								"tableId":   "t",
							},
							"schema": map[string]any{
								"fields": []any{
									map[string]any{"name": "col1", "type": "STRING", "mode": "NULLABLE"},
								},
							},
						},
					},
				},
			},
			want: map[string]any{
				"Schema Resolved": []map[string]any{
					{
						"source_name": "p.d.t",
						"schema": map[string]any{
							"headers": []string{"Column", "Type", "Description", "Mode"},
							"rows":    [][]any{{"col1", "STRING", "", "NULLABLE"}},
						},
					},
				},
			},
		},
		{
			desc:         "empty_response_returns_nil",
			responseDict: map[string]any{},
			want:         nil,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := handleSchemaResponse(tc.responseDict)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("handleSchemaResponse() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAppendMessage(t *testing.T) {
	tcs := []struct {
		desc            string
		initialMessages []map[string]any
		newMessage      map[string]any
		want            []map[string]any
	}{
		{
			desc:            "append when last message is not data",
			initialMessages: []map[string]any{{"Thinking": nil}, {"Schema Resolved": nil}},
			newMessage:      map[string]any{"SQL Generated": "SELECT 1"},
			want:            []map[string]any{{"Thinking": nil}, {"Schema Resolved": nil}, {"SQL Generated": "SELECT 1"}},
		},
		{
			desc:            "replace when last message is data",
			initialMessages: []map[string]any{{"Thinking": nil}, {"Data Retrieved": map[string]any{"rows": []any{}}}},
			newMessage:      map[string]any{"Data Retrieved": map[string]any{"rows": []any{1}}},
			want:            []map[string]any{{"Thinking": nil}, {"Data Retrieved": map[string]any{"rows": []any{1}}}},
		},
		{
			desc:            "append to an empty list",
			initialMessages: []map[string]any{},
			newMessage:      map[string]any{"Answer": "First Message"},
			want:            []map[string]any{{"Answer": "First Message"}},
		},
		{
			desc:            "should not append an empty new message",
			initialMessages: []map[string]any{{"Data Retrieved": nil}},
			newMessage:      nil,
			want:            []map[string]any{{"Data Retrieved": nil}},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := appendMessage(tc.initialMessages, tc.newMessage)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("appendMessage() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestHandleTextResponse(t *testing.T) {
	tcs := []struct {
		desc string
		resp map[string]any
		want map[string]any
	}{
		{
			desc: "multiple parts",
			resp: map[string]any{
				"parts": []any{"The answer", " is 42."},
			},
			want: map[string]any{"Answer": "The answer is 42."},
		},
		{
			desc: "single part",
			resp: map[string]any{
				"parts": []any{"Hello"},
			},
			want: map[string]any{"Answer": "Hello"},
		},
		{
			desc: "empty response",
			resp: map[string]any{},
			want: map[string]any{"Answer": ""},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := handleTextResponse(tc.resp)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("handleTextResponse() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestHandleError(t *testing.T) {
	tcs := []struct {
		desc string
		resp map[string]any
		want map[string]any
	}{
		{
			desc: "full_error_message",
			resp: map[string]any{
				"code":    float64(404),
				"message": "Not Found",
			},
			want: map[string]any{
				"Error": map[string]any{
					"Code":    404,
					"Message": "Not Found",
				},
			},
		},
		{
			desc: "error_with_missing_message",
			resp: map[string]any{
				"code": float64(500),
			},
			want: map[string]any{
				"Error": map[string]any{
					"Code":    500,
					"Message": "",
				},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := handleError(tc.resp)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("handleError() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
