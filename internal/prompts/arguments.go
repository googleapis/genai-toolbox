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

package prompts

// Arguments taken as user input
type Argument struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type,omitempty"`
	Description string `yaml:"description,omitempty"`
	Required    *bool  `yaml:"required,omitempty"`
}
type Arguments = []Argument

// Argument represents arguments when served as part of a ToolManifest.
type ArgumentManifest struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
}

// Argument represents arguments when served through MCP endpoints.
type McpPromptArgs struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
	Required    bool   `yaml:"required,omitempty"`
}

func ArgumentToManifest(args Arguments) []ArgumentManifest {
	manifests := make([]ArgumentManifest, 0, len(args))
	for _, arg := range args {
		required := true
		if arg.Required != nil {
			required = *arg.Required
		}
		manifest := ArgumentManifest{
			Name:        arg.Name,
			Type:        arg.Type,
			Required:    required,
			Description: arg.Description,
		}
		manifests = append(manifests, manifest)
	}

	return manifests
}

func ArgumentToMcpManifest(args Arguments) []McpPromptArgs {
	manifests := make([]McpPromptArgs, 0, len(args))
	for _, arg := range args {
		required := true
		if arg.Required != nil {
			required = *arg.Required
		}
		manifest := McpPromptArgs{
			Name:        arg.Name,
			Required:    required,
			Description: arg.Description,
		}
		manifests = append(manifests, manifest)
	}

	return manifests
}

// // McpPromptsSchema is the representation of input schema for McpManifest.
// type McpPromptsSchema struct {
// 	Type       string                          `json:"type"`
// 	Properties map[string]ParameterMcpManifest `json:"properties"`
// 	Required   []string                        `json:"required"`
// }

// func (ps Arguments) McpManifest() (McpPromptSchema, map[string][]string) {
// 	properties := make(map[string]ParameterMcpManifest)
// 	required := make([]string, 0)

// 	for _, p := range ps {
// 		name := p.GetName()
// 		paramManifest, authParamList := p.McpManifest()
// 		properties[name] = paramManifest
// 		// parameters that doesn't have a default value are added to the required field
// 		if CheckParamRequired(p.GetRequired(), p.GetDefault()) {
// 			required = append(required, name)
// 		}
// 		if len(authParamList) > 0 {
// 			authParam[name] = authParamList
// 		}
// 	}
// 	return McpPromptsSchema{
// 		Type:       "object",
// 		Properties: properties,
// 		Required:   required,
// 	}
// }
