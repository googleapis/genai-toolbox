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
package lookerqueryurl

import (
	"context"
	"fmt"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	lookersrc "github.com/googleapis/genai-toolbox/internal/sources/looker"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/tools/looker/lookercommon"
	"github.com/googleapis/genai-toolbox/internal/util"

	"github.com/looker-open-source/sdk-codegen/go/rtl"
	v4 "github.com/looker-open-source/sdk-codegen/go/sdk/v4"
)

const toolType string = "looker-query-url"

func init() {
	if !tools.Register(toolType, newConfig) {
		panic(fmt.Sprintf("tool type %q already registered", toolType))
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
	Name         string   `yaml:"name" validate:"required"`
	Type         string   `yaml:"type" validate:"required"`
	Source       string   `yaml:"source" validate:"required"`
	Description  string   `yaml:"description" validate:"required"`
	AuthRequired []string `yaml:"authRequired"`
}

// validate interface
var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigType() string {
	return toolType
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
		return nil, fmt.Errorf("invalid source for %q tool: source type must be `looker`", toolType)
	}

	parameters := lookercommon.GetQueryParameters()

	vizParameter := tools.NewMapParameterWithDefault("vis_config",
		map[string]any{},
		"The visualization config for the query",
		"",
	)
	parameters = append(parameters, vizParameter)

	mcpManifest := tools.McpManifest{
		Name:        cfg.Name,
		Description: cfg.Description,
		InputSchema: parameters.McpManifest(),
	}

	// finish tool setup
	return Tool{
		Name:           cfg.Name,
		Type:           toolType,
		Parameters:     parameters,
		AuthRequired:   cfg.AuthRequired,
		UseClientOAuth: s.UseClientOAuth,
		Client:         s.Client,
		ApiSettings:    s.ApiSettings,
		manifest: tools.Manifest{
			Description:  cfg.Description,
			Parameters:   parameters.Manifest(),
			AuthRequired: cfg.AuthRequired,
		},
		mcpManifest: mcpManifest,
	}, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Name           string `yaml:"name"`
	Type           string `yaml:"type"`
	UseClientOAuth bool
	Client         *v4.LookerSDK
	ApiSettings    *rtl.ApiSettings
	AuthRequired   []string         `yaml:"authRequired"`
	Parameters     tools.Parameters `yaml:"parameters"`
	manifest       tools.Manifest
	mcpManifest    tools.McpManifest
}

func (t Tool) Invoke(ctx context.Context, params tools.ParamValues, accessToken tools.AccessToken) (any, error) {
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get logger from ctx: %s", err)
	}
	logger.DebugContext(ctx, "params = ", params)
	wq, err := lookercommon.ProcessQueryArgs(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("error building query request: %w", err)
	}

	paramsMap := params.AsMap()
	visConfig := paramsMap["vis_config"].(map[string]any)
	wq.VisConfig = &visConfig

	sdk, err := lookercommon.GetLookerSDK(t.UseClientOAuth, t.ApiSettings, t.Client, accessToken)
	if err != nil {
		return nil, fmt.Errorf("error getting sdk: %w", err)
	}
	respFields := "id,slug,share_url,expanded_share_url"
	resp, err := sdk.CreateQuery(*wq, respFields, t.ApiSettings)
	if err != nil {
		return nil, fmt.Errorf("error making query request: %s", err)
	}
	logger.DebugContext(ctx, "resp = ", resp)

	data := make(map[string]any)
	if resp.Id != nil {
		data["id"] = *resp.Id
	}
	if resp.Slug != nil {
		data["slug"] = *resp.Slug
	}
	if resp.ShareUrl != nil {
		data["url"] = *resp.ShareUrl
	}
	if resp.ExpandedShareUrl != nil {
		data["long_url"] = *resp.ExpandedShareUrl
	}
	logger.DebugContext(ctx, "data = %v", data)

	return data, nil
}

func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (tools.ParamValues, error) {
	return tools.ParseParams(t.Parameters, data, claims)
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
