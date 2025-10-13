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
	"github.com/googleapis/genai-toolbox/internal/prompts"
	_ "github.com/googleapis/genai-toolbox/internal/prompts/custom"
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

	t.Run("DecodeDefaultsToCustom", func(t *testing.T) {
		decoder := yaml.NewDecoder(strings.NewReader("description: A test prompt"))
		config, err := prompts.DecodeConfig(ctx, "", "testDefaultPrompt", decoder)
		if err != nil {
			t.Fatalf("expected DecodeConfig with empty kind to succeed, but got error: %v", err)
		}
		if config == nil {
			t.Fatal("expected a non-nil config for default kind")
		}
		if config.PromptConfigKind() != "custom" {
			t.Errorf("expected default kind to be 'custom', but got %q", config.PromptConfigKind())
		}
	})
}
