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
	"slices"

	"github.com/googleapis/genai-toolbox/internal/sources"
)

type ToolConfig interface {
	ToolConfigKind() string
	Initialize(map[string]sources.Source) (Tool, error)
}

type Tool interface {
	Invoke(ParamValues) ([]any, error)
	ParseParams(map[string]any, map[string]map[string]any) (ParamValues, error)
	Manifest() Manifest
	Authorized([]string) bool
	MCPTool() MCPTool
}

// Manifest is the representation of tools sent to Client SDKs.
type Manifest struct {
	Description string              `json:"description"`
	Parameters  []ParameterManifest `json:"parameters"`
}

// Definition for a tool the MCP client can call.
type MCPTool struct {
	// The name of the tool.
	Name string `json:"name"`
	// A human-readable description of the tool.
	Description string `json:"desciprtion,omitempty"`
	// A JSON Schema object defining the expected parameters for the tool.
	InputSchema ToolsSchema `json:"inputSchema,omitempty"`
}

// ToolsSchema is the representation of input schema for MCPTool.
type ToolsSchema struct {
	Type       string        `json:"type"`
	Properties []McpProperty `json:"properties"`
	Required   []string      `json:"required"`
}

// ToolsSchema converts Manifest to MCPTool.
func (m Manifest) ToolsSchema() ToolsSchema {
	var properties []McpProperty
	var required []string

	for _, p := range m.Parameters {
		properties = append(properties, p.McpProperty())
		required = append(required, p.Name)
	}
	return ToolsSchema{
		Type:       "object",
		Properties: properties,
		Required:   required,
	}
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
