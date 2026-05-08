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
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/googleapis/mcp-toolbox/internal/prompts"
	"github.com/googleapis/mcp-toolbox/internal/testutils"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
)

func TestPromptset_ContainsPrompt(t *testing.T) {
	t.Parallel()

	promptset := prompts.Promptset{
		PromptsetConfig: prompts.PromptsetConfig{
			Name:        "test-promptset",
			PromptNames: []string{"greet", "summarize"},
		},
	}

	tests := []struct {
		name       string
		promptName string
		want       bool
	}{
		{
			name:       "prompt exists in promptset",
			promptName: "greet",
			want:       true,
		},
		{
			name:       "another prompt exists in promptset",
			promptName: "summarize",
			want:       true,
		},
		{
			name:       "prompt not in promptset",
			promptName: "admin_prompt",
			want:       false,
		},
		{
			name:       "empty prompt name",
			promptName: "",
			want:       false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := promptset.ContainsPrompt(tc.promptName)
			if got != tc.want {
				t.Errorf("ContainsPrompt(%q) = %v, want %v", tc.promptName, got, tc.want)
			}
		})
	}
}

func TestPromptset_ContainsPrompt_EmptyPromptset(t *testing.T) {
	t.Parallel()

	promptset := prompts.Promptset{
		PromptsetConfig: prompts.PromptsetConfig{
			Name:        "empty-promptset",
			PromptNames: []string{},
		},
	}

	if promptset.ContainsPrompt("anything") {
		t.Error("ContainsPrompt should return false for empty promptset")
	}
}

func TestPromptsetConfig_Initialize(t *testing.T) {
	t.Parallel()

	args := prompts.Arguments{
		{Parameter: parameters.NewStringParameter("arg1", "Test argument")},
	}

	promptsMap := map[string]prompts.Prompt{
		"prompt1": testutils.NewMockPrompt("prompt1", "First test prompt", args),
		"prompt2": testutils.NewMockPrompt("prompt2", "Second test prompt", args),
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
				PromptsetConfig: prompts.PromptsetConfig{
					Name:        "default",
					PromptNames: []string{"prompt1", "prompt2"},
				},
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
				PromptsetConfig: prompts.PromptsetConfig{
					Name:        "single",
					PromptNames: []string{"prompt1"},
				},
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
			want: prompts.Promptset{
				PromptsetConfig: prompts.PromptsetConfig{
					Name:        "invalid name",
					PromptNames: []string{"prompt1"},
				},
				Prompts: []*prompts.Prompt{},
				Manifest: prompts.PromptsetManifest{
					ServerVersion:   serverVersion,
					PromptsManifest: map[string]prompts.Manifest{},
				},
				McpManifest: []prompts.McpManifest{},
			},
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
				PromptsetConfig: prompts.PromptsetConfig{
					Name:        "missing_prompt",
					PromptNames: []string{"prompt1", "prompt_does_not_exist"},
				},
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
			wantErr: "prompt does not exist",
		},
		{
			name: "Success case - empty prompt list",
			config: prompts.PromptsetConfig{
				Name:        "empty",
				PromptNames: []string{},
			},
			want: prompts.Promptset{
				PromptsetConfig: prompts.PromptsetConfig{
					Name:        "empty",
					PromptNames: []string{},
				},
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
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.config.Initialize(serverVersion, promptsMap)

			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("Initialize() expected error but got nil")
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("Initialize() error mismatch:\n  want to contain: %q\n  got: %q", tc.wantErr, err.Error())
				}
				if diff := cmp.Diff(tc.want, got, cmp.AllowUnexported(testutils.MockPrompt{}), cmpopts.IgnoreUnexported(prompts.Promptset{})); diff != "" {
					t.Errorf("Initialize() partial result on error mismatch (-want +got):\n%s", diff)
				}
			} else {
				if err != nil {
					t.Fatalf("Initialize() returned unexpected error: %v", err)
				}
				// Using cmp.AllowUnexported because MockPrompt is unexported
				if diff := cmp.Diff(tc.want, got, cmp.AllowUnexported(testutils.MockPrompt{}), cmpopts.IgnoreUnexported(prompts.Promptset{})); diff != "" {
					t.Errorf("Initialize() result mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
