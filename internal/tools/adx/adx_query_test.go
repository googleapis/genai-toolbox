// Copyright 2025 Google LLC
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

package adxquery

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Azure/azure-kusto-go/kusto"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

// mockADXSource implements the compatibleSource interface for testing
type mockADXSource struct {
	results []map[string]interface{}
	err     error
}

func (m *mockADXSource) KustoClient() *kusto.Client {
	return nil // Not needed for this test
}

func (m *mockADXSource) GetDatabase() string {
	return "testdb"
}

func (m *mockADXSource) ExecuteQuery(ctx context.Context, query string) ([]map[string]interface{}, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.results, nil
}

func (m *mockADXSource) SourceKind() string {
	return "adx"
}

func TestConfig_ToolConfigKind(t *testing.T) {
	cfg := Config{}
	if got := cfg.ToolConfigKind(); got != kind {
		t.Errorf("ToolConfigKind() = %v, want %v", got, kind)
	}
}

func TestTool_ToolKind(t *testing.T) {
	tool := &Tool{}
	if got := tool.ToolKind(); got != kind {
		t.Errorf("ToolKind() = %v, want %v", got, kind)
	}
}

func TestNewConfig(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr bool
	}{
		{
			name: "valid config",
			yaml: `
name: test-adx-query
kind: adx-query
source: test-adx
description: Test ADX query tool
query: "TestTable | take 10"
`,
			wantErr: false,
		},
		{
			name: "config with parameters",
			yaml: `
name: test-adx-query
kind: adx-query
source: test-adx
description: Test ADX query tool
query: "TestTable | where timestamp > ago({{.hours}}h) | take 10"
`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoder := yaml.NewDecoder(strings.NewReader(tt.yaml))
			_, err := newConfig(context.Background(), "test", decoder)
			if (err != nil) != tt.wantErr {
				t.Errorf("newConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_Initialize(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		sources map[string]sources.Source
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid source",
			config: Config{
				Name:        "test-tool",
				Source:      "test-adx",
				Description: "Test tool",
				Query:       "TestTable | take 10",
			},
			sources: map[string]sources.Source{
				"test-adx": &mockADXSource{},
			},
			wantErr: false,
		},
		{
			name: "source not found",
			config: Config{
				Name:        "test-tool",
				Source:      "missing-source",
				Description: "Test tool",
				Query:       "TestTable | take 10",
			},
			sources: map[string]sources.Source{},
			wantErr: true,
			errMsg:  `source "missing-source" not found`,
		},
		{
			name: "incompatible source",
			config: Config{
				Name:        "test-tool",
				Source:      "incompatible",
				Description: "Test tool",
				Query:       "TestTable | take 10",
			},
			sources: map[string]sources.Source{
				"incompatible": &incompatibleSource{},
			},
			wantErr: true,
			errMsg:  `source "incompatible" (kind: "other") is not compatible with tool "adx-query", compatible sources: [adx]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool, err := tt.config.Initialize(tt.sources)
			if (err != nil) != tt.wantErr {
				t.Errorf("Initialize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("Initialize() error = %v, want error containing %v", err, tt.errMsg)
				return
			}
			if !tt.wantErr && tool == nil {
				t.Error("Initialize() returned nil tool when expecting success")
			}
		})
	}
}

func TestTool_Invoke(t *testing.T) {
	mockResults := []map[string]interface{}{
		{"id": 1, "name": "test1"},
		{"id": 2, "name": "test2"},
	}

	tests := []struct {
		name       string
		tool       *Tool
		params     map[string]any
		wantErr    bool
		errMsg     string
		wantResult interface{}
	}{
		{
			name: "successful query execution",
			tool: &Tool{
				Query:  "TestTable | take 10",
				source: &mockADXSource{results: mockResults},
			},
			params:     map[string]any{},
			wantErr:    false,
			wantResult: mockResults,
		},
		{
			name: "query with positional parameters",
			tool: &Tool{
				Query:  "TestTable | where name == $1 | take $2",
				source: &mockADXSource{results: mockResults},
			},
			params:     map[string]any{"name": "test", "limit": 5},
			wantErr:    false,
			wantResult: mockResults,
		},
		{
			name: "query execution error",
			tool: &Tool{
				Query:  "TestTable | take 10",
				source: &mockADXSource{err: errors.New("connection failed")},
			},
			params:  map[string]any{},
			wantErr: true,
			errMsg:  "failed to execute ADX query: connection failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock ParamValues
			paramValues := tools.ParamValues{}
			for k, v := range tt.params {
				paramValues = append(paramValues, tools.ParamValue{Name: k, Value: v})
			}
			
			result, err := tt.tool.Invoke(context.Background(), paramValues, "")
			if (err != nil) != tt.wantErr {
				t.Errorf("Invoke() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("Invoke() error = %v, want error %v", err, tt.errMsg)
				return
			}
			if !tt.wantErr && result == nil {
				t.Error("Invoke() returned nil result when expecting success")
			}
		})
	}
}

func TestToolRegistration(t *testing.T) {
	// Test that the tool is properly registered
	toolConfig, err := tools.DecodeConfig(context.Background(), kind, "test", yaml.NewDecoder(strings.NewReader(`
name: test-tool
kind: adx-query
source: test-adx
description: Test ADX query tool
query: "TestTable | take 10"
`)))
	if err != nil {
		t.Fatalf("Failed to create tool config: %v", err)
	}

	if toolConfig.ToolConfigKind() != kind {
		t.Errorf("Tool kind mismatch: got %v, want %v", toolConfig.ToolConfigKind(), kind)
	}
}

// incompatibleSource for testing incompatible source handling
type incompatibleSource struct{}

func (s *incompatibleSource) SourceKind() string {
	return "other"
}