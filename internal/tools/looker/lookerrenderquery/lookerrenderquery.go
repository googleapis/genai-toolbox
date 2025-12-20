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
package lookerrenderquery

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

const kind string = "looker-render-query"

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

	queryidParameter := parameters.NewStringParameter("query_id", "The id of the query to render.")
	// An 8.5 x 11 piece of paper with .25 inch margins works out to 4
	// quarters each 1200 x 1575, assuming 300 dpi. So that is a good
	// default for the size of the rendering.
	// An A4 paper is close enough to accomodate this too.
	widthParameter := parameters.NewIntParameterWithDefault("width", 1575, "The image width. Default 1575")
	heightParameter := parameters.NewIntParameterWithDefault("height", 1200, "The image height. Default 1200")
	formatParameter := parameters.NewStringParameterWithDefault("format", "png", "The image type: png or jpg")

	params := parameters.Parameters{
		queryidParameter,
		widthParameter,
		heightParameter,
		formatParameter,
	}

	annotations := cfg.Annotations
	if annotations == nil {
		readOnlyHint := true
		annotations = &tools.ToolAnnotations{
			ReadOnlyHint: &readOnlyHint,
		}
	}

	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, params, annotations)

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

// validate interface
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

func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get logger from ctx: %s", err)
	}
	logger.DebugContext(ctx, "params = ", params)
	paramsMap := params.AsMap()

	query_id := paramsMap["query_id"].(string)
	width := int64(paramsMap["width"].(int))
	height := int64(paramsMap["height"].(int))
	format := paramsMap["format"].(string)

	var mimeType string
	switch format {
	case "jpg":
		mimeType = "image/jpeg"
	case "png":
		mimeType = "image/png"
	default:
		return nil, fmt.Errorf("format \"%s\" unsupported. must be png or jpg", format)
	}

	applyVis := true

	req := v4.RequestRunQuery{
		QueryId:      query_id,
		ResultFormat: format,
		ApplyVis:     &applyVis,
		ImageWidth:   &width,
		ImageHeight:  &height,
	}

	sdk, err := lookercommon.GetLookerSDK(t.UseClientOAuth, t.ApiSettings, t.Client, accessToken)
	if err != nil {
		return nil, fmt.Errorf("error getting sdk: %w", err)
	}

	resp, err := sdk.RunQuery(req, t.ApiSettings)
	if err != nil {
		return nil, fmt.Errorf("error making run_query request: %s", err)
	}

	return lookercommon.ReturnImage(mimeType, resp), nil
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
