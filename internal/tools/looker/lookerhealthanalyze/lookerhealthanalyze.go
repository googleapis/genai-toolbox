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
package lookerhealthanalyze

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
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
const kind string = "looker-health-analyze"

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

	actionParameter := parameters.NewStringParameterWithRequired("action", "The analysis to run. Can be 'projects', 'models', or 'explores'.", true)
	projectParameter := parameters.NewStringParameterWithRequired("project", "The Looker project to analyze (optional).", false)
	modelParameter := parameters.NewStringParameterWithRequired("model", "The Looker model to analyze (optional).", false)
	exploreParameter := parameters.NewStringParameterWithRequired("explore", "The Looker explore to analyze (optional).", false)
	timeframeParameter := parameters.NewIntParameterWithDefault("timeframe", 90, "The timeframe in days to analyze.")
	minQueriesParameter := parameters.NewIntParameterWithDefault("min_queries", 0, "The minimum number of queries for a model or explore to be considered used.")

	params := parameters.Parameters{
		actionParameter,
		projectParameter,
		modelParameter,
		exploreParameter,
		timeframeParameter,
		minQueriesParameter,
	}

	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, params, cfg.Annotations)

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
	Parameters          parameters.Parameters
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

	paramsMap := params.AsMap()
	timeframe, _ := paramsMap["timeframe"].(int)
	if timeframe == 0 {
		timeframe = 90
	}
	minQueries, _ := paramsMap["min_queries"].(int)
	if minQueries == 0 {
		minQueries = 1
	}

	analyzeTool := &analyzeTool{
		SdkClient:  sdk,
		timeframe:  timeframe,
		minQueries: minQueries,
	}

	action, ok := paramsMap["action"].(string)
	if !ok {
		return nil, fmt.Errorf("action parameter not found")
	}

	switch action {
	case "projects":
		projectId, _ := paramsMap["project"].(string)
		result, err := analyzeTool.projects(ctx, projectId)
		if err != nil {
			return nil, fmt.Errorf("error analyzing projects: %w", err)
		}
		logger.DebugContext(ctx, "result = ", result)
		return result, nil
	case "models":
		projectName, _ := paramsMap["project"].(string)
		modelName, _ := paramsMap["model"].(string)
		result, err := analyzeTool.models(ctx, projectName, modelName)
		if err != nil {
			return nil, fmt.Errorf("error analyzing models: %w", err)
		}
		logger.DebugContext(ctx, "result = ", result)
		return result, nil
	case "explores":
		modelName, _ := paramsMap["model"].(string)
		exploreName, _ := paramsMap["explore"].(string)
		result, err := analyzeTool.explores(ctx, modelName, exploreName)
		if err != nil {
			return nil, fmt.Errorf("error analyzing explores: %w", err)
		}
		logger.DebugContext(ctx, "result = ", result)
		return result, nil
	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}
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
// START LOOKER HEALTH ANALYZE CORE LOGIC
// =================================================================================================================
type analyzeTool struct {
	SdkClient  *v4.LookerSDK
	timeframe  int
	minQueries int
}

func (t *analyzeTool) projects(ctx context.Context, id string) ([]map[string]interface{}, error) {
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get logger from ctx: %s", err)
	}

	var projects []*v4.Project
	if id != "" {
		p, err := t.SdkClient.Project(id, "", nil)
		if err != nil {
			return nil, fmt.Errorf("error fetching project %s: %w", id, err)
		}
		projects = append(projects, &p)
	} else {
		allProjects, err := t.SdkClient.AllProjects("", nil)
		if err != nil {
			return nil, fmt.Errorf("error fetching all projects: %w", err)
		}
		for i := range allProjects {
			projects = append(projects, &allProjects[i])
		}
	}

	var results []map[string]interface{}
	for _, p := range projects {
		pName := *p.Name
		pID := *p.Id
		logger.InfoContext(ctx, fmt.Sprintf("Analyzing project: %s", pName))

		projectFiles, err := t.SdkClient.AllProjectFiles(pID, "", nil)
		if err != nil {
			return nil, fmt.Errorf("error fetching files for project %s: %w", pName, err)
		}

		modelCount := 0
		viewFileCount := 0
		for _, f := range projectFiles {
			if f.Type != nil {
				if *f.Type == "model" {
					modelCount++
				}
				if *f.Type == "view" {
					viewFileCount++
				}
			}
		}

		gitConnectionStatus := "OK"
		if p.GitRemoteUrl == nil {
			gitConnectionStatus = "No repo found"
		} else if strings.Contains(*p.GitRemoteUrl, "/bare_models/") {
			gitConnectionStatus = "Bare repo, no tests required"
		}

		results = append(results, map[string]interface{}{
			"Project":                pName,
			"# Models":               modelCount,
			"# View Files":           viewFileCount,
			"Git Connection Status":  gitConnectionStatus,
			"PR Mode":                string(*p.PullRequestMode),
			"Is Validation Required": *p.ValidationRequired,
		})
	}
	return results, nil
}

