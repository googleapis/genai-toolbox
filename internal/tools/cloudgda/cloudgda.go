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
	"net/http"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/embeddingmodels"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
)

const resourceType string = "cloud-gemini-data-analytics-query"

// Guidance is the tool guidance string.
const Guidance = `Tool guidance:
  Inputs:
    1. query: A natural language formulation of a database query.
  Outputs: (all optional)
    1. disambiguation_question: Clarification questions or comments where the tool needs the users' input.
    2. generated_query: The generated query for the user query.
    3. intent_explanation: An explanation for why the tool produced ` + "`generated_query`" + `.
    4. query_result: The result of executing ` + "`generated_query`" + `.
    5. natural_language_answer: The natural language answer that summarizes the ` + "`query`" + ` and ` + "`query_result`" + `.

Usage guidance:
  1. If ` + "`disambiguation_question`" + ` is produced, then solicit the needed inputs from the user and try the tool with a new ` + "`query`" + ` that has the needed clarification.
  2. If ` + "`natural_language_answer`" + ` is produced, use ` + "`intent_explanation`" + ` and ` + "`generated_query`" + ` to see if you need to clarify any assumptions for the user.`

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
	GetProjectID() string
	UseClientAuthorization() bool
	RunQuery(context.Context, string, []byte) (any, error)
}

type Config struct {
	Name              string             `yaml:"name" validate:"required"`
	Type              string             `yaml:"type" validate:"required"`
	Source            string             `yaml:"source" validate:"required"`
	Description       string             `yaml:"description" validate:"required"`
	Location          string             `yaml:"location" validate:"required"`
	Context           *QueryDataContext  `yaml:"context" validate:"required"`
	GenerationOptions *GenerationOptions `yaml:"generationOptions,omitempty"`
	AuthRequired      []string           `yaml:"authRequired"`
}

// validate interface
var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigType() string {
	return resourceType
}

func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	// Define the parameters for the Gemini Data Analytics Query API
	// The query is the only input parameter.
	allParameters := parameters.Parameters{
		parameters.NewStringParameterWithRequired("query", "A natural language formulation of a database query.", true),
	}
	// The input and outputs are for tool guidance, usage guidance is for multi-turn interaction.
	guidance := Guidance

	if cfg.Description != "" {
		cfg.Description += "\n\n" + guidance
	} else {
		cfg.Description = guidance
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

func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, util.ToolboxError) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Source, t.Name, t.Type)
	if err != nil {
		return nil, util.NewClientServerError("source used is not compatible with the tool", http.StatusInternalServerError, err)
	}

	paramsMap := params.AsMap()
	query, ok := paramsMap["query"].(string)
	if !ok {
		return nil, util.NewAgentError("query parameter not found or not a string", nil)
	}

	// Parse the access token if provided
	var tokenStr string
	if source.UseClientAuthorization() {
		var err error
		tokenStr, err = accessToken.ParseBearerToken()
		if err != nil {
			return nil, util.NewClientServerError("error parsing access token", http.StatusUnauthorized, err)
		}
	}

	// The parent in the request payload uses the tool's configured location.
	payloadParent := fmt.Sprintf("projects/%s/locations/%s", source.GetProjectID(), t.Location)

	payload := &QueryDataRequest{
		Parent:            payloadParent,
		Prompt:            query,
		Context:           t.Context,
		GenerationOptions: t.GenerationOptions,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, util.NewClientServerError("failed to marshal request payload", http.StatusInternalServerError, err)
	}

	resp, err := source.RunQuery(ctx, tokenStr, bodyBytes)
	if err != nil {
		return nil, util.ProcessGcpError(err)
	}
	return resp, nil
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
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Source, t.Name, t.Type)
	if err != nil {
		return false, err
	}
	return source.UseClientAuthorization(), nil
}

func (t Tool) GetAuthTokenHeaderName(_ tools.SourceProvider) (string, error) {
	return "Authorization", nil
}

func (t Tool) GetParameters() parameters.Parameters {
	return t.AllParams
}
