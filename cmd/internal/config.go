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
	"bufio"
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
	"github.com/googleapis/mcp-toolbox/internal/auth/generic"
	"github.com/googleapis/mcp-toolbox/internal/server"
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
	EnvVars         map[string]string
	OptionalEnvVars []string
	requiredEnvVars []string

	// AllowMissingEnvVars, when true, substitutes the variable name for an unset
	// required ${VAR} placeholder instead of erroring. Used by offline flows like
	// skills-generate, where source env vars are needed only to satisfy config
	// parsing/validation, never to connect. A non-empty placeholder is used (not
	// "") so required string fields still pass validation. The served path leaves
	// this false so missing config still fails fast.
	AllowMissingEnvVars bool
}

// parseEnv replaces environment variables ${ENV_NAME} with their values.
// also support ${ENV_NAME:default_value}.
func (p *ConfigParser) parseEnv(input string) (string, error) {
	re := regexp.MustCompile(`\$\{(\w+)(:([^}]*))?\}`)

	if p.EnvVars == nil {
		p.EnvVars = make(map[string]string)
	}

	var err error
	matches := re.FindAllStringSubmatchIndex(input, -1)
	var output strings.Builder
	lastIndex := 0
	for _, match := range matches {
		start, end := match[0], match[1]
		output.WriteString(input[lastIndex:start])

		variableName := input[match[2]:match[3]]
		defaultValue := ""
		defaultProvided := match[4] != -1 && match[5] != -1
		if defaultProvided {
			defaultValue = input[match[6]:match[7]]
		}

		if defaultProvided {
			p.OptionalEnvVars = append(p.OptionalEnvVars, variableName)
		} else {
			p.requiredEnvVars = append(p.requiredEnvVars, variableName)
		}

		if value, found := os.LookupEnv(variableName); found {
			p.EnvVars[variableName] = value
			output.WriteString(value)
		} else if defaultProvided {
			p.EnvVars[variableName] = defaultValue
			output.WriteString(defaultValue)
		} else {
			if p.AllowMissingEnvVars {
				p.EnvVars[variableName] = variableName
				output.WriteString(variableName)
			} else if err == nil {
				line, column := lineColumnAt(input, start)
				err = fmt.Errorf("environment variable not found: %q (line %d, column %d)", variableName, line, column)
			}
		}

		lastIndex = end
	}
	output.WriteString(input[lastIndex:])

	// Filter out OptionalEnvVars that were also found as required
	var finalOptional []string
	for _, v := range p.OptionalEnvVars {
		if !slices.Contains(p.requiredEnvVars, v) && !slices.Contains(finalOptional, v) {
			finalOptional = append(finalOptional, v)
		}
	}
	p.OptionalEnvVars = finalOptional

	return output.String(), err
}

// ParseConfig parses the provided yaml into appropriate configs.
func lineColumnAt(input string, index int) (int, int) {
	line := 1
	column := 1
	for i, r := range input {
		if i >= index {
			break
		}
		if r == '\n' {
			line++
			column = 1
		} else {
			column++
		}
	}
	return line, column
}

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
	config.Sources, config.AuthServices, config.EmbeddingModels, config.Tools, config.Toolsets, config.Prompts, err = server.UnmarshalPrimitiveConfig(ctx, raw)
	if err != nil {
		return config, err
	}
	return config, nil
}

// ConvertConfig converts configuration file to flat format.
func ConvertConfig(raw []byte) ([]byte, error) {
	var buf bytes.Buffer
	// Manually copy top-level comments and empty lines from the source
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// If the line is a comment or whitespace, preserve it
		if strings.HasPrefix(trimmed, "#") || trimmed == "" {
			buf.WriteString(line + "\n")
		} else {
			// Stop at the first line of actual data
			break
		}
	}

	// convert configuration file to flat format
	var input yaml.MapSlice
	decoder := yaml.NewDecoder(bytes.NewReader(raw), yaml.UseOrderedMap())
	encoder := yaml.NewEncoder(&buf, yaml.UseLiteralStyleIfMultiline(true))

	nestedFormatKey := []string{"sources", "authServices", "embeddingModels", "tools", "toolsets", "prompts"}
	docIndex := 0
	for {
		if err := decoder.Decode(&input); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		docIndex++
		for _, item := range input {
			key, ok := item.Key.(string)
			if !ok {
				return nil, fmt.Errorf("doc %d: unexpected non-string key in input: %v", docIndex, item.Key)
			}
			if hasKindField(input) {
				// this doc is already in flat format, encode to buf
				if err := encoder.Encode(input); err != nil {
					return nil, err
				}
				break
			}
			// check if value conversion to yaml.MapSlice successfully
			if slice, ok := item.Value.(yaml.MapSlice); slices.Contains(nestedFormatKey, key) && ok {
				switch key {
				case "authServices":
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
					return nil, fmt.Errorf("doc %d: invalid config format at key %q: %w", docIndex, key, err)
				}
				// encode per-doc
				for _, doc := range transformed {
					if err := encoder.Encode(doc); err != nil {
						return nil, err
					}
				}
			} else {
				return nil, fmt.Errorf("doc %d: invalid config format at key %q: expected nested format keys and type map", docIndex, key)
			}
		}
	}
	return buf.Bytes(), nil
}

// hasKindField is a helper function to check if an input is in flat format
func hasKindField(input yaml.MapSlice) bool {
	for _, item := range input {
		if key, ok := item.Key.(string); ok && key == "kind" {
			return true
		}
	}
	return false
}

// transformDocs transforms the configuration file from nested to flat format
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
