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
package lookergetlooks

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

const kind string = "looker-get-looks"

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

	titleParameter := tools.NewStringParameterWithDefault("title", "", "The title of the look.")
	descParameter := tools.NewStringParameterWithDefault("desc", "", "The description of the look.")
	limitParameter := tools.NewIntParameterWithDefault("limit", 100, "The number of looks to fetch. Default 100")
	offsetParameter := tools.NewIntParameterWithDefault("offset", 0, "The number of looks to skip before fetching. Default 0")
	parameters := tools.Parameters{
		titleParameter,
		descParameter,
		limitParameter,
		offsetParameter,
	}

	mcpManifest := tools.McpManifest{
		Name:        cfg.Name,
		Description: cfg.Description,
		InputSchema: parameters.McpManifest(),
	}

	// finish tool setup
	return Tool{
		Name:         cfg.Name,
		Kind:         kind,
		Parameters:   parameters,
		AuthRequired: cfg.AuthRequired,
		ApiSettings:  s.ApiSettings,
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
	Name         string `yaml:"name"`
	Kind         string `yaml:"kind"`
	ApiSettings  *rtl.ApiSettings
	AuthRequired []string         `yaml:"authRequired"`
	Parameters   tools.Parameters `yaml:"parameters"`
	manifest     tools.Manifest
	mcpManifest  tools.McpManifest
}

func (t Tool) Invoke(ctx context.Context, params tools.ParamValues, accessToken tools.AccessToken) (any, error) {
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get logger from ctx: %s", err)
	}
	paramsMap := params.AsMap()
	title := paramsMap["title"].(string)
	title_ptr := &title
	if *title_ptr == "" {
		title_ptr = nil
	}
	desc := paramsMap["desc"].(string)
	desc_ptr := &desc
	if *desc_ptr == "" {
		desc_ptr = nil
	}
	limit := int64(paramsMap["limit"].(int))
	offset := int64(paramsMap["offset"].(int))

	sdk := lookercommon.GetLookerSDK(t.ApiSettings, accessToken)
	req := v4.RequestSearchLooks{
		Title:       title_ptr,
		Description: desc_ptr,
		Limit:       &limit,
		Offset:      &offset,
	}
	resp, err := sdk.SearchLooks(req, t.ApiSettings)
	if err != nil {
		return nil, fmt.Errorf("error making get_looks request: %s", err)
	}

	var data []any
	for _, v := range resp {
		logger.DebugContext(ctx, "Got response element of %v\n", v)
		vMap := make(map[string]any)
		if v.Id != nil {
			vMap["id"] = *v.Id
		}
		if v.Title != nil {
			vMap["title"] = *v.Title
		}
		if v.Description != nil {
			vMap["description"] = *v.Description
		}
		vMap["model_id"] = *v.Model.Id
		logger.DebugContext(ctx, "Converted to %v\n", vMap)
		data = append(data, vMap)
	}
	logger.DebugContext(ctx, "data = ", data)

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
	return false
}
