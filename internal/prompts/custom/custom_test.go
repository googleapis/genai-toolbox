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

package custom_test

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/prompts"
	"github.com/googleapis/genai-toolbox/internal/prompts/custom"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
)

func TestConfig(t *testing.T) {
	t.Parallel()

	// Setup a shared config for testing its methods
	testArgs := prompts.Arguments{
		{Parameter: parameters.NewStringParameter("name", "The name to use.")},
		{Parameter: parameters.NewStringParameterWithRequired("location", "The location.", false)},
	}

	cfg := custom.Config{
		Name:        "TestConfig",
		Description: "A test config.",
		Messages: []custom.Message{
			{Role: "user", Content: "Hello, my name is {{.name}} and I am in {{.location}}."},
		},
		Arguments: testArgs,
	}

	// initialize and check kind
	p, err := cfg.Initialize()
	if err != nil {
		t.Fatalf("Initialize() failed: %v", err)
	}
	if p == nil {
		t.Fatal("Initialize() returned a nil prompt")
	}
	if cfg.PromptConfigKind() != "custom" {
		t.Errorf("PromptConfigKind() = %q, want %q", cfg.PromptConfigKind(), "custom")
	}

	t.Run("Manifest", func(t *testing.T) {
		want := prompts.Manifest{
			Description: "A test config.",
			Arguments: []parameters.ParameterManifest{
				{Name: "name", Type: "string", Required: true, Description: "The name to use.", AuthServices: []string{}},
				{Name: "location", Type: "string", Required: false, Description: "The location.", AuthServices: []string{}},
			},
		}
		got := p.Manifest()
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("Manifest() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("McpManifest", func(t *testing.T) {
		want := prompts.McpManifest{
			Name:        "TestConfig",
			Description: "A test config.",
			Arguments: []prompts.ArgMcpManifest{
				{Name: "name", Description: "The name to use.", Required: true},
				{Name: "location", Description: "The location.", Required: false},
			},
		}
		got := p.McpManifest()
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("McpManifest() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("SubstituteParams", func(t *testing.T) {
		argValues := parameters.ParamValues{
			{Name: "name", Value: "Alice"},
			{Name: "location", Value: "Wonderland"},
		}
		want := []prompts.Message{
			{Role: "user", Content: "Hello, my name is Alice and I am in Wonderland."},
		}

		got, err := p.SubstituteParams(argValues)
		if err != nil {
			t.Fatalf("SubstituteParams() failed: %v", err)
		}

		gotMessages, ok := got.([]prompts.Message)
		if !ok {
			t.Fatalf("expected result to be of type []prompts.Message, but got %T", got)
		}

		if diff := cmp.Diff(want, gotMessages); diff != "" {
			t.Errorf("SubstituteParams() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("ParseArgs", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			argsIn := map[string]any{
				"name":     "Bob",
				"location": "the Builder",
			}
			want := parameters.ParamValues{
				{Name: "name", Value: "Bob"},
				{Name: "location", Value: "the Builder"},
			}
			got, err := p.ParseArgs(argsIn, nil)
			if err != nil {
				t.Fatalf("ParseArgs() failed: %v", err)
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("ParseArgs() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("FailureMissingRequired", func(t *testing.T) {
			argsIn := map[string]any{
				"location": "missing name",
			}
			_, err := p.ParseArgs(argsIn, nil)
			if err == nil {
				t.Fatal("expected an error for missing required arg, but got nil")
			}
			if !strings.Contains(err.Error(), `parameter "name" is required`) {
				t.Errorf("expected error to be about missing parameter, but got: %v", err)
			}
		})
	})
}
