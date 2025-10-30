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

package prompts_test

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/prompts"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

// mockPrompt is a simple mock implementation of prompts.Prompt for testing.
type mockPrompt struct {
	name        string
	desc        string
	args        prompts.Arguments
	manifest    prompts.Manifest
	mcpManifest prompts.McpManifest
}

func (m *mockPrompt) SubstituteParams(tools.ParamValues) (any, error) { return nil, nil }
func (m *mockPrompt) ParseArgs(map[string]any, map[string]map[string]any) (tools.ParamValues, error) {
	return nil, nil
}
func (m *mockPrompt) Manifest() prompts.Manifest       { return m.manifest }
func (m *mockPrompt) McpManifest() prompts.McpManifest { return m.mcpManifest }

// newMockPrompt creates a new mock prompt for testing.
func newMockPrompt(name, desc string) prompts.Prompt {
	args := prompts.Arguments{
		{Parameter: tools.NewStringParameter("arg1", "Test argument")},
	}
	return &mockPrompt{
		name: name,
		desc: desc,
		args: args,
		manifest: prompts.Manifest{
			Description: desc,
			Arguments: []tools.ParameterManifest{
				{Name: "arg1", Type: "string", Required: true, Description: "Test argument", AuthServices: []string{}},
			},
		},
		mcpManifest: prompts.McpManifest{
			Name:        name,
			Description: desc,
			Arguments: []prompts.ArgMcpManifest{
				{Name: "arg1", Description: "Test argument", Required: true},
			},
		},
	}
}

func TestPromptsetConfig_Initialize(t *testing.T) {
	t.Parallel()

	promptsMap := map[string]prompts.Prompt{
		"prompt1": newMockPrompt("prompt1", "First test prompt"),
		"prompt2": newMockPrompt("prompt2", "Second test prompt"),
	}
	serverVersion := "v1.0.0"

	p1 := promptsMap["prompt1"]
	p2 := promptsMap["prompt2"]
	prompt1Ptr := &p1
	prompt2Ptr := &p2

	testCases := []struct {
		name    string
		config  prompts.PromptsetConfig
		want    prompts.Promptset
		wantErr string
	}{
		{
			name: "Success case",
			config: prompts.PromptsetConfig{
				Name:        "default",
				PromptNames: []string{"prompt1", "prompt2"},
			},
			want: prompts.Promptset{
				Name: "default",
				Prompts: []*prompts.Prompt{
					prompt1Ptr,
					prompt2Ptr,
				},
				Manifest: prompts.PromptsetManifest{
					ServerVersion: serverVersion,
					PromptsManifest: map[string]prompts.Manifest{
						"prompt1": promptsMap["prompt1"].Manifest(),
						"prompt2": promptsMap["prompt2"].Manifest(),
					},
				},
				McpManifest: []prompts.McpManifest{
					promptsMap["prompt1"].McpManifest(),
					promptsMap["prompt2"].McpManifest(),
				},
			},
			wantErr: "",
		},
		{
			name: "Success case with one prompt",
			config: prompts.PromptsetConfig{
				Name:        "single",
				PromptNames: []string{"prompt1"},
			},
			want: prompts.Promptset{
				Name: "single",
				Prompts: []*prompts.Prompt{
					prompt1Ptr,
				},
				Manifest: prompts.PromptsetManifest{
					ServerVersion: serverVersion,
					PromptsManifest: map[string]prompts.Manifest{
						"prompt1": promptsMap["prompt1"].Manifest(),
					},
				},
				McpManifest: []prompts.McpManifest{
					promptsMap["prompt1"].McpManifest(),
				},
			},
			wantErr: "",
		},
		{
			name: "Failure case - invalid promptset name",
			config: prompts.PromptsetConfig{
				Name:        "invalid name", // Contains a space
				PromptNames: []string{"prompt1"},
			},
			want:    prompts.Promptset{Name: "invalid name"}, // Expect partial struct
			wantErr: "invalid promptset name",
		},
		{
			name: "Failure case - prompt not found",
			config: prompts.PromptsetConfig{
				Name:        "missing_prompt",
				PromptNames: []string{"prompt1", "prompt_does_not_exist"},
			},
			// Expect partial struct with fields populated up to the error
			want: prompts.Promptset{
				Name: "missing_prompt",
				Prompts: []*prompts.Prompt{
					prompt1Ptr,
				},
				Manifest: prompts.PromptsetManifest{
					ServerVersion: serverVersion,
					PromptsManifest: map[string]prompts.Manifest{
						"prompt1": promptsMap["prompt1"].Manifest(),
					},
				},
				McpManifest: []prompts.McpManifest{
					promptsMap["prompt1"].McpManifest(),
				},
			},
		},
		{
			name: "Success case - empty prompt list",
			config: prompts.PromptsetConfig{
				Name:        "empty",
				PromptNames: []string{},
			},
			want: prompts.Promptset{
				Name:    "empty",
				Prompts: []*prompts.Prompt{},
				Manifest: prompts.PromptsetManifest{
					ServerVersion:   serverVersion,
					PromptsManifest: map[string]prompts.Manifest{},
				},
				McpManifest: []prompts.McpManifest{},
			},
			wantErr: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.config.Initialize(serverVersion, promptsMap)

			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("Initialize() expected error but got nil")
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("Initialize() error mismatch:\n  want to contain: %q\n  got: %q", tc.wantErr, err.Error())
				}
				// Also check that the partially populated struct matches
				if diff := cmp.Diff(tc.want, got, cmp.AllowUnexported(mockPrompt{})); diff != "" {
					t.Errorf("Initialize() partial result on error mismatch (-want +got):\n%s", diff)
				}
			} else {
				if err != nil {
					t.Fatalf("Initialize() returned unexpected error: %v", err)
				}
				// Using cmp.AllowUnexported because mockPrompt is unexported
				if diff := cmp.Diff(tc.want, got, cmp.AllowUnexported(mockPrompt{})); diff != "" {
					t.Errorf("Initialize() result mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
