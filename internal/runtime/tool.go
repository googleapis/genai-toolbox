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

package runtime

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

// GenericDynamicTool implements tools.Tool for dynamically created tools
type GenericDynamicTool struct {
	mu          sync.RWMutex
	id          string
	name        string
	description string
	source      sources.Source
	query       string
	parameters  []ParameterSpec
	auth        AuthRequirement
	timeout     time.Duration
	manifest    ToolManifest
	manager     *DefaultManager
	
	// Usage tracking
	usageCount int64
	lastUsed   time.Time
}

// Invoke executes the dynamic tool with the provided parameters
func (t *GenericDynamicTool) Invoke(ctx context.Context, params tools.ParamValues) ([]any, error) {
	t.mu.Lock()
	t.usageCount++
	t.lastUsed = time.Now()
	t.mu.Unlock()
	
	// Set timeout if specified
	if t.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, t.timeout)
		defer cancel()
	}
	
	// Prepare SQL request
	req := ArbitrarySQLRequest{
		SourceID:   t.manifest.SourceID,
		Query:      t.query,
		Parameters: params.AsMap(),
		Timeout:    t.timeout,
		MaxRows:    1000, // Default max rows for dynamic tools
		DryRun:     false,
		Context: map[string]interface{}{
			"tool_id":   t.id,
			"tool_name": t.name,
		},
	}
	
	// Execute via manager
	return t.manager.executeSQLQuery(ctx, req, t.source)
}

// ParseParams parses and validates parameters for the tool
func (t *GenericDynamicTool) ParseParams(data map[string]any, claims map[string]map[string]any) (tools.ParamValues, error) {
	// Convert dynamic parameter specs to tools.Parameters format
	toolParams := make(tools.Parameters, len(t.parameters))
	for i, paramSpec := range t.parameters {
		// Create appropriate parameter type based on spec
		var param tools.Parameter
		switch paramSpec.Type {
		case "string":
			param = tools.NewStringParameter(paramSpec.Name, paramSpec.Description)
		case "integer":
			param = tools.NewIntParameter(paramSpec.Name, paramSpec.Description)
		case "float":
			param = tools.NewFloatParameter(paramSpec.Name, paramSpec.Description)
		case "boolean":
			param = tools.NewBooleanParameter(paramSpec.Name, paramSpec.Description)
		case "array":
			// For arrays, default to string items
			stringItem := tools.NewStringParameter("item", "Array item")
			param = tools.NewArrayParameter(paramSpec.Name, paramSpec.Description, stringItem)
		default:
			// Default to string for unknown types
			param = tools.NewStringParameter(paramSpec.Name, paramSpec.Description)
		}
		
		toolParams[i] = param
	}
	
	// Use existing parameter parsing logic
	return tools.ParseParams(toolParams, data, claims)
}

// Manifest returns the tool manifest for API responses
func (t *GenericDynamicTool) Manifest() tools.Manifest {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	// Convert dynamic parameter specs to parameter manifests
	paramManifests := make([]tools.ParameterManifest, len(t.parameters))
	for i, paramSpec := range t.parameters {
		paramManifests[i] = tools.ParameterManifest{
			Name:         paramSpec.Name,
			Type:         paramSpec.Type,
			Description:  paramSpec.Description,
			AuthServices: []string{}, // Dynamic tools don't currently support auth at parameter level
		}
	}
	
	// Convert auth requirement to auth services list
	authRequired := []string{}
	if t.auth.Required {
		authRequired = t.auth.Services
	}
	
	return tools.Manifest{
		Description:  t.description,
		Parameters:   paramManifests,
		AuthRequired: authRequired,
	}
}

// McpManifest returns the MCP-specific manifest for the tool
func (t *GenericDynamicTool) McpManifest() tools.McpManifest {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	// Build input schema for MCP
	properties := make(map[string]tools.ParameterMcpManifest)
	required := []string{}
	
	for _, paramSpec := range t.parameters {
		properties[paramSpec.Name] = tools.ParameterMcpManifest{
			Type:        paramSpec.Type,
			Description: paramSpec.Description,
		}
		
		if paramSpec.Required {
			required = append(required, paramSpec.Name)
		}
	}
	
	inputSchema := tools.McpToolsSchema{
		Type:       "object",
		Properties: properties,
	}
	
	if len(required) > 0 {
		inputSchema.Required = required
	}
	
	return tools.McpManifest{
		Name:        t.name,
		Description: t.description,
		InputSchema: inputSchema,
	}
}

// Authorized checks if the tool call is authorized with the provided auth services
func (t *GenericDynamicTool) Authorized(verifiedAuthServices []string) bool {
	if !t.auth.Required {
		return true
	}
	
	// Check if any required auth service is verified
	for _, requiredService := range t.auth.Services {
		for _, verifiedService := range verifiedAuthServices {
			if requiredService == verifiedService {
				return true
			}
		}
	}
	
	return false
}

// GetManifest returns the detailed manifest with usage statistics
func (t *GenericDynamicTool) GetManifest() ToolManifest {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	manifest := t.manifest
	manifest.UsageCount = t.usageCount
	manifest.LastUsed = t.lastUsed
	manifest.UpdatedAt = time.Now()
	
	return manifest
}

// UpdateQuery updates the tool's query (useful for tool evolution)
func (t *GenericDynamicTool) UpdateQuery(query string) error {
	if query == "" {
		return fmt.Errorf("query cannot be empty")
	}
	
	t.mu.Lock()
	defer t.mu.Unlock()
	
	t.query = query
	t.manifest.UpdatedAt = time.Now()
	
	return nil
}

