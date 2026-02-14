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
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"text/template"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
)

const skillTemplate = `---
name: {{.SkillName}}
description: {{.SkillDescription}}
---

## Usage

All scripts can be executed using Node.js. Replace ` + "`" + `<param_name>` + "`" + ` and ` + "`" + `<param_value>` + "`" + ` with actual values.

**Bash:**
` + "`" + `node scripts/<script_name>.js '{"<param_name>": "<param_value>"}'` + "`" + `

**PowerShell:**
` + "`" + `node scripts/<script_name>.js '{\"<param_name>\": \"<param_value>\"}'` + "`" + `

## Scripts

{{range .Tools}}
### {{.Name}}

{{.Description}}

{{.ParametersSchema}}

---
{{end}}
`

type toolTemplateData struct {
	Name             string
	Description      string
	ParametersSchema string
}

type skillTemplateData struct {
	SkillName        string
	SkillDescription string
	Tools            []toolTemplateData
}

// generateSkillMarkdown generates the content of the SKILL.md file.
// It includes usage instructions and a reference section for each tool in the skill,
// detailing its description and parameters.
func generateSkillMarkdown(skillName, skillDescription string, toolsMap map[string]tools.Tool) (string, error) {
	var toolsData []toolTemplateData

	// Order tools based on name
	var toolNames []string
	for name := range toolsMap {
		toolNames = append(toolNames, name)
	}
	sort.Strings(toolNames)

	for _, name := range toolNames {
		tool := toolsMap[name]
		manifest := tool.Manifest()

		parametersSchema, err := formatParameters(manifest.Parameters)
		if err != nil {
			return "", err
		}

		toolsData = append(toolsData, toolTemplateData{
			Name:             name,
			Description:      manifest.Description,
			ParametersSchema: parametersSchema,
		})
	}

	data := skillTemplateData{
		SkillName:        skillName,
		SkillDescription: skillDescription,
		Tools:            toolsData,
	}

	tmpl, err := template.New("markdown").Parse(skillTemplate)
	if err != nil {
		return "", fmt.Errorf("error parsing markdown template: %w", err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("error executing markdown template: %w", err)
	}

	return buf.String(), nil
}

const nodeScriptTemplate = `#!/usr/bin/env node

const { spawn, execSync } = require('child_process');
const path = require('path');
const fs = require('fs');

const toolName = "{{.Name}}";
const toolsFileName = "{{.ToolsFileName}}";

function getToolboxPath() {
    try {
        const checkCommand = process.platform === 'win32' ? 'where toolbox' : 'which toolbox';
        const globalPath = execSync(checkCommand, { stdio: 'pipe', encoding: 'utf-8' }).trim();
        if (globalPath) {
            return globalPath.split('\n')[0].trim();
        }
    } catch (e) {
        // Ignore error;
    }
    const localPath = path.resolve(__dirname, '../../../toolbox');
    if (fs.existsSync(localPath)) {
        return localPath;
    }
    throw new Error("Toolbox binary not found");
}

let toolboxBinary;
try {
    toolboxBinary = getToolboxPath();
} catch (err) {
    console.error("Error:", err.message);
    process.exit(1);
}

let configArgs = [];
if (toolsFileName) {
  configArgs.push("--tools-file", path.join(__dirname, "..", "assets", toolsFileName));
}

const args = process.argv.slice(2);
const toolboxArgs = [...configArgs, "invoke", toolName, ...args];

const child = spawn(toolboxBinary, toolboxArgs, { stdio: 'inherit' });

child.on('close', (code) => {
  process.exit(code);
});

child.on('error', (err) => {
  console.error("Error executing toolbox:", err);
  process.exit(1);
});
`

type scriptData struct {
	Name          string
	ToolsFileName string
}

// generateScriptContent creates the content for a Node.js wrapper script.
// This script invokes the toolbox CLI with the appropriate configuration
// (using a generated tools file) and arguments to execute the specific tool.
func generateScriptContent(name string, toolsFileName string) (string, error) {
	data := scriptData{
		Name:          name,
		ToolsFileName: toolsFileName,
	}

	tmpl, err := template.New("script").Parse(nodeScriptTemplate)
	if err != nil {
		return "", fmt.Errorf("error parsing script template: %w", err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("error executing script template: %w", err)
	}

	return buf.String(), nil
}

// formatParameters converts a list of parameter manifests into a formatted JSON schema string.
// This schema is used in the skill documentation to describe the input parameters for a tool.
func formatParameters(params []parameters.ParameterManifest) (string, error) {
	if len(params) == 0 {
		return "", nil
	}

	properties := make(map[string]interface{})
	var required []string

	for _, p := range params {
		paramMap := map[string]interface{}{
			"type":        p.Type,
			"description": p.Description,
		}
		if p.Default != nil {
			paramMap["default"] = p.Default
		}
		properties[p.Name] = paramMap
		if p.Required {
			required = append(required, p.Name)
		}
	}

	schema := map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}

	schemaJSON, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return "", fmt.Errorf("error generating parameters schema: %w", err)
	}

	return fmt.Sprintf("#### Parameters\n\n```json\n%s\n```", string(schemaJSON)), nil
}

// generateToolConfigYAML generates the YAML configuration for a single tool and its dependency (source).
// It extracts the relevant tool and source configurations from the server config and formats them
// into a YAML document suitable for inclusion in the skill's assets.
func generateToolConfigYAML(cfg server.ServerConfig, toolName string) ([]byte, error) {
	toolCfg, ok := cfg.ToolConfigs[toolName]
	if !ok {
		return nil, fmt.Errorf("error finding tool config: %s", toolName)
	}

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)

	// Process Tool Config
	toolWrapper := struct {
		Kind   string           `yaml:"kind"`
		Config tools.ToolConfig `yaml:",inline"`
	}{
		Kind:   "tools",
		Config: toolCfg,
	}

	if err := encoder.Encode(toolWrapper); err != nil {
		return nil, fmt.Errorf("error encoding tool config: %w", err)
	}

	// Process Source Config
	var toolMap map[string]interface{}
	b, err := yaml.Marshal(toolCfg)
	if err != nil {
		return nil, fmt.Errorf("error marshaling tool config: %w", err)
	}
	if err := yaml.Unmarshal(b, &toolMap); err != nil {
		return nil, fmt.Errorf("error unmarshaling tool config map: %w", err)
	}

	if sourceName, ok := toolMap["source"].(string); ok && sourceName != "" {
		sourceCfg, ok := cfg.SourceConfigs[sourceName]
		if !ok {
			return nil, fmt.Errorf("error finding source config: %s", sourceName)
		}

		sourceWrapper := struct {
			Kind   string               `yaml:"kind"`
			Config sources.SourceConfig `yaml:",inline"`
		}{
			Kind:   "sources",
			Config: sourceCfg,
		}

		if err := encoder.Encode(sourceWrapper); err != nil {
			return nil, fmt.Errorf("error encoding source config: %w", err)
		}
	}

	return buf.Bytes(), nil
}
