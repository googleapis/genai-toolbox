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
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
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

func TestAdminUpdateEndpoint(t *testing.T) {
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
			expectedStatusCode: http.StatusNoContent,
		},
		{
			name:               "Update Auth Service - Success",
			kind:               "authService",
			resourceName:       "test-auth-service",
			requestBody:        `{"config": {"type": "mock"}}`,
			expectedStatusCode: http.StatusNoContent,
		},
		{
			name:               "Update Embedding Model - Success",
			kind:               "embeddingModel",
			resourceName:       "test-embedding-model",
			requestBody:        `{"config": {"type": "mock"}}`,
			expectedStatusCode: http.StatusNoContent,
		},
		{
			name:               "Update Tool - Success",
			kind:               "tool",
			resourceName:       "test-tool",
			requestBody:        `{"config": {"type": "mock"}}`,
			expectedStatusCode: http.StatusNoContent,
		},
		{
			name:               "Update Toolset - Success",
			kind:               "toolset",
			resourceName:       "test-toolset",
			requestBody:        `{"config": {"name": "test-toolset","tools":["test-tool"]}}`,
			expectedStatusCode: http.StatusNoContent,
		},
		{
			name:               "Update Prompt - Success",
			kind:               "prompt",
			resourceName:       "test-prompt",
			requestBody:        `{"config": {"type": "mock"}}`,
			expectedStatusCode: http.StatusNoContent,
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

func TestAdminDeleteEndpoint(t *testing.T) {
	mockSources := map[string]sources.Source{"test-source": testutils.MockSource{}}
	mockAuthServices := map[string]auth.AuthService{"test-auth-service": testutils.MockAuthService{}}
	mockEmbeddingModel := map[string]embeddingmodels.EmbeddingModel{"test-embedding-model": testutils.MockEmbeddingModel{}}
	mockTool := map[string]tools.Tool{"test-tool": testutils.MockTool{}}
	mockToolset := map[string]tools.Toolset{"test-toolset": tools.Toolset{}}
	mockPrompt := map[string]prompts.Prompt{"test-prompt": testutils.MockPrompt{}}
	r, shutdown := setUpServer(t, "admin", mockSources, mockAuthServices, mockEmbeddingModel, mockTool, mockToolset, mockPrompt, map[string]prompts.Promptset{})
	defer shutdown()
	ts := runServer(r, false)
	defer ts.Close()

	tests := []struct {
		name               string
		kind               string
		resourceName       string
		expectedStatusCode int
	}{
		{
			name:               "Delete Source - Success",
			kind:               "source",
			resourceName:       "test-source",
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "Delete Auth Service - Success",
			kind:               "authService",
			resourceName:       "test-auth-service",
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "Delete Embedding Model - Success",
			kind:               "embeddingModel",
			resourceName:       "test-embedding-model",
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "Delete Tool - Success",
			kind:               "tool",
			resourceName:       "test-tool",
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "Delete Toolset - Success",
			kind:               "toolset",
			resourceName:       "test-toolset",
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "Delete Prompt - Success",
			kind:               "prompt",
			resourceName:       "test-prompt",
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "Delete Non-existent Primitive - Not Found",
			kind:               "source",
			resourceName:       "non-existent-source",
			expectedStatusCode: http.StatusNotFound,
		},
		{
			name:               "Delete with Invalid Kind - Bad Request",
			kind:               "invalidKind",
			resourceName:       "some-name",
			expectedStatusCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, body, err := runRequest(ts, http.MethodDelete, fmt.Sprintf("/%s/%s", tt.kind, tt.resourceName), nil, nil)
			if err != nil {
				t.Fatalf("unexpected error during request: %s", err)
			}
			if resp.StatusCode != tt.expectedStatusCode {
				t.Fatalf("response status code is not %d, got %d, %s", tt.expectedStatusCode, resp.StatusCode, string(body))
			}
		})
	}
}

func TestAdminGetEndpoint(t *testing.T) {
	mockSource := testutils.MockSource{MockSourceConfig: testutils.MockSourceConfig{Foo: "foo", Password: "password"}}
	mockAuthService := testutils.MockAuthService{MockAuthServiceConfig: testutils.MockAuthServiceConfig{Foo: "foo"}}
	mockEmbeddingModel := testutils.MockEmbeddingModel{MockEmbeddingModelConfig: testutils.MockEmbeddingModelConfig{Foo: "foo"}}
	mockTool := testutils.MockTool{MockToolConfig: testutils.MockToolConfig{Foo: "foo"}}
	mockToolset := tools.Toolset{ToolsetConfig: tools.ToolsetConfig{ToolNames: []string{"test-tool"}}}
	mockPrompt := testutils.MockPrompt{MockPromptConfig: testutils.MockPromptConfig{Foo: "foo"}}

	mockSources := map[string]sources.Source{"test-source": mockSource}
	mockAuthServices := map[string]auth.AuthService{"test-auth-service": mockAuthService}
	mockEmbeddingModels := map[string]embeddingmodels.EmbeddingModel{"test-embedding-model": mockEmbeddingModel}
	mockTools := map[string]tools.Tool{"test-tool": mockTool}
	mockToolsets := map[string]tools.Toolset{"test-toolset": mockToolset}
	mockPrompts := map[string]prompts.Prompt{"test-prompt": mockPrompt}

	r, shutdown := setUpServer(t, "admin", mockSources, mockAuthServices, mockEmbeddingModels, mockTools, mockToolsets, mockPrompts, map[string]prompts.Promptset{})
	defer shutdown()
	ts := runServer(r, false)
	defer ts.Close()

	tests := []struct {
		name               string
		kind               string
		want               []string
		expectedStatusCode int
	}{
		{
			name:               "Get Source - Success",
			kind:               "source",
			want:               []string{"test-source"},
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "Get Auth Service - Success",
			kind:               "authService",
			want:               []string{"test-auth-service"},
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "Get Embedding Model - Success",
			kind:               "embeddingModel",
			want:               []string{"test-embedding-model"},
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "Get Tool - Success",
			kind:               "tool",
			want:               []string{"test-tool"},
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "Get Toolset - Success",
			kind:               "toolset",
			want:               []string{"test-toolset"},
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "Get Prompt - Success",
			kind:               "prompt",
			want:               []string{"test-prompt"},
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "Get with Invalid Kind - Bad Request",
			kind:               "invalidKind",
			expectedStatusCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, body, err := runRequest(ts, http.MethodGet, fmt.Sprintf("/%s", tt.kind), nil, nil)
			if err != nil {
				t.Fatalf("unexpected error during request: %s", err)
			}
			if resp.StatusCode != tt.expectedStatusCode {
				t.Fatalf("response status code is not %d, got %d, %s", tt.expectedStatusCode, resp.StatusCode, string(body))
			}
			if tt.expectedStatusCode == http.StatusOK {
				var got []string
				if err := json.Unmarshal(body, &got); err != nil {
					t.Fatalf("error unmarshaling response body")
				}
				if !reflect.DeepEqual(got, tt.want) {
					t.Fatalf("unexpected output: got %+v, want %+v", got, tt.want)
				}
			}
		})
	}
}

