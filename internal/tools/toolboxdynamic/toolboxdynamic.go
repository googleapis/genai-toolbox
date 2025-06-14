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

package toolboxdynamic

import (
	"context"
	"fmt"
	"time"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/runtime"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

const kind string = "toolbox-create-dynamic-tool"

func init() {
	if !tools.Register(kind, newConfig) {
		panic(fmt.Sprintf("tool kind %q already registered", kind))
	}
}

func newConfig(ctx context.Context, name string, decoder *yaml.Decoder) (tools.ToolConfig, error) {
	actual := Config{Name: name}
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return nil, err
	}
	return actual, nil
}

type Config struct {
	Name         string   `yaml:"name" validate:"required"`
	Kind         string   `yaml:"kind" validate:"required"`
	Description  string   `yaml:"description" validate:"required"`
	AuthRequired []string `yaml:"authRequired"`
	Manager      runtime.DynamicToolManager `yaml:"-"` // Injected at runtime
}

// validate interface
var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigKind() string {
	return kind
}

func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	// Default description if not provided
	if cfg.Description == "" {
		cfg.Description = "Create new database tools at runtime with custom SQL queries and parameters"
	}
	
	// Parameters for the MCP tool
	parameters := tools.Parameters{
		tools.NewStringParameter("name", "Unique name for the new tool"),
		tools.NewStringParameter("description", "Description of what the tool does"),
		tools.NewStringParameter("sourceId", "Database source identifier to use for this tool"),
		tools.NewStringParameter("query", "SQL query template for the tool (use $1, $2, etc. for parameters)"),
		tools.NewStringParameter("parameters", "Array of parameter specifications for the tool (JSON array)"),
		tools.NewIntParameter("timeout", "Tool execution timeout in seconds (default 30, max 300)"),
		tools.NewStringParameter("tags", "Optional tags for organizing tools (JSON array)"),
		tools.NewStringParameter("authRequired", "Authentication services required for this tool (JSON array)"),
		tools.NewStringParameter("metadata", "Optional metadata for the tool (JSON object)"),
	}
	
	_, paramManifest, paramMcpManifest := tools.ProcessParameters(tools.Parameters{}, parameters)
	
	mcpManifest := tools.McpManifest{
		Name:        cfg.Name,
		Description: cfg.Description,
		InputSchema: paramMcpManifest,
	}
	
	t := Tool{
		Name:         cfg.Name,
		Kind:         kind,
		AuthRequired: cfg.AuthRequired,
		Parameters:   parameters,
		Sources:      srcs,
		Manager:      cfg.Manager,
		manifest:     tools.Manifest{Description: cfg.Description, Parameters: paramManifest, AuthRequired: cfg.AuthRequired},
		mcpManifest:  mcpManifest,
	}
	
	return &t, nil
}

// validate interface
var _ tools.Tool = &Tool{}

type Tool struct {
	Name         string                      `yaml:"name"`
	Kind         string                      `yaml:"kind"`
	AuthRequired []string                    `yaml:"authRequired"`
	Parameters   tools.Parameters            `yaml:"parameters"`
	Sources      map[string]sources.Source   `yaml:"-"`
	Manager      runtime.DynamicToolManager `yaml:"-"`
	manifest     tools.Manifest
	mcpManifest  tools.McpManifest
}

