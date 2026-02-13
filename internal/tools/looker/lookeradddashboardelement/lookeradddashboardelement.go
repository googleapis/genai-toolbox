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
package lookeradddashboardelement

import (
	"context"
	"fmt"
	"net/http"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/embeddingmodels"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/tools/looker/lookercommon"
	"github.com/googleapis/genai-toolbox/internal/util"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"

	"github.com/looker-open-source/sdk-codegen/go/rtl"
	v4 "github.com/looker-open-source/sdk-codegen/go/sdk/v4"
)

const resourceType string = "looker-add-dashboard-element"

func init() {
	if !tools.Register(resourceType, newConfig) {
		panic(fmt.Sprintf("tool type %q already registered", resourceType))
	}
}

func newConfig(ctx context.Context, name string, decoder *yaml.Decoder) (tools.ToolConfig, error) {
	actual := Config{Name: name}
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return nil, err
	}
	return actual, nil
}

type compatibleSource interface {
	UseClientAuthorization() bool
	GetAuthTokenHeaderName() string
	LookerApiSettings() *rtl.ApiSettings
	GetLookerSDK(string) (*v4.LookerSDK, error)
}

type Config struct {
	Name         string                 `yaml:"name" validate:"required"`
	Type         string                 `yaml:"type" validate:"required"`
	Source       string                 `yaml:"source" validate:"required"`
	Description  string                 `yaml:"description" validate:"required"`
	AuthRequired []string               `yaml:"authRequired"`
	Annotations  *tools.ToolAnnotations `yaml:"annotations,omitempty"`
}

// validate interface
var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigType() string {
	return resourceType
}

func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	params := lookercommon.GetQueryParameters()

	dashIdParameter := parameters.NewStringParameter("dashboard_id", "The id of the dashboard where this tile will exist")
	params = append(params, dashIdParameter)
	titleParameter := parameters.NewStringParameterWithDefault("title", "", "The title of the Dashboard Element")
	params = append(params, titleParameter)
	vizParameter := parameters.NewMapParameterWithDefault("vis_config",
		map[string]any{},
		"The visualization config for the query",
		"",
	)
	params = append(params, vizParameter)
	dashFilters := parameters.NewArrayParameterWithRequired("dashboard_filters",
		`An array of dashboard filters like [{"dashboard_filter_name": "name", "field": "view_name.field_name"}, ...]`,
		false,
		parameters.NewMapParameterWithDefault("dashboard_filter",
			map[string]any{},
			`A dashboard filter like {"dashboard_filter_name": "name", "field": "view_name.field_name"}`,
			"",
		),
	)
	params = append(params, dashFilters)

	annotations := cfg.Annotations
	if annotations == nil {
		readOnlyHint := false
		annotations = &tools.ToolAnnotations{
			ReadOnlyHint: &readOnlyHint,
		}
	}

	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, params, annotations)

	// finish tool setup
	return Tool{
		Config:     cfg,
		Parameters: params,
		manifest: tools.Manifest{
			Description:  cfg.Description,
			Parameters:   params.Manifest(),
			AuthRequired: cfg.AuthRequired,
		},
		mcpManifest: mcpManifest,
	}, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Config
	Parameters  parameters.Parameters `yaml:"parameters"`
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

var (
	dataType string = "data"
	visType  string = "vis"
)

