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

// Group is an initialized group: the source of truth for a named collection of
// tools and prompts. It is a self-contained MCP primitive and does not depend on
// the legacy toolset/promptset types. It keeps O(1) membership sets for its tools
// and prompts; per-tool and per-prompt manifests are generated on demand by
// callers from the resolved tools and prompts maps.
type Group struct {
	GroupConfig
	toolNameSet   map[string]struct{}
	promptNameSet map[string]struct{}
}

// Initialize validates the group name and checks that every declared tool and
// prompt exists in the provided maps, building the membership sets used by
// ContainsTool and ContainsPrompt.
func (gc GroupConfig) Initialize(toolsMap map[string]tools.Tool, promptsMap map[string]prompts.Prompt) (Group, error) {
	if !tools.IsValidName(gc.Name) {
		return Group{}, fmt.Errorf("invalid group name: %s", gc.Name)
	}

	toolNameSet := make(map[string]struct{}, len(gc.ToolNames))
	for _, name := range gc.ToolNames {
		if _, ok := toolsMap[name]; !ok {
			return Group{}, fmt.Errorf("tool does not exist: %s", name)
		}
		toolNameSet[name] = struct{}{}
	}

	promptNameSet := make(map[string]struct{}, len(gc.PromptNames))
	for _, name := range gc.PromptNames {
		if _, ok := promptsMap[name]; !ok {
			return Group{}, fmt.Errorf("prompt does not exist: %s", name)
		}
		promptNameSet[name] = struct{}{}
	}

	return Group{GroupConfig: gc, toolNameSet: toolNameSet, promptNameSet: promptNameSet}, nil
}

// NewGroup builds a Group directly from its config, deriving the membership sets
// from the declared tool and prompt names. It skips the existence checks
// performed by GroupConfig.Initialize and is intended for tests and callers that
// have already validated the names.
func NewGroup(config GroupConfig) Group {
	toolNameSet := make(map[string]struct{}, len(config.ToolNames))
	for _, name := range config.ToolNames {
		toolNameSet[name] = struct{}{}
	}
	promptNameSet := make(map[string]struct{}, len(config.PromptNames))
	for _, name := range config.PromptNames {
		promptNameSet[name] = struct{}{}
	}
	return Group{GroupConfig: config, toolNameSet: toolNameSet, promptNameSet: promptNameSet}
}

// ContainsTool reports whether the group includes a tool with the given name.
func (g Group) ContainsTool(name string) bool {
	_, ok := g.toolNameSet[name]
	return ok
}

// ContainsPrompt reports whether the group includes a prompt with the given name.
func (g Group) ContainsPrompt(name string) bool {
	_, ok := g.promptNameSet[name]
	return ok
}
