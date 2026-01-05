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

package gemini_test

import (
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/embeddingmodels"
	"github.com/googleapis/genai-toolbox/internal/embeddingmodels/gemini"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
)

func TestParseFromYamlGemini(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		want server.EmbeddingModelConfigs
	}{
		{
			desc: "basic example",
			in: `
            embeddingModels:
                my-gemini-model:
                    kind: gemini
                    model: text-embedding-004
            `,
			want: map[string]embeddingmodels.EmbeddingModelConfig{
				"my-gemini-model": gemini.Config{
					Name:  "my-gemini-model",
					Kind:  gemini.EmbeddingModelKind,
					Model: "text-embedding-004",
				},
			},
		},
		{
			desc: "full example with optional fields",
			in: `
            embeddingModels:
                complex-gemini:
                    kind: gemini
                    model: text-embedding-004
                    apiKey: "test-api-key"
                    dimension: 768
            `,
			want: map[string]embeddingmodels.EmbeddingModelConfig{
				"complex-gemini": gemini.Config{
					Name:      "complex-gemini",
					Kind:      gemini.EmbeddingModelKind,
					Model:     "text-embedding-004",
					ApiKey:    "test-api-key",
					Dimension: 768,
				},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := struct {
				Models server.EmbeddingModelConfigs `yaml:"embeddingModels"`
			}{}
			// Parse contents
			err := yaml.Unmarshal(testutils.FormatYaml(tc.in), &got)
			if err != nil {
				t.Fatalf("unable to unmarshal: %s", err)
			}
			if !cmp.Equal(tc.want, got.Models) {
				t.Fatalf("incorrect parse: %v", cmp.Diff(tc.want, got.Models))
			}
		})
	}
}
func TestFailParseFromYamlGemini(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		err  string
	}{
		{
			desc: "missing required model field",
			in: `
            embeddingModels:
                bad-model:
                    kind: gemini
            `,
			// Removed the specific model name from the prefix to match your output
			err: "unable to parse as \"gemini\": Key: 'Config.Model' Error:Field validation for 'Model' failed on the 'required' tag",
		},
		{
			desc: "unknown field",
			in: `
            embeddingModels:
                bad-field:
                    kind: gemini
                    model: text-embedding-004
                    invalid_param: true
            `,
			// Updated to match the specific line-starting format of your error output
			err: "unable to parse as \"gemini\": [1:1] unknown field \"invalid_param\"\n>  1 | invalid_param: true\n       ^\n   2 | kind: gemini\n   3 | model: text-embedding-004",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got := struct {
				Models server.EmbeddingModelConfigs `yaml:"embeddingModels"`
			}{}
			err := yaml.Unmarshal(testutils.FormatYaml(tc.in), &got)
			if err == nil {
				t.Fatalf("expect parsing to fail")
			}
			if err.Error() != tc.err {
				t.Fatalf("unexpected error:\ngot:  %q\nwant: %q", err.Error(), tc.err)
			}
		})
	}
}
