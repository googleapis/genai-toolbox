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

package cloudgda

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/embeddingmodels"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
)

const kind string = "cloud-gemini-data-analytics-query"

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
	GetProjectID() string
	UseClientAuthorization() bool
	RunQuery(context.Context, string, []byte) (any, error)
}

type Config struct {
	Name              string             `yaml:"name" validate:"required"`
	Kind              string             `yaml:"kind" validate:"required"`
	Source            string             `yaml:"source" validate:"required"`
	Description       string             `yaml:"description" validate:"required"`
	Location          string             `yaml:"location" validate:"required"`
	Context           *QueryDataContext  `yaml:"context" validate:"required"`
	GenerationOptions *GenerationOptions `yaml:"generationOptions,omitempty"`
	AuthRequired      []string           `yaml:"authRequired"`
}

// validate interface
var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigKind() string {
	return kind
}

func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	// Define the parameters for the Gemini Data Analytics Query API
	// The prompt is the only input parameter.
	allParameters := parameters.Parameters{
		parameters.NewStringParameterWithRequired("prompt", "The natural language question to ask.", true),
	}

	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, allParameters, nil)

	return Tool{
		Config:      cfg,
		AllParams:   allParameters,
		manifest:    tools.Manifest{Description: cfg.Description, Parameters: allParameters.Manifest(), AuthRequired: cfg.AuthRequired},
		mcpManifest: mcpManifest,
	}, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Config
	AllParams   parameters.Parameters
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

// Invoke executes the tool logic
func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Source, t.Name, t.Kind)
	if err != nil {
		return nil, err
	}

	paramsMap := params.AsMap()
	prompt, ok := paramsMap["prompt"].(string)
	if !ok {
		return nil, fmt.Errorf("prompt parameter not found or not a string")
	}

	// Parse the access token if provided
	var tokenStr string
	if source.UseClientAuthorization() {
		var err error
		tokenStr, err = accessToken.ParseBearerToken()
		if err != nil {
			return nil, fmt.Errorf("error parsing access token: %w", err)
		}
	}

	// The parent in the request payload uses the tool's configured location.
	payloadParent := fmt.Sprintf("projects/%s/locations/%s", source.GetProjectID(), t.Location)

	payload := &QueryDataRequest{
		Parent:            payloadParent,
		Prompt:            prompt,
		Context:           t.Context,
		GenerationOptions: t.GenerationOptions,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request payload: %w", err)
	}
	return source.RunQuery(ctx, tokenStr, bodyBytes)
}

func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (parameters.ParamValues, error) {
	return parameters.ParseParams(t.AllParams, data, claims)
}

func (t Tool) EmbedParams(ctx context.Context, paramValues parameters.ParamValues, embeddingModelsMap map[string]embeddingmodels.EmbeddingModel) (parameters.ParamValues, error) {
	return parameters.EmbedParams(ctx, t.AllParams, paramValues, embeddingModelsMap, nil)
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

func (t Tool) RequiresClientAuthorization(resourceMgr tools.SourceProvider) (bool, error) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Source, t.Name, t.Kind)
	if err != nil {
		return false, err
	}
	return source.UseClientAuthorization(), nil
}

func (t Tool) GetAuthTokenHeaderName(_ tools.SourceProvider) (string, error) {
	return "Authorization", nil
}
