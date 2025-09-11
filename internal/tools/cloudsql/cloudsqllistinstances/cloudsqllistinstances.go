// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cloudsqllistinstances

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	cloudsqladminsrc "github.com/googleapis/genai-toolbox/internal/sources/cloudsqladmin"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

const kind string = "cloud-sql-list-instances"

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

// Config defines the configuration for the list-instance tool.
type Config struct {
	Name         string   `yaml:"name" validate:"required"`
	Kind         string   `yaml:"kind" validate:"required"`
	Source       string   `yaml:"source" validate:"required"`
	Description  string   `yaml:"description" validate:"required"`
	AuthRequired []string `yaml:"authRequired"`
}

// validate interface
var _ tools.ToolConfig = Config{}

// ToolConfigKind returns the kind of the tool.
func (cfg Config) ToolConfigKind() string {
	return kind
}

// Initialize initializes the tool from the configuration.
func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	rawS, ok := srcs[cfg.Source]
	if !ok {
		return nil, fmt.Errorf("no source named %q configured", cfg.Source)
	}
	s, ok := rawS.(*cloudsqladminsrc.Source)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be `cloud-sql-admin`", kind)
	}

	allParameters := tools.Parameters{
		tools.NewStringParameter("project", "The project ID"),
	}
	paramManifest := allParameters.Manifest()

	inputSchema := allParameters.McpManifest()
	inputSchema.Required = []string{"project"}

	mcpManifest := tools.McpManifest{
		Name:        cfg.Name,
		Description: cfg.Description,
		InputSchema: inputSchema,
	}

	return Tool{
		Name:         cfg.Name,
		Kind:         kind,
		AuthRequired: cfg.AuthRequired,
		source:       s,
		AllParams:    allParameters,
		manifest:     tools.Manifest{Description: cfg.Description, Parameters: paramManifest, AuthRequired: cfg.AuthRequired},
		mcpManifest:  mcpManifest,
	}, nil
}

// Tool represents the list-instance tool.
type Tool struct {
	Name         string   `yaml:"name"`
	Kind         string   `yaml:"kind"`
	Description  string   `yaml:"description"`
	AuthRequired []string `yaml:"authRequired"`

	AllParams   tools.Parameters `yaml:"allParams"`
	source      *cloudsqladminsrc.Source
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

// Invoke executes the tool's logic.
func (t Tool) Invoke(ctx context.Context, params tools.ParamValues, accessToken tools.AccessToken) (any, error) {
	paramsMap := params.AsMap()

	project, ok := paramsMap["project"].(string)
	if !ok {
		return nil, fmt.Errorf("missing 'project' parameter")
	}

	client, err := t.source.GetClient(ctx, string(accessToken))
	if err != nil {
		return nil, err
	}

	urlString := fmt.Sprintf("%s/v1/projects/%s/instances", t.source.BaseURL, project)
	req, err := http.NewRequest(http.MethodGet, urlString, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making HTTP request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d, response body: %s", resp.StatusCode, string(body))
	}

	var data any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("error unmarshaling response body: %w", err)
	}

	return data, nil
}

// ParseParams parses the parameters for the tool.
func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (tools.ParamValues, error) {
	return tools.ParseParams(t.AllParams, data, claims)
}

// Manifest returns the tool's manifest.
func (t Tool) Manifest() tools.Manifest {
	return t.manifest
}

// McpManifest returns the tool's MCP manifest.
func (t Tool) McpManifest() tools.McpManifest {
	return t.mcpManifest
}

// Authorized checks if the tool is authorized.
func (t Tool) Authorized(verifiedAuthServices []string) bool {
	return true
}

func (t Tool) RequiresClientAuthorization() bool {
	return t.source.UseClientAuthorization()
}
