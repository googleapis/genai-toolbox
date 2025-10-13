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
	"context"
	"errors"
	"strings"
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/prompts"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

type mockPromptConfig struct {
	name string
	kind string
}

func (m *mockPromptConfig) PromptConfigKind() string            { return m.kind }
func (m *mockPromptConfig) Initialize() (prompts.Prompt, error) { return nil, nil }

var errMockFactory = errors.New("mock factory error")

func mockFactory(ctx context.Context, name string, decoder *yaml.Decoder) (prompts.PromptConfig, error) {
	return &mockPromptConfig{name: name, kind: "mockKind"}, nil
}

func mockErrorFactory(ctx context.Context, name string, decoder *yaml.Decoder) (prompts.PromptConfig, error) {
	return nil, errMockFactory
}

func TestRegistry(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Test case 1: Successful registration and decoding
	t.Run("RegisterAndDecodeSuccess", func(t *testing.T) {
		kind := "testKindSuccess"
		if !prompts.Register(kind, mockFactory) {
			t.Fatal("expected registration to succeed")
		}
		// This should fail because we are registering a duplicate
		if prompts.Register(kind, mockFactory) {
			t.Fatal("expected duplicate registration to fail")
		}

		decoder := yaml.NewDecoder(strings.NewReader(""))
		config, err := prompts.DecodeConfig(ctx, kind, "testPrompt", decoder)
		if err != nil {
			t.Fatalf("expected DecodeConfig to succeed, but got error: %v", err)
		}
		if config == nil {
			t.Fatal("expected a non-nil config")
		}
	})

	// Test case 2: Decoding an unknown kind
	t.Run("DecodeUnknownKind", func(t *testing.T) {
		decoder := yaml.NewDecoder(strings.NewReader(""))
		_, err := prompts.DecodeConfig(ctx, "unregisteredKind", "testPrompt", decoder)
		if err == nil {
			t.Fatal("expected an error for unknown kind, but got nil")
		}
		if !strings.Contains(err.Error(), "unknown prompt kind") {
			t.Errorf("expected error to contain 'unknown prompt kind', but got: %v", err)
		}
	})

	// Test case 3: Factory returns an error
	t.Run("FactoryReturnsError", func(t *testing.T) {
		kind := "testKindError"
		if !prompts.Register(kind, mockErrorFactory) {
			t.Fatal("expected registration to succeed")
		}

		decoder := yaml.NewDecoder(strings.NewReader(""))
		_, err := prompts.DecodeConfig(ctx, kind, "testPrompt", decoder)
		if err == nil {
			t.Fatal("expected an error from the factory, but got nil")
		}
		if !errors.Is(err, errMockFactory) {
			t.Errorf("expected error to wrap mock factory error, but it didn't")
		}
	})
}

func TestGetMcpManifest(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name        string
		promptName  string
		description string
		args        prompts.Arguments
		want        prompts.McpManifest
	}{
		{
			name:        "No arguments or metadata",
			promptName:  "test-prompt",
			description: "A test prompt.",
			args:        prompts.Arguments{},
			want: prompts.McpManifest{
				Name:        "test-prompt",
				Description: "A test prompt.",
				Arguments:   []prompts.McpArgManifest{},
				Metadata:    nil,
			},
		},
		{
			name:        "With arguments",
			promptName:  "arg-prompt",
			description: "Prompt with args.",
			args: prompts.Arguments{
				prompts.Argument{Parameter: tools.NewStringParameter("param1", "First param")},
				prompts.Argument{Parameter: tools.NewIntParameterWithRequired("param2", "Second param", false)},
			},
			want: prompts.McpManifest{
				Name:        "arg-prompt",
				Description: "Prompt with args.",
				Arguments: []prompts.McpArgManifest{
					{Name: "param1", Description: "First param", Required: true},
					{Name: "param2", Description: "Second param", Required: false},
				},
				Metadata: nil,
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := prompts.GetMcpManifest(tc.promptName, tc.description, tc.args)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("GetMcpManifest() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestConfig_Methods(t *testing.T) {
	t.Parallel()

	// Setup a shared config for testing its methods
	testArgs := prompts.Arguments{
		prompts.Argument{Parameter: tools.NewStringParameter("name", "The name to use.")},
		prompts.Argument{Parameter: tools.NewStringParameterWithRequired("location", "The location.", false)},
	}

	cfg := prompts.Config{
		Name:        "TestConfig",
		Kind:        "test",
		Description: "A test config.",
		Messages: []prompts.Message{
			{Role: "user", Content: "Hello, my name is {{.name}} and I am in {{.location}}."},
		},
		Arguments: testArgs,
	}

	t.Run("Initialize and Kind", func(t *testing.T) {
		p, err := cfg.Initialize()
		if err != nil {
			t.Fatalf("Initialize() failed: %v", err)
		}
		if p == nil {
			t.Fatal("Initialize() returned a nil prompt")
		}
		if cfg.PromptConfigKind() != "test" {
			t.Errorf("PromptConfigKind() = %q, want %q", cfg.PromptConfigKind(), "test")
		}
	})

	t.Run("Manifest", func(t *testing.T) {
		want := prompts.Manifest{
			Description: "A test config.",
			Arguments: []tools.ParameterManifest{
				{Name: "name", Type: "string", Required: true, Description: "The name to use.", AuthServices: []string{}},
				{Name: "location", Type: "string", Required: false, Description: "The location.", AuthServices: []string{}},
			},
		}
		got := cfg.Manifest()
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("Manifest() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("McpManifest", func(t *testing.T) {
		want := prompts.McpManifest{
			Name:        "TestConfig",
			Description: "A test config.",
			Arguments: []prompts.McpArgManifest{
				{Name: "name", Description: "The name to use.", Required: true},
				{Name: "location", Description: "The location.", Required: false},
			},
		}
		got := cfg.McpManifest()
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("McpManifest() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("SubstituteParams", func(t *testing.T) {
		argValues := tools.ParamValues{
			{Name: "name", Value: "Alice"},
			{Name: "location", Value: "Wonderland"},
		}
		want := []prompts.Message{
			{Role: "user", Content: "Hello, my name is Alice and I am in Wonderland."},
		}

		got, err := cfg.SubstituteParams(argValues)
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
			want := tools.ParamValues{
				{Name: "name", Value: "Bob"},
				{Name: "location", Value: "the Builder"},
			}
			got, err := cfg.ParseArgs(argsIn, nil)
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
			_, err := cfg.ParseArgs(argsIn, nil)
			if err == nil {
				t.Fatal("expected an error for missing required arg, but got nil")
			}
			if !strings.Contains(err.Error(), `parameter "name" is required`) {
				t.Errorf("expected error to be about missing parameter, but got: %v", err)
			}
		})
	})
}
