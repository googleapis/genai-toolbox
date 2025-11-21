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
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
)

type Message = prompts.Message

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

// Interface compliance checks.
var _ prompts.PromptConfig = Config{}
var _ prompts.Prompt = Prompt{}

func (c Config) PromptConfigKind() string {
	return kind
}

func (c Config) Initialize() (prompts.Prompt, error) {
	p := Prompt{
		Config:      c,
		manifest:    prompts.GetManifest(c.Description, c.Arguments),
		mcpManifest: prompts.GetMcpManifest(c.Name, c.Description, c.Arguments),
	}
	return p, nil
}

type Prompt struct {
	Config
	manifest    prompts.Manifest
	mcpManifest prompts.McpManifest
}

func (p Prompt) ToConfig() prompts.PromptConfig {
	return p.Config
}

func (p Prompt) Manifest() prompts.Manifest {
	return p.manifest
}

func (p Prompt) McpManifest() prompts.McpManifest {
	return p.mcpManifest
}

func (p Prompt) SubstituteParams(argValues parameters.ParamValues) (any, error) {
	return prompts.SubstituteMessages(p.Messages, p.Arguments, argValues)
}

func (p Prompt) ParseArgs(args map[string]any, data map[string]map[string]any) (parameters.ParamValues, error) {
	return prompts.ParseArguments(p.Arguments, args, data)
}
