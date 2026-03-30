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
	"net/http"

	"github.com/googleapis/genai-toolbox/internal/auth"
	"github.com/googleapis/genai-toolbox/internal/embeddingmodels"
	"github.com/googleapis/genai-toolbox/internal/prompts"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
	"go.opentelemetry.io/otel/trace"
)

type MockSource struct {
	MockSourceConfig
	Name string
}

func (s MockSource) SourceType() string {
	return "mock"
}

func (s MockSource) ToConfig() sources.SourceConfig {
	return &s.MockSourceConfig
}

type MockAuthService struct {
	MockAuthServiceConfig
	Name string
}

func (a MockAuthService) AuthServiceType() string {
	return "mock"
}

func (a MockAuthService) GetName() string {
	return a.Name
}

func (a MockAuthService) GetClaimsFromHeader(ctx context.Context, h http.Header) (map[string]any, error) {
	return nil, nil
}

func (a MockAuthService) ToConfig() auth.AuthServiceConfig {
	return &a.MockAuthServiceConfig
}

type MockEmbeddingModel struct {
	MockEmbeddingModelConfig
	Name string
}

func (e MockEmbeddingModel) EmbeddingModelType() string {
	return "mock"
}

func (e MockEmbeddingModel) ToConfig() embeddingmodels.EmbeddingModelConfig {
	return &e.MockEmbeddingModelConfig
}

func (e MockEmbeddingModel) EmbedParameters(ctx context.Context, t []string) ([][]float32, error) {
	return nil, nil
}

// MockTool is used to mock tools in tests
type MockTool struct {
	MockToolConfig
	Name                         string
	Description                  string
	Params                       []parameters.Parameter
	ToolManifest                 tools.Manifest
	Unauthorized                 bool
	RequiresClientAuthrorization bool
}

func (t MockTool) Invoke(context.Context, tools.SourceProvider, parameters.ParamValues, tools.AccessToken) (any, util.ToolboxError) {
	mock := []any{t.Name}
	return mock, nil
}

func (t MockTool) ToConfig() tools.ToolConfig {
	return &t.MockToolConfig
}

// claims is a map of user info decoded from an auth token
func (t MockTool) ParseParams(data map[string]any, claimsMap map[string]map[string]any) (parameters.ParamValues, error) {
	return parameters.ParseParams(t.Params, data, claimsMap)
}

func (t MockTool) EmbedParams(ctx context.Context, paramValues parameters.ParamValues, embeddingModelsMap map[string]embeddingmodels.EmbeddingModel) (parameters.ParamValues, error) {
	return parameters.EmbedParams(ctx, t.Params, paramValues, embeddingModelsMap, nil)
}

func (t MockTool) Manifest() tools.Manifest {
	pMs := make([]parameters.ParameterManifest, 0, len(t.Params))
	for _, p := range t.Params {
		pMs = append(pMs, p.Manifest())
	}
	return tools.Manifest{Description: t.Description, Parameters: pMs}
}

func (t MockTool) Authorized(verifiedAuthServices []string) bool {
	// defaulted to true
	return !t.Unauthorized
}

func (t MockTool) RequiresClientAuthorization(tools.SourceProvider) (bool, error) {
	// defaulted to false
	return t.RequiresClientAuthrorization, nil
}

func (t MockTool) GetParameters() parameters.Parameters {
	return t.Params
}

func (t MockTool) McpManifest() tools.McpManifest {
	properties := make(map[string]parameters.ParameterMcpManifest)
	required := make([]string, 0)
	authParams := make(map[string][]string)

	for _, p := range t.Params {
		name := p.GetName()
		paramManifest, authParamList := p.McpManifest()
		properties[name] = paramManifest
		required = append(required, name)

		if len(authParamList) > 0 {
			authParams[name] = authParamList
		}
	}

	toolsSchema := parameters.McpToolsSchema{
		Type:       "object",
		Properties: properties,
		Required:   required,
	}

	mcpManifest := tools.McpManifest{
		Name:        t.Name,
		Description: t.Description,
		InputSchema: toolsSchema,
	}

	if len(authParams) > 0 {
		mcpManifest.Metadata = map[string]any{
			"toolbox/authParams": authParams,
		}
	}

	return mcpManifest
}

func (t MockTool) GetAuthTokenHeaderName(tools.SourceProvider) (string, error) {
	return "Authorization", nil
}

// MockPrompt is used to mock prompts in tests
type MockPrompt struct {
	MockPromptConfig
	Name        string
	Description string
	Args        prompts.Arguments
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
	return &p.MockPromptConfig
}

type MockSourceConfig struct {
	Foo      string      `yaml:"foo"`
	Password util.Secret `yaml:"password"`
	Type     string      `yaml:"type"`
}

func (m *MockSourceConfig) SourceConfigType() string {
	return "mock"
}

func (m *MockSourceConfig) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	return MockSource{MockSourceConfig: *m}, nil
}

type MockAuthServiceConfig struct {
	Foo  string `yaml:"foo"`
	Type string `yaml:"type"`
}

func (m *MockAuthServiceConfig) AuthServiceConfigType() string {
	return "mock"
}

func (m *MockAuthServiceConfig) Initialize() (auth.AuthService, error) {
	return MockAuthService{MockAuthServiceConfig: *m}, nil
}

type MockEmbeddingModelConfig struct {
	Foo  string `yaml:"foo"`
	Type string `yaml:"type"`
}

func (m *MockEmbeddingModelConfig) EmbeddingModelConfigType() string {
	return "mock"
}

func (m *MockEmbeddingModelConfig) Initialize(ctx context.Context) (embeddingmodels.EmbeddingModel, error) {
	return MockEmbeddingModel{MockEmbeddingModelConfig: *m}, nil
}

type MockToolConfig struct {
	Foo          string   `yaml:"foo"`
	Type         string   `yaml:"type"`
	AuthRequired []string `yaml:"authRequired"`
}

func (m *MockToolConfig) ToolConfigType() string {
	return "mock"
}

func (m *MockToolConfig) Initialize(map[string]sources.Source) (tools.Tool, error) {
	return MockTool{MockToolConfig: *m}, nil
}

type MockToolsetConfig struct {
	tools.ToolsetConfig
}

func (m *MockToolsetConfig) Initialize(serverVersion string, toolsMap map[string]tools.Tool) (tools.Toolset, error) {
	return tools.Toolset{ToolsetConfig: m.ToolsetConfig}, nil
}

type MockPromptConfig struct {
	Foo  string `yaml:"foo"`
	Type string `yaml:"type"`
}

func (m *MockPromptConfig) PromptConfigType() string {
	return "mock"
}

func (m *MockPromptConfig) Initialize() (prompts.Prompt, error) {
	return MockPrompt{MockPromptConfig: *m}, nil
}
