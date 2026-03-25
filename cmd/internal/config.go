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

package internal

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/auth/generic"
	"github.com/googleapis/genai-toolbox/internal/server"
)

type Config struct {
	Sources         server.SourceConfigs         `yaml:"sources"`
	AuthServices    server.AuthServiceConfigs    `yaml:"authServices"`
	EmbeddingModels server.EmbeddingModelConfigs `yaml:"embeddingModels"`
	Tools           server.ToolConfigs           `yaml:"tools"`
	Toolsets        server.ToolsetConfigs        `yaml:"toolsets"`
	Prompts         server.PromptConfigs         `yaml:"prompts"`
}

type ConfigParser struct {
	EnvVars map[string]string
}

// parseEnv replaces environment variables ${ENV_NAME} with their values.
// also support ${ENV_NAME:default_value}.
func (p *ConfigParser) parseEnv(input string) (string, error) {
	re := regexp.MustCompile(`\$\{(\w+)(:([^}]*))?\}`)

	if p.EnvVars == nil {
		p.EnvVars = make(map[string]string)
	}

	var err error
	output := re.ReplaceAllStringFunc(input, func(match string) string {
		parts := re.FindStringSubmatch(match)

		// extract the variable name
		variableName := parts[1]
		if value, found := os.LookupEnv(variableName); found {
			p.EnvVars[variableName] = value
			return value
		}
		if len(parts) >= 4 && parts[2] != "" {
			value := parts[3]
			p.EnvVars[variableName] = value
			return value
		}
		err = fmt.Errorf("environment variable not found: %q", variableName)
		return ""
	})
	return output, err
}

// ParseConfig parses the provided yaml into appropriate configs.
func (p *ConfigParser) ParseConfig(ctx context.Context, raw []byte) (Config, error) {
	var config Config
	// Replace environment variables if found
	output, err := p.parseEnv(string(raw))
	if err != nil {
		return config, fmt.Errorf("error parsing environment variables: %s", err)
	}
	raw = []byte(output)

	raw, err = ConvertConfig(raw)
	if err != nil {
		return config, fmt.Errorf("error converting config file: %s", err)
	}

	// Parse contents
	config.Sources, config.AuthServices, config.EmbeddingModels, config.Tools, config.Toolsets, config.Prompts, err = server.UnmarshalResourceConfig(ctx, raw)
	if err != nil {
		return config, err
	}
	return config, nil
}

// ConvertConfig converts configuration file to flat format.
func ConvertConfig(raw []byte) ([]byte, error) {
	var input yaml.MapSlice
	decoder := yaml.NewDecoder(bytes.NewReader(raw), yaml.UseOrderedMap())

	// convert to config file v2
	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)

	v1keys := []string{"sources", "authServices", "embeddingModels", "tools", "toolsets", "prompts"}
	for {
		if err := decoder.Decode(&input); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		for _, item := range input {
			key, ok := item.Key.(string)
			if !ok {
				return nil, fmt.Errorf("unexpected non-string key in input: %v", item.Key)
			}
			// check if the key is config file v1's key
			if slices.Contains(v1keys, key) {
				// check if value conversion to yaml.MapSlice successfully
				// fields such as "tools" in toolsets might pass the first check but
				// fail to convert to MapSlice
				if slice, ok := item.Value.(yaml.MapSlice); ok {
					// Deprecated: convert authSources to authServices
					switch key {
					case "authSources", "authServices":
						key = "authService"
					case "sources":
						key = "source"
					case "embeddingModels":
						key = "embeddingModel"
					case "tools":
						key = "tool"
					case "toolsets":
						key = "toolset"
					case "prompts":
						key = "prompt"
					}
					transformed, err := transformDocs(key, slice)
					if err != nil {
						return nil, err
					}
					// encode per-doc
					for _, doc := range transformed {
						if err := encoder.Encode(doc); err != nil {
							return nil, err
						}
					}
				} else {
					// invalid input will be ignored
					// we don't want to throw error here since the config could
					// be valid but with a different order such as:
					// ---
					// tools:
					// - tool_a
					// kind: toolset
					// ---
					continue
				}
			} else {
				// this doc is already v2, encode to buf
				if err := encoder.Encode(input); err != nil {
					return nil, err
				}
				break
			}
		}
	}
	return buf.Bytes(), nil
}

