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
	"strings"
	"testing"

	"github.com/googleapis/mcp-toolbox/internal/group"
	"github.com/googleapis/mcp-toolbox/internal/prompts"
	"github.com/googleapis/mcp-toolbox/internal/testutils"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
)

const serverVersion = "test-version"

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
			wantPrompts: []string{},
		},
		{
			name: "prompts only",
			config: group.GroupConfig{
				Name:        "promptsonly",
				PromptNames: []string{"prompt1"},
			},
			wantTools:   []string{},
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
			wantErr: "tool does not exist: nope",
		},
		{
			name: "missing prompt",
			config: group.GroupConfig{
				Name:        "g",
				PromptNames: []string{"nope"},
			},
			wantErr: "prompt does not exist: nope",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g, err := tc.config.Initialize(serverVersion, toolsMap, promptsMap)
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
			if got := toolNames(g.Tools); !equalStrings(got, tc.wantTools) {
				t.Errorf("tools = %v, want %v", got, tc.wantTools)
			}
			if len(g.Prompts) != len(tc.wantPrompts) {
				t.Errorf("got %d prompts, want %d", len(g.Prompts), len(tc.wantPrompts))
			}
			ps := g.ToPromptset()
			for _, name := range tc.wantPrompts {
				if !ps.ContainsPrompt(name) {
					t.Errorf("derived promptset missing prompt %q", name)
				}
			}
			if g.ToolsManifest.ServerVersion != serverVersion {
				t.Errorf("tools manifest server version = %q, want %q", g.ToolsManifest.ServerVersion, serverVersion)
			}
			if g.PromptsManifest.ServerVersion != serverVersion {
				t.Errorf("prompts manifest server version = %q, want %q", g.PromptsManifest.ServerVersion, serverVersion)
			}
		})
	}
}

func TestGroup_Projections(t *testing.T) {
	t.Parallel()
	toolsMap, promptsMap := testFixtures()

	g, err := group.GroupConfig{
		Name:        "mygroup",
		Description: "a group",
		ToolNames:   []string{"tool1", "tool2"},
		PromptNames: []string{"prompt1", "prompt2"},
	}.Initialize(serverVersion, toolsMap, promptsMap)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ts := g.ToToolset()
	if ts.Name != "mygroup" {
		t.Errorf("toolset name = %q, want %q", ts.Name, "mygroup")
	}
	if !ts.ContainsTool("tool1") || !ts.ContainsTool("tool2") {
		t.Errorf("derived toolset missing expected tools")
	}
	if ts.ContainsTool("tool3") {
		t.Errorf("derived toolset reports an absent tool")
	}

	ps := g.ToPromptset()
	if ps.Name != "mygroup" {
		t.Errorf("promptset name = %q, want %q", ps.Name, "mygroup")
	}
	if !ps.ContainsPrompt("prompt1") || !ps.ContainsPrompt("prompt2") {
		t.Errorf("derived promptset missing expected prompts")
	}
	if ps.ContainsPrompt("prompt3") {
		t.Errorf("derived promptset reports an absent prompt")
	}
}

func toolNames(ts []*tools.Tool) []string {
	names := make([]string, 0, len(ts))
	for _, t := range ts {
		names = append(names, (*t).GetName())
	}
	return names
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