func (t *analyzeTool) models(ctx context.Context, project, model string) ([]map[string]interface{}, error) {
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get logger from ctx: %s", err)
	}
	logger.InfoContext(ctx, "Analyzing models...")

	usedModels, err := t.getUsedModels(ctx)
	if err != nil {
		return nil, err
	}

	lookmlModels, err := t.SdkClient.AllLookmlModels(v4.RequestAllLookmlModels{}, nil)
	if err != nil {
		return nil, fmt.Errorf("error fetching LookML models: %w", err)
	}

	var results []map[string]interface{}
	for _, m := range lookmlModels {
		if (project == "" || (m.ProjectName != nil && *m.ProjectName == project)) &&
			(model == "" || (m.Name != nil && *m.Name == model)) {

			queryCount := 0
			if qc, ok := usedModels[*m.Name]; ok {
				queryCount = qc
			}

			exploreCount := 0
			if m.Explores != nil {
				exploreCount = len(*m.Explores)
			}

			results = append(results, map[string]interface{}{
				"Project":     *m.ProjectName,
				"Model":       *m.Name,
				"# Explores":  exploreCount,
				"Query Count": queryCount,
			})
		}
	}
	return results, nil
}

func (t *analyzeTool) getUsedModels(ctx context.Context) (map[string]int, error) {
	limit := "5000"
	query := &v4.WriteQuery{
		Model:  "system__activity",
		View:   "history",
		Fields: &[]string{"history.query_run_count", "query.model"},
		Filters: &map[string]any{
			"history.created_date":    fmt.Sprintf("%d days", t.timeframe),
			"query.model":             "-system__activity, -i__looker",
			"history.query_run_count": fmt.Sprintf(">%d", t.minQueries-1),
			"user.dev_branch_name":    "NULL",
		},
		Limit: &limit,
	}
	raw, err := lookercommon.RunInlineQuery(ctx, t.SdkClient, query, "json", nil)
	if err != nil {
		return nil, err
	}

	var data []map[string]interface{}
	_ = json.Unmarshal([]byte(raw), &data)

	results := make(map[string]int)
	for _, row := range data {
		model, _ := row["query.model"].(string)
		count, _ := row["history.query_run_count"].(float64)
		results[model] = int(count)
	}
	return results, nil
}

func (t *analyzeTool) getUsedExploreFields(ctx context.Context, model, explore string) (map[string]int, error) {
	limit := "5000"
	query := &v4.WriteQuery{
		Model:  "system__activity",
		View:   "history",
		Fields: &[]string{"query.formatted_fields", "query.filters", "history.query_run_count"},
		Filters: &map[string]any{
			"history.created_date":   fmt.Sprintf("%d days", t.timeframe),
			"query.model":            strings.ReplaceAll(model, "_", "^_"),
			"query.view":             strings.ReplaceAll(explore, "_", "^_"),
			"query.formatted_fields": "-NULL",
			"history.workspace_id":   "production",
		},
		Limit: &limit,
	}
	raw, err := lookercommon.RunInlineQuery(ctx, t.SdkClient, query, "json", nil)
	if err != nil {
		return nil, err
	}

	var data []map[string]interface{}
	_ = json.Unmarshal([]byte(raw), &data)

	results := make(map[string]int)
	fieldRegex := regexp.MustCompile(`(\w+\.\w+)`)

	for _, row := range data {
		count, _ := row["history.query_run_count"].(float64)
		formattedFields, _ := row["query.formatted_fields"].(string)
		filters, _ := row["query.filters"].(string)

		usedFields := make(map[string]bool)

		for _, field := range fieldRegex.FindAllString(formattedFields, -1) {
			results[field] += int(count)
			usedFields[field] = true
		}

		for _, field := range fieldRegex.FindAllString(filters, -1) {
			if _, ok := usedFields[field]; !ok {
				results[field] += int(count)
			}
		}
	}
	return results, nil
}

