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
package lookergetprojectfiles

import (
	"context"
	"fmt"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/tools/looker/lookercommon"
	"github.com/googleapis/genai-toolbox/internal/util"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"

	"github.com/looker-open-source/sdk-codegen/go/rtl"
	v4 "github.com/looker-open-source/sdk-codegen/go/sdk/v4"
)

const kind string = "looker-get-project-files"

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
	projectIdParameter := parameters.NewStringParameter("project_id", "The id of the project containing the files")
	params := parameters.Parameters{projectIdParameter}

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
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Source, t.Name, t.Kind)
	if err != nil {
		return nil, err
	}

	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get logger from ctx: %s", err)
	}

	sdk, err := lookercommon.GetLookerSDK(source.UseClientAuthorization(), source.LookerApiSettings(), source.LookerClient(), accessToken)
	if err != nil {
		return nil, fmt.Errorf("error getting sdk: %w", err)
	}

	mapParams := params.AsMap()
	projectId, ok := mapParams["project_id"].(string)
	if !ok {
		return nil, fmt.Errorf("'project_id' must be a string, got %T", mapParams["project_id"])
	}

	resp, err := sdk.AllProjectFiles(projectId, "", source.LookerApiSettings())
	if err != nil {
		return nil, fmt.Errorf("error making get_project_files request: %s", err)
	}

	var data []any
	for _, v := range resp {
		logger.DebugContext(ctx, "Got response element of %v\n", v)
		vMap := make(map[string]any)
		if v.Id != nil {
			vMap["id"] = *v.Id
		}
		if v.Path != nil {
			vMap["path"] = *v.Path
		}
		if v.Title != nil {
			vMap["title"] = *v.Title
		}
		if v.Type != nil {
			vMap["type"] = *v.Type
		}
		if v.Extension != nil {
			vMap["extension"] = *v.Extension
		}
		if v.Editable != nil {
			vMap["editable"] = *v.Editable
		}
		logger.DebugContext(ctx, "Converted to %v\n", vMap)
		data = append(data, vMap)
	}
	logger.DebugContext(ctx, "data = ", data)

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
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Source, t.Name, t.Kind)
	if err != nil {
		return false, err
	}
	return source.UseClientAuthorization(), nil
}

func (t Tool) Authorized(verifiedAuthServices []string) bool {
	return tools.IsAuthorized(t.AuthRequired, verifiedAuthServices)
}

func (t Tool) GetAuthTokenHeaderName(resourceMgr tools.SourceProvider) (string, error) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Source, t.Name, t.Kind)
	if err != nil {
		return "", err
	}
	return source.GetAuthTokenHeaderName(), nil
}
