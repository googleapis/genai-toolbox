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

package prebuiltconfigs

import (
	"embed"
	"fmt"
	"path"
	"strings"
)

var (
	//go:embed tools/*.yaml
	prebuiltConfigsFS embed.FS

	// Map of sources to their prebuilt tools
	prebuiltToolYAMLs map[string][]byte
	// List of sources with prebuilt tools
	prebuiltToolsSources []string
)

func init() {
	var err error
	prebuiltToolYAMLs, prebuiltToolsSources, err = loadPrebuiltToolYAMLs()
	if err != nil {
		panic(fmt.Sprintf("Unexpected Error: %v\n", err))
	}
}

// Getter for the prebuiltToolsSources
func GetPrebuiltSources() []string {
	return prebuiltToolsSources
}

// Get prebuilt tools for a source
func Get(prebuiltSourceConfig string) ([]byte, error) {
	content, ok := prebuiltToolYAMLs[prebuiltSourceConfig]
	if !ok {
		prebuiltHelpSuffix := "no prebuilt configurations found."
		if len(prebuiltToolsSources) > 0 {
			prebuiltHelpSuffix = fmt.Sprintf("available: %s", strings.Join(prebuiltToolsSources, ", "))
		}
		errMsg := fmt.Errorf("prebuilt source tool for '%s' not found. %s", prebuiltSourceConfig, prebuiltHelpSuffix)
		return nil, errMsg
	}
	return content, nil
}

// Load all available pre built tools
func loadPrebuiltToolYAMLs() (map[string][]byte, []string, error) {
	toolYAMLs := make(map[string][]byte)
	var sourceTypes []string
	entries, err := prebuiltConfigsFS.ReadDir("tools")
	if err != nil {
		errMsg := fmt.Errorf("failed to read prebuilt tools %w", err)
		return nil, nil, errMsg
	}

	for _, entry := range entries {
		lowerName := strings.ToLower(entry.Name())
		if !entry.IsDir() && (strings.HasSuffix(lowerName, ".yaml")) {
			filePathInFS := path.Join("tools", entry.Name())
			content, err := prebuiltConfigsFS.ReadFile(filePathInFS)
			if err != nil {
				errMsg := fmt.Errorf("failed to read a prebuilt tool %w", err)
				return nil, nil, errMsg
			}
			sourceTypeKey := entry.Name()[:len(entry.Name())-len(".yaml")]

			sourceTypes = append(sourceTypes, sourceTypeKey)
			toolYAMLs[sourceTypeKey] = content
		}
	}
	if len(toolYAMLs) == 0 {
		errMsg := fmt.Errorf("no prebuilt tool configurations were loaded.%w", err)
		return nil, nil, errMsg
	}

	return toolYAMLs, sourceTypes, nil
}
