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
	"fmt"
	"slices"
	"strings"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/util"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
)

// ToolConfigFactory defines the signature for a function that creates and
// decodes a specific tool's configuration. It takes the context, the tool's
// name, and a YAML decoder to parse the config.
type ToolConfigFactory func(ctx context.Context, name string, decoder *yaml.Decoder) (ToolConfig, error)

var toolRegistry = make(map[string]ToolConfigFactory)

// Register allows individual tool packages to register their configuration
// factory function. This is typically called from an init() function in the
// tool's package. It associates a 'kind' string with a function that can
// produce the specific ToolConfig type. It returns true if the registration was
// successful, and false if a tool with the same kind was already registered.
func Register(kind string, factory ToolConfigFactory) bool {
	if _, exists := toolRegistry[kind]; exists {
		// Tool with this kind already exists, do not overwrite.
		return false
	}
	toolRegistry[kind] = factory
	return true
}

// DecodeConfig looks up the registered factory for the given kind and uses it
// to decode the tool configuration.
func DecodeConfig(ctx context.Context, kind string, name string, decoder *yaml.Decoder) (ToolConfig, error) {
	factory, found := toolRegistry[kind]
	if !found {
		return nil, fmt.Errorf("unknown tool kind: %q", kind)
	}
	toolConfig, err := factory(ctx, name, decoder)
	if err != nil {
		return nil, fmt.Errorf("unable to parse tool %q as kind %q: %w", name, kind, err)
	}
	return toolConfig, nil
}

type ToolConfig interface {
	ToolConfigKind() string
	Initialize(map[string]sources.Source) (Tool, error)
}

// https://modelcontextprotocol.io/specification/2025-06-18/schema#toolannotations
type ToolAnnotations struct {
	DestructiveHint *bool `json:"destructiveHint,omitempty" yaml:"destructiveHint,omitempty"`
	IdempotentHint  *bool `json:"idempotentHint,omitempty" yaml:"idempotentHint,omitempty"`
	OpenWorldHint   *bool `json:"openWorldHint,omitempty" yaml:"openWorldHint,omitempty"`
	ReadOnlyHint    *bool `json:"readOnlyHint,omitempty" yaml:"readOnlyHint,omitempty"`
}

type AccessToken string

func (token AccessToken) ParseBearerToken() (string, error) {
	headerParts := strings.Split(string(token), " ")
	if len(headerParts) != 2 || strings.ToLower(headerParts[0]) != "bearer" {
		return "", fmt.Errorf("authorization header must be in the format 'Bearer <token>': %w", util.ErrUnauthorized)
	}
	return headerParts[1], nil
}

type Tool interface {
	Invoke(context.Context, parameters.ParamValues, AccessToken) (any, error)
	ParseParams(map[string]any, map[string]map[string]any) (parameters.ParamValues, error)
	Manifest() Manifest
	McpManifest() McpManifest
	Authorized([]string) bool
	RequiresClientAuthorization() bool
	ToConfig() ToolConfig
	GetAuthTokenHeaderName() string
}

// Manifest is the representation of tools sent to Client SDKs.
type Manifest struct {
	Description  string                         `json:"description"`
	Parameters   []parameters.ParameterManifest `json:"parameters"`
	AuthRequired []string                       `json:"authRequired"`
}

// Definition for a tool the MCP client can call.
type McpManifest struct {
	// The name of the tool.
	Name string `json:"name"`
	// A human-readable description of the tool.
	Description string           `json:"description,omitempty"`
	Annotations *ToolAnnotations `json:"annotations,omitempty"`
	// A JSON Schema object defining the expected parameters for the tool.
	InputSchema parameters.McpToolsSchema `json:"inputSchema,omitempty"`
	Metadata    map[string]any            `json:"_meta,omitempty"`
}

func GetMcpManifest(name, desc string, authInvoke []string, params parameters.Parameters, annotations *ToolAnnotations) McpManifest {
	inputSchema, authParams := params.McpManifest()
	mcpManifest := McpManifest{
		Name:        name,
		Description: desc,
		InputSchema: inputSchema,
		Annotations: annotations,
	}

	// construct metadata, if applicable
	metadata := make(map[string]any)
	if len(authInvoke) > 0 {
		metadata["toolbox/authInvoke"] = authInvoke
	}
	if len(authParams) > 0 {
		metadata["toolbox/authParam"] = authParams
	}
	if len(metadata) > 0 {
		mcpManifest.Metadata = metadata
	}
	return mcpManifest
}

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
