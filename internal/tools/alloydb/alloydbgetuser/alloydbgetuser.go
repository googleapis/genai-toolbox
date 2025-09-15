
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

package alloydbgetuser

import (
	"context"
	"fmt"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	alloydbadmin "github.com/googleapis/genai-toolbox/internal/sources/alloydbadmin"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

const kind string = "alloydb-get-user"

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

// Configuration for the get-user tool.
type Config struct {
	Name         string            `yaml:"name" validate:"required"`
	Kind         string            `yaml:"kind" validate:"required"`
	Source       string            `yaml:"source" validate:"required"`
	Description  string            `yaml:"description" validate:"required"`
	AuthRequired []string          `yaml:"authRequired"`
	BaseURL      string            `yaml:"baseURL"`
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
		return nil, fmt.Errorf("source %q not found", cfg.Source)
	}

	s, ok := rawS.(*alloydbadmin.Source)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be `%s`", kind, alloydbadmin.SourceKind)
	}

	allParameters := tools.Parameters{
		tools.NewStringParameter("projectId", "The GCP project ID."),
		tools.NewStringParameter("locationId", "The location of the cluster (e.g., 'us-central1')."),
		tools.NewStringParameter("clusterId", "The ID of the cluster."),
		tools.NewStringParameter("userId", "The ID of the user."),
	}
	paramManifest := allParameters.Manifest()

	inputSchema := allParameters.McpManifest()
	inputSchema.Required = []string{"projectId", "locationId", "clusterId", "userId"}

	mcpManifest := tools.McpManifest{
		Name:        cfg.Name,
		Description: cfg.Description,
		InputSchema: inputSchema,
	}

	return Tool{
		Name:         cfg.Name,
		Kind:         kind,
		Source:       s,
		AllParams:    allParameters,
		manifest:     tools.Manifest{Description: cfg.Description, Parameters: paramManifest},
		mcpManifest:  mcpManifest,
	}, nil
}

// Tool represents the get-user tool.
type Tool struct {
	Name string `yaml:"name"`
	Kind string `yaml:"kind"`

	Source    *alloydbadmin.Source
	AllParams tools.Parameters

	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

// Invoke executes the tool's logic.
func (t Tool) Invoke(ctx context.Context, params tools.ParamValues, accessToken tools.AccessToken) (any, error) {
	paramsMap := params.AsMap()

	projectId, ok := paramsMap["projectId"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid or missing 'projectId' parameter; expected a string")
	}
	locationId, ok := paramsMap["locationId"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid 'locationId' parameter; expected a string")
	}
	clusterId, ok := paramsMap["clusterId"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid 'clusterId' parameter; expected a string")
	}
	userId, ok := paramsMap["userId"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid 'userId' parameter; expected a string")
	}

	service, err := t.Source.GetService(ctx, string(accessToken))
	if err != nil {
		return nil, err
	}

	urlString := fmt.Sprintf("projects/%s/locations/%s/clusters/%s/users/%s", projectId, locationId, clusterId, userId)

	resp, err := service.Projects.Locations.Clusters.Users.Get(urlString).Do()
	if err != nil {
		return nil, fmt.Errorf("error getting AlloyDB user: %w", err)
	}

	return resp, nil
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
	return t.Source.UseClientAuthorization()
}
