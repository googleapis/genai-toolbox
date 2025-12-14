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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	cloudgdasrc "github.com/googleapis/genai-toolbox/internal/sources/cloudgda"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
)

const Kind string = "cloud-gemini-data-analytics-query"

func init() {
	if !tools.Register(Kind, newConfig) {
		panic(fmt.Sprintf("tool kind %q already registered", Kind))
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
	return Kind
}

func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	// verify source exists
	rawS, ok := srcs[cfg.Source]
	if !ok {
		return nil, fmt.Errorf("no source named %q configured", cfg.Source)
	}

	// verify the source is compatible
	s, ok := rawS.(*cloudgdasrc.Source)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be `cloud-gemini-data-analytics`", Kind)
	}

	// Define the parameters for the Gemini Data Analytics Query API
	// The prompt is the only input parameter.
	allParameters := parameters.Parameters{
		parameters.NewStringParameterWithRequired("prompt", "The natural language question to ask.", true),
	}

	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, allParameters, nil)

	return Tool{
		Config:      cfg,
		AllParams:   allParameters,
		Source:      s,
		manifest:    tools.Manifest{Description: cfg.Description, Parameters: allParameters.Manifest(), AuthRequired: cfg.AuthRequired},
		mcpManifest: mcpManifest,
	}, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Config
	AllParams   parameters.Parameters
	Source      *cloudgdasrc.Source
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

// Invoke executes the tool logic
func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	paramsMap := params.AsMap()
	prompt, ok := paramsMap["prompt"].(string)
	if !ok {
		return nil, fmt.Errorf("prompt parameter not found or not a string")
	}

	// The API endpoint itself always uses the "global" location.
	apiLocation := "global"
	apiParent := fmt.Sprintf("projects/%s/locations/%s", t.Source.ProjectID, apiLocation)
	apiURL := fmt.Sprintf("%s/v1beta/%s:queryData", t.Source.BaseURL, apiParent)

	// The parent in the request payload uses the tool's configured location.
	payloadParent := fmt.Sprintf("projects/%s/locations/%s", t.Source.ProjectID, t.Location)

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

	// Debugging output
	fmt.Printf("API URL: %s\n", apiURL)
	fmt.Printf("Request Body: %s\n", string(bodyBytes))

	client, err := t.Source.GetClient(ctx, string(accessToken))
	if err != nil {
		return nil, fmt.Errorf("failed to get HTTP client: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return result, nil
}

func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (parameters.ParamValues, error) {
	return parameters.ParseParams(t.AllParams, data, claims)
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
	return t.Source.UseClientAuthorization()
}

func (t Tool) GetAuthTokenHeaderName() string {
	return "Authorization"
}
