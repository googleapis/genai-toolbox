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

package bigquerychat

import (
	"strings"
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
)

func TestParseFromYamlBigQueryChat(t *testing.T) {
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
					kind: bigquery-chat
					source: my-instance
					description: some description
			`,
			want: server.ToolConfigs{
				"example_tool": Config{
					Name:         "example_tool",
					Kind:         "bigquery-chat",
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
		wantContains string
	}{
		{
			desc: "format_generated_sql",
			responseDict: map[string]any{
				"generatedSql": "SELECT * FROM my_table;",
			},
			maxRows:      100,
			wantContains: "## SQL Generated\n```sql\nSELECT * FROM my_table;\n```",
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
			maxRows:      100,
			wantContains: "| 1 | A |",
		},
		{
			desc: "check_data_truncation_message",
			responseDict: func() map[string]any {
				data := make([]any, 105)
				for i := 0; i < 105; i++ {
					data[i] = map[string]any{"id": i}
				}
				return map[string]any{
					"result": map[string]any{
						"schema": map[string]any{"fields": []any{map[string]any{"name": "id"}}},
						"data":   data,
					},
				}
			}(),
			maxRows:      100,
			wantContains: "... *and 5 more rows*.",
		},
		{
			desc:         "unhandled_response_returns_empty_string",
			responseDict: map[string]any{"invalid_key": "some_value"},
			maxRows:      100,
			wantContains: "",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := handleDataResponse(tc.responseDict, tc.maxRows)
			if !strings.Contains(got, tc.wantContains) {
				t.Errorf("handleDataResponse() = %q, want to contain %q", got, tc.wantContains)
			}
		})
	}
}

func TestHandleSchemaResponse(t *testing.T) {
	tcs := []struct {
		desc         string
		responseDict map[string]any
		wantContains string
	}{
		{
			desc: "schema_query_path",
			responseDict: map[string]any{
				"query": map[string]any{
					"question": "What is the schema?",
				},
			},
			wantContains: "What is the schema?",
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
									map[string]any{"name": "col1", "type": "STRING"},
								},
							},
						},
					},
				},
			},
			wantContains: "## Schema Resolved",
		},
		{
			desc:         "empty_response_returns_empty_string",
			responseDict: map[string]any{},
			wantContains: "",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := handleSchemaResponse(tc.responseDict)
			if !strings.Contains(got, tc.wantContains) {
				t.Errorf("handleSchemaResponse() = %q, want to contain %q", got, tc.wantContains)
			}
		})
	}
}

func TestAppendMessage(t *testing.T) {
	tcs := []struct {
		desc            string
		initialMessages []string
		newMessage      string
		want            []string
	}{
		{
			desc:            "append when last message is not data",
			initialMessages: []string{"## Thinking", "## Schema Resolved"},
			newMessage:      "## SQL Generated",
			want:            []string{"## Thinking", "## Schema Resolved", "## SQL Generated"},
		},
		{
			desc:            "replace when last message is data",
			initialMessages: []string{"## Thinking", "## Data Retrieved\n|...table...|"},
			newMessage:      "## Chart\n...",
			want:            []string{"## Thinking", "## Chart\n..."},
		},
		{
			desc:            "append to an empty list",
			initialMessages: []string{},
			newMessage:      "## First Message",
			want:            []string{"## First Message"},
		},
		{
			desc:            "should not append an empty new message",
			initialMessages: []string{"## Data Retrieved\n|...|"},
			newMessage:      "",
			want:            []string{"## Data Retrieved\n|...|"},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			// The function can modify the slice in place, so we pass a copy to be safe.
			initialCopy := make([]string, len(tc.initialMessages))
			copy(initialCopy, tc.initialMessages)

			got := appendMessage(initialCopy, tc.newMessage)
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
		want string
	}{
		{
			desc: "multiple parts",
			resp: map[string]any{
				"parts": []any{"The answer", " is 42."},
			},
			want: "Answer: The answer is 42.",
		},
		{
			desc: "single part",
			resp: map[string]any{
				"parts": []any{"Hello"},
			},
			want: "Answer: Hello",
		},
		{
			desc: "empty response",
			resp: map[string]any{},
			want: "Answer: Not provided.",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := handleTextResponse(tc.resp)
			if got != tc.want {
				t.Errorf("handleTextResponse() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestHandleError(t *testing.T) {
	tcs := []struct {
		desc string
		resp map[string]any
		want string
	}{
		{
			desc: "full_error_message",
			resp: map[string]any{
				"code":    404,
				"message": "Not Found",
			},
			want: "## Error\n**Code:** 404\n**Message:** Not Found",
		},
		{
			desc: "error_with_missing_message",
			resp: map[string]any{
				"code": 500,
			},
			want: "## Error\n**Code:** 500\n**Message:** No message provided.",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := handleError(tc.resp)
			if got != tc.want {
				t.Errorf("handleError() = %q, want %q", got, tc.want)
			}
		})
	}
}
