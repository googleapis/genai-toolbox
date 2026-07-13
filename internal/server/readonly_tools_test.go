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
	"os"
	"testing"

	"github.com/googleapis/mcp-toolbox/internal/log"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
)

type mockSource struct {
	readOnly bool
}

func (m *mockSource) Initialize(ctx context.Context) error {
	return nil
}
func (m *mockSource) Cleanup() error {
	return nil
}
func (m *mockSource) IsReadOnlyMode() bool {
	return m.readOnly
}
func (m *mockSource) SourceType() string {
	return "mock-db"
}
func (m *mockSource) ToConfig() sources.SourceConfig {
	return nil
}

type mockToolConfig struct {
	tools.ConfigBase
	source string
}

func (m *mockToolConfig) ToolConfigType() string {
	return "mock-tool"
}

func (m *mockToolConfig) Initialize(ctx context.Context) (tools.Tool, error) {
	readOnlyHint := m.Name == "readonly-tool"
	var annotations *tools.ToolAnnotations
	if m.Name != "unannotated-tool" {
		annotations = &tools.ToolAnnotations{ReadOnlyHint: &readOnlyHint}
	}
	return &mockTool{
		BaseTool: tools.NewBaseTool(m.ConfigBase, annotations, tools.Manifest{}, nil),
		source:   m.source,
	}, nil
}

type mockTool struct {
	tools.BaseTool[tools.ConfigBase]
	source string
}

func (m *mockTool) Invoke(ctx context.Context, sp tools.SourceProvider, pv parameters.ParamValues, token tools.AccessToken) (any, util.ToolboxError) {
	return nil, nil
}

func (m *mockTool) ToConfig() tools.ToolConfig {
	return nil
}

func TestInitializeTools_ReadOnlySuppression(t *testing.T) {
	ctx := context.Background()

	sourcesMap := map[string]sources.Source{
		"mock-db": &mockSource{readOnly: true},
	}

	cfg := ServerConfig{
		ToolConfigs: map[string]tools.ToolConfig{
			"write-tool": &mockToolConfig{
				ConfigBase: tools.ConfigBase{Name: "write-tool"},
				source:     "mock-db",
			},
			"readonly-tool": &mockToolConfig{
				ConfigBase: tools.ConfigBase{Name: "readonly-tool"},
				source:     "mock-db",
			},
			"unannotated-tool": &mockToolConfig{
				ConfigBase: tools.ConfigBase{Name: "unannotated-tool"},
				source:     "mock-db",
			},
		},
	}

	testLogger, _ := log.NewStdLogger(os.Stdout, os.Stderr, "info")
	instr, _ := telemetry.CreateTelemetryInstrumentation("test")

	toolsMap, err := initializeTools(ctx, cfg, sourcesMap, instr, testLogger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 1. readonly-tool should be present
	if _, ok := toolsMap["readonly-tool"]; !ok {
		t.Errorf("readonly-tool should NOT be suppressed")
	}

	// 2. write-tool should be suppressed
	if _, ok := toolsMap["write-tool"]; ok {
		t.Errorf("write-tool should be suppressed")
	}

	// 3. unannotated-tool should NOT be suppressed (graceful fallback)
	if _, ok := toolsMap["unannotated-tool"]; !ok {
		t.Errorf("unannotated-tool should NOT be suppressed")
	}
}

func TestInitializeTools_WriteModeNoSuppression(t *testing.T) {
	ctx := context.Background()

	sourcesMap := map[string]sources.Source{
		"mock-db": &mockSource{readOnly: false}, // Write mode!
	}

	cfg := ServerConfig{
		ToolConfigs: map[string]tools.ToolConfig{
			"write-tool": &mockToolConfig{
				ConfigBase: tools.ConfigBase{Name: "write-tool"},
				source:     "mock-db",
			},
		},
	}

	testLogger, _ := log.NewStdLogger(os.Stdout, os.Stderr, "info")

	toolsMap, err := initializeTools(ctx, cfg, sourcesMap, telemetry.NewNoopInstrumentation(), testLogger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// write-tool should be present because the source is NOT read-only
	if _, ok := toolsMap["write-tool"]; !ok {
		t.Errorf("write-tool should NOT be suppressed in write mode")
	}
}
