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

import (
	"context"
	"fmt"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

// PromptConfigFactory defines the signature for a function that creates and
// decodes a specific prompt's configuration.
type PromptConfigFactory func(ctx context.Context, name string, decoder *yaml.Decoder) (Config, error)

var promptRegistry = make(map[string]PromptConfigFactory)

// Register allows individual prompt packages to register their configuration
// factory function. This is typically called from an init() function in the
// prompt's package. It associates a 'kind' string with a function that can
// produce the specific PromptConfig type. It returns true if the registration was
// successful, and false if a prompt with the same kind was already registered.
func Register(kind string, factory PromptConfigFactory) bool {
	if _, exists := promptRegistry[kind]; exists {
		// Prompt with this kind already exists, do not overwrite.
		return false
	}
	promptRegistry[kind] = factory
	return true
}

// DecodeConfig looks up the registered factory for the given kind and uses it
// to decode the prompt configuration.
func DecodeConfig(ctx context.Context, kind, name string, decoder *yaml.Decoder) (PromptConfig, error) {
	factory, found := promptRegistry[kind]
	if !found {
		return nil, fmt.Errorf("unknown prompt kind: %q", kind)
	}
	promptConfig, err := factory(ctx, name, decoder)
	if err != nil {
		return nil, fmt.Errorf("unable to parse prompt %q as kind %q: %w", name, kind, err)
	}
	return promptConfig, nil
}

type PromptConfig interface {
	PromptConfigKind() string
	Initialize() (Prompt, error)
}

type Prompt interface {
	SubstituteParams(tools.ParamValues) (any, error)
	ParseArgs(map[string]any, map[string]map[string]any) (tools.ParamValues, error)
	Manifest() Manifest
	McpManifest() McpManifest
}

// Manifest is the representation of prompts sent to Client SDKs.
type Manifest struct {
	Description string                    `json:"description"`
	Arguments   []tools.ParameterManifest `json:"arguments"`
}

// McpPromptArg defines the simplified structure for a prompt's argument in the MCP manifest.
type McpPromptArg struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required"`
}

// McpManifest is the definition for a prompt the MCP client can get.
type McpManifest struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Arguments   []McpPromptArg `json:"inputSchema,omitempty"`
	Metadata    map[string]any `json:"_meta,omitempty"`
}

func GetMcpManifest(name, desc string, args Arguments) McpManifest {
	mcpManifest := McpManifest{
		Name:        name,
		Description: desc,
		Arguments:   args.McpManifest(),
	}

	// construct metadata, if applicable
	metadata := make(map[string]any)
	if len(metadata) > 0 {
		mcpManifest.Metadata = metadata
	}
	return mcpManifest
}

type Message struct {
	Role    string `yaml:"role,omitempty"`
	Content string `yaml:"content"`
}

// Config is the configuration for a prompt.
type Config struct {
	Name        string    `yaml:"name"`
	Kind        string    `yaml:"kind,omitempty"`
	Description string    `yaml:"description,omitempty"`
	Messages    []Message `yaml:"messages"`
	Arguments   Arguments `yaml:"arguments,omitempty"`
}

// ArgValues is an ordered list of ArgValue
type ArgValues []ArgValue

// ArgValue represents the parameter's name and value.
type ArgValue struct {
	Name  string
	Value any
}

// SystemPromptConfig is the configuration for a system prompt.
type SystemPromptConfig struct {
	Description string    `yaml:"description,omitempty"`
	Messages    []Message `yaml:"messages"`
	Arguments   Arguments `yaml:"arguments,omitempty"`
}