func (t *analyzeTool) explores(ctx context.Context, model, explore string) ([]map[string]interface{}, error) {
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get logger from ctx: %s", err)
	}
	logger.InfoContext(ctx, "Analyzing explores...")

	lookmlModels, err := t.SdkClient.AllLookmlModels(v4.RequestAllLookmlModels{}, nil)
	if err != nil {
		return nil, fmt.Errorf("error fetching LookML models: %w", err)
	}

	var results []map[string]interface{}
	for _, m := range lookmlModels {
		if model != "" && (m.Name == nil || *m.Name != model) {
			continue
		}
		if m.Explores == nil {
			continue
		}

		for _, e := range *m.Explores {
			if explore != "" && (e.Name == nil || *e.Name != explore) {
				continue
			}
			if e.Name == nil {
				continue
			}

			// Get detailed explore info to count fields and joins
			req := v4.RequestLookmlModelExplore{
				LookmlModelName: *m.Name,
				ExploreName:     *e.Name,
			}
			exploreDetail, err := t.SdkClient.LookmlModelExplore(req, nil)
			if err != nil {
				// Log the error but continue to the next explore if possible
				logger.ErrorContext(ctx, fmt.Sprintf("Error fetching detail for explore %s.%s: %v", *m.Name, *e.Name, err))
				continue
			}

			fieldCount := 0
			if exploreDetail.Fields != nil {
				fieldCount = len(*exploreDetail.Fields.Dimensions) + len(*exploreDetail.Fields.Measures)
			}

			joinCount := 0
			if exploreDetail.Joins != nil {
				joinCount = len(*exploreDetail.Joins)
			}

			usedFields, err := t.getUsedExploreFields(ctx, *m.Name, *e.Name)
			if err != nil {
				logger.ErrorContext(ctx, fmt.Sprintf("Error fetching used fields for explore %s.%s: %v", *m.Name, *e.Name, err))
				continue
			}

			allFields := []string{}
			if exploreDetail.Fields != nil {
				for _, d := range *exploreDetail.Fields.Dimensions {
					if !*d.Hidden {
						allFields = append(allFields, *d.Name)
					}
				}
				for _, ms := range *exploreDetail.Fields.Measures {
					if !*ms.Hidden {
						allFields = append(allFields, *ms.Name)
					}
				}
			}

			unusedFieldsCount := 0
			for _, field := range allFields {
				if _, ok := usedFields[field]; !ok {
					unusedFieldsCount++
				}
			}

			joinStats := make(map[string]int)
			if exploreDetail.Joins != nil {
				for field, queryCount := range usedFields {
					join := strings.Split(field, ".")[0]
					joinStats[join] += queryCount
				}
				for _, join := range *exploreDetail.Joins {
					if _, ok := joinStats[*join.Name]; !ok {
						joinStats[*join.Name] = 0
					}
				}
			}

			unusedJoinsCount := 0
			for _, count := range joinStats {
				if count == 0 {
					unusedJoinsCount++
				}
			}

			// Use an inline query to get query count for the explore
			limit := "1"
			queryCountQueryBody := &v4.WriteQuery{
				Model:  "system__activity",
				View:   "history",
				Fields: &[]string{"history.query_run_count"},
				Filters: &map[string]any{
					"query.model":             *m.Name,
					"query.view":              *e.Name,
					"history.created_date":    fmt.Sprintf("%d days", t.timeframe),
					"history.query_run_count": fmt.Sprintf(">%d", t.minQueries-1),
					"user.dev_branch_name":    "NULL",
				},
				Limit: &limit,
			}

			rawQueryCount, err := lookercommon.RunInlineQuery(ctx, t.SdkClient, queryCountQueryBody, "json", nil)
			if err != nil {
				return nil, err
			}
			queryCount := 0
			var data []map[string]interface{}
			_ = json.Unmarshal([]byte(rawQueryCount), &data)
			if len(data) > 0 {
				if count, ok := data[0]["history.query_run_count"].(float64); ok {
					queryCount = int(count)
				}
			}

			results = append(results, map[string]interface{}{
				"Model":           *m.Name,
				"Explore":         *e.Name,
				"Is Hidden":       *e.Hidden,
				"Has Description": e.Description != nil && *e.Description != "",
				"# Joins":         joinCount,
				"# Unused Joins":  unusedJoinsCount,
				"# Unused Fields": unusedFieldsCount,
				"# Fields":        fieldCount,
				"Query Count":     queryCount,
			})
		}
	}
	return results, nil
}

// =================================================================================================================
// END LOOKER HEALTH ANALYZE CORE LOGIC
// =================================================================================================================

func (t Tool) GetAuthTokenHeaderName() string {
	return t.AuthTokenHeaderName
}
