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

package hfinference_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/embeddingmodels"
	"github.com/googleapis/genai-toolbox/internal/embeddingmodels/hfinference"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/testutils"
)

func TestParseFromYAMLHFInference(t *testing.T) {
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
			type: hf-inference
			model: sentence-transformers/all-MiniLM-L6-v2
            `,
			want: map[string]embeddingmodels.EmbeddingModelConfig{
				"my-hf-model": hfinference.Config{
					Name:  "my-hf-model",
					Type:  hfinference.EmbeddingModelType,
					Model: "sentence-transformers/all-MiniLM-L6-v2",
				},
			},
		},
		{
			desc: "full example",
			in: `
			kind: embeddingModels
			name: my-hf-model
			type: hf-inference
			model: sentence-transformers/all-MiniLM-L6-v2
			apiKey: test-token
			baseUrl: https://router.huggingface.co/hf-inference
			task: feature-extraction
            `,
			want: map[string]embeddingmodels.EmbeddingModelConfig{
				"my-hf-model": hfinference.Config{
					Name:    "my-hf-model",
					Type:    hfinference.EmbeddingModelType,
					Model:   "sentence-transformers/all-MiniLM-L6-v2",
					APIKey:  "test-token",
					BaseURL: "https://router.huggingface.co/hf-inference",
					Task:    "feature-extraction",
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

func TestParseFromYAMLHFInferenceInvalidBaseURL(t *testing.T) {
	in := `
	kind: embeddingModels
	name: my-hf-model
	type: hf-inference
	model: sentence-transformers/all-MiniLM-L6-v2
	baseUrl: ://bad-url
	`

	_, _, _, _, _, _, err := server.UnmarshalResourceConfig(context.Background(), testutils.FormatYaml(in))
	if err == nil {
		t.Fatal("expected invalid baseUrl error")
	}
	if !strings.Contains(err.Error(), "Key: 'Config.BaseURL' Error:Field validation for 'BaseURL' failed on the 'url' tag") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEmbedParameters(t *testing.T) {
	var gotAuth string
	var gotPath string
	var gotReq struct {
		Inputs []string `json:"inputs"`
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotReq); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([][]float32{{1, 2}, {3, 4}})
	}))
	defer srv.Close()

	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("failed to create logger context: %v", err)
	}
	ctx = testutils.ContextWithUserAgent(ctx, "toolbox-test")

	model, err := (hfinference.Config{
		Name:    "hf",
		Type:    hfinference.EmbeddingModelType,
		Model:   "sentence-transformers/all-MiniLM-L6-v2",
		APIKey:  "secret",
		BaseURL: srv.URL,
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
	if gotPath != "/models/sentence-transformers/all-MiniLM-L6-v2/pipeline/feature-extraction" {
		t.Fatalf("unexpected path: %q", gotPath)
	}
	if diff := cmp.Diff([]string{"hello", "world"}, gotReq.Inputs); diff != "" {
		t.Fatalf("unexpected inputs (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([][]float32{{1, 2}, {3, 4}}, embeddings); diff != "" {
		t.Fatalf("unexpected embeddings (-want +got):\n%s", diff)
	}
}

func TestEmbedParametersPoolsPerTokenEmbeddingsBatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([][][]float32{
			{{1, 3}, {3, 5}},
			{{2, 4}, {4, 6}},
		})
	}))
	defer srv.Close()

	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("failed to create logger context: %v", err)
	}
	ctx = testutils.ContextWithUserAgent(ctx, "toolbox-test")

	model, err := (hfinference.Config{
		Name:    "hf",
		Type:    hfinference.EmbeddingModelType,
		Model:   "google-bert/bert-base-uncased",
		BaseURL: srv.URL,
	}).Initialize(ctx)
	if err != nil {
		t.Fatalf("failed to initialize model: %v", err)
	}

	embeddings, err := model.EmbedParameters(ctx, []string{"hello", "world"})
	if err != nil {
		t.Fatalf("failed to embed parameters: %v", err)
	}

	want := [][]float32{{2, 4}, {3, 5}}
	if diff := cmp.Diff(want, embeddings); diff != "" {
		t.Fatalf("unexpected pooled embeddings (-want +got):\n%s", diff)
	}
}

func TestEmbedParametersPoolsPipelineListStyleOutputs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([][][][]float32{
			{{{1, 3}, {3, 5}}},
			{{{2, 4}, {4, 6}}},
		})
	}))
	defer srv.Close()

	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("failed to create logger context: %v", err)
	}
	ctx = testutils.ContextWithUserAgent(ctx, "toolbox-test")

	model, err := (hfinference.Config{
		Name:    "hf",
		Type:    hfinference.EmbeddingModelType,
		Model:   "google-bert/bert-base-uncased",
		BaseURL: srv.URL,
	}).Initialize(ctx)
	if err != nil {
		t.Fatalf("failed to initialize model: %v", err)
	}

	embeddings, err := model.EmbedParameters(ctx, []string{"hello", "world"})
	if err != nil {
		t.Fatalf("failed to embed parameters: %v", err)
	}

	want := [][]float32{{2, 4}, {3, 5}}
	if diff := cmp.Diff(want, embeddings); diff != "" {
		t.Fatalf("unexpected pooled embeddings (-want +got):\n%s", diff)
	}
}

func TestEmbedParametersPoolsPerTokenEmbeddingsSingleInput(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([][]float32{{1, 2}, {3, 4}, {5, 6}})
	}))
	defer srv.Close()

	ctx, err := testutils.ContextWithNewLogger()
	if err != nil {
		t.Fatalf("failed to create logger context: %v", err)
	}
	ctx = testutils.ContextWithUserAgent(ctx, "toolbox-test")

	model, err := (hfinference.Config{
		Name:    "hf",
		Type:    hfinference.EmbeddingModelType,
		Model:   "google-bert/bert-base-uncased",
		BaseURL: srv.URL,
	}).Initialize(ctx)
	if err != nil {
		t.Fatalf("failed to initialize model: %v", err)
	}

	embeddings, err := model.EmbedParameters(ctx, []string{"hello"})
	if err != nil {
		t.Fatalf("failed to embed parameters: %v", err)
	}

	want := [][]float32{{3, 4}}
	if diff := cmp.Diff(want, embeddings); diff != "" {
		t.Fatalf("unexpected pooled embedding (-want +got):\n%s", diff)
	}
}
