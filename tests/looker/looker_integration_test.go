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

	"github.com/looker-open-source/sdk-codegen/go/rtl"
	v4 "github.com/looker-open-source/sdk-codegen/go/sdk/v4"
)

var (
	LookerSourceType   = "looker"
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
		"type":          LookerSourceType,
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
				"type":        "looker-get-models",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"get_explores": map[string]any{
				"type":        "looker-get-explores",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"get_dimensions": map[string]any{
				"type":        "looker-get-dimensions",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"get_measures": map[string]any{
				"type":        "looker-get-measures",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"get_filters": map[string]any{
				"type":        "looker-get-filters",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"get_parameters": map[string]any{
				"type":        "looker-get-parameters",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"query": map[string]any{
				"type":        "looker-query",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"query_sql": map[string]any{
				"type":        "looker-query-sql",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"query_url": map[string]any{
				"type":        "looker-query-url",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"get_looks": map[string]any{
				"type":        "looker-get-looks",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"make_look": map[string]any{
				"type":        "looker-make-look",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"get_dashboards": map[string]any{
				"type":        "looker-get-dashboards",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"make_dashboard": map[string]any{
				"type":        "looker-make-dashboard",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"add_dashboard_filter": map[string]any{
				"type":        "looker-add-dashboard-filter",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"add_dashboard_element": map[string]any{
				"type":        "looker-add-dashboard-element",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"conversational_analytics": map[string]any{
				"type":        "looker-conversational-analytics",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"health_pulse": map[string]any{
				"type":        "looker-health-pulse",
				"source":      "my-instance",
				"description": "Checks the health of a Looker instance by running a series of checks on the system.",
			},
			"health_analyze": map[string]any{
				"type":        "looker-health-analyze",
				"source":      "my-instance",
				"description": "Provides analysis of a Looker instance's projects, models, or explores.",
			},
			"health_vacuum": map[string]any{
				"type":        "looker-health-vacuum",
				"source":      "my-instance",
				"description": "Vacuums unused content from a Looker instance.",
			},
			"dev_mode": map[string]any{
				"type":        "looker-dev-mode",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"get_projects": map[string]any{
				"type":        "looker-get-projects",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"get_project_files": map[string]any{
				"type":        "looker-get-project-files",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"get_project_file": map[string]any{
				"type":        "looker-get-project-file",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"create_project_file": map[string]any{
				"type":        "looker-create-project-file",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"update_project_file": map[string]any{
				"type":        "looker-update-project-file",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"delete_project_file": map[string]any{
				"type":        "looker-delete-project-file",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"get_project_directories": map[string]any{
				"type":        "looker-get-project-directories",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"create_project_directory": map[string]any{
				"type":        "looker-create-project-directory",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"delete_project_directory": map[string]any{
				"type":        "looker-delete-project-directory",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"validate_project": map[string]any{
				"type":        "looker-validate-project",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"generate_embed_url": map[string]any{
				"type":        "looker-generate-embed-url",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"get_connections": map[string]any{
				"type":        "looker-get-connections",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"get_connection_schemas": map[string]any{
				"type":        "looker-get-connection-schemas",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"get_connection_databases": map[string]any{
				"type":        "looker-get-connection-databases",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"get_connection_tables": map[string]any{
				"type":        "looker-get-connection-tables",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"get_connection_table_columns": map[string]any{
				"type":        "looker-get-connection-table-columns",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"get_lookml_tests": map[string]any{
				"type":        "looker-get-lookml-tests",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"run_lookml_tests": map[string]any{
				"type":        "looker-run-lookml-tests",
				"source":      "my-instance",
				"description": "Simple tool to test end to end functionality.",
			},
			"create_view_from_table": map[string]any{
				"type":        "looker-create-view-from-table",
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

	wantResult := "{\"connections\":[],\"label\":\"System Activity\",\"name\":\"system__activity\",\"project_name\":\"system__activity\"}"
	tests.RunMCPToolInvokeSimpleTest(t, "get_models", wantResult)

	wantResult = "{\"description\":\"Data about Look and dashboard usage, including frequency of views, favoriting, scheduling, embedding, and access via the API. Also includes details about individual Looks and dashboards.\",\"group_label\":\"System Activity\",\"label\":\"Content Usage\",\"name\":\"content_usage\"}"
	tests.RunMCPToolInvokeParametersTest(t, "get_explores", []byte(`{"model": "system__activity"}`), wantResult)

	wantResult = "{\"description\":\"Number of times this content has been viewed via the Looker API\",\"label\":\"Content Usage API Count\",\"label_short\":\"API Count\",\"name\":\"content_usage.api_count\",\"type\":\"number\"}"
	tests.RunMCPToolInvokeParametersTest(t, "get_dimensions", []byte(`{"model": "system__activity", "explore": "content_usage"}`), wantResult)

	wantResult = "{\"description\":\"The total number of views via the Looker API\",\"label\":\"Content Usage API Total\",\"label_short\":\"API Total\",\"name\":\"content_usage.api_total\",\"type\":\"sum\"}"
	tests.RunMCPToolInvokeParametersTest(t, "get_measures", []byte(`{"model": "system__activity", "explore": "content_usage"}`), wantResult)

	wantResult = "[]"
	tests.RunMCPToolInvokeParametersTest(t, "get_filters", []byte(`{"model": "system__activity", "explore": "content_usage"}`), wantResult)

	wantResult = "[]"
	tests.RunMCPToolInvokeParametersTest(t, "get_parameters", []byte(`{"model": "system__activity", "explore": "content_usage"}`), wantResult)

	wantResult = "{\"look.count\":"
	tests.RunMCPToolInvokeParametersTest(t, "query", []byte(`{"model": "system__activity", "explore": "look", "fields": ["look.count"]}`), wantResult)

	wantResult = "SELECT"
	tests.RunMCPToolInvokeParametersTest(t, "query_sql", []byte(`{"model": "system__activity", "explore": "look", "fields": ["look.count"]}`), wantResult)

	wantResult = "system__activity"
	tests.RunMCPToolInvokeParametersTest(t, "query_url", []byte(`{"model": "system__activity", "explore": "look", "fields": ["look.count"]}`), wantResult)

	// A system that is just being used for testing has no looks or dashboards
	wantResult = "null"
	tests.RunMCPToolInvokeParametersTest(t, "get_looks", []byte(`{"title": "FOO", "desc": "BAR"}`), wantResult)

	wantResult = "null"
	tests.RunMCPToolInvokeParametersTest(t, "get_dashboards", []byte(`{"title": "FOO", "desc": "BAR"}`), wantResult)

	wantResult = "\"Connection\":\"thelook\""
	tests.RunMCPToolInvokeParametersTest(t, "health_pulse", []byte(`{"action": "check_db_connections"}`), wantResult)

	wantResult = "[]"
	tests.RunMCPToolInvokeParametersTest(t, "health_pulse", []byte(`{"action": "check_schedule_failures"}`), wantResult)

	wantResult = "[{\"Feature\":\"Unsupported in Looker (Google Cloud core)\"}]"
	tests.RunMCPToolInvokeParametersTest(t, "health_pulse", []byte(`{"action": "check_legacy_features"}`), wantResult)

	wantResult = "\"Project\":\"the_look\""
	tests.RunMCPToolInvokeParametersTest(t, "health_analyze", []byte(`{"action": "projects"}`), wantResult)

	wantResult = "\"Model\":\"the_look\""
	tests.RunMCPToolInvokeParametersTest(t, "health_analyze", []byte(`{"action": "explores", "project": "the_look", "model": "the_look", "explore": "inventory_items"}`), wantResult)

	wantResult = "\"Model\":\"the_look\""
	tests.RunMCPToolInvokeParametersTest(t, "health_vacuum", []byte(`{"action": "models"}`), wantResult)

	wantResult = "the_look"
	tests.RunMCPToolInvokeSimpleTest(t, "get_projects", wantResult)

	wantResult = "order_items.view"
	tests.RunMCPToolInvokeParametersTest(t, "get_project_files", []byte(`{"project_id": "the_look"}`), wantResult)

	wantResult = "view"
	tests.RunMCPToolInvokeParametersTest(t, "get_project_file", []byte(`{"project_id": "the_look", "file_path": "order_items.view.lkml"}`), wantResult)

	wantResult = "dev"
	tests.RunMCPToolInvokeParametersTest(t, "dev_mode", []byte(`{"devMode": true}`), wantResult)

	wantResult = "created"
	tests.RunMCPToolInvokeParametersTest(t, "create_project_file", []byte(`{"project_id": "the_look", "file_path": "foo.view.lkml", "file_content": "view"}`), wantResult)

	wantResult = "updated"
	tests.RunMCPToolInvokeParametersTest(t, "update_project_file", []byte(`{"project_id": "the_look", "file_path": "foo.view.lkml", "file_content": "model"}`), wantResult)

	wantResult = "deleted"
	tests.RunMCPToolInvokeParametersTest(t, "delete_project_file", []byte(`{"project_id": "the_look", "file_path": "foo.view.lkml"}`), wantResult)

	wantResult = "Created"
	tests.RunMCPToolInvokeParametersTest(t, "create_project_directory", []byte(`{"project_id": "the_look", "directory_path": "views"}`), wantResult)

	wantResult = "views"
	tests.RunMCPToolInvokeParametersTest(t, "get_project_directories", []byte(`{"project_id": "the_look"}`), wantResult)

	// Add test back when infrastructure for testing supports it.
	// wantResult = "{\"status\":  \"success\", \"message\": \"Triggered view generation for project the_look in folder views\"}"
	// tests.RunMCPToolInvokeParametersTest(t, "create_view_from_table", []byte(`{"project_id": "the_look", "connection": "thelook", "tables": [{"schema": "demo_db", "table_name": "Employees"}]}`), wantResult)

	wantResult = "Deleted"
	tests.RunMCPToolInvokeParametersTest(t, "delete_project_directory", []byte(`{"project_id": "the_look", "directory_path": "views"}`), wantResult)

	wantResult = "\"errors\":[]"
	tests.RunMCPToolInvokeParametersTest(t, "validate_project", []byte(`{"project_id": "the_look"}`), wantResult)

	wantResult = "[]"
	tests.RunMCPToolInvokeParametersTest(t, "get_lookml_tests", []byte(`{"project_id": "the_look"}`), wantResult)

	wantResult = "[]"
	tests.RunMCPToolInvokeParametersTest(t, "run_lookml_tests", []byte(`{"project_id": "the_look"}`), wantResult)

	wantResult = "production"
	tests.RunMCPToolInvokeParametersTest(t, "dev_mode", []byte(`{"devMode": false}`), wantResult)

	wantResult = "thelook"
	tests.RunMCPToolInvokeSimpleTest(t, "get_connections", wantResult)

	wantResult = "{\"name\":\"demo_db\",\"is_default\":true}"
	tests.RunMCPToolInvokeParametersTest(t, "get_connection_schemas", []byte(`{"conn": "thelook"}`), wantResult)

	wantResult = "[]"
	tests.RunMCPToolInvokeParametersTest(t, "get_connection_databases", []byte(`{"conn": "thelook"}`), wantResult)

	wantResult = "Employees"
	tests.RunMCPToolInvokeParametersTest(t, "get_connection_tables", []byte(`{"conn": "thelook", "schema": "demo_db"}`), wantResult)

	wantResult = "{\"column_name\":\"EmpID\",\"data_type_database\":\"int\",\"data_type_looker\":\"number\",\"sql_escaped_column_name\":\"EmpID\"}"
	tests.RunMCPToolInvokeParametersTest(t, "get_connection_table_columns", []byte(`{"conn": "thelook", "schema": "demo_db", "tables": "Employees"}`), wantResult)

	wantResult = "/login/embed?t=" // testing for specific substring, since url is dynamic
	tests.RunMCPToolInvokeParametersTest(t, "generate_embed_url", []byte(`{"type": "dashboards", "id": "1"}`), wantResult)

	runConversationalAnalytics(t, "system__activity", "content_usage")

	deleteLook := testMakeLook(t)
	defer deleteLook()

	dashboardId, deleteDashboard := testMakeDashboard(t)
	defer deleteDashboard()
	testAddDashboardFilter(t, dashboardId)
	testAddDashboardElement(t, dashboardId)
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
			url := "http://127.0.0.1:5000/mcp"
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

func newLookerTestSDK(t *testing.T) *v4.LookerSDK {
	t.Helper()
	cfg := rtl.ApiSettings{
		BaseUrl:      LookerBaseUrl,
		ApiVersion:   "4.0",
		VerifySsl:    LookerVerifySsl == "true",
		Timeout:      120,
		ClientId:     LookerClientId,
		ClientSecret: LookerClientSecret,
	}
	return v4.NewLookerSDK(rtl.NewAuthSession(cfg))
}

func testMakeLook(t *testing.T) func() {
	var id string
	t.Run("TestMakeLook", func(t *testing.T) {
		reqBody := []byte(`{"model": "system__activity", "explore": "look", "fields": ["look.count"], "title": "TestLook"}`)

		url := "http://127.0.0.1:5000/mcp"
		resp, bodyBytes := tests.RunRequest(t, http.MethodPost, url, bytes.NewBuffer(reqBody), nil)

		if resp.StatusCode != 200 {
			t.Fatalf("unexpected status code: got %d, want %d. Body: %s", resp.StatusCode, 200, string(bodyBytes))
		}

		var respBody map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &respBody); err != nil {
			t.Fatalf("error parsing response body: %v", err)
		}

		result := respBody["result"].(string)
		if err := json.Unmarshal([]byte(result), &respBody); err != nil {
			t.Fatalf("error parsing result body: %v", err)
		}

		var ok bool
		if id, ok = respBody["id"].(string); !ok || id == "" {
			t.Fatalf("didn't get TestLook id, got %s", string(bodyBytes))
		}
	})

	return func() {
		sdk := newLookerTestSDK(t)

		if _, err := sdk.DeleteLook(id, nil); err != nil {
			t.Fatalf("error deleting look: %v", err)
		}
		t.Logf("deleted Look %s", id)
	}
}

func testAddDashboardFilter(t *testing.T, dashboardId string) {
	t.Run("TestAddDashboardFilter", func(t *testing.T) {
		reqBody := []byte(fmt.Sprintf(`{"dashboard_id": "%s", "model": "system__activity", "explore": "look", "dimension": "look.created_year", "name": "test_filter", "title": "TestDashboardFilter"}`, dashboardId))

		url := "http://127.0.0.1:5000/mcp"
		resp, bodyBytes := tests.RunRequest(t, http.MethodPost, url, bytes.NewBuffer(reqBody), nil)

		if resp.StatusCode != 200 {
			t.Fatalf("unexpected status code: got %d, want %d. Body: %s", resp.StatusCode, 200, string(bodyBytes))
		}

		t.Logf("got %s", string(bodyBytes))
	})
}

func testAddDashboardElement(t *testing.T, dashboardId string) {
	t.Run("TestAddDashboardElement", func(t *testing.T) {
		reqBody := []byte(fmt.Sprintf(`{"dashboard_id": "%s", "model": "system__activity", "explore": "look", "fields": ["look.count"], "title": "TestDashboardElement"}`, dashboardId))

		url := "http://127.0.0.1:5000/mcp"
		resp, bodyBytes := tests.RunRequest(t, http.MethodPost, url, bytes.NewBuffer(reqBody), nil)

		if resp.StatusCode != 200 {
			t.Fatalf("unexpected status code: got %d, want %d. Body: %s", resp.StatusCode, 200, string(bodyBytes))
		}

		t.Logf("got %s", string(bodyBytes))
	})
}

func testMakeDashboard(t *testing.T) (string, func()) {
	var id string
	t.Run("TestMakeDashboard", func(t *testing.T) {
		reqBody := []byte(`{"title": "TestDashboard"}`)

		url := "http://127.0.0.1:5000/mcp"
		resp, bodyBytes := tests.RunRequest(t, http.MethodPost, url, bytes.NewBuffer(reqBody), nil)

		if resp.StatusCode != 200 {
			t.Fatalf("unexpected status code: got %d, want %d. Body: %s", resp.StatusCode, 200, string(bodyBytes))
		}

		var respBody map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &respBody); err != nil {
			t.Fatalf("error parsing response body: %v", err)
		}

		result := respBody["result"].(string)
		if err := json.Unmarshal([]byte(result), &respBody); err != nil {
			t.Fatalf("error parsing result body: %v", err)
		}

		var ok bool
		if id, ok = respBody["id"].(string); !ok || id == "" {
			t.Fatalf("didn't get TestDashboard id, got %s", string(bodyBytes))
		}
	})

	return id, func() {
		sdk := newLookerTestSDK(t)

		if _, err := sdk.DeleteDashboard(id, nil); err != nil {
			t.Fatalf("error deleting dashboard: %v", err)
		}
		t.Logf("deleted Dashboard %s", id)
	}
}
