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

package hfinferenceoai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/googleapis/genai-toolbox/internal/embeddingmodels"
	"github.com/googleapis/genai-toolbox/internal/util"
)

const EmbeddingModelType = "hf-inference-oai"

var _ embeddingmodels.EmbeddingModelConfig = Config{}

type Config struct {
	Name    string `yaml:"name" validate:"required"`
	Type    string `yaml:"type" validate:"required"`
	BaseURL string `yaml:"baseUrl" validate:"required,url"`
	Model   string `yaml:"model" validate:"required"`
	APIKey  string `yaml:"apiKey"`
}

func (cfg Config) EmbeddingModelConfigType() string {
	return EmbeddingModelType
}

func (cfg Config) Initialize(ctx context.Context) (embeddingmodels.EmbeddingModel, error) {
	ua, err := util.UserAgentFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get user agent from context: %w", err)
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

	body, err := json.Marshal(embedRequest{
		Model: m.Model,
		Input: parameters,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal embedding request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(m.BaseURL, "/")+"/v1/embeddings", bytes.NewReader(body))
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

	var out embedResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("failed to decode embedding response: %w", err)
	}
	if len(out.Data) != len(parameters) {
		return nil, fmt.Errorf("embedding endpoint returned %d embeddings for %d inputs", len(out.Data), len(parameters))
	}

	embeddings := make([][]float32, len(out.Data))
	seen := make([]bool, len(out.Data))
	for responsePos, item := range out.Data {
		targetPos := responsePos
		if item.Index != nil {
			targetPos = *item.Index
		} else if len(out.Data) > 1 {
			return nil, fmt.Errorf("embedding endpoint omitted index for batched response item %d", responsePos)
		}
		if targetPos < 0 || targetPos >= len(parameters) {
			return nil, fmt.Errorf("embedding endpoint returned out-of-range index %d for %d inputs", targetPos, len(parameters))
		}
		if seen[targetPos] {
			return nil, fmt.Errorf("embedding endpoint returned duplicate index %d", targetPos)
		}
		embeddings[targetPos] = item.Embedding
		seen[targetPos] = true
	}

	logger.InfoContext(ctx, fmt.Sprintf("Successfully embedded %d text parameters using model %s", len(parameters), m.Model))
	return embeddings, nil
}

type embedRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type embedResponse struct {
	Data []embedResponseItem `json:"data"`
}

type embedResponseItem struct {
	Index     *int      `json:"index"`
	Embedding []float32 `json:"embedding"`
}
