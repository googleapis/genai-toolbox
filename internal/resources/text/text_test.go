// Copyright 2024 Google LLC
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

package text

import (
	"bytes"
	"context"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/mcp-toolbox/internal/resources"
)

func floatPtr(f float64) *float64 { return &f }

func TestTextResourceInitialization(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		config      Config
		wantError   bool
		errContains string
		wantMime    string
		wantPrior   float64
	}{
		{
			name: "success with defaults",
			config: Config{
				BaseConfig: resources.BaseConfig{Name: "test1"},
				Text:       "Hello, world!",
			},
			wantError: false,
			wantMime:  "text/plain",
			wantPrior: 1.0,
		},
		{
			name: "success with overrides",
			config: Config{
				BaseConfig: resources.BaseConfig{
					Name:        "test2",
					MimeType:    "application/json",
					Annotations: &resources.ResourceAnnotations{Priority: floatPtr(0.5)},
				},
				Text: `{"hello":"world"}`,
			},
			wantError: false,
			wantMime:  "application/json",
			wantPrior: 0.5,
		},
		{
			name: "error missing text payload",
			config: Config{
				BaseConfig: resources.BaseConfig{Name: "test3"},
				Text:       "",
			},
			wantError:   true,
			errContains: "missing required 'text' field",
		},
		{
			name: "explicit 0.0 priority",
			config: Config{
				BaseConfig: resources.BaseConfig{
					Name:        "test-priority",
					Annotations: &resources.ResourceAnnotations{Priority: floatPtr(0.0)},
				},
				Text: "priority test",
			},
			wantError: false,
			wantMime:  "text/plain",
			wantPrior: 0.0,
		},
		{
			name: "multi-byte unicode size calculation",
			config: Config{
				BaseConfig: resources.BaseConfig{Name: "test-unicode"},
				Text:       "Hello 🌍",
			},
			wantError: false,
			wantMime:  "text/plain",
			wantPrior: 1.0,
		},
		{
			name: "pure whitespace payload",
			config: Config{
				BaseConfig: resources.BaseConfig{Name: "test-whitespace"},
				Text:       "   \n  ",
			},
			wantError: false,
			wantMime:  "text/plain",
			wantPrior: 1.0,
		},
		{
			name: "explicit empty mimetype defaults to text/plain",
			config: Config{
				BaseConfig: resources.BaseConfig{
					Name:     "test-empty-mime",
					MimeType: "",
				},
				Text: "hello",
			},
			wantError: false,
			wantMime:  "text/plain",
			wantPrior: 1.0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res, err := tc.config.Initialize(ctx)
			if tc.wantError {
				if err == nil {
					t.Fatalf("Initialize() expected error, got nil")
				}
				if tc.errContains != "" {
					expectedErr := "missing required 'text' field for text resource \"test3\""
					if err.Error() != expectedErr {
						t.Errorf("Initialize() err = %v, want %v", err, expectedErr)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("Initialize() unexpected error: %v", err)
			}

			// Verify execution (Read)
			data, err := res.Read(ctx, nil)
			if err != nil {
				t.Fatalf("Read() unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.config.Text, data); diff != "" {
				t.Errorf("Read() mismatch (-want +got):\n%s", diff)
			}

			// Verify mutated config defaults and Size calculation
			cfg := res.ToConfig().(*Config)
			if cfg.MimeType != tc.wantMime {
				t.Errorf("MimeType = %v, want %v", cfg.MimeType, tc.wantMime)
			}

			if cfg.Annotations.Priority == nil || *cfg.Annotations.Priority != tc.wantPrior {
				t.Errorf("Annotations.Priority = %v, want %v", cfg.Annotations.Priority, tc.wantPrior)
			}

			if cfg.Size == nil {
				t.Fatalf("Size is nil, expected dynamic calculation")
			}
			expectedSize := int64(len([]byte(tc.config.Text)))
			if *cfg.Size != expectedSize {
				t.Errorf("Size = %d, want %d", *cfg.Size, expectedSize)
			}
		})
	}
}

func TestTextResourceYAMLUnmarshaling(t *testing.T) {
	ctx := context.Background()

	t.Run("Valid YAML", func(t *testing.T) {
		yamlData := []byte(`
name: test-yaml
type: text
uri: info://test
annotations:
  priority: 0.9
  audience:
    - user
text: |
  Line 1
  Line 2
`)

		dec := yaml.NewDecoder(bytes.NewReader(yamlData), yaml.Strict())

		resCfg, err := newConfig(ctx, "test-yaml", dec)
		if err != nil {
			t.Fatalf("unexpected error decoding text resource: %v", err)
		}

		res, err := resCfg.Initialize(ctx)
		if err != nil {
			t.Fatalf("unexpected error initializing text resource: %v", err)
		}

		cfg := res.ToConfig().(*Config)
		if cfg.Text != "Line 1\nLine 2\n" {
			t.Errorf("unexpected text payload: %q", cfg.Text)
		}
		
		if cfg.Annotations == nil || cfg.Annotations.Priority == nil || *cfg.Annotations.Priority != 0.9 {
			t.Errorf("unexpected priority: %v", cfg.Annotations)
		}

		if cfg.Size == nil || *cfg.Size != 14 { // "Line 1\nLine 2\n" is 14 bytes
			t.Errorf("unexpected size, got: %v", cfg.Size)
		}
	})

	t.Run("Strict Decoder Validation", func(t *testing.T) {
		yamlData := []byte(`
name: test-invalid
type: text
textContent: "hello" # invalid field
`)

		dec := yaml.NewDecoder(bytes.NewReader(yamlData), yaml.Strict())
		_, err := newConfig(ctx, "test-invalid", dec)
		if err == nil {
			t.Fatalf("expected strict decoder error for unknown field 'textContent', got nil")
		}
	})
}
