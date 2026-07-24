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

package group_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/googleapis/mcp-toolbox/internal/group"
	"github.com/googleapis/mcp-toolbox/internal/prompts"
	"github.com/googleapis/mcp-toolbox/internal/server/primitives"
	"github.com/googleapis/mcp-toolbox/internal/testutils"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
)

func testFixtures() (map[string]tools.Tool, map[string]prompts.Prompt) {
	toolsMap := map[string]tools.Tool{
		"tool1": testutils.NewMockTool("tool1", "first tool", []parameters.Parameter{}, false, false),
		"tool2": testutils.NewMockTool("tool2", "second tool", []parameters.Parameter{}, false, false),
	}
	promptsMap := map[string]prompts.Prompt{
		"prompt1": testutils.NewMockPrompt("prompt1", "first prompt", prompts.Arguments{}),
		"prompt2": testutils.NewMockPrompt("prompt2", "second prompt", prompts.Arguments{}),
	}
	return toolsMap, promptsMap
}

func TestGroupConfig_Initialize(t *testing.T) {
	t.Parallel()
	toolsMap, promptsMap := testFixtures()

	testCases := []struct {
		name        string
		config      group.GroupConfig
		wantTools   []string
		wantPrompts []string
		wantErr     string
	}{
		{
			name: "tools and prompts",
			config: group.GroupConfig{
				Name:        "mygroup",
				Description: "a group",
				ToolNames:   []string{"tool1", "tool2"},
				PromptNames: []string{"prompt1", "prompt2"},
			},
			wantTools:   []string{"tool1", "tool2"},
			wantPrompts: []string{"prompt1", "prompt2"},
		},
		{
			name: "tools only",
			config: group.GroupConfig{
				Name:      "toolsonly",
				ToolNames: []string{"tool1"},
			},
			wantTools:   []string{"tool1"},
			wantPrompts: nil,
		},
		{
			name: "prompts only",
			config: group.GroupConfig{
				Name:        "promptsonly",
				PromptNames: []string{"prompt1"},
			},
			wantTools:   nil,
			wantPrompts: []string{"prompt1"},
		},
		{
			name: "default nameless group",
			config: group.GroupConfig{
				Name:        "",
				ToolNames:   []string{"tool1"},
				PromptNames: []string{"prompt1"},
			},
			wantTools:   []string{"tool1"},
			wantPrompts: []string{"prompt1"},
		},
		{
			name: "invalid group name",
			config: group.GroupConfig{
				Name:      "bad name!",
				ToolNames: []string{"tool1"},
			},
			wantErr: "invalid group name",
		},
		{
			name: "missing tool",
			config: group.GroupConfig{
				Name:      "g",
				ToolNames: []string{"nope"},
			},
			wantErr: "tool does not exist: \"nope\"",
		},
		{
			name: "missing prompt",
			config: group.GroupConfig{
				Name:        "g",
				PromptNames: []string{"nope"},
			},
			wantErr: "prompt does not exist: \"nope\"",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g, err := tc.config.Initialize(toolsMap, promptsMap)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error = %q, want it to contain %q", err.Error(), tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !slices.Equal(g.ToolNames, tc.wantTools) {
				t.Errorf("tools = %v, want %v", g.ToolNames, tc.wantTools)
			}
			if !slices.Equal(g.PromptNames, tc.wantPrompts) {
				t.Errorf("prompts = %v, want %v", g.PromptNames, tc.wantPrompts)
			}
			for _, name := range tc.wantTools {
				if !g.ContainsTool(name) {
					t.Errorf("group missing tool %q", name)
				}
			}
			for _, name := range tc.wantPrompts {
				if !g.ContainsPrompt(name) {
					t.Errorf("group missing prompt %q", name)
				}
			}
		})
	}
}

func TestGroup_ToolsetManifest(t *testing.T) {
	t.Parallel()
	toolsMap, _ := testFixtures()

	g := group.NewGroup(group.GroupConfig{
		Name:      "mygroup",
		ToolNames: []string{"tool1", "tool2"},
	})
	mgr := primitives.NewPrimitiveManager(nil, nil, nil, toolsMap, nil, nil)

	manifest, err := g.ToolsetManifest("v1.2.3", mgr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if manifest.ServerVersion != "v1.2.3" {
		t.Errorf("ServerVersion = %q, want %q", manifest.ServerVersion, "v1.2.3")
	}
	wantTools := []string{"tool1", "tool2"}
	gotTools := make([]string, 0, len(manifest.ToolsManifest))
	for name := range manifest.ToolsManifest {
		gotTools = append(gotTools, name)
	}
	slices.Sort(gotTools)
	if !slices.Equal(gotTools, wantTools) {
		t.Errorf("tools manifest keys = %v, want %v", gotTools, wantTools)
	}
	if manifest.ToolsManifest["tool1"].Description != "first tool" {
		t.Errorf("tool1 description = %q, want %q", manifest.ToolsManifest["tool1"].Description, "first tool")
	}

	// Missing tool error path.
	missing := group.NewGroup(group.GroupConfig{Name: "g", ToolNames: []string{"nope"}})
	if _, err := missing.ToolsetManifest("v1", mgr); err == nil {
		t.Fatal("expected error for missing tool, got nil")
	} else if !strings.Contains(err.Error(), "tool does not exist: nope") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "tool does not exist: nope")
	}
}

func TestGroup_Contains(t *testing.T) {
	t.Parallel()
	toolsMap, promptsMap := testFixtures()

	g, err := group.GroupConfig{
		Name:        "mygroup",
		Description: "a group",
		ToolNames:   []string{"tool1", "tool2"},
		PromptNames: []string{"prompt1", "prompt2"},
	}.Initialize(toolsMap, promptsMap)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !g.ContainsTool("tool1") || !g.ContainsTool("tool2") {
		t.Errorf("group missing expected tools")
	}
	if g.ContainsTool("tool3") {
		t.Errorf("group reports an absent tool")
	}
	if !g.ContainsPrompt("prompt1") || !g.ContainsPrompt("prompt2") {
		t.Errorf("group missing expected prompts")
	}
	if g.ContainsPrompt("prompt3") {
		t.Errorf("group reports an absent prompt")
	}
}
