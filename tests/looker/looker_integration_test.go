// Copyright 2024 Google LLC
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

package looker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/googleapis/genai-toolbox/internal/log"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/util"
	"github.com/googleapis/genai-toolbox/tests"
)

var (
	LookerSourceKind   = "looker"
	LookerBaseUrl      = os.Getenv("LOOKER_BASE_URL")
	LookerVerifySsl    = os.Getenv("LOOKER_VERIFY_SSL")
	LookerClientId     = os.Getenv("LOOKER_CLIENT_ID")
	LookerClientSecret = os.Getenv("LOOKER_CLIENT_SECRET")
	LookerProject      = os.Getenv("LOOKER_PROJECT")
	LookerLocation     = os.Getenv("LOOKER_LOCATION")
)

func getLookerVars(t *testing.T) map[string]any {
	switch "" {
	case LookerBaseUrl:
		t.Fatal("'LOOKER_BASE_URL' not set")
	case LookerVerifySsl:
		t.Fatal("'LOOKER_VERIFY_SSL' not set")
	case LookerClientId:
		t.Fatal("'LOOKER_CLIENT_ID' not set")
	case LookerClientSecret:
		t.Fatal("'LOOKER_CLIENT_SECRET' not set")
	case LookerProject:
		t.Fatal("'LOOKER_PROJECT' not set")
	case LookerLocation:
		t.Fatal("'LOOKER_LOCATION' not set")
	}

	return map[string]any{
		"kind":          LookerSourceKind,
		"base_url":      LookerBaseUrl,
		"verify_ssl":    (LookerVerifySsl == "true"),
		"client_id":     LookerClientId,
		"client_secret": LookerClientSecret,
		"project":       LookerProject,
		"location":      LookerLocation,
	}
}

