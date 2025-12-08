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
package lookeradddashboardfilter

import (
	"context"
	"fmt"

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

const kind string = "looker-add-dashboard-filter"

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
	Annotations  *tools.ToolAnnotations `yaml:"annotations,omitempty"`
}

// validate interface
var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigKind() string {
	return kind
}

func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	// verify source exists
	rawS, ok := srcs[cfg.Source]
	if !ok {
		return nil, fmt.Errorf("no source named %q configured", cfg.Source)
	}

	// verify the source is compatible
	s, ok := rawS.(*lookersrc.Source)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be `looker`", kind)
	}

	params := parameters.Parameters{}

	dashIdParameter := parameters.NewStringParameter("dashboard_id", "The id of the dashboard where this filter will exist")
	params = append(params, dashIdParameter)
	nameParameter := parameters.NewStringParameter("name", "The name of the Dashboard Filter")
	params = append(params, nameParameter)
	titleParameter := parameters.NewStringParameter("title", "The title of the Dashboard Filter")
	params = append(params, titleParameter)
	typeParameter := parameters.NewStringParameterWithDefault("type", "field", "The type of the Dashboard Filter: date_filter, number_filter, string_filter, field_filter (default field_filter)")
	params = append(params, typeParameter)
	defaultParameter := parameters.NewStringParameterWithRequired("default_value", "The default_value of the Dashboard Filter (optional)", false)
	params = append(params, defaultParameter)
	modelParameter := parameters.NewStringParameterWithRequired("model", "The model of a field type Dashboard Filter (required if type field)", false)
	params = append(params, modelParameter)
	exploreParameter := parameters.NewStringParameterWithRequired("explore", "The explore of a field type Dashboard Filter (required if type field)", false)
	params = append(params, exploreParameter)
	dimensionParameter := parameters.NewStringParameterWithRequired("dimension", "The dimension of a field type Dashboard Filter (required if type field)", false)
	params = append(params, dimensionParameter)
	multiValueParameter := parameters.NewBooleanParameterWithDefault("allow_multiple_values", true, "The Dashboard Filter should allow multiple values (default true)")
	params = append(params, multiValueParameter)
	requiredParameter := parameters.NewBooleanParameterWithDefault("required", false, "The Dashboard Filter is required to run dashboard (default false)")
	params = append(params, requiredParameter)

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
		Config:              cfg,
		Name:                cfg.Name,
		Kind:                kind,
		UseClientOAuth:      s.UseClientAuthorization(),
		AuthTokenHeaderName: s.GetAuthTokenHeaderName(),
		Client:              s.Client,
		ApiSettings:         s.ApiSettings,
		Parameters:          params,
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
	Name                string `yaml:"name"`
	Kind                string `yaml:"kind"`
	UseClientOAuth      bool
	AuthTokenHeaderName string
	Client              *v4.LookerSDK
	ApiSettings         *rtl.ApiSettings
	AuthRequired        []string              `yaml:"authRequired"`
	Parameters          parameters.Parameters `yaml:"parameters"`
	manifest            tools.Manifest
	mcpManifest         tools.McpManifest
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get logger from ctx: %s", err)
	}
	logger.DebugContext(ctx, "params = ", params)

	paramsMap := params.AsMap()
	dashboard_id := paramsMap["dashboard_id"].(string)
	name := paramsMap["name"].(string)
	title := paramsMap["title"].(string)
	filterType := paramsMap["type"].(string)
	switch filterType {
	case "date_filter":
	case "number_filter":
	case "string_filter":
	case "field_filter":
	default:
		return nil, fmt.Errorf("invalid filter type: %s. Must be one of date_filter, number_filter, string_filter, field_filter", filterType)
	}
	allowMultipleValues := paramsMap["allow_multiple_values"].(bool)
	required := paramsMap["required"].(bool)

	req := v4.WriteCreateDashboardFilter{
		DashboardId:         dashboard_id,
		Name:                name,
		Title:               title,
		Type:                filterType,
		AllowMultipleValues: &allowMultipleValues,
		Required:            &required,
	}

	if v, ok := paramsMap["default_value"]; ok {
		if v != nil {
			defaultValue := paramsMap["default_value"].(string)
			req.DefaultValue = &defaultValue
		}
	}

	if filterType == "field_filter" {
		model, ok := paramsMap["model"].(string)
		if !ok || model == "" {
			return nil, fmt.Errorf("model must be specified for field_filter type")
		}
		explore, ok := paramsMap["explore"].(string)
		if !ok || explore == "" {
			return nil, fmt.Errorf("explore must be specified for field_filter type")
		}
		dimension, ok := paramsMap["dimension"].(string)
		if !ok || dimension == "" {
			return nil, fmt.Errorf("dimension must be specified for field_filter type")
		}

		req.Model = &model
		req.Explore = &explore
		req.Dimension = &dimension
	}

	sdk, err := lookercommon.GetLookerSDK(t.UseClientOAuth, t.ApiSettings, t.Client, accessToken)
	if err != nil {
		return nil, fmt.Errorf("error getting sdk: %w", err)
	}

	resp, err := sdk.CreateDashboardFilter(req, "name", t.ApiSettings)
	if err != nil {
		return nil, fmt.Errorf("error making create dashboard filter request: %s", err)
	}
	logger.DebugContext(ctx, "resp = %v", resp)

	data := make(map[string]any)

	data["result"] = fmt.Sprintf("Dashboard filter \"%s\" added to dashboard %s", *resp.Name, dashboard_id)

	return data, nil
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

func (t Tool) RequiresClientAuthorization(resourceMgr tools.SourceProvider) bool {
	return t.UseClientOAuth
}

func (t Tool) GetAuthTokenHeaderName() string {
	return t.AuthTokenHeaderName
}
