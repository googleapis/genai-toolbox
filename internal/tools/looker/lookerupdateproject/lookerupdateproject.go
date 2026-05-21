// Copyright 2026 Google LLC
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
package lookerupdateproject

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/mcp-toolbox/internal/embeddingmodels"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"

	"github.com/looker-open-source/sdk-codegen/go/rtl"
	v4 "github.com/looker-open-source/sdk-codegen/go/sdk/v4"
)

const resourceType string = "looker-update-project"

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
	Name           string                 `yaml:"name" validate:"required"`
	Type           string                 `yaml:"type" validate:"required"`
	Source         string                 `yaml:"source" validate:"required"`
	Description    string                 `yaml:"description" validate:"required"`
	AuthRequired   []string               `yaml:"authRequired"`
	Annotations    *tools.ToolAnnotations `yaml:"annotations,omitempty"`
	ScopesRequired []string               `yaml:"scopesRequired"`
}

// validate interface
var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigType() string {
	return resourceType
}

func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	projectIdParameter := parameters.NewStringParameterWithRequired("project_id", "The ID of the Looker project to update.", true)
	gitRemoteUrlParameter := parameters.NewStringParameterWithRequired("git_remote_url", "Git remote repository URL.", false)
	gitServiceNameParameter := parameters.NewStringParameterWithRequired("git_service_name", "Name of the git service provider (e.g., 'bare', 'github').", false)
	gitProductionBranchNameParameter := parameters.NewStringParameterWithRequired("git_production_branch_name", "Git production branch name.", false)
	pullRequestModeParameter := parameters.NewStringParameterWithRequired("pull_request_mode", "The git pull request policy for this project. Valid values: 'off', 'links', 'recommended', 'required'.", false)
	validationRequiredParameter := parameters.NewBooleanParameterWithRequired("validation_required", "Validation policy: If true, must pass validation before committing.", false)
	gitReleaseMgmtEnabledParameter := parameters.NewBooleanParameterWithRequired("git_release_mgmt_enabled", "If true, advanced git release management is enabled.", false)
	allowWarningsParameter := parameters.NewBooleanParameterWithRequired("allow_warnings", "Validation policy: If true, allow committing with warnings.", false)

	params := parameters.Parameters{
		projectIdParameter,
		gitRemoteUrlParameter,
		gitServiceNameParameter,
		gitProductionBranchNameParameter,
		pullRequestModeParameter,
		validationRequiredParameter,
		gitReleaseMgmtEnabledParameter,
		allowWarningsParameter,
	}

	return Tool{
		Config:     cfg,
		Parameters: params,
		manifest: tools.Manifest{
			Description:  cfg.Description,
			Parameters:   params.Manifest(),
			AuthRequired: cfg.AuthRequired,
		},
	}, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Config
	Parameters parameters.Parameters `yaml:"parameters"`
	manifest   tools.Manifest
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, util.ToolboxError) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Source, t.Name, t.Type)
	if err != nil {
		return nil, util.NewClientServerError("source used is not compatible with the tool", http.StatusInternalServerError, err)
	}

	sdk, err := source.GetLookerSDK(string(accessToken))
	if err != nil {
		return nil, util.NewClientServerError(fmt.Sprintf("error getting sdk: %v", err), http.StatusInternalServerError, err)
	}

	mapParams := params.AsMap()
	projectId, ok := mapParams["project_id"].(string)
	if !ok || projectId == "" {
		return nil, util.NewClientServerError("project_id must be specified", http.StatusBadRequest, nil)
	}

	body := v4.WriteProject{}

	if val, ok := mapParams["git_remote_url"].(string); ok && val != "" {
		body.GitRemoteUrl = &val
	}

	if val, ok := mapParams["git_service_name"].(string); ok && val != "" {
		body.GitServiceName = &val
	}

	if val, ok := mapParams["git_production_branch_name"].(string); ok && val != "" {
		body.GitProductionBranchName = &val
	}

	if val, ok := mapParams["pull_request_mode"].(string); ok && val != "" {
		prm := v4.PullRequestMode(val)
		body.PullRequestMode = &prm
	}

	if val, ok := mapParams["validation_required"].(bool); ok {
		body.ValidationRequired = &val
	}

	if val, ok := mapParams["git_release_mgmt_enabled"].(bool); ok {
		body.GitReleaseMgmtEnabled = &val
	}

	if val, ok := mapParams["allow_warnings"].(bool); ok {
		body.AllowWarnings = &val
	}

	resp, err := sdk.UpdateProject(projectId, body, "", source.LookerApiSettings())
	if err != nil {
		if strings.Contains(err.Error(), "status=401") {
			return nil, util.NewClientServerError("unauthorized error", http.StatusUnauthorized, err)
		}
		return nil, util.ProcessGeneralError(err)
	}
	return resp, nil
}

func (t Tool) EmbedParams(ctx context.Context, paramValues parameters.ParamValues, embeddingModelsMap map[string]embeddingmodels.EmbeddingModel) (parameters.ParamValues, error) {
	return parameters.EmbedParams(ctx, t.Parameters, paramValues, embeddingModelsMap, nil)
}

func (t Tool) Manifest() tools.Manifest {
	return t.manifest
}

func (t Tool) GetName() string {
	return t.Name
}

func (t Tool) GetDescription() string {
	return t.Description
}

func (t Tool) GetAuthRequired() []string {
	return t.AuthRequired
}

func (t Tool) GetAnnotations() *tools.ToolAnnotations {
	annotations := t.Annotations
	if annotations == nil {
		readOnlyHint := false
		annotations = &tools.ToolAnnotations{
			ReadOnlyHint: &readOnlyHint,
		}
	}
	return annotations
}

func (t Tool) GetScopesRequired() []string {
	return t.ScopesRequired
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
