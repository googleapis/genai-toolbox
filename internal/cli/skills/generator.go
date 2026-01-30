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
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/template"

	"github.com/goccy/go-yaml"
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

func generateSkillMarkdown(skillName, skillDescription string, toolsMap map[string]Tool) (string, error) {
	var toolsData []toolTemplateData

	// Order tools based on name
	var tools []Tool
	for _, tool := range toolsMap {
		tools = append(tools, tool)
	}
	sort.Slice(tools, func(i, j int) bool {
		return tools[i].Name < tools[j].Name
	})

	for _, tool := range tools {
		parametersSchema, err := formatParameters(tool.Parameters)
		if err != nil {
			return "", err
		}

		toolsData = append(toolsData, toolTemplateData{
			Name:             tool.Name,
			Description:      tool.Description,
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

const { spawn } = require('child_process');
const path = require('path');

const toolName = "{{.Name}}";
const prebuiltNames = {{.PrebuiltNamesJSON}};
const toolsFileName = "{{.ToolsFileName}}";

let configArgs = [];
if (prebuiltNames.length > 0) {
  prebuiltNames.forEach(name => {
    configArgs.push("--prebuilt", name);
  });
}

if (toolsFileName) {
  configArgs.push("--tools-file", path.join(__dirname, "..", "assets", toolsFileName));
}

const args = process.argv.slice(2);
const toolboxArgs = [...configArgs, "invoke", toolName, ...args];

const command = process.platform === 'win32' ? 'toolbox.exe' : 'toolbox';

const child = spawn(command, toolboxArgs, { stdio: 'inherit' });

child.on('close', (code) => {
  process.exit(code);
});

child.on('error', (err) => {
  console.error("Error executing toolbox:", err);
  process.exit(1);
});
`

type scriptData struct {
	Name              string
	PrebuiltNamesJSON string
	ToolsFileName     string
}

func generateScriptContent(name string, config serverConfig, toolsFileName string) (string, error) {
	prebuiltJSON, err := json.Marshal(config.prebuiltConfigs)
	if err != nil {
		return "", fmt.Errorf("error marshaling prebuilt configs: %w", err)
	}

	data := scriptData{
		Name:              name,
		PrebuiltNamesJSON: string(prebuiltJSON),
		ToolsFileName:     toolsFileName,
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

func formatParameters(params []Parameter) (string, error) {
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

	return fmt.Sprintf("## Parameters\n\n```json\n%s\n```", string(schemaJSON)), nil
}

func generateFilteredConfig(toolsFile, toolName string) ([]byte, error) {
	data, err := os.ReadFile(toolsFile)
	if err != nil {
		return nil, fmt.Errorf("error reading %s: %w", toolsFile, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("error parsing YAML from %s: %w", toolsFile, err)
	}

	if _, ok := cfg.Tools[toolName]; !ok {
		return nil, nil // Tool not found in this file
	}

	filteredCfg := Config{
		Tools: map[string]map[string]interface{}{
			toolName: cfg.Tools[toolName],
		},
	}

	// Add relevant source if exists
	if src, ok := cfg.Tools[toolName]["source"].(string); ok && src != "" {
		if sourceData, exists := cfg.Sources[src]; exists {
			if filteredCfg.Sources == nil {
				filteredCfg.Sources = make(map[string]interface{})
			}
			filteredCfg.Sources[src] = sourceData
		}
	}

	filteredData, err := yaml.Marshal(filteredCfg)
	if err != nil {
		return nil, fmt.Errorf("error marshaling filtered tools for %s: %w", toolName, err)
	}
	return filteredData, nil
}
