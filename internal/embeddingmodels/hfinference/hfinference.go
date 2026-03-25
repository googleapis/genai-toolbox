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

package hfinference

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/googleapis/genai-toolbox/internal/embeddingmodels"
	"github.com/googleapis/genai-toolbox/internal/util"
)

const (
	EmbeddingModelType = "hf-inference"
	defaultBaseURL     = "https://router.huggingface.co/hf-inference"
	defaultTask        = "feature-extraction"
)

var _ embeddingmodels.EmbeddingModelConfig = Config{}

type Config struct {
	Name    string `yaml:"name" validate:"required"`
	Type    string `yaml:"type" validate:"required"`
	Model   string `yaml:"model" validate:"required"`
	APIKey  string `yaml:"apiKey"`
	BaseURL string `yaml:"baseUrl" validate:"omitempty,url"`
	Task    string `yaml:"task" validate:"omitempty,oneof=feature-extraction"`
}

func (cfg Config) EmbeddingModelConfigType() string {
	return EmbeddingModelType
}

func (cfg Config) Initialize(ctx context.Context) (embeddingmodels.EmbeddingModel, error) {
	ua, err := util.UserAgentFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get user agent from context: %w", err)
	}

	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}
	if cfg.Task == "" {
		cfg.Task = defaultTask
	}

	return &EmbeddingModel{
		Config: cfg,
		Client: &http.Client{
			Timeout:   time.Minute,
			Transport: util.NewUserAgentRoundTripper(ua, http.DefaultTransport),
		},
	}, nil
}

var _ embeddingmodels.EmbeddingModel = (*EmbeddingModel)(nil)

type EmbeddingModel struct {
	Client *http.Client
	Config
}

func (m EmbeddingModel) EmbeddingModelType() string {
	return EmbeddingModelType
}

func (m EmbeddingModel) ToConfig() embeddingmodels.EmbeddingModelConfig {
	return m.Config
}

func (m EmbeddingModel) EmbedParameters(ctx context.Context, parameters []string) ([][]float32, error) {
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get logger from ctx: %w", err)
	}

	body, err := json.Marshal(map[string]any{"inputs": parameters})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal embedding request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.endpointURL(), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if m.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+m.APIKey)
	}

	resp, err := m.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call embedding endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return nil, fmt.Errorf("embedding endpoint returned %s: %s", resp.Status, strings.TrimSpace(string(msg)))
	}

	var raw any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("failed to decode embedding response: %w", err)
	}

	embeddings, err := parseEmbeddings(raw, len(parameters))
	if err != nil {
		return nil, err
	}

logger.InfoContext(ctx, "Successfully embedded %d text parameters using model %s", len(parameters), m.Model)
	return embeddings, nil
}

func (m EmbeddingModel) endpointURL() string {
	return strings.TrimRight(m.BaseURL, "/") + "/models/" + escapePath(m.Model) + "/pipeline/" + m.Task
}

func escapePath(s string) string {
	parts := strings.Split(s, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}

func parseEmbeddings(raw any, want int) ([][]float32, error) {
	items, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("unexpected embedding response type: %T", raw)
	}
	if len(items) == 0 {
		return [][]float32{}, nil
	}

	switch items[0].(type) {
	case float64:
		if want != 1 {
			return nil, fmt.Errorf("embedding endpoint returned 1 embedding for %d inputs", want)
		}
		vector, err := parseVector(items)
		if err != nil {
			return nil, err
		}
		return [][]float32{vector}, nil
	case []any:
		if len(items) != want {
			if want != 1 {
				return nil, fmt.Errorf("embedding endpoint returned %d embeddings for %d inputs", len(items), want)
			}

			vector, err := parseEmbeddingItem(items)
			if err != nil {
				return nil, err
			}
			return [][]float32{vector}, nil
		}

		embeddings := make([][]float32, 0, len(items))
		for _, item := range items {
			vectorItems, ok := item.([]any)
			if !ok {
				return nil, fmt.Errorf("unexpected embedding vector type: %T", item)
			}
			vector, err := parseEmbeddingItem(vectorItems)
			if err != nil {
				return nil, err
			}
			embeddings = append(embeddings, vector)
		}
		return embeddings, nil
	default:
		return nil, fmt.Errorf("unexpected embedding response item type: %T", items[0])
	}
}

func parseEmbeddingItem(items []any) ([]float32, error) {
	if len(items) == 0 {
		return []float32{}, nil
	}
	// transformers 5.3.0.dev0 FeatureExtractionPipeline returns raw model_outputs[0].tolist(),
	// so token-level outputs commonly arrive as [1][seq][dim] and list inputs may add another singleton wrapper.
	if len(items) == 1 {
		if nested, ok := items[0].([]any); ok {
			return parseEmbeddingItem(nested)
		}
	}

	switch items[0].(type) {
	case float64:
		return parseVector(items)
	case []any:
		return meanPoolVectors(items)
	default:
		return nil, fmt.Errorf("unexpected embedding item type: %T", items[0])
	}
}

func parseVector(items []any) ([]float32, error) {
	vector := make([]float32, 0, len(items))
	for _, item := range items {
		f, ok := item.(float64)
		if !ok {
			return nil, fmt.Errorf("unexpected embedding scalar type: %T", item)
		}
		vector = append(vector, float32(f))
	}
	return vector, nil
}

func meanPoolVectors(items []any) ([]float32, error) {
	if len(items) == 0 {
		return []float32{}, nil
	}

	var sums []float32
	for i, item := range items {
		vectorItems, ok := item.([]any)
		if !ok {
			return nil, fmt.Errorf("unexpected token embedding type: %T", item)
		}

		vector, err := parseVector(vectorItems)
		if err != nil {
			return nil, err
		}

		if i == 0 {
			sums = make([]float32, len(vector))
		} else if len(vector) != len(sums) {
			return nil, fmt.Errorf("inconsistent token embedding length: got %d, want %d", len(vector), len(sums))
		}

		for j, value := range vector {
			sums[j] += value
		}
	}

	scale := float32(len(items))
	for i := range sums {
		sums[i] /= scale
	}
	return sums, nil
}
