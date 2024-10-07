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
	"slices"

	"gopkg.in/yaml.v3"
)

type Toolset struct {
	Name  string   `yaml:"name"`
	Tools []string `yaml:",inline"`
}

type ToolsetConfigs map[string]Toolset

// validate interface
var _ yaml.Unmarshaler = &ToolsetConfigs{}

func (c *ToolsetConfigs) UnmarshalYAML(node *yaml.Node) error {
	*c = make(ToolsetConfigs)

	var raw map[string][]string
	if err := node.Decode(&raw); err != nil {
		return err
	}

	for name, tools := range raw {
		(*c)[name] = Toolset{Name: name, Tools: tools}
	}

	return nil
}

func (t Toolset) Initialize(toolsMap map[string]Tool) (Toolset, error) {
	// finish toolset setup
	// fetch existing tool names
	toolNames := make([]string, 0, len(toolsMap))
	for n := range toolsMap {
		toolNames = append(toolNames, n)
	}
	// Validate each declared tool name exists
	for _, name := range t.Tools {
		exists := slices.Contains(toolNames, name) // exists will be true
		if !exists {
			return t, fmt.Errorf("invalide tool name: %s", t)
		}

	}
	return t, nil
}