func TestLooker(t *testing.T) {
	sourceConfig := getLookerVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	testLogger, err := log.NewStdLogger(os.Stdout, os.Stderr, "info")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	ctx = util.WithLogger(ctx, testLogger)

	var args []string

	// Write config into a file and pass it to command
	toolsFile := map[string]any{
		"sources": map[string]any{
			"my-instance": sourceConfig,
		},
		"tools": map[string]any{
			"get_models": map[string]any{
				"kind":        "looker-get-models",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"get_explores": map[string]any{
				"kind":        "looker-get-explores",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"get_dimensions": map[string]any{
				"kind":        "looker-get-dimensions",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"get_measures": map[string]any{
				"kind":        "looker-get-measures",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"get_filters": map[string]any{
				"kind":        "looker-get-filters",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"get_parameters": map[string]any{
				"kind":        "looker-get-parameters",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"query": map[string]any{
				"kind":        "looker-query",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"query_sql": map[string]any{
				"kind":        "looker-query-sql",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"query_url": map[string]any{
				"kind":        "looker-query-url",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"get_looks": map[string]any{
				"kind":        "looker-get-looks",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"get_dashboards": map[string]any{
				"kind":        "looker-get-dashboards",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"conversational_analytics": map[string]any{
				"kind":        "looker-conversational-analytics",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
		},
	}

	cmd, cleanup, err := tests.StartCmd(ctx, toolsFile, args...)
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

	tests.RunToolGetTestByName(t, "get_models",
		map[string]any{
			"get_models": map[string]any{
				"description":  "Simple tool to test end to end functionality.",
				"authRequired": []any{},
				"parameters":   []any{},
			},
		},
	)
	tests.RunToolGetTestByName(t, "get_explores",
		map[string]any{
			"get_explores": map[string]any{
				"description":  "Simple tool to test end to end functionality.",
				"authRequired": []any{},
				"parameters": []any{
					map[string]any{
						"authSources": []any{},
						"description": "The model containing the explores.",
						"name":        "model",
						"required":    true,
						"type":        "string",
					},
				},
			},
		},
	)
	tests.RunToolGetTestByName(t, "get_dimensions",
		map[string]any{
			"get_dimensions": map[string]any{
				"description":  "Simple tool to test end to end functionality.",
				"authRequired": []any{},
				"parameters": []any{
					map[string]any{
						"authSources": []any{},
						"description": "The model containing the explore.",
						"name":        "model",
						"required":    true,
						"type":        "string",
					},
					map[string]any{
						"authSources": []any{},
						"description": "The explore containing the fields.",
						"name":        "explore",
						"required":    true,
						"type":        "string",
					},
				},
			},
		},
	)
	tests.RunToolGetTestByName(t, "get_measures",
		map[string]any{
			"get_measures": map[string]any{
				"description":  "Simple tool to test end to end functionality.",
				"authRequired": []any{},
				"parameters": []any{
					map[string]any{
						"authSources": []any{},
						"description": "The model containing the explore.",
						"name":        "model",
						"required":    true,
						"type":        "string",
					},
					map[string]any{
						"authSources": []any{},
						"description": "The explore containing the fields.",
						"name":        "explore",
						"required":    true,
						"type":        "string",
					},
				},
			},
		},
	)
	tests.RunToolGetTestByName(t, "get_parameters",
		map[string]any{
			"get_parameters": map[string]any{
				"description":  "Simple tool to test end to end functionality.",
				"authRequired": []any{},
				"parameters": []any{
					map[string]any{
						"authSources": []any{},
						"description": "The model containing the explore.",
						"name":        "model",
						"required":    true,
						"type":        "string",
					},
					map[string]any{
						"authSources": []any{},
						"description": "The explore containing the fields.",
						"name":        "explore",
						"required":    true,
						"type":        "string",
					},
				},
			},
		},
	)
	tests.RunToolGetTestByName(t, "get_filters",
		map[string]any{
			"get_filters": map[string]any{
				"description":  "Simple tool to test end to end functionality.",
				"authRequired": []any{},
				"parameters": []any{
					map[string]any{
						"authSources": []any{},
						"description": "The model containing the explore.",
						"name":        "model",
						"required":    true,
						"type":        "string",
					},
					map[string]any{
						"authSources": []any{},
						"description": "The explore containing the fields.",
						"name":        "explore",
						"required":    true,
						"type":        "string",
					},
				},
			},
		},
	)
	tests.RunToolGetTestByName(t, "query",
		map[string]any{
			"query": map[string]any{
				"description":  "Simple tool to test end to end functionality.",
				"authRequired": []any{},
				"parameters": []any{
					map[string]any{
						"authSources": []any{},
						"description": "The model containing the explore.",
						"name":        "model",
						"required":    true,
						"type":        "string",
					},
					map[string]any{
						"authSources": []any{},
						"description": "The explore to be queried.",
						"name":        "explore",
						"required":    true,
						"type":        "string",
					},
					map[string]any{
						"authSources": []any{},
						"description": "The fields to be retrieved.",
						"items": map[string]any{
							"authSources": []any{},
							"description": "A field to be returned in the query",
							"name":        "field",
							"required":    true,
							"type":        "string",
						},
						"name":     "fields",
						"required": true,
						"type":     "array",
					},
					map[string]any{
						"additionalProperties": true,
						"authSources":          []any{},
						"description":          "The filters for the query",
						"name":                 "filters",
						"required":             false,
						"type":                 "object",
					},
					map[string]any{
						"authSources": []any{},
						"description": "The query pivots (must be included in fields as well).",
						"items": map[string]any{
							"authSources": []any{},
							"description": "A field to be used as a pivot in the query",
							"name":        "pivot_field",
							"required":    false,
							"type":        "string",
						},
						"name":     "pivots",
						"required": false,
						"type":     "array",
					},
					map[string]any{
						"authSources": []any{},
						"description": "The sorts like \"field.id desc 0\".",
						"items": map[string]any{
							"authSources": []any{},
							"description": "A field to be used as a sort in the query",
							"name":        "sort_field",
							"required":    false,
							"type":        "string",
						},
						"name":     "sorts",
						"required": false,
						"type":     "array",
					},
					map[string]any{
						"authSources": []any{},
						"description": "The row limit.",
						"name":        "limit",
						"required":    false,
						"type":        "integer",
					},
					map[string]any{
						"authSources": []any{},
						"description": "The query timezone.",
						"name":        "tz",
						"required":    false,
						"type":        "string",
					},
				},
			},
		},
	)
	tests.RunToolGetTestByName(t, "query_sql",
		map[string]any{
			"query_sql": map[string]any{
				"description":  "Simple tool to test end to end functionality.",
				"authRequired": []any{},
				"parameters": []any{
					map[string]any{
						"authSources": []any{},
						"description": "The model containing the explore.",
						"name":        "model",
						"required":    true,
						"type":        "string",
					},
					map[string]any{
						"authSources": []any{},
						"description": "The explore to be queried.",
						"name":        "explore",
						"required":    true,
						"type":        "string",
					},
					map[string]any{
						"authSources": []any{},
						"description": "The fields to be retrieved.",
						"items": map[string]any{
							"authSources": []any{},
							"description": "A field to be returned in the query",
							"name":        "field",
							"required":    true,
							"type":        "string",
						},
						"name":     "fields",
						"required": true,
						"type":     "array",
					},
					map[string]any{
						"additionalProperties": true,
						"authSources":          []any{},
						"description":          "The filters for the query",
						"name":                 "filters",
						"required":             false,
						"type":                 "object",
					},
					map[string]any{
						"authSources": []any{},
						"description": "The query pivots (must be included in fields as well).",
						"items": map[string]any{
							"authSources": []any{},
							"description": "A field to be used as a pivot in the query",
							"name":        "pivot_field",
							"required":    false,
							"type":        "string",
						},
						"name":     "pivots",
						"required": false,
						"type":     "array",
					},
					map[string]any{
						"authSources": []any{},
						"description": "The sorts like \"field.id desc 0\".",
						"items": map[string]any{
							"authSources": []any{},
							"description": "A field to be used as a sort in the query",
							"name":        "sort_field",
							"required":    false,
							"type":        "string",
						},
						"name":     "sorts",
						"required": false,
						"type":     "array",
					},
					map[string]any{
						"authSources": []any{},
						"description": "The row limit.",
						"name":        "limit",
						"required":    false,
						"type":        "integer",
					},
					map[string]any{
						"authSources": []any{},
						"description": "The query timezone.",
						"name":        "tz",
						"required":    false,
						"type":        "string",
					},
				},
			},
		},
	)
	tests.RunToolGetTestByName(t, "query_url",
		map[string]any{
			"query_url": map[string]any{
				"description":  "Simple tool to test end to end functionality.",
				"authRequired": []any{},
				"parameters": []any{
					map[string]any{
						"authSources": []any{},
						"description": "The model containing the explore.",
						"name":        "model",
						"required":    true,
						"type":        "string",
					},
					map[string]any{
						"authSources": []any{},
						"description": "The explore to be queried.",
						"name":        "explore",
						"required":    true,
						"type":        "string",
					},
					map[string]any{
						"authSources": []any{},
						"description": "The fields to be retrieved.",
						"items": map[string]any{
							"authSources": []any{},
							"description": "A field to be returned in the query",
							"name":        "field",
							"required":    true,
							"type":        "string",
						},
						"name":     "fields",
						"required": true,
						"type":     "array",
					},
					map[string]any{
						"additionalProperties": true,
						"authSources":          []any{},
						"description":          "The filters for the query",
						"name":                 "filters",
						"required":             false,
						"type":                 "object",
					},
					map[string]any{
						"authSources": []any{},
						"description": "The query pivots (must be included in fields as well).",
						"items": map[string]any{
							"authSources": []any{},
							"description": "A field to be used as a pivot in the query",
							"name":        "pivot_field",
							"required":    false,
							"type":        "string",
						},
						"name":     "pivots",
						"required": false,
						"type":     "array",
					},
					map[string]any{
						"authSources": []any{},
						"description": "The sorts like \"field.id desc 0\".",
						"items": map[string]any{
							"authSources": []any{},
							"description": "A field to be used as a sort in the query",
							"name":        "sort_field",
							"required":    false,
							"type":        "string",
						},
						"name":     "sorts",
						"required": false,
						"type":     "array",
					},
					map[string]any{
						"authSources": []any{},
						"description": "The row limit.",
						"name":        "limit",
						"required":    false,
						"type":        "integer",
					},
					map[string]any{
						"authSources": []any{},
						"description": "The query timezone.",
						"name":        "tz",
						"required":    false,
						"type":        "string",
					},
					map[string]any{
						"additionalProperties": true,
						"authSources":          []any{},
						"description":          "The visualization config for the query",
						"name":                 "vis_config",
						"required":             false,
						"type":                 "object",
					},
				},
			},
		},
	)
	tests.RunToolGetTestByName(t, "get_looks",
		map[string]any{
			"get_looks": map[string]any{
				"description":  "Simple tool to test end to end functionality.",
				"authRequired": []any{},
				"parameters": []any{
					map[string]any{
						"authSources": []any{},
						"description": "The title of the look.",
						"name":        "title",
						"required":    false,
						"type":        "string",
					},
					map[string]any{
						"authSources": []any{},
						"description": "The description of the look.",
						"name":        "desc",
						"required":    false,
						"type":        "string",
					},
					map[string]any{
						"authSources": []any{},
						"description": "The number of looks to fetch. Default 100",
						"name":        "limit",
						"required":    false,
						"type":        "integer",
					},
					map[string]any{
						"authSources": []any{},
						"description": "The number of looks to skip before fetching. Default 0",
						"name":        "offset",
						"required":    false,
						"type":        "integer",
					},
				},
			},
		},
	)
	tests.RunToolGetTestByName(t, "get_dashboards",
		map[string]any{
			"get_dashboards": map[string]any{
				"description":  "Simple tool to test end to end functionality.",
				"authRequired": []any{},
				"parameters": []any{
					map[string]any{
						"authSources": []any{},
						"description": "The title of the dashboard.",
						"name":        "title",
						"required":    false,
						"type":        "string",
					},
					map[string]any{
						"authSources": []any{},
						"description": "The description of the dashboard.",
						"name":        "desc",
						"required":    false,
						"type":        "string",
					},
					map[string]any{
						"authSources": []any{},
						"description": "The number of dashboards to fetch. Default 100",
						"name":        "limit",
						"required":    false,
						"type":        "integer",
					},
					map[string]any{
						"authSources": []any{},
						"description": "The number of dashboards to skip before fetching. Default 0",
						"name":        "offset",
						"required":    false,
						"type":        "integer",
					},
				},
			},
		},
	)
	tests.RunToolGetTestByName(t, "conversational_analytics",
		map[string]any{
			"conversational_analytics": map[string]any{
				"description":  "Simple tool to test end to end functionality.",
				"authRequired": []any{},
				"parameters": []any{
					map[string]any{
						"authSources": []any{},
						"description": "The user's question, potentially including conversation history and system instructions for context.",
						"name":        "user_query_with_context",
						"required":    true,
						"type":        "string",
					},
					map[string]any{
						"authSources": []any{},
						"description": "An Array of at least one and up to 5 explore references like [{'model': 'MODEL_NAME', 'explore': 'EXPLORE_NAME'}]",
						"items": map[string]any{
							"additionalProperties": true,
							"authSources":          []any{},
							"name":                 "explore_reference",
							"description":          "An explore reference like {'model': 'MODEL_NAME', 'explore': 'EXPLORE_NAME'}",
							"required":             true,
							"type":                 "object",
						},
						"name":     "explore_references",
						"required": true,
						"type":     "array",
					},
				},
			},
		},
	)

	wantResult := "{\"label\":\"System Activity\",\"name\":\"system__activity\",\"project_name\":\"system__activity\"}"
	tests.RunToolInvokeSimpleTest(t, "get_models", wantResult)

	wantResult = "{\"description\":\"Data about Look and dashboard usage, including frequency of views, favoriting, scheduling, embedding, and access via the API. Also includes details about individual Looks and dashboards.\",\"group_label\":\"System Activity\",\"label\":\"Content Usage\",\"name\":\"content_usage\"}"
	tests.RunToolInvokeParametersTest(t, "get_explores", []byte(`{"model": "system__activity"}`), wantResult)

	wantResult = "{\"description\":\"Number of times this content has been viewed via the Looker API\",\"label\":\"Content Usage API Count\",\"label_short\":\"API Count\",\"name\":\"content_usage.api_count\",\"type\":\"number\"}"
	tests.RunToolInvokeParametersTest(t, "get_dimensions", []byte(`{"model": "system__activity", "explore": "content_usage"}`), wantResult)

	wantResult = "{\"description\":\"The total number of views via the Looker API\",\"label\":\"Content Usage API Total\",\"label_short\":\"API Total\",\"name\":\"content_usage.api_total\",\"type\":\"sum\"}"
	tests.RunToolInvokeParametersTest(t, "get_measures", []byte(`{"model": "system__activity", "explore": "content_usage"}`), wantResult)

	wantResult = "[]"
	tests.RunToolInvokeParametersTest(t, "get_filters", []byte(`{"model": "system__activity", "explore": "content_usage"}`), wantResult)

	wantResult = "[]"
	tests.RunToolInvokeParametersTest(t, "get_parameters", []byte(`{"model": "system__activity", "explore": "content_usage"}`), wantResult)

	wantResult = "{\"look.count\":"
	tests.RunToolInvokeParametersTest(t, "query", []byte(`{"model": "system__activity", "explore": "look", "fields": ["look.count"]}`), wantResult)

	wantResult = "SELECT"
	tests.RunToolInvokeParametersTest(t, "query_sql", []byte(`{"model": "system__activity", "explore": "look", "fields": ["look.count"]}`), wantResult)

	wantResult = "system__activity"
	tests.RunToolInvokeParametersTest(t, "query_url", []byte(`{"model": "system__activity", "explore": "look", "fields": ["look.count"]}`), wantResult)

	// A system that is just being used for testing has no looks or dashboards
	wantResult = "null"
	tests.RunToolInvokeParametersTest(t, "get_looks", []byte(`{"title": "FOO", "desc": "BAR"}`), wantResult)

	wantResult = "null"
	tests.RunToolInvokeParametersTest(t, "get_dashboards", []byte(`{"title": "FOO", "desc": "BAR"}`), wantResult)

	runConversationalAnalytics(t, "system__activity", "content_usage")
}

func runConversationalAnalytics(t *testing.T, modelName, exploreName string) {
	exploreRefsJSON := fmt.Sprintf(`[{"model":"%s","explore":"%s"}]`, modelName, exploreName)

	var refs []map[string]any
	if err := json.Unmarshal([]byte(exploreRefsJSON), &refs); err != nil {
		t.Fatalf("failed to unmarshal explore refs: %v", err)
	}

	testCases := []struct {
		name           string
		exploreRefs    []map[string]any
		wantStatusCode int
		wantInResult   string
		wantInError    string
	}{
		{
			name:           "invoke conversational analytics with explore",
			exploreRefs:    refs,
			wantStatusCode: http.StatusOK,
			wantInResult:   `Answer`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			requestBodyMap := map[string]any{
				"user_query_with_context": "What is in the explore?",
				"explore_references":      tc.exploreRefs,
			}
			bodyBytes, err := json.Marshal(requestBodyMap)
			if err != nil {
				t.Fatalf("failed to marshal request body: %v", err)
			}
			url := "http://127.0.0.1:5000/api/tool/conversational_analytics/invoke"
			resp, bodyBytes := tests.RunRequest(t, http.MethodPost, url, bytes.NewBuffer(bodyBytes), nil)

			if resp.StatusCode != tc.wantStatusCode {
				t.Fatalf("unexpected status code: got %d, want %d. Body: %s", resp.StatusCode, tc.wantStatusCode, string(bodyBytes))
			}

			if tc.wantInResult != "" {
				var respBody map[string]interface{}
				if err := json.Unmarshal(bodyBytes, &respBody); err != nil {
					t.Fatalf("error parsing response body: %v", err)
				}
				got, ok := respBody["result"].(string)
				if !ok {
					t.Fatalf("unable to find result in response body")
				}
				if !strings.Contains(got, tc.wantInResult) {
					t.Errorf("unexpected result: got %q, want to contain %q", got, tc.wantInResult)
				}
			}

			if tc.wantInError != "" {
				if !strings.Contains(string(bodyBytes), tc.wantInError) {
					t.Errorf("unexpected error message: got %q, want to contain %q", string(bodyBytes), tc.wantInError)
				}
			}
		})
	}
}