// transformDocs transforms the configuration file from v1 format to v2
// yaml.MapSlice will preserve the order in a map
func transformDocs(kind string, input yaml.MapSlice) ([]yaml.MapSlice, error) {
	var transformed []yaml.MapSlice
	for _, entry := range input {
		entryName, ok := entry.Key.(string)
		if !ok {
			return nil, fmt.Errorf("unexpected non-string key for entry in '%s': %v", kind, entry.Key)
		}
		entryBody := processValue(entry.Value, kind == "toolset")

		currentTransformed := yaml.MapSlice{
			{Key: "kind", Value: kind},
			{Key: "name", Value: entryName},
		}

		// Merge the transformed body into our result
		if bodySlice, ok := entryBody.(yaml.MapSlice); ok {
			currentTransformed = append(currentTransformed, bodySlice...)
		} else {
			return nil, fmt.Errorf("unable to convert entryBody to MapSlice")
		}
		transformed = append(transformed, currentTransformed)
	}
	return transformed, nil
}

// processValue recursively looks for MapSlices to rename 'kind' -> 'type'
func processValue(v any, isToolset bool) any {
	switch val := v.(type) {
	case yaml.MapSlice:
		// creating a new MapSlice is safer for recursive transformation
		newVal := make(yaml.MapSlice, len(val))
		for i, item := range val {
			// Perform renaming
			if item.Key == "kind" {
				item.Key = "type"
			}
			// Recursive call for nested values (e.g., nested objects or lists)
			item.Value = processValue(item.Value, false)
			newVal[i] = item
		}
		return newVal
	case []any:
		// Process lists: If it's a toolset top-level list, wrap it.
		if isToolset {
			return yaml.MapSlice{{Key: "tools", Value: val}}
		}
		// Otherwise, recurse into list items (to catch nested objects)
		newVal := make([]any, len(val))
		for i := range val {
			newVal[i] = processValue(val[i], false)
		}
		return newVal
	default:
		return val
	}
}