func TestAdminGetByNameEndpoint(t *testing.T) {
	mockSource := testutils.MockSource{MockSourceConfig: testutils.MockSourceConfig{Foo: "foo", Password: "password"}}
	mockSourceConfigMasked := testutils.MockSourceConfig{Foo: "foo", Password: "***"}
	mockAuthService := testutils.MockAuthService{MockAuthServiceConfig: testutils.MockAuthServiceConfig{Foo: "foo"}}
	mockEmbeddingModel := testutils.MockEmbeddingModel{MockEmbeddingModelConfig: testutils.MockEmbeddingModelConfig{Foo: "foo"}}
	mockTool := testutils.MockTool{MockToolConfig: testutils.MockToolConfig{Foo: "foo"}}
	mockToolset := tools.Toolset{ToolsetConfig: tools.ToolsetConfig{ToolNames: []string{"test-tool"}}}
	mockPrompt := testutils.MockPrompt{MockPromptConfig: testutils.MockPromptConfig{Foo: "foo"}}

	mockSources := map[string]sources.Source{"test-source": mockSource}
	mockAuthServices := map[string]auth.AuthService{"test-auth-service": mockAuthService}
	mockEmbeddingModels := map[string]embeddingmodels.EmbeddingModel{"test-embedding-model": mockEmbeddingModel}
	mockTools := map[string]tools.Tool{"test-tool": mockTool}
	mockToolsets := map[string]tools.Toolset{"test-toolset": mockToolset}
	mockPrompts := map[string]prompts.Prompt{"test-prompt": mockPrompt}

	r, shutdown := setUpServer(t, "admin", mockSources, mockAuthServices, mockEmbeddingModels, mockTools, mockToolsets, mockPrompts, map[string]prompts.Promptset{})
	defer shutdown()
	ts := runServer(r, false)
	defer ts.Close()

	tests := []struct {
		name               string
		kind               string
		resourceName       string
		want               any
		expectedStatusCode int
	}{
		{
			name:               "Get Source - Success",
			kind:               "source",
			resourceName:       "test-source",
			want:               mockSourceConfigMasked,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "Get Auth Service - Success",
			kind:               "authService",
			resourceName:       "test-auth-service",
			want:               mockAuthService.ToConfig(),
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "Get Embedding Model - Success",
			kind:               "embeddingModel",
			resourceName:       "test-embedding-model",
			want:               mockEmbeddingModel.ToConfig(),
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "Get Tool - Success",
			kind:               "tool",
			resourceName:       "test-tool",
			want:               mockTool.ToConfig(),
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "Get Toolset - Success",
			kind:               "toolset",
			resourceName:       "test-toolset",
			want:               mockToolset.ToConfig(),
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "Get Prompt - Success",
			kind:               "prompt",
			resourceName:       "test-prompt",
			want:               mockPrompt.ToConfig(),
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "Get Non-existent Primitive - Not Found",
			kind:               "source",
			resourceName:       "non-existent-source",
			expectedStatusCode: http.StatusNotFound,
		},
		{
			name:               "Get with Invalid Kind - Bad Request",
			kind:               "invalidKind",
			resourceName:       "some-name",
			expectedStatusCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, body, err := runRequest(ts, http.MethodGet, fmt.Sprintf("/%s/%s", tt.kind, tt.resourceName), nil, nil)
			if err != nil {
				t.Fatalf("unexpected error during request: %s", err)
			}
			if resp.StatusCode != tt.expectedStatusCode {
				t.Fatalf("response status code is not %d, got %d, %s", tt.expectedStatusCode, resp.StatusCode, string(body))
			}
			if tt.expectedStatusCode == http.StatusOK {
				var got any
				if err := json.Unmarshal(body, &got); err != nil {
					t.Fatalf("error unmarshaling response body")
				}
				var want any
				wantBytes, err := json.Marshal(tt.want)
				if err != nil {
					t.Fatalf("error marshaling want struct")
				}
				if err = json.Unmarshal(wantBytes, &want); err != nil {
					t.Fatalf("error unmarshaling want bytes")
				}
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("unexpected output: got %+v, want %+v", got, want)
				}
			}
		})
	}
}
