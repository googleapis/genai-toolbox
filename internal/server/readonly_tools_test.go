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
	"testing"

	"github.com/googleapis/mcp-toolbox/internal/log"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/telemetry"
	"github.com/googleapis/mcp-toolbox/internal/tools"
)

type mockSource struct {
	readOnly bool
}

func (m *mockSource) SourceType() string {
	return "mock-source"
}

func (m *mockSource) ToConfig() sources.SourceConfig {
	return nil
}

func (m *mockSource) IsReadOnlyMode() bool {
	return m.readOnly
}

type mockToolConfig struct {
	source string
	Source string
	name   string
}

func (m *mockToolConfig) ToolConfigType() string {
	return "mock-tool"
}

func (m *mockToolConfig) Initialize(ctx context.Context) (tools.Tool, error) {
	return &mockTool{
		source: m.source,
		name:   m.name,
	}, nil
}

type mockTool struct {
	source string
	name   string
}

func (m *mockTool) Name() string {
	return m.name
}

func (m *mockTool) Description() string {
	return "mock tool"
}

func (m *mockTool) Execute(ctx context.Context, params map[string]any) (*tools.ToolResponse, error) {
	return nil, nil
}

func (m *mockTool) InputSchema() map[string]any {
	return nil
}

func (m *mockTool) GetLinkedSource() sources.Source {
	return nil
}

func (m *mockTool) GetAnnotations() *tools.Annotations {
	if m.name == "unannotated-tool" {
		return nil
	}
	readOnlyHint := false
	if m.name == "readonly-tool" {
		readOnlyHint = true
	}
	return &tools.Annotations{
		ReadOnlyHint: &readOnlyHint,
	}
}

func TestInitializeTools_ReadOnlySuppression(t *testing.T) {
	ctx := context.Background()
	tracer := telemetry.NewNoopInstrumentation()

	sourcesMap := map[string]sources.Source{
		"mock-db": &mockSource{readOnly: true},
	}

	cfg := ServerConfig{
		ToolConfigs: map[string]tools.ToolConfig{
			"write-tool": &mockToolConfig{
				source: "mock-db",
				Source: "mock-db",
				name:   "write-tool",
			},
			"readonly-tool": &mockToolConfig{
				source: "mock-db",
				Source: "mock-db",
				name:   "readonly-tool",
			},
			"unannotated-tool": &mockToolConfig{
				source: "mock-db",
				Source: "mock-db",
				name:   "unannotated-tool",
			},
		},
	}

	toolsMap, err := initializeTools(ctx, cfg, sourcesMap, tracer, log.NewLogger(true))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := toolsMap["write-tool"]; ok {
		t.Errorf("expected write-tool to be suppressed")
	}

	if _, ok := toolsMap["readonly-tool"]; !ok {
		t.Errorf("expected readonly-tool to be present")
	}

	if _, ok := toolsMap["unannotated-tool"]; !ok {
		t.Errorf("expected unannotated-tool to be present (but emit a warning)")
	}
}
