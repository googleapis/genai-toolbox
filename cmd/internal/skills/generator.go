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
	"fmt"
	"sort"
	"strings"
	"text/template"

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
` + "`" + `node <skill_dir>/scripts/<script_name>.js '{"<param_name>": "<param_value>"}'` + "`" + `

**PowerShell:**
` + "`" + `node <skill_dir>/scripts/<script_name>.js '{\"<param_name>\": \"<param_value>\"}'` + "`" + `
{{if .AdditionalNotes}}
{{.AdditionalNotes}}
{{end}}

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
	AdditionalNotes  string
	Tools            []toolTemplateData
}

// generateSkillMarkdown generates the content of the SKILL.md file.
// It includes usage instructions and a reference section for each tool in the skill,
// detailing its description and parameters.
func generateSkillMarkdown(skillName, skillDescription, additionalNotes string, toolsMap map[string]tools.Tool, envVars map[string]string) (string, error) {
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

		parametersSchema, err := formatParameters(manifest.Parameters, envVars)
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
		AdditionalNotes:  additionalNotes,
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
{{if .LicenseHeader}}
{{.LicenseHeader}}
{{end}}
const { spawn, execSync } = require('child_process');
const path = require('path');
const fs = require('fs');

const toolName = "{{.Name}}";
const configArgs = [{{.ConfigArgs}}];

function getToolboxPath() {
    if (process.env.GEMINI_CLI === '1') {
        const ext = process.platform === 'win32' ? '.exe' : '';
        const localPath = path.resolve(__dirname, '../../../toolbox' + ext);
        if (fs.existsSync(localPath)) {
            return localPath;
        }
    }
    try {
        const checkCommand = process.platform === 'win32' ? 'where toolbox' : 'which toolbox';
        const globalPath = execSync(checkCommand, { stdio: 'pipe', encoding: 'utf-8' }).trim();
        if (globalPath) {
            return globalPath.split('\n')[0].trim();
        }
        throw new Error("Toolbox binary not found");
    } catch (e) {
        throw new Error("Toolbox binary not found");
    }
}

let toolboxBinary;
try {
    toolboxBinary = getToolboxPath();
} catch (err) {
    console.error("Error:", err.message);
    process.exit(1);
}

function getEnv() {
    const envPath = path.resolve(__dirname, '../../../.env');
    const env = { ...process.env };
    if (fs.existsSync(envPath)) {
        const envContent = fs.readFileSync(envPath, 'utf-8');
        envContent.split('\n').forEach(line => {
            const trimmed = line.trim();
            if (trimmed && !trimmed.startsWith('#')) {
                const splitIdx = trimmed.indexOf('=');
                if (splitIdx !== -1) {
                    const key = trimmed.slice(0, splitIdx).trim();
                    let value = trimmed.slice(splitIdx + 1).trim();
                    value = value.replace(/(^['"]|['"]$)/g, '');
                    if (env[key] === undefined) {
                        env[key] = value;
                    }
                }
            }
        });
    }
    return env;
}

let env = process.env;
let userAgent = "skills";
if (process.env.GEMINI_CLI === '1') {
    env = getEnv();
    userAgent = "skills-geminicli";
}

const args = process.argv.slice(2);

const toolboxArgs = ["--log-level", "error", ...configArgs, "invoke", toolName, "--user-agent-metadata", userAgent, ...args];

const child = spawn(toolboxBinary, toolboxArgs, { stdio: 'inherit', env });

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
	ConfigArgs    string
	LicenseHeader string
}

// generateScriptContent creates the content for a Node.js wrapper script.
// This script invokes the toolbox CLI with the appropriate configuration
// (using a generated config) and arguments to execute the specific tool.
func generateScriptContent(name string, configArgs string, licenseHeader string) (string, error) {
	data := scriptData{
		Name:          name,
		ConfigArgs:    configArgs,
		LicenseHeader: licenseHeader,
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
func formatParameters(params []parameters.ParameterManifest, envVars map[string]string) (string, error) {
	if len(params) == 0 {
		return "", nil
	}

	var sb strings.Builder
	sb.WriteString("#### Parameters\n\n")
	sb.WriteString("| Name | Type | Description | Required | Default |\n")
	sb.WriteString("| :--- | :--- | :--- | :--- | :--- |\n")

	for _, p := range params {
		required := "No"
		if p.Required {
			required = "Yes"
		}
		defaultValue := ""
		if p.Default != nil {
			defaultValue = fmt.Sprintf("`%v`", p.Default)
			// Check if default value matches any env var
			if strVal, ok := p.Default.(string); ok {
				for _, envVal := range envVars {
					if envVal == strVal {
						defaultValue = ""
						break
					}
				}
			}
		}
		fmt.Fprintf(&sb, "| %s | %s | %s | %s | %s |\n", p.Name, p.Type, p.Description, required, defaultValue)
	}

	return sb.String(), nil
}
