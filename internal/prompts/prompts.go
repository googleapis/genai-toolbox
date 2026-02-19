// Copyright 2025 Google LLC
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
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
)

// PromptConfigFactory defines the signature for a function that creates and
// decodes a specific prompt's configuration.
type PromptConfigFactory func(ctx context.Context, name string, decoder *yaml.Decoder) (PromptConfig, error)

var promptRegistry = make(map[string]PromptConfigFactory)

// Register allows individual prompt packages to register their configuration
// factory function. This is typically called from an init() function in the
// prompt's package. It associates a 'type' string with a function that can
// produce the specific PromptConfig type. It returns true if the registration was
// successful, and false if a prompt with the same type was already registered.
func Register(resourceType string, factory PromptConfigFactory) bool {
	if _, exists := promptRegistry[resourceType]; exists {
		// Prompt with this type already exists, do not overwrite.
		return false
	}
	promptRegistry[resourceType] = factory
	return true
}

// DecodeConfig looks up the registered factory for the given type and uses it
// to decode the prompt configuration.
func DecodeConfig(ctx context.Context, resourceType, name string, decoder *yaml.Decoder) (PromptConfig, error) {
	factory, found := promptRegistry[resourceType]
	if !found && resourceType == "" {
		resourceType = "custom"
		factory, found = promptRegistry[resourceType]
	}

	if !found {
		return nil, fmt.Errorf("unknown prompt type: %q", resourceType)
	}

	promptConfig, err := factory(ctx, name, decoder)
	if err != nil {
		return nil, fmt.Errorf("unable to parse prompt %q as resourceType %q: %w", name, resourceType, err)
	}
	return promptConfig, nil
}

type PromptConfig interface {
	PromptConfigType() string
	Initialize() (Prompt, error)
}

type Prompt interface {
	SubstituteParams(parameters.ParamValues) (any, error)
	ParseArgs(map[string]any, map[string]map[string]any) (parameters.ParamValues, error)
	Manifest() Manifest
	McpManifest() McpManifest
	ToConfig() PromptConfig
}

// Manifest is the representation of prompts sent to Client SDKs.
type Manifest struct {
	Description string                         `json:"description"`
	Arguments   []parameters.ParameterManifest `json:"arguments"`
}

// McpManifest is the definition for a prompt the MCP client can get.
type McpManifest struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Arguments   []ArgMcpManifest `json:"arguments,omitempty"`
}

func GetMcpManifest(name, desc string, args Arguments) McpManifest {
	mcpArgs := make([]ArgMcpManifest, 0, len(args))
	for _, arg := range args {
		mcpArgs = append(mcpArgs, arg.McpManifest())
	}
	return McpManifest{
		Name:        name,
		Description: desc,
		Arguments:   mcpArgs,
	}
}

func GetManifest(desc string, args Arguments) Manifest {
	paramManifests := make([]parameters.ParameterManifest, 0, len(args))
	for _, arg := range args {
		paramManifests = append(paramManifests, arg.Manifest())
	}
	return Manifest{
		Description: desc,
		Arguments:   paramManifests,
	}
}
