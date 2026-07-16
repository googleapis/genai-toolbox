// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package custom

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/mcp-toolbox/internal/prompts"
	promptcustom "github.com/googleapis/mcp-toolbox/internal/prompts/custom"
	"github.com/googleapis/mcp-toolbox/internal/testutils"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
)

// The prompts coverage gate is computed solely from this integration test
// binary (see .ci/test_prompts_with_coverage.sh). The promptset lifecycle and
// the custom prompt config round-trip are not reachable through the server's
// HTTP surface, so they are exercised directly here to keep them under the gate.

func newPromptsMap() map[string]prompts.Prompt {
	args := prompts.Arguments{
		{Parameter: parameters.NewStringParameter("arg1", "first argument")},
	}
	return map[string]prompts.Prompt{
		"prompt1": testutils.NewMockPrompt("prompt1", "first test prompt", args),
		"prompt2": testutils.NewMockPrompt("prompt2", "second test prompt", args),
	}
}

func TestPromptsetInitialize(t *testing.T) {
	promptsMap := newPromptsMap()
	const serverVersion = "v1.0.0"

	testCases := []struct {
		name    string
		config  prompts.PromptsetConfig
		wantErr string
	}{
		{
			name:   "success",
			config: prompts.PromptsetConfig{Name: "default", PromptNames: []string{"prompt1", "prompt2"}},
		},
		{
			name:   "empty prompt list",
			config: prompts.PromptsetConfig{Name: "empty", PromptNames: []string{}},
		},
		{
			name:    "invalid promptset name",
			config:  prompts.PromptsetConfig{Name: "invalid name", PromptNames: []string{"prompt1"}},
			wantErr: "invalid promptset name",
		},
		{
			name:    "prompt does not exist",
			config:  prompts.PromptsetConfig{Name: "missing", PromptNames: []string{"prompt1", "nope"}},
			wantErr: "prompt does not exist",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.config.Initialize(serverVersion, promptsMap)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("Initialize() expected error containing %q, got nil", tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("Initialize() error = %q, want to contain %q", err.Error(), tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Initialize() unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.config, got.ToConfig()); diff != "" {
				t.Errorf("ToConfig() mismatch (-want +got):\n%s", diff)
			}
			for _, name := range tc.config.PromptNames {
				if !got.ContainsPrompt(name) {
					t.Errorf("ContainsPrompt(%q) = false, want true", name)
				}
			}
			if got.ContainsPrompt("absent") {
				t.Errorf("ContainsPrompt(%q) = true, want false", "absent")
			}
		})
	}
}

func TestPromptsetContainsPromptNilSet(t *testing.T) {
	// A Promptset built directly (not via Initialize) has a nil PromptNameSet
	// and must report no membership rather than panic.
	ps := prompts.Promptset{
		PromptsetConfig: prompts.PromptsetConfig{Name: "manual", PromptNames: []string{"prompt1"}},
	}
	if ps.ContainsPrompt("prompt1") {
		t.Errorf("ContainsPrompt on nil PromptNameSet = true, want false")
	}
}

func TestCustomPromptConfigRoundTrip(t *testing.T) {
	cfg := promptcustom.Config{
		Name:        "greet",
		Description: "greets the user",
		Messages: []prompts.Message{
			{Role: "user", Content: "hello {{.arg1}}"},
		},
		Arguments: prompts.Arguments{
			{Parameter: parameters.NewStringParameter("arg1", "the name to greet")},
		},
	}

	prompt, err := cfg.Initialize()
	if err != nil {
		t.Fatalf("Initialize() unexpected error: %v", err)
	}

	got, ok := prompt.ToConfig().(promptcustom.Config)
	if !ok {
		t.Fatalf("ToConfig() returned %T, want custom.Config", prompt.ToConfig())
	}
	if diff := cmp.Diff(cfg, got); diff != "" {
		t.Errorf("ToConfig() round-trip mismatch (-want +got):\n%s", diff)
	}
}

func TestRegisterDuplicate(t *testing.T) {
	// "custom" is already registered via the blank import in the integration
	// test; re-registering must fail rather than overwrite.
	if prompts.Register("custom", nil) {
		t.Errorf("Register(\"custom\") = true, want false for an already-registered type")
	}
}
