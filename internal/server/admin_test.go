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

package server

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/auth"
	"github.com/googleapis/genai-toolbox/internal/embeddingmodels"
	"github.com/googleapis/genai-toolbox/internal/prompts"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

func init() {
	sources.Register("mock", func(ctx context.Context, name string, decoder *yaml.Decoder) (sources.SourceConfig, error) {
		var c testutils.MockSourceConfig
		if err := decoder.DecodeContext(ctx, &c); err != nil {
			return nil, err
		}
		return &c, nil
	})
	auth.Register("mock", func(ctx context.Context, name string, decoder *yaml.Decoder) (auth.AuthServiceConfig, error) {
		var c testutils.MockAuthServiceConfig
		if err := decoder.DecodeContext(ctx, &c); err != nil {
			return nil, err
		}
		return &c, nil
	})
	embeddingmodels.Register("mock", func(ctx context.Context, name string, decoder *yaml.Decoder) (embeddingmodels.EmbeddingModelConfig, error) {
		var c testutils.MockEmbeddingModelConfig
		if err := decoder.DecodeContext(ctx, &c); err != nil {
			return nil, err
		}
		return &c, nil
	})
	tools.Register("mock", func(ctx context.Context, name string, decoder *yaml.Decoder) (tools.ToolConfig, error) {
		var c testutils.MockToolConfig
		if err := decoder.DecodeContext(ctx, &c); err != nil {
			return nil, err
		}
		return &c, nil
	})
	prompts.Register("mock", func(ctx context.Context, name string, decoder *yaml.Decoder) (prompts.PromptConfig, error) {
		var c testutils.MockPromptConfig
		if err := decoder.DecodeContext(ctx, &c); err != nil {
			return nil, err
		}
		return &c, nil
	})
}

func TestUpdateEndpoint(t *testing.T) {
	r, shutdown := setUpServer(t, "admin", map[string]sources.Source{}, map[string]auth.AuthService{}, map[string]embeddingmodels.EmbeddingModel{}, map[string]tools.Tool{}, map[string]tools.Toolset{}, map[string]prompts.Prompt{}, map[string]prompts.Promptset{})
	defer shutdown()
	ts := runServer(r, false)
	defer ts.Close()

	tests := []struct {
		name               string
		kind               string
		resourceName       string
		requestBody        string
		expectedStatusCode int
	}{
		{
			name:               "Update Source - Success",
			kind:               "source",
			resourceName:       "test-source",
			requestBody:        `{"config": {"type": "mock"}}`,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "Update Auth Service - Success",
			kind:               "authService",
			resourceName:       "test-auth-service",
			requestBody:        `{"config": {"type": "mock"}}`,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "Update Embedding Model - Success",
			kind:               "embeddingModel",
			resourceName:       "test-embedding-model",
			requestBody:        `{"config": {"type": "mock"}}`,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "Update Tool - Success",
			kind:               "tool",
			resourceName:       "test-tool",
			requestBody:        `{"config": {"type": "mock"}}`,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "Update Toolset - Success",
			kind:               "toolset",
			resourceName:       "test-toolset",
			requestBody:        `{"config": {"name": "test-toolset","tools":["test-tool"]}}`,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "Update Prompt - Success",
			kind:               "prompt",
			resourceName:       "test-prompt",
			requestBody:        `{"config": {"type": "mock"}}`,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "Update unknown primitives - Error",
			kind:               "foo",
			resourceName:       "test-foo",
			requestBody:        `{"config": {"type": "mock"}}`,
			expectedStatusCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, body, err := runRequest(ts, http.MethodPut, fmt.Sprintf("/%s/%s", tt.kind, tt.resourceName), bytes.NewBuffer([]byte(tt.requestBody)), nil)
			if err != nil {
				t.Fatalf("unexpected error during request: %s", err)
			}
			if resp.StatusCode != tt.expectedStatusCode {
				t.Fatalf("response status code is not %d, got %d, %s", tt.expectedStatusCode, resp.StatusCode, string(body))
			}
		})
	}
}
