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

package lookersearchpermissionsets

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

const (
	kind = "looker-search-permission-sets"
)

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

func (c Config) ToolConfigKind() string {
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

	params := parameters.Parameters{
		parameters.NewStringParameterWithRequired("name", "The name of the permission set.", false),
		parameters.NewIntParameterWithRequired("id", "The unique id of the permission set.", false),
		parameters.NewIntParameterWithDefault("limit", 100, "The number of permission sets to fetch. Default is 100"),
		parameters.NewIntParameterWithDefault("offset", 0, "The number of permission sets to skip before fetching. Default 0"),
	}

	annotations := cfg.Annotations
	if annotations == nil {
		readOnlyHint := true
		annotations = &tools.ToolAnnotations{
			ReadOnlyHint: &readOnlyHint,
		}
	}

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
		mcpManifest: tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, params, annotations),
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
	Parameters          parameters.Parameters
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

	paramsMap := params.AsMap()

	var namePtr *string
	if name, ok := paramsMap["name"].(string); ok && name != "" {
		namePtr = &name
	}

	var idPtr *string
	if id, ok := paramsMap["id"].(int); ok {
		idStr := fmt.Sprintf("%d", id)
		idPtr = &idStr
	}

	var limitPtr *int64
	if limit, ok := paramsMap["limit"].(int); ok {
		limit64 := int64(limit)
		limitPtr = &limit64
	}

	var offsetPtr *int64
	if offset, ok := paramsMap["offset"].(int); ok {
		offset64 := int64(offset)
		offsetPtr = &offset64
	}

	sdk, err := lookercommon.GetLookerSDK(t.UseClientOAuth, t.ApiSettings, t.Client, accessToken)
	if err != nil {
		return nil, fmt.Errorf("error getting sdk: %w", err)
	}

	req := v4.RequestSearchPermissionSets{
		Name:   namePtr,
		Id:     idPtr,
		Limit:  limitPtr,
		Offset: offsetPtr,
	}

	resp, err := sdk.SearchPermissionSets(req, t.ApiSettings)
	if err != nil {
		return nil, fmt.Errorf("error searching permission sets: %w", err)
	}

	var data []map[string]any
	for _, v := range resp {
		vMap := make(map[string]any)
		if v.Id != nil {
			vMap["id"] = *v.Id
		}
		if v.Name != nil {
			vMap["name"] = *v.Name
		}
		if v.Permissions != nil {
			vMap["permissions"] = *v.Permissions
		}
		if v.AllAccess != nil {
			vMap["all_access"] = *v.AllAccess
		}
		logger.DebugContext(ctx, "Converted to %v\n", vMap)
		data = append(data, vMap)
	}

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
