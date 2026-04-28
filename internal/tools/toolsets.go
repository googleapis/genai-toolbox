// Copyright 2024 Google LLC
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

package tools

import (
	"fmt"
	"regexp"
)

type ToolsetConfig struct {
	Name      string   `yaml:"name"`
	ToolNames []string `yaml:",inline"`
}

type Toolset struct {
	ToolsetConfig
	Tools       []*Tool         `yaml:",inline"`
	Manifest    ToolsetManifest `yaml:",inline"`
	McpManifest []McpManifest   `yaml:",inline"`
}

func (t Toolset) ToConfig() ToolsetConfig {
	return t.ToolsetConfig
}

func (ts *Toolset) Initialize(serverVersion string, toolsMap map[string]Tool) error {
	ts.Tools = make([]*Tool, 0, len(ts.ToolNames))
	ts.Manifest = ToolsetManifest{
		ServerVersion: serverVersion,
		ToolsManifest: make(map[string]Manifest),
	}
	for _, toolName := range ts.ToolNames {
		tool, ok := toolsMap[toolName]
		if !ok {
			return fmt.Errorf("tool does not exist: %s", toolName)
		}
		t := tool
		ts.Tools = append(ts.Tools, &t)
		ts.Manifest.ToolsManifest[toolName] = tool.Manifest()
		ts.McpManifest = append(ts.McpManifest, tool.McpManifest())
	}
	return nil
}

type ToolsetManifest struct {
	ServerVersion string              `json:"serverVersion"`
	ToolsManifest map[string]Manifest `json:"tools"`
}

func (t ToolsetConfig) Initialize(serverVersion string, toolsMap map[string]Tool) (Toolset, error) {
	// finish toolset setup
	// Check each declared tool name exists
	toolset := &Toolset{ToolsetConfig: t}
	toolset.Name = t.Name
	if !IsValidName(toolset.Name) {
		return *toolset, fmt.Errorf("invalid toolset name: %s", toolset.Name)
	}
	err := toolset.Initialize(serverVersion, toolsMap)
	if err != nil {
		return *toolset, err
	}
	return *toolset, nil
}

var validName = regexp.MustCompile(`^[a-zA-Z0-9_-]*$`)

func IsValidName(s string) bool {
	return validName.MatchString(s)
}
