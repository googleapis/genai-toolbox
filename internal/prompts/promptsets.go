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
	"fmt"

	"github.com/googleapis/mcp-toolbox/internal/tools"
)

type PromptsetConfig struct {
	Name        string   `yaml:"name"`
	PromptNames []string `yaml:",inline"`
}

type Promptset struct {
	PromptsetConfig
	Prompts       []*Prompt         `yaml:",inline"`
	Manifest      PromptsetManifest `yaml:",inline"`
	McpManifest   []McpManifest     `yaml:",inline"`
	promptNameSet map[string]struct{}
}

func (p Promptset) ToConfig() PromptsetConfig {
	return p.PromptsetConfig
}

// ContainsPrompt reports whether the promptset includes a prompt with the given name.
// When built via Initialize, lookups are O(1) via promptNameSet; for Promptsets
// constructed directly (e.g., in tests), falls back to a linear scan of PromptNames.
func (p Promptset) ContainsPrompt(name string) bool {
	if p.promptNameSet != nil {
		_, ok := p.promptNameSet[name]
		return ok
	}
	for _, n := range p.PromptNames {
		if n == name {
			return true
		}
	}
	return false
}

type PromptsetManifest struct {
	ServerVersion   string              `json:"serverVersion"`
	PromptsManifest map[string]Manifest `json:"prompts"`
}

func (p PromptsetConfig) Initialize(serverVersion string, promptsMap map[string]Prompt) (Promptset, error) {
	// Check each declared prompt name exists
	promptset := Promptset{
		PromptsetConfig: p,
		Prompts:         make([]*Prompt, 0, len(p.PromptNames)),
		Manifest: PromptsetManifest{
			ServerVersion:   serverVersion,
			PromptsManifest: make(map[string]Manifest, len(p.PromptNames)),
		},
		McpManifest:   make([]McpManifest, 0, len(p.PromptNames)),
		promptNameSet: make(map[string]struct{}, len(p.PromptNames)),
	}
	if !tools.IsValidName(promptset.Name) {
		return promptset, fmt.Errorf("invalid promptset name: %s", promptset.Name)
	}
	for _, promptName := range p.PromptNames {
		prompt, ok := promptsMap[promptName]
		if !ok {
			return promptset, fmt.Errorf("prompt does not exist: %s", promptName)
		}
		promptset.Prompts = append(promptset.Prompts, &prompt)
		promptset.Manifest.PromptsManifest[promptName] = prompt.Manifest()
		promptset.McpManifest = append(promptset.McpManifest, prompt.McpManifest())
		promptset.promptNameSet[promptName] = struct{}{}
	}

	return promptset, nil
}
