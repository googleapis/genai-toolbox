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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
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
			"health_pulse": map[string]any{
				"kind":        "looker-health-pulse",
				"source":      "my-instance",
				"description": "Checks the health of a Looker instance by running a series of checks on the system.",
			},
			"health_analyze": map[string]any{
				"kind":        "looker-health-analyze",
				"source":      "my-instance",
				"description": "Provides analysis of a Looker instance's projects, models, or explores.",
			},
			"health_vacuum": map[string]any{
				"kind":        "looker-health-vacuum",
				"source":      "my-instance",
				"description": "Vacuums unused content from a Looker instance.",
			},
			"dev_mode": map[string]any{
				"kind":        "looker-dev-mode",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"get_projects": map[string]any{
				"kind":        "looker-get-projects",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"get_project_files": map[string]any{
				"kind":        "looker-get-project-files",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"get_project_file": map[string]any{
				"kind":        "looker-get-project-file",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"create_project_file": map[string]any{
				"kind":        "looker-create-project-file",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"update_project_file": map[string]any{
				"kind":        "looker-update-project-file",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"delete_project_file": map[string]any{
				"kind":        "looker-delete-project-file",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"generate_embed_url": map[string]any{
				"kind":        "looker-generate-embed-url",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"get_connections": map[string]any{
				"kind":        "looker-get-connections",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"get_connection_schemas": map[string]any{
				"kind":        "looker-get-connection-schemas",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"get_connection_databases": map[string]any{
				"kind":        "looker-get-connection-databases",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"get_connection_tables": map[string]any{
				"kind":        "looker-get-connection-tables",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"get_connection_table_columns": map[string]any{
				"kind":        "looker-get-connection-table-columns",
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
	tests.RunToolGetTestByName(t, "health_pulse",
		map[string]any{
			"health_pulse": map[string]any{
				"description":  "Checks the health of a Looker instance by running a series of checks on the system.",
				"authRequired": []any{},
				"parameters": []any{
					map[string]any{
						"authSources": []any{},
						"description": "The health check to run. Can be either: `check_db_connections`, `check_dashboard_performance`,`check_dashboard_errors`,`check_explore_performance`,`check_schedule_failures`, or `check_legacy_features`",
						"name":        "action",
						"required":    true,
						"type":        "string",
					},
				},
			},
		},
	)
	tests.RunToolGetTestByName(t, "health_analyze",
		map[string]any{
			"health_analyze": map[string]any{
				"description":  "Provides analysis of a Looker instance's projects, models, or explores.",
				"authRequired": []any{},
				"parameters": []any{
					map[string]any{
						"authSources": []any{},
						"description": "The analysis to run. Can be 'projects', 'models', or 'explores'.",
						"name":        "action",
						"required":    true,
						"type":        "string",
					},
					map[string]any{
						"authSources": []any{},
						"description": "The Looker project to analyze (optional).",
						"name":        "project",
						"required":    false,
						"type":        "string",
					},
					map[string]any{
						"authSources": []any{},
						"description": "The Looker model to analyze (optional).",
						"name":        "model",
						"required":    false,
						"type":        "string",
					},
					map[string]any{
						"authSources": []any{},
						"description": "The Looker explore to analyze (optional).",
						"name":        "explore",
						"required":    false,
						"type":        "string",
					},
					map[string]any{
						"authSources": []any{},
						"description": "The timeframe in days to analyze.",
						"name":        "timeframe",
						"required":    false,
						"type":        "integer",
					},
					map[string]any{
						"authSources": []any{},
						"description": "The minimum number of queries for a model or explore to be considered used.",
						"name":        "min_queries",
						"required":    false,
						"type":        "integer",
					},
				},
			},
		},
	)
	tests.RunToolGetTestByName(t, "health_vacuum",
		map[string]any{
			"health_vacuum": map[string]any{
				"description":  "Vacuums unused content from a Looker instance.",
				"authRequired": []any{},
				"parameters": []any{
					map[string]any{
						"authSources": []any{},
						"description": "The vacuum action to run. Can be 'models', or 'explores'.",
						"name":        "action",
						"required":    true,
						"type":        "string",
					},
					map[string]any{
						"authSources": []any{},
						"description": "The Looker project to vacuum (optional).",
						"name":        "project",
						"required":    false,
						"type":        "string",
					},
					map[string]any{
						"authSources": []any{},
						"description": "The Looker model to vacuum (optional).",
						"name":        "model",
						"required":    false,
						"type":        "string",
					},
					map[string]any{
						"authSources": []any{},
						"description": "The Looker explore to vacuum (optional).",
						"name":        "explore",
						"required":    false,
						"type":        "string",
					},
					map[string]any{
						"authSources": []any{},
						"description": "The timeframe in days to analyze.",
						"name":        "timeframe",
						"required":    false,
						"type":        "integer",
					},
					map[string]any{
						"authSources": []any{},
						"description": "The minimum number of queries for a model or explore to be considered used.",
						"name":        "min_queries",
						"required":    false,
						"type":        "integer",
					},
				},
			},
		},
	)
	tests.RunToolGetTestByName(t, "dev_mode",
		map[string]any{
			"dev_mode": map[string]any{
				"description":  "Simple tool to test end to end functionality.",
				"authRequired": []any{},
				"parameters": []any{
					map[string]any{
						"authSources": []any{},
						"description": "Whether to set Dev Mode.",
						"name":        "devMode",
						"required":    false,
						"type":        "boolean",
					},
				},
			},
		},
	)
	tests.RunToolGetTestByName(t, "get_projects",
		map[string]any{
			"get_projects": map[string]any{
				"description":  "Simple tool to test end to end functionality.",
				"authRequired": []any{},
				"parameters":   []any{},
			},
		},
	)
	tests.RunToolGetTestByName(t, "get_project_files",
		map[string]any{
			"get_project_files": map[string]any{
				"description":  "Simple tool to test end to end functionality.",
				"authRequired": []any{},
				"parameters": []any{
					map[string]any{
						"authSources": []any{},
						"description": "The id of the project containing the files",
						"name":        "project_id",
						"required":    true,
						"type":        "string",
					},
				},
			},
		},
	)
	tests.RunToolGetTestByName(t, "get_project_file",
		map[string]any{
			"get_project_file": map[string]any{
				"description":  "Simple tool to test end to end functionality.",
				"authRequired": []any{},
				"parameters": []any{
					map[string]any{
						"authSources": []any{},
						"description": "The id of the project containing the files",
						"name":        "project_id",
						"required":    true,
						"type":        "string",
					},
					map[string]any{
						"authSources": []any{},
						"description": "The path of the file within the project",
						"name":        "file_path",
						"required":    true,
						"type":        "string",
					},
				},
			},
		},
	)
	tests.RunToolGetTestByName(t, "create_project_file",
		map[string]any{
			"create_project_file": map[string]any{
				"description":  "Simple tool to test end to end functionality.",
				"authRequired": []any{},
				"parameters": []any{
					map[string]any{
						"authSources": []any{},
						"description": "The id of the project containing the files",
						"name":        "project_id",
						"required":    true,
						"type":        "string",
					},
					map[string]any{
						"authSources": []any{},
						"description": "The path of the file within the project",
						"name":        "file_path",
						"required":    true,
						"type":        "string",
					},
					map[string]any{
						"authSources": []any{},
						"description": "The content of the file",
						"name":        "file_content",
						"required":    true,
						"type":        "string",
					},
				},
			},
		},
	)
	tests.RunToolGetTestByName(t, "update_project_file",
		map[string]any{
			"update_project_file": map[string]any{
				"description":  "Simple tool to test end to end functionality.",
				"authRequired": []any{},
				"parameters": []any{
					map[string]any{
						"authSources": []any{},
						"description": "The id of the project containing the files",
						"name":        "project_id",
						"required":    true,
						"type":        "string",
					},
					map[string]any{
						"authSources": []any{},
						"description": "The path of the file within the project",
						"name":        "file_path",
						"required":    true,
						"type":        "string",
					},
					map[string]any{
						"authSources": []any{},
						"description": "The content of the file",
						"name":        "file_content",
						"required":    true,
						"type":        "string",
					},
				},
			},
		},
	)
	tests.RunToolGetTestByName(t, "delete_project_file",
		map[string]any{
			"delete_project_file": map[string]any{
				"description":  "Simple tool to test end to end functionality.",
				"authRequired": []any{},
				"parameters": []any{
					map[string]any{
						"authSources": []any{},
						"description": "The id of the project containing the files",
						"name":        "project_id",
						"required":    true,
						"type":        "string",
					},
					map[string]any{
						"authSources": []any{},
						"description": "The path of the file within the project",
						"name":        "file_path",
						"required":    true,
						"type":        "string",
					},
				},
			},
		},
	)
	tests.RunToolGetTestByName(t, "generate_embed_url",
		map[string]any{
			"generate_embed_url": map[string]any{
				"description":  "Simple tool to test end to end functionality.",
				"authRequired": []any{},
				"parameters": []any{
					map[string]any{
						"authSources": []any{},
						"description": "Type of Looker content to embed (ie. dashboards, looks, query-visualization)",
						"name":        "type",
						"required":    false,
						"type":        "string",
					},
					map[string]any{
						"authSources": []any{},
						"description": "The ID of the content to embed.",
						"name":        "id",
						"required":    false,
						"type":        "string",
					},
				},
			},
		},
	)
	tests.RunToolGetTestByName(t, "get_connections",
		map[string]any{
			"get_connections": map[string]any{
				"description":  "Simple tool to test end to end functionality.",
				"authRequired": []any{},
				"parameters":   []any{},
			},
		},
	)
	tests.RunToolGetTestByName(t, "get_connection_schemas",
		map[string]any{
			"get_connection_schemas": map[string]any{
				"description":  "Simple tool to test end to end functionality.",
				"authRequired": []any{},
				"parameters": []any{
					map[string]any{
						"authSources": []any{},
						"description": "The connection containing the schemas.",
						"name":        "conn",
						"required":    true,
						"type":        "string",
					},
					map[string]any{
						"authSources": []any{},
						"description": "The optional database to search",
						"name":        "db",
						"required":    false,
						"type":        "string",
					},
				},
			},
		},
	)
	tests.RunToolGetTestByName(t, "get_connection_databases",
		map[string]any{
			"get_connection_databases": map[string]any{
				"description":  "Simple tool to test end to end functionality.",
				"authRequired": []any{},
				"parameters": []any{
					map[string]any{
						"authSources": []any{},
						"description": "The connection containing the databases.",
						"name":        "conn",
						"required":    true,
						"type":        "string",
					},
				},
			},
		},
	)
	tests.RunToolGetTestByName(t, "get_connection_tables",
		map[string]any{
			"get_connection_tables": map[string]any{
				"description":  "Simple tool to test end to end functionality.",
				"authRequired": []any{},
				"parameters": []any{
					map[string]any{
						"authSources": []any{},
						"description": "The connection containing the tables.",
						"name":        "conn",
						"required":    true,
						"type":        "string",
					},
					map[string]any{
						"authSources": []any{},
						"description": "The optional database to search",
						"name":        "db",
						"required":    false,
						"type":        "string",
					},
					map[string]any{
						"authSources": []any{},
						"description": "The schema containing the tables.",
						"name":        "schema",
						"required":    true,
						"type":        "string",
					},
				},
			},
		},
	)
	tests.RunToolGetTestByName(t, "get_connection_table_columns",
		map[string]any{
			"get_connection_table_columns": map[string]any{
				"description":  "Simple tool to test end to end functionality.",
				"authRequired": []any{},
				"parameters": []any{
					map[string]any{
						"authSources": []any{},
						"description": "The connection containing the tables.",
						"name":        "conn",
						"required":    true,
						"type":        "string",
					},
					map[string]any{
						"authSources": []any{},
						"description": "The optional database to search",
						"name":        "db",
						"required":    false,
						"type":        "string",
					},
					map[string]any{
						"authSources": []any{},
						"description": "The schema containing the tables.",
						"name":        "schema",
						"required":    true,
						"type":        "string",
					},
					map[string]any{
						"authSources": []any{},
						"description": "A comma separated list of tables containing the columns.",
						"name":        "tables",
						"required":    true,
						"type":        "string",
					},
				},
			},
		},
	)

	wantResult := "{\"connections\":[],\"label\":\"System Activity\",\"name\":\"system__activity\",\"project_name\":\"system__activity\"}"
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

	wantResult = "\"Connection\":\"thelook\""
	tests.RunToolInvokeParametersTest(t, "health_pulse", []byte(`{"action": "check_db_connections"}`), wantResult)

	wantResult = "[]"
	tests.RunToolInvokeParametersTest(t, "health_pulse", []byte(`{"action": "check_schedule_failures"}`), wantResult)

	wantResult = "[{\"Feature\":\"Unsupported in Looker (Google Cloud core)\"}]"
	tests.RunToolInvokeParametersTest(t, "health_pulse", []byte(`{"action": "check_legacy_features"}`), wantResult)

	wantResult = "\"Project\":\"the_look\""
	tests.RunToolInvokeParametersTest(t, "health_analyze", []byte(`{"action": "projects"}`), wantResult)

	wantResult = "\"Model\":\"the_look\""
	tests.RunToolInvokeParametersTest(t, "health_analyze", []byte(`{"action": "explores", "project": "the_look", "model": "the_look", "explore": "inventory_items"}`), wantResult)

	wantResult = "\"Model\":\"the_look\""
	tests.RunToolInvokeParametersTest(t, "health_vacuum", []byte(`{"action": "models"}`), wantResult)

	wantResult = "the_look"
	tests.RunToolInvokeSimpleTest(t, "get_projects", wantResult)

	wantResult = "order_items.view"
	tests.RunToolInvokeParametersTest(t, "get_project_files", []byte(`{"project_id": "the_look"}`), wantResult)

	wantResult = "view"
	tests.RunToolInvokeParametersTest(t, "get_project_file", []byte(`{"project_id": "the_look", "file_path": "order_items.view.lkml"}`), wantResult)

	wantResult = "dev"
	tests.RunToolInvokeParametersTest(t, "dev_mode", []byte(`{"devMode": true}`), wantResult)

	wantResult = "created"
	tests.RunToolInvokeParametersTest(t, "create_project_file", []byte(`{"project_id": "the_look", "file_path": "foo.view.lkml", "file_content": "view"}`), wantResult)

	wantResult = "updated"
	tests.RunToolInvokeParametersTest(t, "update_project_file", []byte(`{"project_id": "the_look", "file_path": "foo.view.lkml", "file_content": "model"}`), wantResult)

	wantResult = "deleted"
	tests.RunToolInvokeParametersTest(t, "delete_project_file", []byte(`{"project_id": "the_look", "file_path": "foo.view.lkml"}`), wantResult)

	wantResult = "production"
	tests.RunToolInvokeParametersTest(t, "dev_mode", []byte(`{"devMode": false}`), wantResult)

	wantResult = "thelook"
	tests.RunToolInvokeSimpleTest(t, "get_connections", wantResult)

	wantResult = "{\"name\":\"demo_db\",\"is_default\":true}"
	tests.RunToolInvokeParametersTest(t, "get_connection_schemas", []byte(`{"conn": "thelook"}`), wantResult)

	wantResult = "[]"
	tests.RunToolInvokeParametersTest(t, "get_connection_databases", []byte(`{"conn": "thelook"}`), wantResult)

	wantResult = "Employees"
	tests.RunToolInvokeParametersTest(t, "get_connection_tables", []byte(`{"conn": "thelook", "schema": "demo_db"}`), wantResult)

	wantResult = "{\"column_name\":\"EmpID\",\"data_type_database\":\"int\",\"data_type_looker\":\"number\",\"sql_escaped_column_name\":\"EmpID\"}"
	tests.RunToolInvokeParametersTest(t, "get_connection_table_columns", []byte(`{"conn": "thelook", "schema": "demo_db", "tables": "Employees"}`), wantResult)

	wantResult = "/login/embed?t=" // testing for specific substring, since url is dynamic
	tests.RunToolInvokeParametersTest(t, "generate_embed_url", []byte(`{"type": "dashboards", "id": "1"}`), wantResult)
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
