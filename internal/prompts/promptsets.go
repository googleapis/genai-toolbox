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

	"github.com/googleapis/genai-toolbox/internal/tools"
)

type PromptsetConfig struct {
	Name        string   `yaml:"name"`
	PromptNames []string `yaml:",inline"`
}

type Promptset struct {
	PromptsetConfig
	Prompts     []*Prompt         `yaml:",inline"`
	Manifest    PromptsetManifest `yaml:",inline"`
	McpManifest []McpManifest     `yaml:",inline"`
}

func (p Promptset) ToConfig() PromptsetConfig {
	return p.PromptsetConfig
}

func (ps *Promptset) Initialize(serverVersion string, promptsMap map[string]Prompt) error {
	ps.Prompts = make([]*Prompt, 0, len(ps.PromptNames))
	ps.McpManifest = make([]McpManifest, 0, len(ps.PromptNames))
	ps.Manifest = PromptsetManifest{
		ServerVersion:   serverVersion,
		PromptsManifest: make(map[string]Manifest, len(ps.PromptNames)),
	}
	for _, promptName := range ps.PromptNames {
		prompt, ok := promptsMap[promptName]
		if !ok {
			return fmt.Errorf("prompt does not exist: %s", promptName)
		}
		p := prompt
		ps.Prompts = append(ps.Prompts, &p)
		ps.Manifest.PromptsManifest[promptName] = prompt.Manifest()
		ps.McpManifest = append(ps.McpManifest, prompt.McpManifest())
	}
	return nil
}

type PromptsetManifest struct {
	ServerVersion   string              `json:"serverVersion"`
	PromptsManifest map[string]Manifest `json:"prompts"`
}

func (p PromptsetConfig) Initialize(serverVersion string, promptsMap map[string]Prompt) (Promptset, error) {
	// Check each declared prompt name exists
	promptset := &Promptset{PromptsetConfig: p}
	promptset.Name = p.Name
	if !tools.IsValidName(promptset.Name) {
		return *promptset, fmt.Errorf("invalid promptset name: %s", promptset.Name)
	}
	err := promptset.Initialize(serverVersion, promptsMap)
	if err != nil {
		return *promptset, err
	}
	return *promptset, nil
}
