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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
)

func TestFormatParameters(t *testing.T) {
	tests := []struct {
		name         string
		params       []Parameter
		wantContains []string
		wantErr      bool
	}{
		{
			name:         "empty parameters",
			params:       []Parameter{},
			wantContains: []string{""},
		},
		{
			name: "single required string parameter",
			params: []Parameter{
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
			params: []Parameter{
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
	tools := map[string]Tool{
		"tool1": {
			Name:        "tool1",
			Description: "First tool",
			Parameters: []Parameter{
				{Name: "p1", Type: "string", Description: "d1", Required: true},
			},
		},
	}

	got, err := generateSkillMarkdown("MySkill", "My Description", tools)
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
		"`node scripts/<script_name>.js '{\\\"<param_name>\\\": \\\"<param_value>\\\"}'`",
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

func TestGenerateShellScriptContent(t *testing.T) {
	tests := []struct {
		name          string
		toolName      string
		config        serverConfig
		toolsFileName string
		wantContains  []string
	}{
		{
			name:     "basic script",
			toolName: "test-tool",
			config: serverConfig{
				prebuiltConfigs: []string{},
			},
			toolsFileName: "",
			wantContains: []string{
				`const toolName = "test-tool";`,
				`const prebuiltNames = [];`,
				`const toolsFileName = "";`,
				`const toolboxArgs = [...configArgs, "invoke", toolName, ...args];`,
			},
		},
		{
			name:     "script with prebuilts and tools file",
			toolName: "complex-tool",
			config: serverConfig{
				prebuiltConfigs: []string{"pre1", "pre2"},
			},
			toolsFileName: "tools.yaml",
			wantContains: []string{
				`const toolName = "complex-tool";`,
				`const prebuiltNames = ["pre1","pre2"];`,
				`const toolsFileName = "tools.yaml";`,
				`configArgs.push("--prebuilt", name);`,
				`configArgs.push("--tools-file", path.join(__dirname, "..", "assets", toolsFileName));`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := generateScriptContent(tt.toolName, tt.config, tt.toolsFileName)
			if err != nil {
				t.Fatalf("generateShellScriptContent() error = %v", err)
			}

			for _, s := range tt.wantContains {
				if !strings.Contains(got, s) {
					t.Errorf("generateShellScriptContent() missing substring %q\nGot:\n%s", s, got)
				}
			}
		})
	}
}

func TestGenerateFilteredConfig(t *testing.T) {
	// Setup temporary directory and file
	tmpDir := t.TempDir()
	toolsFile := filepath.Join(tmpDir, "tools.yaml")

	configContent := `
sources:
  src1:
    type: "postgres"
    connection_string: "conn1"
  src2:
    type: "mysql"
    connection_string: "conn2"
tools:
  tool1:
    source: "src1"
    query: "SELECT 1"
  tool2:
    source: "src2"
    query: "SELECT 2"
  tool3:
    type: "http" # No source
`
	if err := os.WriteFile(toolsFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create temp tools file: %v", err)
	}

	tests := []struct {
		name     string
		toolName string
		wantCfg  Config
		wantErr  bool
		wantNil  bool
	}{
		{
			name:     "tool with source",
			toolName: "tool1",
			wantCfg: Config{
				Sources: map[string]interface{}{
					"src1": map[string]interface{}{
						"type":              "postgres",
						"connection_string": "conn1",
					},
				},
				Tools: map[string]map[string]interface{}{
					"tool1": {
						"source": "src1",
						"query":  "SELECT 1",
					},
				},
			},
		},
		{
			name:     "tool without source",
			toolName: "tool3",
			wantCfg: Config{
				Tools: map[string]map[string]interface{}{
					"tool3": {
						"type": "http",
					},
				},
			},
		},
		{
			name:     "non-existent tool",
			toolName: "missing-tool",
			wantNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBytes, err := generateFilteredConfig(toolsFile, tt.toolName)
			if (err != nil) != tt.wantErr {
				t.Errorf("generateFilteredConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			if tt.wantNil {
				if gotBytes != nil {
					t.Errorf("generateFilteredConfig() expected nil, got %s", string(gotBytes))
				}
				return
			}

			var gotCfg Config
			if err := yaml.Unmarshal(gotBytes, &gotCfg); err != nil {
				t.Errorf("Failed to unmarshal result: %v", err)
			}

			if diff := cmp.Diff(tt.wantCfg, gotCfg); diff != "" {
				t.Errorf("generateFilteredConfig() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