func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, util.ToolboxError) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Source, t.Name, t.Type)
	if err != nil {
		return nil, util.NewClientServerError("source used is not compatible with the tool", http.StatusInternalServerError, err)
	}

	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return nil, util.NewClientServerError("unable to get logger from ctx", http.StatusInternalServerError, err)
	}

	logger.DebugContext(ctx, "params = ", params)

	wq, err := lookercommon.ProcessQueryArgs(ctx, params)
	if err != nil {
		return nil, util.NewAgentError("error building query request", err)
	}

	paramsMap := params.AsMap()

	dashboard_id, ok := paramsMap["dashboard_id"].(string)
	if !ok {
		return nil, util.NewAgentError("dashboard_id parameter missing or invalid", nil)
	}

	title, ok := paramsMap["title"].(string)
	if !ok {
		title = ""
	}

	visConfig, ok := paramsMap["vis_config"].(map[string]any)
	if !ok {
		visConfig = make(map[string]any)
	}
	wq.VisConfig = &visConfig

	sdk, err := source.GetLookerSDK(string(accessToken))
	if err != nil {
		return nil, util.NewClientServerError("error getting sdk", http.StatusInternalServerError, err)
	}

	qresp, err := sdk.CreateQuery(*wq, "id", source.LookerApiSettings())
	if err != nil {
		return nil, util.ProcessGeneralError(err)
	}

	dashFilters := []any{}
	if v, ok := paramsMap["dashboard_filters"]; ok {
		if v != nil {
			if df, ok := v.([]any); ok {
				dashFilters = df
			}
		}
	}

	var filterables []v4.ResultMakerFilterables
	for _, m := range dashFilters {
		f, ok := m.(map[string]any)
		if !ok {
			return nil, util.NewAgentError("invalid dashboard filter structure", nil)
		}
		name, ok := f["dashboard_filter_name"].(string)
		if !ok {
			return nil, util.NewAgentError("error processing dashboard filter: missing dashboard_filter_name", nil)
		}
		field, ok := f["field"].(string)
		if !ok {
			return nil, util.NewAgentError("error processing dashboard filter: missing field", nil)
		}
		listener := v4.ResultMakerFilterablesListen{
			DashboardFilterName: &name,
			Field:               &field,
		}
		listeners := []v4.ResultMakerFilterablesListen{listener}

		filter := v4.ResultMakerFilterables{
			Listen: &listeners,
		}

		filterables = append(filterables, filter)
	}

	if len(filterables) == 0 {
		filterables = nil
	}

	wrm := v4.WriteResultMakerWithIdVisConfigAndDynamicFields{
		Query:       wq,
		VisConfig:   &visConfig,
		Filterables: &filterables,
	}
	wde := v4.WriteDashboardElement{
		DashboardId: &dashboard_id,
		Title:       &title,
		ResultMaker: &wrm,
		Query:       wq,
		QueryId:     qresp.Id,
	}

	switch len(visConfig) {
	case 0:
		wde.Type = &dataType
	default:
		wde.Type = &visType
	}

	fields := ""

	req := v4.RequestCreateDashboardElement{
		Body:   wde,
		Fields: &fields,
	}

	resp, err := sdk.CreateDashboardElement(req, source.LookerApiSettings())
	if err != nil {
		return nil, util.ProcessGeneralError(err)
	}
	logger.DebugContext(ctx, "resp = %v", resp)

	data := make(map[string]any)

	data["result"] = fmt.Sprintf("Dashboard element added to dashboard %s", dashboard_id)

	return data, nil
}

func (t Tool) EmbedParams(ctx context.Context, paramValues parameters.ParamValues, embeddingModelsMap map[string]embeddingmodels.EmbeddingModel) (parameters.ParamValues, error) {
	return parameters.EmbedParams(ctx, t.Parameters, paramValues, embeddingModelsMap, nil)
}

func (t Tool) Manifest() tools.Manifest {
	return t.manifest
}

func (t Tool) McpManifest() tools.McpManifest {
	return t.mcpManifest
}

func (t Tool) RequiresClientAuthorization(resourceMgr tools.SourceProvider) (bool, error) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Source, t.Name, t.Type)
	if err != nil {
		return false, err
	}
	return source.UseClientAuthorization(), nil
}

func (t Tool) Authorized(verifiedAuthServices []string) bool {
	return tools.IsAuthorized(t.AuthRequired, verifiedAuthServices)
}

func (t Tool) GetAuthTokenHeaderName(resourceMgr tools.SourceProvider) (string, error) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Source, t.Name, t.Type)
	if err != nil {
		return "", err
	}
	return source.GetAuthTokenHeaderName(), nil
}

func (t Tool) GetParameters() parameters.Parameters {
	return t.Parameters
}
