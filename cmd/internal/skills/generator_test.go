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

package skills

import (
	"context"
	"strings"
	"testing"

	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
	"go.opentelemetry.io/otel/trace"
)

type MockToolConfig struct {
	Name       string                `yaml:"name"`
	Type       string                `yaml:"type"`
	Source     string                `yaml:"source"`
	Other      string                `yaml:"other"`
	Parameters parameters.Parameters `yaml:"parameters"`
}

func (m MockToolConfig) ToolConfigType() string {
	return m.Type
}

func (m MockToolConfig) Initialize(map[string]sources.Source) (tools.Tool, error) {
	return nil, nil
}

type MockSourceConfig struct {
	Name             string `yaml:"name"`
	Type             string `yaml:"type"`
	ConnectionString string `yaml:"connection_string"`
}

func (m MockSourceConfig) SourceConfigType() string {
	return m.Type
}

func (m MockSourceConfig) Initialize(context.Context, trace.Tracer) (sources.Source, error) {
	return nil, nil
}

func TestFormatParameters(t *testing.T) {
	tests := []struct {
		name         string
		params       []parameters.ParameterManifest
		wantContains []string
		wantErr      bool
	}{
		{
			name:         "empty parameters",
			params:       []parameters.ParameterManifest{},
			wantContains: []string{""},
		},
		{
			name: "single required string parameter",
			params: []parameters.ParameterManifest{
				{
					Name:        "param1",
					Description: "A test parameter",
					Type:        "string",
					Required:    true,
				},
			},
			wantContains: []string{
				"## Parameters",
				"```json",
				`"type": "object"`,
				`"properties": {`,
				`"param1": {`,
				`"type": "string"`,
				`"description": "A test parameter"`,
				`"required": [`,
				`"param1"`,
			},
		},
		{
			name: "mixed parameters with defaults",
			params: []parameters.ParameterManifest{
				{
					Name:        "param1",
					Description: "Param 1",
					Type:        "string",
					Required:    true,
				},
				{
					Name:        "param2",
					Description: "Param 2",
					Type:        "integer",
					Default:     42,
					Required:    false,
				},
			},
			wantContains: []string{
				`"param1": {`,
				`"param2": {`,
				`"default": 42`,
				`"required": [`,
				`"param1"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := formatParameters(tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("formatParameters() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			if len(tt.params) == 0 {
				if got != "" {
					t.Errorf("formatParameters() = %v, want empty string", got)
				}
				return
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("formatParameters() result missing expected string: %s\nGot:\n%s", want, got)
				}
			}
		})
	}
}

func TestGenerateSkillMarkdown(t *testing.T) {
	toolsMap := map[string]tools.Tool{
		"tool1": server.MockTool{
			Description: "First tool",
			Params: []parameters.Parameter{
				parameters.NewStringParameter("p1", "d1"),
			},
		},
	}

	got, err := generateSkillMarkdown("MySkill", "My Description", toolsMap)
	if err != nil {
		t.Fatalf("generateSkillMarkdown() error = %v", err)
	}

	expectedSubstrings := []string{
		"name: MySkill",
		"description: My Description",
		"## Usage",
		"All scripts can be executed using Node.js",
		"**Bash:**",
		"`node scripts/<script_name>.js '{\"<param_name>\": \"<param_value>\"}'`",
		"**PowerShell:**",
		"`node scripts/<script_name>.js '{\"<param_name>\": \"<param_value>\"}'`",
		"## Scripts",
		"### tool1",
		"First tool",
		"## Parameters",
	}

	for _, s := range expectedSubstrings {
		if !strings.Contains(got, s) {
			t.Errorf("generateSkillMarkdown() missing substring %q", s)
		}
	}
}

func TestGenerateScriptContent(t *testing.T) {
	tests := []struct {
		name          string
		toolName      string
		toolsFileName string
		wantContains  []string
	}{
		{
			name:          "basic script",
			toolName:      "test-tool",
			toolsFileName: "",
			wantContains: []string{
				`const toolName = "test-tool";`,
				`const toolsFileName = "";`,
				`const toolboxArgs = [...configArgs, "invoke", toolName, ...args];`,
			},
		},
		{
			name:          "script with tools file",
			toolName:      "complex-tool",
			toolsFileName: "tools.yaml",
			wantContains: []string{
				`const toolName = "complex-tool";`,
				`const toolsFileName = "tools.yaml";`,
				`configArgs.push("--tools-file", path.join(__dirname, "..", "assets", toolsFileName));`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := generateScriptContent(tt.toolName, tt.toolsFileName)
			if err != nil {
				t.Fatalf("generateScriptContent() error = %v", err)
			}

			for _, s := range tt.wantContains {
				if !strings.Contains(got, s) {
					t.Errorf("generateScriptContent() missing substring %q\nGot:\n%s", s, got)
				}
			}
		})
	}
}

func TestGenerateToolConfigYAML(t *testing.T) {
	cfg := server.ServerConfig{
		ToolConfigs: server.ToolConfigs{
			"tool1": MockToolConfig{
				Name:   "tool1",
				Type:   "custom-tool",
				Source: "src1",
				Other:  "foo",
			},
			"toolNoSource": MockToolConfig{
				Name: "toolNoSource",
				Type: "http",
			},
			"toolWithParams": MockToolConfig{
				Name: "toolWithParams",
				Type: "custom-tool",
				Parameters: []parameters.Parameter{
					parameters.NewStringParameter("param1", "desc1"),
				},
			},
			"toolWithMissingSource": MockToolConfig{
				Name:   "toolWithMissingSource",
				Type:   "custom-tool",
				Source: "missing-src",
			},
		},
		SourceConfigs: server.SourceConfigs{
			"src1": MockSourceConfig{
				Name:             "src1",
				Type:             "postgres",
				ConnectionString: "conn1",
			},
		},
	}

	tests := []struct {
		name         string
		toolName     string
		wantContains []string
		wantErr      bool
		wantNil      bool
	}{
		{
			name:     "tool with source",
			toolName: "tool1",
			wantContains: []string{
				"kind: tools",
				"name: tool1",
				"type: custom-tool",
				"source: src1",
				"other: foo",
				"---",
				"kind: sources",
				"name: src1",
				"type: postgres",
				"connection_string: conn1",
			},
		},
		{
			name:     "tool without source",
			toolName: "toolNoSource",
			wantContains: []string{
				"kind: tools",
				"name: toolNoSource",
				"type: http",
			},
		},
		{
			name:     "tool with parameters",
			toolName: "toolWithParams",
			wantContains: []string{
				"kind: tools",
				"name: toolWithParams",
				"type: custom-tool",
				"parameters:",
				"- name: param1",
				"type: string",
				"description: desc1",
			},
		},
		{
			name:     "non-existent tool",
			toolName: "missing-tool",
			wantErr:  true,
		},
		{
			name:     "tool with missing source config",
			toolName: "toolWithMissingSource",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBytes, err := generateToolConfigYAML(cfg, tt.toolName)
			if (err != nil) != tt.wantErr {
				t.Errorf("generateToolConfigYAML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			if tt.wantNil {
				if gotBytes != nil {
					t.Errorf("generateToolConfigYAML() expected nil, got %s", string(gotBytes))
				}
				return
			}

			got := string(gotBytes)
			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("generateToolConfigYAML() result missing expected string: %q\nGot:\n%s", want, got)
				}
			}
		})
	}
}
