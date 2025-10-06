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
	"text/template"

	"github.com/googleapis/genai-toolbox/internal/prompts"
)

// Configuration for the create-cluster tool.
type Config struct {
	Name        string             `yaml:"name" validate:"required"`
	Kind        string             `yaml:"kind"`
	Description string             `yaml:"description"`
	Arguments   []prompts.Argument `yaml:"arguments"`
	Messages    []prompts.Message  `yaml:"messages"`
}

// Manifest implements the Manifest method of the Prompt interface.
func (p Config) Manifest() prompts.Manifest {
	manifest := prompts.Manifest{
		Description: p.Description,
		Arguments:   prompts.ArgumentToManifest(p.Arguments),
	}
	return manifest
}

// Manifest implements the Manifest method of the Prompt interface.
func (p Config) McpManifest() prompts.McpManifest {
	mcpManifest := prompts.McpManifest{
		Name:        p.Name,
		Description: p.Description,
		Arguments:   prompts.ArgumentToMcpManifest(p.Arguments),
	}
	metadata := make(map[string]any)
	if len(metadata) > 0 {
		mcpManifest.Metadata = metadata
	}
	return mcpManifest
}

func (p Config) Initialize() (prompts.Prompt, error) {
	if p.Kind == "" {
		p.Kind = "custom"
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

func (p Config) SubstituteParams(argValues prompts.ArgValues) (any, error) {
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

func (p Config) ParseArgs(args map[string]any, data map[string]map[string]any) (prompts.ArgValues, error) {
	argValues := make(prompts.ArgValues, 0, len(p.Arguments))

	for _, arg := range p.Arguments {
		val, ok := args[arg.Name]
		if !ok && arg.Required != nil && *arg.Required {
			return nil, fmt.Errorf("required argument %q not provided", arg.Name)
		}

		if ok {
			argValues = append(argValues, prompts.ArgValue{
				Name:  arg.Name,
				Value: val,
			})
		}
	}

	return argValues, nil
}