// mergeConfigs merges multiple Config structs into one.
// Detects and raises errors for resource conflicts in sources, authServices, tools, and toolsets.
// All resource names (sources, authServices, tools, toolsets) must be unique across all files.
func mergeConfigs(files ...Config) (Config, error) {
	merged := Config{
		Sources:         make(server.SourceConfigs),
		AuthServices:    make(server.AuthServiceConfigs),
		EmbeddingModels: make(server.EmbeddingModelConfigs),
		Tools:           make(server.ToolConfigs),
		Toolsets:        make(server.ToolsetConfigs),
		Prompts:         make(server.PromptConfigs),
	}

	var conflicts []string

	for fileIndex, file := range files {
		// Check for conflicts and merge sources
		for name, source := range file.Sources {
			if mergedSource, exists := merged.Sources[name]; exists {
				if !cmp.Equal(mergedSource, source) {
					conflicts = append(conflicts, fmt.Sprintf("source '%s' (file #%d)", name, fileIndex+1))
				}
			} else {
				merged.Sources[name] = source
			}
		}

		// Check for conflicts and merge authServices
		for name, authService := range file.AuthServices {
			if _, exists := merged.AuthServices[name]; exists {
				conflicts = append(conflicts, fmt.Sprintf("authService '%s' (file #%d)", name, fileIndex+1))
			} else {
				merged.AuthServices[name] = authService
			}
		}

		// Check for conflicts and merge embeddingModels
		for name, em := range file.EmbeddingModels {
			if _, exists := merged.EmbeddingModels[name]; exists {
				conflicts = append(conflicts, fmt.Sprintf("embedding model '%s' (file #%d)", name, fileIndex+1))
			} else {
				merged.EmbeddingModels[name] = em
			}
		}

		// Check for conflicts and merge tools
		for name, tool := range file.Tools {
			if _, exists := merged.Tools[name]; exists {
				conflicts = append(conflicts, fmt.Sprintf("tool '%s' (file #%d)", name, fileIndex+1))
			} else {
				merged.Tools[name] = tool
			}
		}

		// Check for conflicts and merge toolsets
		for name, toolset := range file.Toolsets {
			if _, exists := merged.Toolsets[name]; exists {
				conflicts = append(conflicts, fmt.Sprintf("toolset '%s' (file #%d)", name, fileIndex+1))
			} else {
				merged.Toolsets[name] = toolset
			}
		}

		// Check for conflicts and merge prompts
		for name, prompt := range file.Prompts {
			if _, exists := merged.Prompts[name]; exists {
				conflicts = append(conflicts, fmt.Sprintf("prompt '%s' (file #%d)", name, fileIndex+1))
			} else {
				merged.Prompts[name] = prompt
			}
		}
	}

	// If conflicts were detected, return an error
	if len(conflicts) > 0 {
		return Config{}, fmt.Errorf("resource conflicts detected:\n  - %s\n\nPlease ensure each source, authService, tool, toolset and prompt has a unique name across all files", strings.Join(conflicts, "\n  - "))
	}

	// Ensure only one authService has mcpEnabled = true
	var mcpEnabledAuthServers []string
	for name, authService := range merged.AuthServices {
		// Only generic type has McpEnabled right now
		if genericService, ok := authService.(generic.Config); ok && genericService.McpEnabled {
			mcpEnabledAuthServers = append(mcpEnabledAuthServers, name)
		}
	}
	if len(mcpEnabledAuthServers) > 1 {
		return Config{}, fmt.Errorf("multiple authServices with mcpEnabled=true detected: %s. Only one MCP authorization server is currently supported", strings.Join(mcpEnabledAuthServers, ", "))
	}

	return merged, nil
}

// LoadAndMergeConfigs loads multiple YAML files and merges them
func (p *ConfigParser) LoadAndMergeConfigs(ctx context.Context, filePaths []string) (Config, error) {
	var configs []Config

	for _, filePath := range filePaths {
		buf, err := os.ReadFile(filePath)
		if err != nil {
			return Config{}, fmt.Errorf("unable to read config file at %q: %w", filePath, err)
		}

		config, err := p.ParseConfig(ctx, buf)
		if err != nil {
			return Config{}, fmt.Errorf("unable to parse config file at %q: %w", filePath, err)
		}

		configs = append(configs, config)
	}
	if len(configs) == 0 {
		return Config{}, fmt.Errorf("no YAML files found")
	}
	if len(configs) > 1 {
		mergedFile, err := mergeConfigs(configs...)
		if err != nil {
			return Config{}, fmt.Errorf("unable to merge config files: %w", err)
		}
		return mergedFile, nil
	}
	return configs[0], nil
}

// GetPathsFromConfigFolder loads all YAML files from a directory and merges them
func GetPathsFromConfigFolder(ctx context.Context, folderPath string) ([]string, error) {
	// Check if directory exists
	info, err := os.Stat(folderPath)
	if err != nil {
		return nil, fmt.Errorf("unable to access config folder at %q: %w", folderPath, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path %q is not a directory", folderPath)
	}

	// Find all YAML files in the directory
	pattern := filepath.Join(folderPath, "*.yaml")
	yamlFiles, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("error finding YAML files in %q: %w", folderPath, err)
	}

	// Also find .yml files
	ymlPattern := filepath.Join(folderPath, "*.yml")
	ymlFiles, err := filepath.Glob(ymlPattern)
	if err != nil {
		return nil, fmt.Errorf("error finding YML files in %q: %w", folderPath, err)
	}

	// Combine both file lists
	allFiles := append(yamlFiles, ymlFiles...)
	return allFiles, nil
}
