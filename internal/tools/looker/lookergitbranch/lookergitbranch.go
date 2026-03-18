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
package lookergitbranch

import (
	"context"
	"fmt"
	"net/http"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/embeddingmodels"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"

	"github.com/looker-open-source/sdk-codegen/go/rtl"
	v4 "github.com/looker-open-source/sdk-codegen/go/sdk/v4"
)

const resourceType string = "looker-git-branch"

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
	projectIdParameter := parameters.NewStringParameter("project_id", "The project_id")
	operationParameter := parameters.NewStringParameter("operation", "The operation, one of `list`, `get`, `create`, `switch`, or `delete`")
	branchParameter := parameters.NewStringParameterWithDefault("branch", "", "The git branch on which to operate. Not required for `list` or `get` operations.")
	refParameter := parameters.NewStringParameterWithDefault("ref", "", "The ref to use as the start of a new branch. If not specified for a `create` operation it will default to HEAD of current branch. If supplied with a `switch` operation will `reset --hard` the branch.")
	params := parameters.Parameters{projectIdParameter, operationParameter, branchParameter, refParameter}

	annotations := cfg.Annotations

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

func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, util.ToolboxError) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Source, t.Name, t.Type)
	if err != nil {
		return nil, util.NewClientServerError("source used is not compatible with the tool", http.StatusInternalServerError, err)
	}

	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return nil, util.NewClientServerError("unable to get logger from ctx", http.StatusInternalServerError, err)
	}

	sdk, err := source.GetLookerSDK(string(accessToken))
	if err != nil {
		return nil, util.NewClientServerError(fmt.Sprintf("error getting sdk: %v", err), http.StatusInternalServerError, err)
	}

	mapParams := params.AsMap()
	logger.DebugContext(ctx, "looker_git_branch params = ", mapParams)
	projectId := mapParams["project_id"].(string)
	operation := mapParams["operation"].(string)
	branch := mapParams["branch"].(string)
	ref := mapParams["ref"].(string)

	switch operation {
	case "list":
		resp, err := sdk.AllGitBranches(projectId, source.LookerApiSettings())
		if err != nil {
			return nil, util.NewClientServerError(fmt.Sprintf("error making list_git_branches request: %s", err), http.StatusInternalServerError, err)
		}
		return resp, nil
	case "get":
		resp, err := sdk.GitBranch(projectId, source.LookerApiSettings())
		if err != nil {
			return nil, util.NewClientServerError(fmt.Sprintf("error making get_git_branch request: %s", err), http.StatusInternalServerError, err)
		}
		return resp, nil
	case "create":
		if branch == "" {
			return nil, util.NewClientServerError(fmt.Sprintf("%s operation: branch must be specified", operation), http.StatusInternalServerError, nil)
		}
		body := v4.WriteGitBranch{
			Name: &branch,
		}
		if ref != "" {
			body.Ref = &ref
		}
		resp, err := sdk.CreateGitBranch(projectId, body, source.LookerApiSettings())
		if err != nil {
			return nil, util.NewClientServerError(fmt.Sprintf("error making create_git_branch request: %s", err), http.StatusInternalServerError, err)
		}
		return resp, nil
	case "switch":
		if branch == "" {
			return nil, util.NewClientServerError(fmt.Sprintf("%s operation: branch must be specified", operation), http.StatusInternalServerError, nil)
		}
		body := v4.WriteGitBranch{
			Name: &branch,
		}
		if ref != "" {
			body.Ref = &ref
		}
		resp, err := sdk.UpdateGitBranch(projectId, body, source.LookerApiSettings())
		if err != nil {
			return nil, util.NewClientServerError(fmt.Sprintf("error making update_git_branch request: %s", err), http.StatusInternalServerError, err)
		}
		return resp, nil
	case "delete":
		if branch == "" {
			return nil, util.NewClientServerError(fmt.Sprintf("%s operation: branch must be specified", operation), http.StatusInternalServerError, nil)
		}
		_, err := sdk.DeleteGitBranch(projectId, branch, source.LookerApiSettings())
		if err != nil {
			return nil, util.NewClientServerError(fmt.Sprintf("error making delete_git_branch request: %s", err), http.StatusInternalServerError, err)
		}
		return fmt.Sprintf("Deleted branch %s", branch), nil
	default:
		return nil, util.NewClientServerError(fmt.Sprintf("unknown operation: %s. Must be one of `list`, `get`, `create`, `switch`, or `delete`", operation), http.StatusInternalServerError, nil)
	}
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
