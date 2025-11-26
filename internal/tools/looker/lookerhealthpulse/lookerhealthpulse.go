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
package lookerhealthpulse

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	lookersrc "github.com/googleapis/genai-toolbox/internal/sources/looker"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/tools/looker/lookercommon"
	"github.com/googleapis/genai-toolbox/internal/util"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"

	"github.com/looker-open-source/sdk-codegen/go/rtl"
	v4 "github.com/looker-open-source/sdk-codegen/go/sdk/v4"
)

// =================================================================================================================
// START MCP SERVER CORE LOGIC
// =================================================================================================================
const kind string = "looker-health-pulse"

func init() {
	if !tools.Register(kind, newConfig) {
		panic(fmt.Sprintf("tool kind %q already registered", kind))
	}
}

func newConfig(ctx context.Context, name string, decoder *yaml.Decoder) (tools.ToolConfig, error) {
	actual := Config{Name: name}
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return nil, err
	}
	return actual, nil
}

type Config struct {
	Name         string                 `yaml:"name" validate:"required"`
	Kind         string                 `yaml:"kind" validate:"required"`
	Source       string                 `yaml:"source" validate:"required"`
	Description  string                 `yaml:"description" validate:"required"`
	AuthRequired []string               `yaml:"authRequired"`
	Parameters   map[string]any         `yaml:"parameters"`
	Annotations  *tools.ToolAnnotations `yaml:"annotations,omitempty"`
}

var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigKind() string {
	return kind
}

func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	rawS, ok := srcs[cfg.Source]
	if !ok {
		return nil, fmt.Errorf("no source named %q configured", cfg.Source)
	}

	s, ok := rawS.(*lookersrc.Source)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be `looker`", kind)
	}

	actionParameter := parameters.NewStringParameterWithRequired("action", "The health check to run. Can be either: `check_db_connections`, `check_dashboard_performance`,`check_dashboard_errors`,`check_explore_performance`,`check_schedule_failures`, or `check_legacy_features`", true)

	params := parameters.Parameters{
		actionParameter,
	}

	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, params, cfg.Annotations)

	// finish tool setup
	return Tool{
		Config:              cfg,
		Parameters:          params,
		UseClientOAuth:      s.UseClientAuthorization(),
		AuthTokenHeaderName: s.GetAuthTokenHeaderName(),
		Client:              s.Client,
		ApiSettings:         s.ApiSettings,
		manifest: tools.Manifest{
			Description:  cfg.Description,
			Parameters:   params.Manifest(),
			AuthRequired: cfg.AuthRequired,
		},
		mcpManifest: mcpManifest,
	}, nil
}

var _ tools.Tool = Tool{}

type Tool struct {
	Config
	UseClientOAuth      bool
	AuthTokenHeaderName string
	Client              *v4.LookerSDK
	ApiSettings         *rtl.ApiSettings
	Parameters          parameters.Parameters `yaml:"parameters"`
	manifest            tools.Manifest
	mcpManifest         tools.McpManifest
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

func (t Tool) Invoke(ctx context.Context, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get logger from ctx: %s", err)
	}

	sdk, err := lookercommon.GetLookerSDK(t.UseClientOAuth, t.ApiSettings, t.Client, accessToken)
	if err != nil {
		return nil, fmt.Errorf("error getting sdk: %w", err)
	}

	pulseTool := &pulseTool{
		ApiSettings: t.ApiSettings,
		SdkClient:   sdk,
	}

	paramsMap := params.AsMap()
	action, ok := paramsMap["action"].(string)
	if !ok {
		return nil, fmt.Errorf("action parameter not found")
	}

	pulseParams := PulseParams{
		Action: action,
	}

	result, err := pulseTool.RunPulse(ctx, pulseParams)
	if err != nil {
		return nil, fmt.Errorf("error running pulse: %w", err)
	}

	logger.DebugContext(ctx, "result = ", result)

	return result, nil
}

func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (parameters.ParamValues, error) {
	return parameters.ParseParams(t.Parameters, data, claims)
}

func (t Tool) Manifest() tools.Manifest {
	return t.manifest
}

func (t Tool) McpManifest() tools.McpManifest {
	return t.mcpManifest
}

func (t Tool) Authorized(verifiedAuthServices []string) bool {
	return tools.IsAuthorized(t.AuthRequired, verifiedAuthServices)
}

func (t Tool) RequiresClientAuthorization() bool {
	return t.UseClientOAuth
}

// =================================================================================================================
// END MCP SERVER CORE LOGIC
// =================================================================================================================

