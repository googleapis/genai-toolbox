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

package custom

import (
	"context"
	"fmt"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/prompts"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

const kind = "custom"

// init registers this prompt kind with the prompt framework.
func init() {
	if !prompts.Register(kind, newConfig) {
		panic(fmt.Sprintf("prompt kind %q already registered", kind))
	}
}

// newConfig is the factory function for creating a custom prompt configuration.
func newConfig(ctx context.Context, name string, decoder *yaml.Decoder) (prompts.PromptConfig, error) {
	cfg := &Config{Name: name}
	if err := decoder.DecodeContext(ctx, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Config is the configuration for a custom prompt.
// It implements both the prompts.PromptConfig and prompts.Prompt interfaces.
type Config struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description,omitempty"`
	Messages    []Message         `yaml:"messages"`
	Arguments   prompts.Arguments `yaml:"arguments,omitempty"`
}

// Message represents a single message in a prompt.
type Message struct {
	Role    string `yaml:"role,omitempty"`
	Content string `yaml:"content"`
}

// Interface compliance checks.
var _ prompts.PromptConfig = (*Config)(nil)
var _ prompts.Prompt = (*Config)(nil)

func (c *Config) PromptConfigKind() string {
	return kind
}

func (c *Config) Initialize() (prompts.Prompt, error) {
	// For this simple prompt, the config is also the runnable prompt.
	return c, nil
}

func (c *Config) Manifest() prompts.Manifest {
	var paramManifests []tools.ParameterManifest
	for _, arg := range c.Arguments {
		paramManifests = append(paramManifests, arg.Manifest())
	}
	return prompts.Manifest{
		Description: c.Description,
		Arguments:   paramManifests,
	}
}

func (c *Config) McpManifest() prompts.McpManifest {
	return getMcpManifest(c.Name, c.Description, c.Arguments)
}

func (c *Config) SubstituteParams(argValues tools.ParamValues) (any, error) {
	substitutedMessages := []Message{}
	argsMap := argValues.AsMap()

	var parameters tools.Parameters
	for _, arg := range c.Arguments {
		parameters = append(parameters, arg)
	}

	for _, msg := range c.Messages {
		substitutedContent, err := tools.ResolveTemplateParams(parameters, msg.Content, argsMap)
		if err != nil {
			return nil, fmt.Errorf("error substituting params for message: %w", err)
		}
		substitutedMessages = append(substitutedMessages, Message{
			Role:    msg.Role,
			Content: substitutedContent,
		})
	}
	return substitutedMessages, nil
}

func (c *Config) ParseArgs(args map[string]any, data map[string]map[string]any) (tools.ParamValues, error) {
	var parameters tools.Parameters
	for _, arg := range c.Arguments {
		parameters = append(parameters, arg)
	}
	return tools.ParseParams(parameters, args, data)
}

func getMcpManifest(name, desc string, args prompts.Arguments) prompts.McpManifest {
	mcpArgs := make([]prompts.McpArgManifest, 0, len(args))
	for _, arg := range args {
		mcpArgs = append(mcpArgs, arg.McpArgManifest())
	}
	return prompts.McpManifest{
		Name:        name,
		Description: desc,
		Arguments:   mcpArgs,
	}
}