func (t Tool) Invoke(ctx context.Context, params tools.ParamValues) ([]any, error) {
	paramsMap := params.AsMap()
	
	// Extract required parameters
	toolName, ok := paramsMap["name"].(string)
	if !ok || toolName == "" {
		return nil, fmt.Errorf("name is required and must be a string")
	}
	
	description, ok := paramsMap["description"].(string)
	if !ok || description == "" {
		return nil, fmt.Errorf("description is required and must be a string")
	}
	
	sourceID, ok := paramsMap["sourceId"].(string)
	if !ok || sourceID == "" {
		return nil, fmt.Errorf("sourceId is required and must be a string")
	}
	
	query, ok := paramsMap["query"].(string)
	if !ok || query == "" {
		return nil, fmt.Errorf("query is required and must be a string")
	}
	
	// Verify source exists
	if _, exists := t.Sources[sourceID]; !exists {
		return nil, fmt.Errorf("source %q not found", sourceID)
	}
	
	// Extract optional parameters
	timeout := 30 * time.Second
	if timeoutSec, ok := paramsMap["timeout"].(int); ok {
		if timeoutSec <= 0 || timeoutSec > 300 {
			return nil, fmt.Errorf("timeout must be between 1 and 300 seconds")
		}
		timeout = time.Duration(timeoutSec) * time.Second
	}
	
	// Extract parameter specifications
	var paramSpecs []runtime.ParameterSpec
	if paramArray, ok := paramsMap["parameters"].([]interface{}); ok {
		for i, param := range paramArray {
			paramMap, ok := param.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("parameter %d must be an object", i)
			}
			
			paramSpec, err := parseParameterSpec(paramMap)
			if err != nil {
				return nil, fmt.Errorf("parameter %d: %w", i, err)
			}
			
			paramSpecs = append(paramSpecs, paramSpec)
		}
	}
	
	// Extract tags
	var tags []string
	if tagArray, ok := paramsMap["tags"].([]interface{}); ok {
		for _, tag := range tagArray {
			if tagStr, ok := tag.(string); ok {
				tags = append(tags, tagStr)
			}
		}
	}
	
	// Extract auth requirements
	var authServices []string
	if authArray, ok := paramsMap["authRequired"].([]interface{}); ok {
		for _, auth := range authArray {
			if authStr, ok := auth.(string); ok {
				authServices = append(authServices, authStr)
			}
		}
	}
	
	// Extract metadata
	metadata := make(map[string]interface{})
	if metaMap, ok := paramsMap["metadata"].(map[string]interface{}); ok {
		metadata = metaMap
	}
	
	// Create auth requirement
	authReq := runtime.AuthRequirement{
		Required: len(authServices) > 0,
		Services: authServices,
	}
	
	// Check if manager is available
	if t.Manager == nil {
		return nil, fmt.Errorf("dynamic tool manager not available")
	}
	
	// Create tool specification
	spec := runtime.DynamicToolSpec{
		Name:        toolName,
		Description: description,
		SourceID:    sourceID,
		Query:       query,
		Parameters:  paramSpecs,
		Auth:        authReq,
		Timeout:     timeout,
		Tags:        tags,
		Metadata:    metadata,
	}
	
	// Create the dynamic tool
	createdTool, err := t.Manager.CreateDynamicTool(ctx, spec)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic tool: %w", err)
	}
	
	// Return success response with tool information
	result := []any{
		map[string]any{
			"status":      "success",
			"message":     fmt.Sprintf("Dynamic tool '%s' created successfully", toolName),
			"toolName":    toolName,
			"sourceId":    sourceID,
			"description": description,
			"parameters":  len(paramSpecs),
			"createdAt":   time.Now().UTC(),
		},
	}
	
	// Add tool manifest if available
	if manifestTool, ok := createdTool.(interface{ GetManifest() runtime.ToolManifest }); ok {
		manifest := manifestTool.GetManifest()
		result[0].(map[string]any)["manifest"] = manifest
	}
	
	return result, nil
}

func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (tools.ParamValues, error) {
	return tools.ParseParams(t.Parameters, data, claims)
}

func (t Tool) Manifest() tools.Manifest {
	return t.manifest
}

func (t Tool) McpManifest() tools.McpManifest {
	return t.mcpManifest
}

func (t Tool) Authorized(verifiedAuthServices []string) bool {
	return tools.IsAuthorized(t.AuthRequired, verifiedAuthServices)
}

// SetManager allows injection of the dynamic tool manager
func (t *Tool) SetManager(manager runtime.DynamicToolManager) {
	t.Manager = manager
}

// parseParameterSpec parses a parameter specification from map data
func parseParameterSpec(paramMap map[string]interface{}) (runtime.ParameterSpec, error) {
	spec := runtime.ParameterSpec{}
	
	// Extract name
	name, ok := paramMap["name"].(string)
	if !ok || name == "" {
		return spec, fmt.Errorf("parameter name is required and must be a string")
	}
	spec.Name = name
	
	// Extract type
	paramType, ok := paramMap["type"].(string)
	if !ok || paramType == "" {
		return spec, fmt.Errorf("parameter type is required and must be a string")
	}
	spec.Type = paramType
	
	// Extract description
	if description, ok := paramMap["description"].(string); ok {
		spec.Description = description
	}
	
	// Extract required flag
	if required, ok := paramMap["required"].(bool); ok {
		spec.Required = required
	}
	
	// Extract default value
	if defaultValue, exists := paramMap["default"]; exists {
		spec.Default = defaultValue
	}
	
	// Extract validation rules
	if validation, exists := paramMap["validation"]; exists {
		spec.Validation = validation
	}
	
	return spec, nil
}