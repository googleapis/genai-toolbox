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

package text

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/mcp-toolbox/internal/resources"
)

func floatPtr(f float64) *float64 { return &f }

func TestTextResourceInitialization(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		yamlStr     string
		wantError   bool
		errContains string
		wantMime    string
		wantPrior   *float64
		wantText    string
	}{
		{
			name: "success with defaults (no URI specified)",
			yamlStr: `
name: test1
type: text
text: "Hello, world!"
`,
			wantError: false,
			wantMime:  "text/plain",
			wantPrior: nil,
			wantText:  "Hello, world!",
		},
		{
			name: "success with overrides",
			yamlStr: `
name: test2
type: text
mimeType: application/json
annotations:
  priority: 0.5
text: '{"hello":"world"}'
`,
			wantError: false,
			wantMime:  "application/json",
			wantPrior: floatPtr(0.5),
			wantText:  `{"hello":"world"}`,
		},
		{
			name: "error empty text payload explicitly defined",
			yamlStr: `
name: test4
type: text
text: ""
`,
			wantError:   true,
			errContains: "Field validation for 'Text' failed",
		},
		{
			name: "error missing text payload",
			yamlStr: `
name: test3
type: text
`,
			wantError:   true,
			errContains: "Field validation for 'Text' failed",
		},
		{
			name: "explicit 0.0 priority",
			yamlStr: `
name: test-priority
type: text
annotations:
  priority: 0.0
text: "priority test"
`,
			wantError: false,
			wantMime:  "text/plain",
			wantPrior: floatPtr(0.0),
			wantText:  "priority test",
		},
		{
			name: "multi-byte unicode size calculation",
			yamlStr: `
name: test-unicode
type: text
text: "Hello 🌍"
`,
			wantError: false,
			wantMime:  "text/plain",
			wantPrior: nil,
			wantText:  "Hello 🌍",
		},
		{
			name: "pure whitespace payload",
			yamlStr: `
name: test-whitespace
type: text
text: "   \n  "
`,
			wantError: false,
			wantMime:  "text/plain",
			wantPrior: nil,
			wantText:  "   \n  ",
		},
		{
			name: "explicit empty mimetype defaults to text/plain",
			yamlStr: `
name: test-empty-mime
type: text
mimeType: ""
text: "hello"
`,
			wantError: false,
			wantMime:  "text/plain",
			wantPrior: nil,
			wantText:  "hello",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dec := yaml.NewDecoder(bytes.NewReader([]byte(tc.yamlStr)), yaml.Strict(), yaml.Validator(validator.New()))
			
			// parse resourceName
			var pre map[string]any
			if err := yaml.Unmarshal([]byte(tc.yamlStr), &pre); err != nil {
			    t.Fatalf("failed to pre-parse yaml: %v", err)
			}
			resName := pre["name"].(string)
			
			config, err := newConfig(ctx, resName, dec)
			var res resources.Resource
			if err == nil {
			    err = config.Validate()
			}
			if err == nil {
				res, err = config.Initialize(ctx)
			}
			
			if tc.wantError {
				if err == nil {
					t.Fatalf("Expected error, got nil")
				}
				if tc.errContains != "" {
					if !strings.Contains(err.Error(), tc.errContains) {
						t.Errorf("err = %v, want to contain %q", err, tc.errContains)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("Initialize() unexpected error: %v", err)
			}

			// Verify URI default
			if config.GetURI() != "text://"+resName {
			    t.Errorf("expected URI to default to %q, got %q", "text://"+resName, config.GetURI())
			}

			// Verify execution (Read)
			data, err := res.Read(ctx, nil)
			if err != nil {
				t.Fatalf("Read() unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.wantText, data); diff != "" {
				t.Errorf("Read() mismatch (-want +got):\n%s", diff)
			}

			textRes := res.(*Resource)
			expectedSize := int64(len(tc.wantText))
			if textRes.Size != expectedSize {
				t.Errorf("Size = %d, want %d", textRes.Size, expectedSize)
			}
		})
	}
}
func TestTextResourceYAMLUnmarshaling(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name         string
		yamlData     string
		wantText     string
		wantPriority *float64
		wantSize     *int64
	}{
		{
			name: "Valid YAML",
			yamlData: `
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
`,
			wantText:     "Line 1\nLine 2\n",
			wantPriority: floatPtr(0.9),
			wantSize:     func(i int64) *int64 { return &i }(14),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dec := yaml.NewDecoder(bytes.NewReader([]byte(tc.yamlData)), yaml.Strict(), yaml.Validator(validator.New()))
			resCfg, err := newConfig(ctx, "test-yaml", dec)
			if err != nil {
				t.Fatalf("unexpected error decoding text resource: %v", err)
			}

			cfg := resCfg.(*Config)
			if cfg.Text != tc.wantText {
				t.Errorf("unexpected text payload: %q", cfg.Text)
			}

			if tc.wantPriority != nil {
				if cfg.Annotations == nil || cfg.Annotations.Priority == nil || *cfg.Annotations.Priority != *tc.wantPriority {
					t.Errorf("unexpected priority: %v", cfg.Annotations)
				}
			}

			// We need to initialize it to get the size calculated
			res, err := cfg.Initialize(ctx)
			if err != nil {
				t.Fatalf("unexpected error initializing text resource: %v", err)
			}

			if tc.wantSize != nil {
				if res.(*Resource).Size != *tc.wantSize {
					t.Errorf("unexpected size: got %v, want %v", res.(*Resource).Size, tc.wantSize)
				}
			}
		})
	}
}

func TestTextResourceYAMLUnmarshaling_Fail(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		yamlData    string
		errContains string
	}{
		{
			name: "Strict Decoder Validation",
			yamlData: `
name: test-invalid
type: text
textContent: "hello" # invalid field
`,
			errContains: "unknown field",
		},
		{
			name: "Missing required text field",
			yamlData: `
name: test-missing-text
type: text
`,
			errContains: "Field validation for 'Text' failed",
		},
		{
			name: "Empty text field",
			yamlData: `
name: test-empty-text
type: text
text: ""
`,
			errContains: "Field validation for 'Text' failed",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dec := yaml.NewDecoder(bytes.NewReader([]byte(tc.yamlData)), yaml.Strict(), yaml.Validator(validator.New()))
			resCfg, err := newConfig(ctx, "test-invalid", dec)
			if err == nil {
				err = resCfg.Validate()
			}

			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if tc.errContains != "" && !strings.Contains(err.Error(), tc.errContains) {
				t.Errorf("expected error to contain %q, got: %v", tc.errContains, err)
			}
		})
	}
}