// UpdateDescription updates the tool's description
func (t *GenericDynamicTool) UpdateDescription(description string) error {
	if description == "" {
		return fmt.Errorf("description cannot be empty")
	}
	
	t.mu.Lock()
	defer t.mu.Unlock()
	
	t.description = description
	t.manifest.Description = description
	t.manifest.UpdatedAt = time.Now()
	
	return nil
}

// UpdateParameters updates the tool's parameters
func (t *GenericDynamicTool) UpdateParameters(parameters []ParameterSpec) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	t.parameters = parameters
	t.manifest.UpdatedAt = time.Now()
	
	return nil
}

// GetUsageStats returns usage statistics for the tool
func (t *GenericDynamicTool) GetUsageStats() ToolUsageStats {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	return ToolUsageStats{
		ToolID:     t.id,
		ToolName:   t.name,
		UsageCount: t.usageCount,
		LastUsed:   t.lastUsed,
		CreatedAt:  t.manifest.CreatedAt,
		UpdatedAt:  t.manifest.UpdatedAt,
	}
}

// ToolUsageStats represents usage statistics for a dynamic tool
type ToolUsageStats struct {
	ToolID     string    `json:"toolId" yaml:"toolId"`
	ToolName   string    `json:"toolName" yaml:"toolName"`
	UsageCount int64     `json:"usageCount" yaml:"usageCount"`
	LastUsed   time.Time `json:"lastUsed" yaml:"lastUsed"`
	CreatedAt  time.Time `json:"createdAt" yaml:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt" yaml:"updatedAt"`
}

// DynamicToolBuilder helps build dynamic tools with validation
type DynamicToolBuilder struct {
	spec    DynamicToolSpec
	errors  []error
}

// NewDynamicToolBuilder creates a new builder for dynamic tools
func NewDynamicToolBuilder() *DynamicToolBuilder {
	return &DynamicToolBuilder{
		spec:   DynamicToolSpec{},
		errors: []error{},
	}
}

// WithName sets the tool name
func (b *DynamicToolBuilder) WithName(name string) *DynamicToolBuilder {
	if name == "" {
		b.errors = append(b.errors, fmt.Errorf("tool name cannot be empty"))
	}
	b.spec.Name = name
	return b
}

// WithDescription sets the tool description
func (b *DynamicToolBuilder) WithDescription(description string) *DynamicToolBuilder {
	if description == "" {
		b.errors = append(b.errors, fmt.Errorf("tool description cannot be empty"))
	}
	b.spec.Description = description
	return b
}

// WithSourceID sets the source ID
func (b *DynamicToolBuilder) WithSourceID(sourceID string) *DynamicToolBuilder {
	if sourceID == "" {
		b.errors = append(b.errors, fmt.Errorf("source ID cannot be empty"))
	}
	b.spec.SourceID = sourceID
	return b
}

// WithQuery sets the SQL query
func (b *DynamicToolBuilder) WithQuery(query string) *DynamicToolBuilder {
	if query == "" {
		b.errors = append(b.errors, fmt.Errorf("query cannot be empty"))
	}
	b.spec.Query = query
	return b
}

// WithParameter adds a parameter to the tool
func (b *DynamicToolBuilder) WithParameter(param ParameterSpec) *DynamicToolBuilder {
	if param.Name == "" {
		b.errors = append(b.errors, fmt.Errorf("parameter name cannot be empty"))
	}
	if param.Type == "" {
		b.errors = append(b.errors, fmt.Errorf("parameter type cannot be empty"))
	}
	b.spec.Parameters = append(b.spec.Parameters, param)
	return b
}

// WithTimeout sets the execution timeout
func (b *DynamicToolBuilder) WithTimeout(timeout time.Duration) *DynamicToolBuilder {
	if timeout > 5*time.Minute {
		b.errors = append(b.errors, fmt.Errorf("timeout cannot exceed 5 minutes"))
	}
	b.spec.Timeout = timeout
	return b
}

// WithAuth sets authentication requirements
func (b *DynamicToolBuilder) WithAuth(auth AuthRequirement) *DynamicToolBuilder {
	b.spec.Auth = auth
	return b
}

// WithTags sets tool tags
func (b *DynamicToolBuilder) WithTags(tags []string) *DynamicToolBuilder {
	b.spec.Tags = tags
	return b
}

// WithMetadata sets tool metadata
func (b *DynamicToolBuilder) WithMetadata(metadata map[string]interface{}) *DynamicToolBuilder {
	b.spec.Metadata = metadata
	return b
}

// Build creates the dynamic tool specification
func (b *DynamicToolBuilder) Build() (DynamicToolSpec, error) {
	// Check for required fields that weren't set
	if b.spec.Name == "" {
		b.errors = append(b.errors, fmt.Errorf("tool name is required"))
	}
	if b.spec.Description == "" {
		b.errors = append(b.errors, fmt.Errorf("tool description is required"))
	}
	if b.spec.SourceID == "" {
		b.errors = append(b.errors, fmt.Errorf("source ID is required"))
	}
	if b.spec.Query == "" {
		b.errors = append(b.errors, fmt.Errorf("query is required"))
	}
	
	if len(b.errors) > 0 {
		return DynamicToolSpec{}, fmt.Errorf("validation errors: %v", b.errors)
	}
	
	// Set defaults
	if b.spec.Timeout == 0 {
		b.spec.Timeout = 30 * time.Second
	}
	
	if b.spec.Tags == nil {
		b.spec.Tags = []string{}
	}
	
	if b.spec.Metadata == nil {
		b.spec.Metadata = make(map[string]interface{})
	}
	
	return b.spec, nil
}