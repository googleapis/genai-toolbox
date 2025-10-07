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
	"fmt"

	"github.com/googleapis/genai-toolbox/internal/tools"
)

type PromptsetConfig struct {
	Name        string   `yaml:"name"`
	PromptNames []string `yaml:",inline"`
}

type Promptset struct {
	Name        string            `yaml:"name"`
	Prompts     []*Prompt         `yaml:",inline"`
	Manifest    PromptsetManifest `yaml:",inline"`
	McpManifest []McpManifest     `yaml:",inline"`
}

type PromptsetManifest struct {
	ServerVersion   string              `json:"serverVersion"`
	PromptsManifest map[string]Manifest `json:"prompts"`
}

func (t PromptsetConfig) Initialize(serverVersion string, promptsMap map[string]Prompt) (Promptset, error) {
	// Check each declared prompt name exists
	var promptset Promptset
	promptset.Name = t.Name
	if !tools.IsValidName(promptset.Name) {
		return promptset, fmt.Errorf("invalid promptset name: %s", t)
	}
	promptset.Prompts = make([]*Prompt, 0, len(t.PromptNames))
	promptset.McpManifest = make([]McpManifest, 0, len(t.PromptNames))
	promptset.Manifest = PromptsetManifest{
		ServerVersion:   serverVersion,
		PromptsManifest: make(map[string]Manifest, len(t.PromptNames)),
	}
	for _, promptName := range t.PromptNames {
		prompt, ok := promptsMap[promptName]
		if !ok {
			return promptset, fmt.Errorf("prompt does not exist: %s", t)
		}
		promptset.Prompts = append(promptset.Prompts, &prompt)
		promptset.Manifest.PromptsManifest[promptName] = prompt.Manifest()
		promptset.McpManifest = append(promptset.McpManifest, prompt.McpManifest())
	}

	return promptset, nil
}
