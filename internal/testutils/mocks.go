// Copyright 2026 Google LLC
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

package testutils

import (
	"context"
	"fmt"

	"github.com/googleapis/mcp-toolbox/internal/embeddingmodels"
	"github.com/googleapis/mcp-toolbox/internal/prompts"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
)

// MockTool is used to mock tools in tests
type MockTool struct {
	Name                       string
	Description                string
	Params                     []parameters.Parameter
	manifest                   tools.Manifest
	mcpManifest                tools.McpManifest
	unauthorized               bool
	requireClientAuthorization bool
}

// NewMockTool creates a new mock prompt for testing.
func NewMockTool(name, desc string, params []parameters.Parameter, unauthorized, requireClientAuthorization bool) MockTool {
	pMs := make([]parameters.ParameterManifest, 0, len(params))
	for _, p := range params {
		pMs = append(pMs, p.Manifest())
	}
	manifest := tools.Manifest{Description: desc, Parameters: pMs}

	properties := make(map[string]parameters.ParameterMcpManifest)
	required := make([]string, 0)
	authParams := make(map[string][]string)

	for _, p := range params {
		pName := p.GetName()
		paramManifest, authParamList := p.McpManifest()
		properties[pName] = paramManifest
		required = append(required, pName)

		if len(authParamList) > 0 {
			authParams[pName] = authParamList
		}
	}

	toolsSchema := parameters.McpToolsSchema{
		Type:       "object",
		Properties: properties,
		Required:   required,
	}

	mcpManifest := tools.McpManifest{
		Name:        name,
		Description: desc,
		InputSchema: toolsSchema,
	}

	if len(authParams) > 0 {
		mcpManifest.Metadata = map[string]any{
			"toolbox/authParams": authParams,
		}
	}
	return MockTool{
		Name:                       name,
		Description:                desc,
		Params:                     params,
		manifest:                   manifest,
		mcpManifest:                mcpManifest,
		unauthorized:               unauthorized,
		requireClientAuthorization: requireClientAuthorization,
	}
}

func (t MockTool) Invoke(context.Context, tools.SourceProvider, parameters.ParamValues, tools.AccessToken) (any, util.ToolboxError) {
	mock := []any{t.Name}
	return mock, nil
}

func (t MockTool) ToConfig() tools.ToolConfig {
	return nil
}

// claims is a map of user info decoded from an auth token
func (t MockTool) ParseParams(data map[string]any, claimsMap map[string]map[string]any) (parameters.ParamValues, error) {
	return parameters.ParseParams(t.Params, data, claimsMap)
}

func (t MockTool) EmbedParams(ctx context.Context, paramValues parameters.ParamValues, embeddingModelsMap map[string]embeddingmodels.EmbeddingModel) (parameters.ParamValues, error) {
	return parameters.EmbedParams(ctx, t.Params, paramValues, embeddingModelsMap, nil)
}

func (t MockTool) Manifest() tools.Manifest {
	return t.manifest
}

func (t MockTool) Authorized(verifiedAuthServices []string) bool {
	// defaulted to true
	return !t.unauthorized
}

func (t MockTool) RequiresClientAuthorization(tools.SourceProvider) (bool, error) {
	// defaulted to false
	return t.requireClientAuthorization, nil
}

func (t MockTool) GetParameters() parameters.Parameters {
	return t.Params
}

func (t MockTool) McpManifest() tools.McpManifest {
	return t.mcpManifest
}

func (t MockTool) GetAuthTokenHeaderName(tools.SourceProvider) (string, error) {
	return "Authorization", nil
}

func (t MockTool) GetScopesRequired() []string {
	return nil
}

// MockPrompt is used to mock prompts in tests
type MockPrompt struct {
	Name        string
	Description string
	Args        prompts.Arguments
	manifest    prompts.Manifest
	mcpManifest prompts.McpManifest
}

func (p MockPrompt) SubstituteParams(vals parameters.ParamValues) (any, error) {
	return []prompts.Message{
		{
			Role:    "user",
			Content: fmt.Sprintf("substituted %s", p.Name),
		},
	}, nil
}

func (p MockPrompt) ParseArgs(data map[string]any, claimsMap map[string]map[string]any) (parameters.ParamValues, error) {
	var params parameters.Parameters
	for _, arg := range p.Args {
		params = append(params, arg.Parameter)
	}
	return parameters.ParseParams(params, data, claimsMap)
}

func (p MockPrompt) Manifest() prompts.Manifest {
	var argManifests []parameters.ParameterManifest
	for _, arg := range p.Args {
		argManifests = append(argManifests, arg.Manifest())
	}
	return prompts.Manifest{
		Description: p.Description,
		Arguments:   argManifests,
	}
}

func (p MockPrompt) McpManifest() prompts.McpManifest {
	return prompts.GetMcpManifest(p.Name, p.Description, p.Args)
}

func (p MockPrompt) ToConfig() prompts.PromptConfig {
	return nil
}

// NewMockPrompt creates a new mock prompt for testing.
func NewMockPrompt(name, desc string, args prompts.Arguments) MockPrompt {
	var argManifests []parameters.ParameterManifest
	for _, arg := range args {
		argManifests = append(argManifests, arg.Manifest())
	}
	manifest := prompts.Manifest{
		Description: desc,
		Arguments:   argManifests,
	}
	mcpManifest := prompts.GetMcpManifest(name, desc, args)
	return MockPrompt{
		Name:        name,
		Description: desc,
		Args:        args,
		manifest:    manifest,
		mcpManifest: mcpManifest,
	}
}