// =================================================================================================================
// START LOOKER HEALTH PULSE CORE LOGIC
// =================================================================================================================
type PulseParams struct {
	Action string
	// Optionally add more parameters if needed
}

// pulseTool holds Looker API settings and client
type pulseTool struct {
	ApiSettings *rtl.ApiSettings
	SdkClient   *v4.LookerSDK
}

func (t *pulseTool) RunPulse(ctx context.Context, params PulseParams) (interface{}, error) {
	switch params.Action {
	case "check_db_connections":
		return t.checkDBConnections(ctx)
	case "check_dashboard_performance":
		return t.checkDashboardPerformance(ctx)
	case "check_dashboard_errors":
		return t.checkDashboardErrors(ctx)
	case "check_explore_performance":
		return t.checkExplorePerformance(ctx)
	case "check_schedule_failures":
		return t.checkScheduleFailures(ctx)
	case "check_legacy_features":
		return t.checkLegacyFeatures(ctx)
	default:
		return nil, fmt.Errorf("unknown action: %s", params.Action)
	}
}

// Check DB connections and run tests
func (t *pulseTool) checkDBConnections(ctx context.Context) (interface{}, error) {
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get logger from ctx: %s", err)
	}
	logger.InfoContext(ctx, "Test 1/6: Checking connections")

	reservedNames := map[string]struct{}{
		"looker__internal__analytics__replica": {},
		"looker__internal__analytics":          {},
		"looker":                               {},
		"looker__ilooker":                      {},
	}

	connections, err := t.SdkClient.AllConnections("", t.ApiSettings)
	if err != nil {
		return nil, fmt.Errorf("error fetching connections: %w", err)
	}

	var filteredConnections []v4.DBConnection
	for _, c := range connections {
		if _, reserved := reservedNames[*c.Name]; !reserved {
			filteredConnections = append(filteredConnections, c)
		}
	}
	if len(filteredConnections) == 0 {
		return nil, fmt.Errorf("no connections found")
	}

	var results []map[string]interface{}
	for _, conn := range filteredConnections {
		var errors []string
		// Test connection (simulate test_connection endpoint)
		resp, err := t.SdkClient.TestConnection(*conn.Name, nil, t.ApiSettings)
		if err != nil {
			errors = append(errors, "API JSONDecode Error")
		} else {
			for _, r := range resp {
				if *r.Status == "error" {
					errors = append(errors, *r.Message)
				}
			}
		}

		// Run inline query for connection activity
		limit := "1"
		query := &v4.WriteQuery{
			Model:  "system__activity",
			View:   "history",
			Fields: &[]string{"history.query_run_count"},
			Filters: &map[string]any{
				"history.connection_name": *conn.Name,
				"history.created_date":    "90 days",
				"user.dev_branch_name":    "NULL",
			},
			Limit: &limit,
		}
		raw, err := lookercommon.RunInlineQuery(ctx, t.SdkClient, query, "json", t.ApiSettings)
		if err != nil {
			return nil, err
		}
		var queryRunCount interface{}
		var data []map[string]interface{}
		_ = json.Unmarshal([]byte(raw), &data)
		if len(data) > 0 {
			queryRunCount = data[0]["history.query_run_count"]
		}

		results = append(results, map[string]interface{}{
			"Connection":  *conn.Name,
			"Status":      "OK",
			"Errors":      errors,
			"Query Count": queryRunCount,
		})
	}
	return results, nil
}

func (t *pulseTool) checkDashboardPerformance(ctx context.Context) (interface{}, error) {
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get logger from ctx: %s", err)
	}
	logger.InfoContext(ctx, "Test 2/6: Checking for dashboards with queries slower than 30 seconds in the last 7 days")

	limit := "20"
	query := &v4.WriteQuery{
		Model:  "system__activity",
		View:   "history",
		Fields: &[]string{"dashboard.title", "query.count"},
		Filters: &map[string]any{
			"history.created_date": "7 days",
			"history.real_dash_id": "-NULL",
			"history.runtime":      ">30",
			"history.status":       "complete",
		},
		Sorts: &[]string{"query.count desc"},
		Limit: &limit,
	}
	raw, err := lookercommon.RunInlineQuery(ctx, t.SdkClient, query, "json", t.ApiSettings)
	if err != nil {
		return nil, err
	}
	var dashboards []map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &dashboards); err != nil {
		return nil, err
	}
	return dashboards, nil
}

