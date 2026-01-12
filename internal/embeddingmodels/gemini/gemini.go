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

package gemini

import (
	"context"
	"fmt"

	"github.com/googleapis/genai-toolbox/internal/embeddingmodels"
	"github.com/googleapis/genai-toolbox/internal/util"
	"google.golang.org/genai"
)

const EmbeddingModelKind string = "gemini"

// validate interface
var _ embeddingmodels.EmbeddingModelConfig = Config{}

type Config struct {
	Name      string `yaml:"name" validate:"required"`
	Kind      string `yaml:"kind" validate:"required"`
	Model     string `yaml:"model" validate:"required"`
	ApiKey    string `yaml:"apiKey"`
	Dimension int32  `yaml:"dimension"`
}

// Returns the embedding model kind
func (cfg Config) EmbeddingModelConfigKind() string {
	return EmbeddingModelKind
}

// Initialize a Gemini embedding model
func (cfg Config) Initialize(ctx context.Context) (embeddingmodels.EmbeddingModel, error) {
	// Get client configs
	configs := &genai.ClientConfig{}
	if cfg.ApiKey != "" {
		configs.APIKey = cfg.ApiKey
	}

	// Create new Gemini API client
	client, err := genai.NewClient(ctx, configs)
	if err != nil {
		return nil, fmt.Errorf("unable to create Gemini API client")
	}

	m := &EmbeddingModel{
		Config: cfg,
		Client: client,
	}
	return m, nil
}

var _ embeddingmodels.EmbeddingModel = EmbeddingModel{}

type EmbeddingModel struct {
	Client *genai.Client
	Config
}

// Returns the embedding model kind
func (m EmbeddingModel) EmbeddingModelKind() string {
	return EmbeddingModelKind
}

func (m EmbeddingModel) ToConfig() embeddingmodels.EmbeddingModelConfig {
	return m.Config
}

func (m EmbeddingModel) EmbedParameters(ctx context.Context, parameters []string) ([][]float32, error) {
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get logger from ctx: %s", err)
	}

	contents := convertStringsToContents(parameters)

	embedConfig := &genai.EmbedContentConfig{
		TaskType: "SEMANTIC_SIMILARITY",
	}

	if m.Dimension > 0 {
		embedConfig.OutputDimensionality = genai.Ptr(m.Dimension)
	}

	result, err := m.Client.Models.EmbedContent(ctx, m.Model, contents, embedConfig)
	if err != nil {
		logger.ErrorContext(ctx, "Error calling EmbedContent for model %s: %v", m.Model, err)
		return nil, err
	}

	embeddings := make([][]float32, 0, len(result.Embeddings))
	for _, embedding := range result.Embeddings {
		embeddings = append(embeddings, embedding.Values)
	}

	logger.InfoContext(ctx, "Successfully embedded %d text parameters using model %s", len(parameters), m.Model)

	return embeddings, nil
}

// convertStringsToContents takes a slice of strings and converts it into a slice of *genai.Content objects.
func convertStringsToContents(texts []string) []*genai.Content {
	contents := make([]*genai.Content, 0, len(texts))

	for _, text := range texts {
		content := genai.NewContentFromText(text, "")
		contents = append(contents, content)
	}
	return contents
}
