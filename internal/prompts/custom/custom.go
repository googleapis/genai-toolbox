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

package custom_prompts

import (
	"bytes"
	"fmt"
	"html/template"

	"github.com/googleapis/genai-toolbox/internal/prompts"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

// Configuration for the custom prompt
type Config struct {
	Name        string            `yaml:"name" validate:"required"`
	Kind        string            `yaml:"kind"`
	Description string            `yaml:"description"`
	Arguments   prompts.Arguments `yaml:"arguments"`
	Messages    []prompts.Message `yaml:"messages"`
}

// validate interface
var _ prompts.Prompt = Config{}

// Manifest implements the Manifest method of the Prompt interface.
func (p Config) Manifest() prompts.Manifest {
	var paramManifests []tools.ParameterManifest
	for _, arg := range p.Arguments {
		paramManifests = append(paramManifests, arg.Manifest())
	}
	return prompts.Manifest{
		Description: p.Description,
		Arguments:   paramManifests,
	}
}

func (p Config) Initialize() (prompts.Prompt, error) {
	if p.Kind == "" {
		p.Kind = "custom"
	} else if p.Kind != "custom" {
		return nil, fmt.Errorf("kind must be 'custom'")
	}

	for i := range p.Messages {
		if p.Messages[i].Role == "" {
			p.Messages[i].Role = "user"
		}
	}
	for i := range p.Arguments {
		if p.Arguments[i].Type == "" {
			p.Arguments[i].Type = "any"
		}
		if p.Arguments[i].Required == nil {
			b := true
			p.Arguments[i].Required = &b
		}
	}
	return p, nil
}

func (p Config) SubstituteParams(argValues tools.ParamValues) (any, error) {
	substitutedMessages := []prompts.Message{}

	argsMap := make(map[string]any)
	for _, arg := range argValues {
		argsMap[arg.Name] = arg.Value
	}

	for _, msg := range p.Messages {
		tpl, err := template.New("message").Option("missingkey=error").Parse(msg.Content)
		if err != nil {
			return nil, err
		}

		var buf bytes.Buffer
		if err := tpl.Execute(&buf, argsMap); err != nil {
			return nil, err
		}

		substitutedMessages = append(substitutedMessages, prompts.Message{
			Role:    msg.Role,
			Content: buf.String(),
		})
	}

	return substitutedMessages, nil
}

func (p Config) ParseArgs(args map[string]any, data map[string]map[string]any) (tools.ParamValues, error) {
	var parameters tools.Parameters
	for _, arg := range p.Arguments {
		parameters = append(parameters, arg)
	}
	return tools.ParseParams(parameters, args, data)
}

func (p Config) McpManifest() prompts.McpManifest {
	var mcpArgs []prompts.McpPromptArg
	for _, arg := range p.Arguments {
		mcpArgs = append(mcpArgs, arg.McpPromptManifest())
	}

	return prompts.McpManifest{
		Name:        p.Name,
		Description: p.Description,
		Arguments:   mcpArgs,
	}
}
