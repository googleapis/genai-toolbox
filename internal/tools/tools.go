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

package tools

import (
	"context"
	"errors"
	"fmt"
	"slices"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
)

// ToolConfigFactory defines the signature for a function that creates and
// decodes a specific tool's configuration. It takes the context, the tool's
// name, and a YAML decoder to parse the config.
type ToolConfigFactory func(ctx context.Context, name string, decoder *yaml.Decoder) (ToolConfig, error)

var toolRegistry = make(map[string]ToolConfigFactory)

// Register allows individual tool packages to register their configuration
// factory function. This is typically called from an init() function in the
// tool's package. It associates a 'type' string with a function that can
// produce the specific ToolConfig type. It returns true if the registration was
// successful, and false if a tool with the same type was already registered.
func Register(resourceType string, factory ToolConfigFactory) bool {
	if _, exists := toolRegistry[resourceType]; exists {
		// Tool with this type already exists, do not overwrite.
		return false
	}
	toolRegistry[resourceType] = factory
	return true
}

// DecodeConfig looks up the registered factory for the given type and uses it
// to decode the tool configuration.
func DecodeConfig(ctx context.Context, resourceType string, name string, decoder *yaml.Decoder) (ToolConfig, error) {
	factory, found := toolRegistry[resourceType]
	if !found {
		return nil, fmt.Errorf("unknown tool type: %q", resourceType)
	}
	toolConfig, err := factory(ctx, name, decoder)
	if err != nil {
		return nil, fmt.Errorf("unable to parse tool %q as kind %q: %w", name, resourceType, err)
	}
	return toolConfig, nil
}

type ToolConfig interface {
	ToolConfigType() string
	Initialize(map[string]sources.Source) (Tool, error)
}

type AccessToken string

type Tool interface {
	Invoke(context.Context, ParamValues, AccessToken) (any, error)
	ParseParams(map[string]any, map[string]map[string]any) (ParamValues, error)
	Manifest() Manifest
	McpManifest() McpManifest
	Authorized([]string) bool
	RequiresClientAuthorization() bool
}

// Manifest is the representation of tools sent to Client SDKs.
type Manifest struct {
	Description  string              `json:"description"`
	Parameters   []ParameterManifest `json:"parameters"`
	AuthRequired []string            `json:"authRequired"`
}

// Definition for a tool the MCP client can call.
type McpManifest struct {
	// The name of the tool.
	Name string `json:"name"`
	// A human-readable description of the tool.
	Description string `json:"description,omitempty"`
	// A JSON Schema object defining the expected parameters for the tool.
	InputSchema McpToolsSchema `json:"inputSchema,omitempty"`
}

var ErrUnauthorized = errors.New("unauthorized")

// Helper function that returns if a tool invocation request is authorized
func IsAuthorized(authRequiredSources []string, verifiedAuthServices []string) bool {
	if len(authRequiredSources) == 0 {
		// no authorization requirement
		return true
	}
	for _, a := range authRequiredSources {
		if slices.Contains(verifiedAuthServices, a) {
			return true
		}
	}
	return false
}
