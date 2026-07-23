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
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"

	"github.com/googleapis/mcp-toolbox/internal/embeddingmodels"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
)

type emulatorMockFile struct {
	Mocks []emulatorMock `json:"mocks"`
}

type emulatorMock struct {
	ToolName    string         `json:"tool_name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters"`
	Response    any            `json:"response"`
}

type emulatorTool struct {
	name  string
	base  tools.Tool
	mocks []emulatorMock
}

func loadEmulatorMocks(path string) (map[string][]emulatorMock, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("unable to read emulator mocks file %q: %w", path, err)
	}

	var file emulatorMockFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("unable to parse emulator mocks file %q: %w", path, err)
	}

	mocks := make(map[string][]emulatorMock)
	for i := range file.Mocks {
		m := &file.Mocks[i]
		if m.ToolName == "" {
			return nil, fmt.Errorf("invalid emulator mock at index %d: tool_name is required", i)
		}
		if m.Parameters == nil {
			m.Parameters = map[string]any{}
		}
		normalized, err := normalizeJSONValue(m.Parameters)
		if err != nil {
			return nil, fmt.Errorf("failed to normalize parameters for mock at index %d: %w", i, err)
		}
		normalizedMap, ok := normalized.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("failed to normalize parameters for mock at index %d: expected object parameters", i)
		}
		m.Parameters = normalizedMap
		mocks[m.ToolName] = append(mocks[m.ToolName], *m)
	}

	return mocks, nil
}

func wrapToolsForEmulator(toolMap map[string]tools.Tool, mocksByTool map[string][]emulatorMock) map[string]tools.Tool {
	wrapped := make(map[string]tools.Tool, len(toolMap))
	for toolName, t := range toolMap {
		wrapped[toolName] = emulatorTool{
			name:  toolName,
			base:  t,
			mocks: mocksByTool[toolName],
		}
	}
	return wrapped
}

func normalizeJSONValue(v any) (any, error) {
	buf, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var normalized any
	if err := json.Unmarshal(buf, &normalized); err != nil {
		return nil, err
	}
	return normalized, nil
}

func (e emulatorTool) findMock(inMap map[string]any) (any, bool) {
	for _, mock := range e.mocks {
		if reflect.DeepEqual(inMap, mock.Parameters) {
			return mock.Response, true
		}
	}
	return nil, false
}

func (e emulatorTool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, util.ToolboxError) {
	in, err := normalizeJSONValue(params.AsMap())
	if err != nil {
		return nil, util.NewAgentError("emulator mode: failed to normalize input parameters", err)
	}
	inMap, ok := in.(map[string]any)
	if !ok {
		return nil, util.NewAgentError("emulator mode: failed to normalize input parameters", nil)
	}
	if response, found := e.findMock(inMap); found {
		return response, nil
	}
	return nil, util.NewAgentError(fmt.Sprintf("emulator mode: no mock matched for tool %q with parameters %v", e.name, inMap), nil)
}

func (e emulatorTool) EmbedParams(ctx context.Context, params parameters.ParamValues, embeddingModels map[string]embeddingmodels.EmbeddingModel) (parameters.ParamValues, error) {
	return e.base.EmbedParams(ctx, params, embeddingModels)
}

func (e emulatorTool) GetName() string {
	return e.base.GetName()
}

func (e emulatorTool) GetDescription() string {
	return e.base.GetDescription()
}

func (e emulatorTool) GetAuthRequired() []string {
	return e.base.GetAuthRequired()
}

func (e emulatorTool) GetAnnotations() *tools.ToolAnnotations {
	return e.base.GetAnnotations()
}

func (e emulatorTool) Manifest(srcs map[string]sources.Source) (tools.Manifest, error) {
	return e.base.Manifest(srcs)
}

func (e emulatorTool) StaticManifest() tools.Manifest {
	return e.base.StaticManifest()
}

func (e emulatorTool) Authorized(verifiedAuthServices []string) bool {
	return e.base.Authorized(verifiedAuthServices)
}

func (e emulatorTool) RequiresClientAuthorization(resourceMgr tools.SourceProvider) (bool, error) {
	return e.base.RequiresClientAuthorization(resourceMgr)
}

func (e emulatorTool) ToConfig() tools.ToolConfig {
	return e.base.ToConfig()
}

func (e emulatorTool) GetAuthTokenHeaderName(resourceMgr tools.SourceProvider) (string, error) {
	return e.base.GetAuthTokenHeaderName(resourceMgr)
}

func (e emulatorTool) GetParameters(srcs map[string]sources.Source) (parameters.Parameters, error) {
	return e.base.GetParameters(srcs)
}

func (e emulatorTool) GetScopesRequired() []string {
	return e.base.GetScopesRequired()
}
