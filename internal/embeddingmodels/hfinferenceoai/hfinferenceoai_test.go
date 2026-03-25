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

package hfinferenceoai_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/embeddingmodels"
	"github.com/googleapis/genai-toolbox/internal/embeddingmodels/hfinferenceoai"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
)

func TestParseFromYAMLHFInferenceOAI(t *testing.T) {
	tcs := []struct {
		desc string
		in   string
		want server.EmbeddingModelConfigs
	}{
		{
			desc: "basic example",
			in: `
			kind: embeddingModels
			name: my-hf-model
			type: hf-inference-oai
			baseUrl: https://example.com
			model: Qwen/Qwen3-Embedding-0.6B
            `,
			want: map[string]embeddingmodels.EmbeddingModelConfig{
				"my-hf-model": hfinferenceoai.Config{
					Name:    "my-hf-model",
					Type:    hfinferenceoai.EmbeddingModelType,
					BaseURL: "https://example.com",
					Model:   "Qwen/Qwen3-Embedding-0.6B",
				},
			},
		},
		{
			desc: "full example with optional api key",
			in: `
			kind: embeddingModels
			name: secured-hf-model
			type: hf-inference-oai
			baseUrl: https://example.com
			model: Qwen/Qwen3-Embedding-0.6B
			apiKey: test-token
            `,
			want: map[string]embeddingmodels.EmbeddingModelConfig{
				"secured-hf-model": hfinferenceoai.Config{
					Name:    "secured-hf-model",
					Type:    hfinferenceoai.EmbeddingModelType,
					BaseURL: "https://example.com",
					Model:   "Qwen/Qwen3-Embedding-0.6B",
					APIKey:  "test-token",
				},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			_, _, got, _, _, _, err := server.UnmarshalResourceConfig(context.Background(), testutils.FormatYaml(tc.in))
			if err != nil {
				t.Fatalf("unable to unmarshal: %s", err)
			}
			if !cmp.Equal(tc.want, got) {
				t.Fatalf("incorrect parse: %v", cmp.Diff(tc.want, got))
			}
		})
	}
}

func TestEmbedParameters(t *testing.T) {
	var gotAuth string
	var gotReq struct {
		Model string   `json:"model"`
		Input []string `json:"input"`
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/v1/embeddings" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotReq); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		first := 1
		second := 0
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"index": first, "embedding": []float32{3, 4}},
				{"index": second, "embedding": []float32{1, 2}},
			},
		})
	}))
	defer srv.Close()

	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("failed to create logger context: %v", err)
	}
	ctx = testutils.ContextWithUserAgent(ctx, "toolbox-test")

	model, err := (hfinferenceoai.Config{
		Name:    "hf",
		Type:    hfinferenceoai.EmbeddingModelType,
		BaseURL: srv.URL,
		Model:   "Qwen/Qwen3-Embedding-0.6B",
		APIKey:  "secret",
	}).Initialize(ctx)
	if err != nil {
		t.Fatalf("failed to initialize model: %v", err)
	}

	embeddings, err := model.EmbedParameters(ctx, []string{"hello", "world"})
	if err != nil {
		t.Fatalf("failed to embed parameters: %v", err)
	}

	if gotAuth != "Bearer secret" {
		t.Fatalf("unexpected auth header: %q", gotAuth)
	}
	if diff := cmp.Diff([]string{"hello", "world"}, gotReq.Input); diff != "" {
		t.Fatalf("unexpected inputs (-want +got):\n%s", diff)
	}
	if gotReq.Model != "Qwen/Qwen3-Embedding-0.6B" {
		t.Fatalf("unexpected model: %q", gotReq.Model)
	}
	if diff := cmp.Diff([][]float32{{1, 2}, {3, 4}}, embeddings); diff != "" {
		t.Fatalf("unexpected embeddings (-want +got):\n%s", diff)
	}
}
