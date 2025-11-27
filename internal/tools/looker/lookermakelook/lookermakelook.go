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
package lookermakelook

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/tools/looker/lookercommon"
	"github.com/googleapis/genai-toolbox/internal/util"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"

	"github.com/looker-open-source/sdk-codegen/go/rtl"
	v4 "github.com/looker-open-source/sdk-codegen/go/sdk/v4"
)

const kind string = "looker-make-look"

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

type compatibleSource interface {
	UseClientAuthorization() bool
	GetAuthTokenHeaderName() string
	LookerClient() *v4.LookerSDK
	LookerApiSettings() *rtl.ApiSettings
}

type Config struct {
	Name         string   `yaml:"name" validate:"required"`
	Kind         string   `yaml:"kind" validate:"required"`
	Source       string   `yaml:"source" validate:"required"`
	Description  string   `yaml:"description" validate:"required"`
	AuthRequired []string `yaml:"authRequired"`
}

// validate interface
var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigKind() string {
	return kind
}

func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	params := lookercommon.GetQueryParameters()

	titleParameter := parameters.NewStringParameter("title", "The title of the Look")
	params = append(params, titleParameter)
	descParameter := parameters.NewStringParameterWithDefault("description", "", "The description of the Look")
	params = append(params, descParameter)
	vizParameter := parameters.NewMapParameterWithDefault("vis_config",
		map[string]any{},
		"The visualization config for the query",
		"",
	)
	params = append(params, vizParameter)

	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, params)

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

func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	s, ok := resourceMgr.GetSource(t.Source)
	if !ok {
		return nil, fmt.Errorf("unable to retrieve source %s in tool %s", t.Source, t.Name)
	}
	source, ok := s.(compatibleSource)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source %q not compatible", kind, t.Source)
	}

	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get logger from ctx: %s", err)
	}
	logger.DebugContext(ctx, "params = ", params)
	wq, err := lookercommon.ProcessQueryArgs(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("error building query request: %w", err)
	}

	sdk, err := lookercommon.GetLookerSDK(source.UseClientAuthorization(), source.LookerApiSettings(), source.LookerClient(), accessToken)
	if err != nil {
		return nil, fmt.Errorf("error getting sdk: %w", err)
	}
	mrespFields := "id,personal_folder_id"
	mresp, err := sdk.Me(mrespFields, source.LookerApiSettings())
	if err != nil {
		return nil, fmt.Errorf("error making me request: %s", err)
	}

	paramsMap := params.AsMap()
	title := paramsMap["title"].(string)
	description := paramsMap["description"].(string)

	looks, err := sdk.FolderLooks(*mresp.PersonalFolderId, "title", source.LookerApiSettings())
	if err != nil {
		return nil, fmt.Errorf("error getting existing looks in folder: %s", err)
	}

	lookTitles := []string{}
	for _, look := range looks {
		lookTitles = append(lookTitles, *look.Title)
	}
	if slices.Contains(lookTitles, title) {
		lt, _ := json.Marshal(lookTitles)
		return nil, fmt.Errorf("title %s already used in user's folder. Currently used titles are %v. Make the call again with a unique title", title, string(lt))
	}

	visConfig := paramsMap["vis_config"].(map[string]any)
	wq.VisConfig = &visConfig

	qrespFields := "id"
	qresp, err := sdk.CreateQuery(*wq, qrespFields, source.LookerApiSettings())
	if err != nil {
		return nil, fmt.Errorf("error making create query request: %s", err)
	}

	wlwq := v4.WriteLookWithQuery{
		Title:       &title,
		UserId:      mresp.Id,
		Description: &description,
		QueryId:     qresp.Id,
		FolderId:    mresp.PersonalFolderId,
	}
	resp, err := sdk.CreateLook(wlwq, "", source.LookerApiSettings())
	if err != nil {
		return nil, fmt.Errorf("error making create look request: %s", err)
	}
	logger.DebugContext(ctx, "resp = %v", resp)

	setting, err := sdk.GetSetting("host_url", source.LookerApiSettings())
	if err != nil {
		logger.ErrorContext(ctx, "error getting settings: %s", err)
	}

	data := make(map[string]any)
	if resp.Id != nil {
		data["id"] = *resp.Id
	}
	if resp.ShortUrl != nil {
		if setting.HostUrl != nil {
			data["short_url"] = *setting.HostUrl + *resp.ShortUrl
		} else {
			data["short_url"] = *resp.ShortUrl
		}
	}
	logger.DebugContext(ctx, "data = %v", data)

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

func (t Tool) RequiresClientAuthorization(resourceMgr tools.SourceProvider) (bool, error) {
	s, ok := resourceMgr.GetSource(t.Source)
	if !ok {
		return false, fmt.Errorf("unable to retrieve source %s in tool %s", t.Source, t.Name)
	}

	source, ok := s.(compatibleSource)
	if !ok {
		return false, fmt.Errorf("invalid source for %q tool: source %q not compatible", kind, t.Source)
	}
	return source.UseClientAuthorization(), nil
}

func (t Tool) Authorized(verifiedAuthServices []string) bool {
	return tools.IsAuthorized(t.AuthRequired, verifiedAuthServices)
}

func (t Tool) GetAuthTokenHeaderName(resourceMgr tools.SourceProvider) (string, error) {
	s, ok := resourceMgr.GetSource(t.Source)
	if !ok {
		return "", fmt.Errorf("unable to retrieve source %s in tool %s", t.Source, t.Name)
	}

	source, ok := s.(compatibleSource)
	if !ok {
		return "", fmt.Errorf("invalid source for %q tool: source %q not compatible", kind, t.Source)
	}
	return source.GetAuthTokenHeaderName(), nil
}
