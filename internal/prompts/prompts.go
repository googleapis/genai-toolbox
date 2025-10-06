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

type Message struct {
	Role    string `yaml:"role,omitempty"`
	Content string `yaml:"content"`
}

// Prompt config as defined by the user
type PromptConfig struct {
	Name        string     `yaml:"name"`
	Description string     `yaml:"description"`
	Messages    []Message  `yaml:"messages"`
	Arguments   []Argument `yaml:"arguments,omitempty"`
}

// ParamValues is an ordered list of ParamValue
type ArgValues []ArgValue

// ParamValue represents the parameter's name and value.
type ArgValue struct {
	Name  string
	Value any
}

type Prompt interface {
	Initialize() (Prompt, error)
	SubstituteParams(ArgValues) (any, error)
	ParseArgs(map[string]any, map[string]map[string]any) (ArgValues, error)
	Manifest() Manifest
	McpManifest() McpManifest
}

// Manifest is the representation of prompts sent to Client SDKs.
type Manifest struct {
	Description string             `json:"description"`
	Arguments   []ArgumentManifest `json:"arguments"`
}

// Definition for a prompt the MCP client can call.
type McpManifest struct {
	// The name of the prompt.
	Name string `json:"name"`
	// A human-readable description of the prompt.
	Description string `json:"description,omitempty"`
	// A JSON Schema object defining the expected arguments for the prompt.
	Arguments []McpPromptArgs `json:"inputSchema,omitempty"`
	Metadata  map[string]any  `json:"_meta,omitempty"`
}
