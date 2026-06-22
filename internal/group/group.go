// Copyright 2026 Google LLC
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

package group

import (
	"fmt"

	"github.com/googleapis/mcp-toolbox/internal/prompts"
	"github.com/googleapis/mcp-toolbox/internal/tools"
)

// GroupConfig is the parsed configuration for a group: a single named collection
// that holds both tools and prompts. Its description doubles as the MCP server
// instructions for clients connected to the group.
type GroupConfig struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	ToolNames   []string `yaml:"tools"`
	PromptNames []string `yaml:"prompts"`
}

// Group is an initialized group and the source of truth from which the legacy
// toolset and promptset views are derived (see ToToolset and ToPromptset).
type Group struct {
	GroupConfig
	Tools           []*tools.Tool
	ToolsManifest   tools.ToolsetManifest
	Prompts         []*prompts.Prompt
	PromptsManifest prompts.PromptsetManifest
}

// Initialize validates the declared tools and prompts against the provided maps
// and builds the derived tool/prompt slices and manifests. It delegates to
// tools.ToolsetConfig.Initialize and prompts.PromptsetConfig.Initialize so that
// validation and manifest-building stay identical to the legacy types.
func (gc GroupConfig) Initialize(serverVersion string, toolsMap map[string]tools.Tool, promptsMap map[string]prompts.Prompt) (Group, error) {
	if !tools.IsValidName(gc.Name) {
		return Group{}, fmt.Errorf("invalid group name: %s", gc.Name)
	}

	toolset, err := tools.ToolsetConfig{Name: gc.Name, ToolNames: gc.ToolNames}.Initialize(serverVersion, toolsMap)
	if err != nil {
		return Group{}, err
	}
	promptset, err := prompts.PromptsetConfig{Name: gc.Name, PromptNames: gc.PromptNames}.Initialize(serverVersion, promptsMap)
	if err != nil {
		return Group{}, err
	}

	return Group{
		GroupConfig:     gc,
		Tools:           toolset.Tools,
		ToolsManifest:   toolset.Manifest,
		Prompts:         promptset.Prompts,
		PromptsManifest: promptset.Manifest,
	}, nil
}

// ToToolset returns the derived toolset view, keyed by the group's name, so that
// existing toolset call sites keep working unchanged.
func (g Group) ToToolset() tools.Toolset {
	return tools.Toolset{
		ToolsetConfig: tools.ToolsetConfig{Name: g.Name, ToolNames: g.ToolNames},
		Tools:         g.Tools,
		Manifest:      g.ToolsManifest,
	}
}

// ToPromptset returns the derived promptset view, keyed by the group's name, so
// that prompts scope to the connected group.
func (g Group) ToPromptset() prompts.Promptset {
	promptNameSet := make(map[string]struct{}, len(g.PromptNames))
	for _, name := range g.PromptNames {
		promptNameSet[name] = struct{}{}
	}
	return prompts.Promptset{
		PromptsetConfig: prompts.PromptsetConfig{Name: g.Name, PromptNames: g.PromptNames},
		Prompts:         g.Prompts,
		Manifest:        g.PromptsManifest,
		PromptNameSet:   promptNameSet,
	}
}
