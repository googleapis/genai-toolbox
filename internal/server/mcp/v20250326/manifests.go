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

package v20250326

import (
	"fmt"

	"github.com/googleapis/mcp-toolbox/internal/prompts"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
)

// generateToolManifest generates Tool for list tools result
func generateToolManifest(name, desc string, authInvoke []string, params parameters.Parameters, annotations *tools.ToolAnnotations, supportsSecureParams bool) (Tool, bool) {
	stdSchema, secureSchema, authParams := generateParamManifests(params, supportsSecureParams)

	// If the tool has secure parameters, and the client does NOT support secure parameters,
	// we should omit this tool entirely.
	hasSecureParams := false
	for _, p := range params {
		if p.GetSecure() {
			hasSecureParams = true
			break
		}
	}
	if hasSecureParams && !supportsSecureParams {
		return Tool{}, false
	}

	var toolAnnotations *ToolAnnotations
	if annotations != nil {
		toolAnnotations = &ToolAnnotations{
			DestructiveHint: annotations.DestructiveHint,
			IdempotentHint:  annotations.IdempotentHint,
			OpenWorldHint:   annotations.OpenWorldHint,
			ReadOnlyHint:    annotations.ReadOnlyHint,
		}
	}
	mcpManifest := Tool{
		Name:            name,
		Description:     desc,
		ToolInputSchema: stdSchema,
		Annotations:     toolAnnotations,
	}
	if secureSchema != nil {
		mcpManifest.SecureInputSchema = secureSchema
	}

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
	return mcpManifest, true
}

// generateParamManifests splits parameters into standard and secure manifests.
func generateParamManifests(ps parameters.Parameters, supportsSecureParams bool) (InputSchema, *InputSchema, map[string][]string) {
	stdProps := make(map[string]parameters.ParameterMcpManifest)
	stdReq := make([]string, 0)

	secProps := make(map[string]parameters.ParameterMcpManifest)
	secReq := make([]string, 0)

	authParam := make(map[string][]string)

	for _, p := range ps {
		// If the parameter is sourced from another param, skip it in the MCP manifest
		if p.GetValueFromParam() != "" {
			continue
		}

		name := p.GetName()
		paramManifest, authParamList := p.McpManifest()
		defaultV := p.GetDefault()
		if defaultV != nil {
			paramManifest.Default = defaultV
		}

		if len(authParamList) > 0 {
			authParam[name] = authParamList
		}

		if p.GetSecure() {
			if supportsSecureParams {
				secProps[name] = paramManifest
				if parameters.CheckParamRequired(p.GetRequired(), defaultV) {
					secReq = append(secReq, name)
				}
			}
		} else {
			stdProps[name] = paramManifest
			if parameters.CheckParamRequired(p.GetRequired(), defaultV) {
				stdReq = append(stdReq, name)
			}
		}
	}

	var secureSchema *InputSchema
	if supportsSecureParams && len(secProps) > 0 {
		secureSchema = &InputSchema{
			Type:       "object",
			Properties: secProps,
			Required:   secReq,
		}
	}

	return InputSchema{
		Type:       "object",
		Properties: stdProps,
		Required:   stdReq,
	}, secureSchema, authParam
}

// GenerateListToolsResult generates tools/list method result according to mcp schema
func GenerateListToolsResult(t tools.Toolset, toolsMap map[string]tools.Tool, supportsSecureParams bool) (ListToolsResult, error) {
	mcpManifest := make([]Tool, 0, len(t.ToolNames))
	for _, toolName := range t.ToolNames {
		tool, ok := toolsMap[toolName]
		if !ok {
			return ListToolsResult{}, fmt.Errorf("tool does not exist: %s", toolName)
		}
		toolManifest, ok := generateToolManifest(toolName, tool.GetDescription(), tool.GetAuthRequired(), tool.GetParameters(), tool.GetAnnotations(), supportsSecureParams)
		if !ok {
			continue
		}
		mcpManifest = append(mcpManifest, toolManifest)
	}
	return ListToolsResult{Tools: mcpManifest}, nil
}

// generatePromptManifest generates a version-specific Prompt manifest for list/prompts
func generatePromptManifest(name, desc string, args prompts.Arguments) Prompt {
	mcpArgs := make([]PromptArgument, 0, len(args))
	for _, arg := range args {
		promptArg := PromptArgument{
			Name:        arg.GetName(),
			Description: arg.GetDesc(),
			Required:    parameters.CheckParamRequired(arg.GetRequired(), arg.GetDefault()),
		}
		mcpArgs = append(mcpArgs, promptArg)
	}
	return Prompt{
		Name:        name,
		Description: desc,
		Arguments:   mcpArgs,
	}
}

// GenerateListPromptsResult generates the list/prompts result
func GenerateListPromptsResult(p prompts.Promptset, promptsMap map[string]prompts.Prompt) (ListPromptsResult, error) {
	mcpManifest := make([]Prompt, 0, len(p.PromptNames))
	for _, promptName := range p.PromptNames {
		prompt, ok := promptsMap[promptName]
		if !ok {
			return ListPromptsResult{}, fmt.Errorf("prompt does not exist: %s", promptName)
		}
		promptManifest := generatePromptManifest(promptName, prompt.GetDesc(), prompt.GetArguments())
		mcpManifest = append(mcpManifest, promptManifest)
	}
	return ListPromptsResult{Prompts: mcpManifest}, nil
}