func (t *pulseTool) checkDashboardErrors(ctx context.Context) (interface{}, error) {
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get logger from ctx: %s", err)
	}
	logger.InfoContext(ctx, "Test 3/6: Checking for dashboards with erroring queries in the last 7 days")

	limit := "20"
	query := &v4.WriteQuery{
		Model:  "system__activity",
		View:   "history",
		Fields: &[]string{"dashboard.title", "history.query_run_count"},
		Filters: &map[string]any{
			"dashboard.title":           "-NULL",
			"history.created_date":      "7 days",
			"history.dashboard_session": "-NULL",
			"history.status":            "error",
		},
		Sorts: &[]string{"history.query_run_count desc"},
		Limit: &limit,
	}
	raw, err := lookercommon.RunInlineQuery(ctx, t.SdkClient, query, "json", t.ApiSettings)
	if err != nil {
		return nil, err
	}
	var dashboards []map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &dashboards); err != nil {
		return nil, err
	}
	return dashboards, nil
}

func (t *pulseTool) checkExplorePerformance(ctx context.Context) (interface{}, error) {
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get logger from ctx: %s", err)
	}
	logger.InfoContext(ctx, "Test 4/6: Checking for the slowest explores in the past 7 days")

	limit := "20"
	query := &v4.WriteQuery{
		Model:  "system__activity",
		View:   "history",
		Fields: &[]string{"query.model", "query.view", "history.average_runtime"},
		Filters: &map[string]any{
			"history.created_date": "7 days",
			"query.model":          "-NULL, -system^_^_activity",
		},
		Sorts: &[]string{"history.average_runtime desc"},
		Limit: &limit,
	}
	raw, err := lookercommon.RunInlineQuery(ctx, t.SdkClient, query, "json", t.ApiSettings)
	if err != nil {
		return nil, err
	}
	var explores []map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &explores); err != nil {
		return nil, err
	}

	// Average query runtime
	query.Fields = &[]string{"history.average_runtime"}
	rawAvg, err := lookercommon.RunInlineQuery(ctx, t.SdkClient, query, "json", t.ApiSettings)
	if err != nil {
		return nil, err
	}
	var avgData []map[string]interface{}
	if err := json.Unmarshal([]byte(rawAvg), &avgData); err == nil {
		if len(avgData) > 0 {
			if avgRuntime, ok := avgData[0]["history.average_runtime"].(float64); ok {
				logger.InfoContext(ctx, fmt.Sprintf("For context, the average query runtime is %.4fs", avgRuntime))
			}
		}
	}
	return explores, nil
}

func (t *pulseTool) checkScheduleFailures(ctx context.Context) (interface{}, error) {
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get logger from ctx: %s", err)
	}
	logger.InfoContext(ctx, "Test 5/6: Checking for failing schedules")

	limit := "500"
	query := &v4.WriteQuery{
		Model:  "system__activity",
		View:   "scheduled_plan",
		Fields: &[]string{"scheduled_job.name", "scheduled_job.count"},
		Filters: &map[string]any{
			"scheduled_job.created_date": "7 days",
			"scheduled_job.status":       "failure",
		},
		Sorts: &[]string{"scheduled_job.count desc"},
		Limit: &limit,
	}
	raw, err := lookercommon.RunInlineQuery(ctx, t.SdkClient, query, "json", t.ApiSettings)
	if err != nil {
		return nil, err
	}
	var schedules []map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &schedules); err != nil {
		return nil, err
	}
	return schedules, nil
}

func (t *pulseTool) checkLegacyFeatures(ctx context.Context) (interface{}, error) {
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get logger from ctx: %s", err)
	}
	logger.InfoContext(ctx, "Test 6/6: Checking for enabled legacy features")

	features, err := t.SdkClient.AllLegacyFeatures(t.ApiSettings)
	if err != nil {
		if strings.Contains(err.Error(), "Unsupported in Looker (Google Cloud core)") {
			return []map[string]string{{"Feature": "Unsupported in Looker (Google Cloud core)"}}, nil
		}
		logger.ErrorContext(ctx, err.Error())
		return []map[string]string{{"Feature": "Unable to pull legacy features due to SDK error"}}, nil
	}
	var legacyFeatures []map[string]string
	for _, f := range features {
		if *f.Enabled {
			legacyFeatures = append(legacyFeatures, map[string]string{"Feature": *f.Name})
		}
	}
	return legacyFeatures, nil
}

// =================================================================================================================
// END LOOKER HEALTH PULSE CORE LOGIC
// =================================================================================================================

func (t Tool) GetAuthTokenHeaderName() string {
	return t.AuthTokenHeaderName
}
