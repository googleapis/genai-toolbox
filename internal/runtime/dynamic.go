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
	"go.opentelemetry.io/otel/trace"
)

// DynamicToolManager manages runtime creation and execution of dynamic tools
type DynamicToolManager interface {
	// CreateDynamicTool creates a new tool at runtime with the given specification
	CreateDynamicTool(ctx context.Context, spec DynamicToolSpec) (tools.Tool, error)
	
	// ExecuteArbitrarySQL executes arbitrary SQL queries against configured sources
	ExecuteArbitrarySQL(ctx context.Context, req ArbitrarySQLRequest) ([]any, error)
	
	// ListDynamicTools returns a list of all dynamically created tools
	ListDynamicTools(ctx context.Context) ([]ToolManifest, error)
	
	// RemoveDynamicTool removes a dynamically created tool
	RemoveDynamicTool(ctx context.Context, toolID string) error
	
	// GetDynamicTool retrieves a dynamically created tool by ID
	GetDynamicTool(ctx context.Context, toolID string) (tools.Tool, error)
	
	// Cleanup performs periodic cleanup of unused dynamic resources
	Cleanup(ctx context.Context) error
}

// RuntimeSourceManager manages dynamic source creation and lifecycle
type RuntimeSourceManager interface {
	// CreateSource creates a new source at runtime
	CreateSource(ctx context.Context, config sources.SourceConfig) (sources.Source, error)
	
	// ListSources returns a list of all sources (static + dynamic)
	ListSources(ctx context.Context) ([]SourceManifest, error)
	
	// RemoveSource removes a dynamically created source
	RemoveSource(ctx context.Context, sourceID string) error
	
	// TestConnection tests connectivity for a source
	TestConnection(ctx context.Context, sourceID string) error
	
	// GetSource retrieves a source by ID
	GetSource(ctx context.Context, sourceID string) (sources.Source, error)
}

// ConfigManager manages dynamic configuration persistence and validation
type ConfigManager interface {
	// SaveConfiguration persists dynamic configuration
	SaveConfiguration(ctx context.Context, config DynamicConfig) error
	
	// LoadConfiguration loads dynamic configuration
	LoadConfiguration(ctx context.Context) (DynamicConfig, error)
	
	// ValidateConfiguration validates configuration without applying
	ValidateConfiguration(ctx context.Context, config DynamicConfig) error
	
	// ExportConfiguration exports configuration in specified format
	ExportConfiguration(ctx context.Context, format string) ([]byte, error)
	
	// ImportConfiguration imports configuration from data
	ImportConfiguration(ctx context.Context, data []byte, format string) error
}

// DynamicToolSpec defines the specification for creating a dynamic tool
type DynamicToolSpec struct {
	Name        string                 `json:"name" yaml:"name"`
	Description string                 `json:"description" yaml:"description"`
	SourceID    string                 `json:"sourceId" yaml:"sourceId"`
	Query       string                 `json:"query" yaml:"query"`
	Parameters  []ParameterSpec        `json:"parameters" yaml:"parameters"`
	Auth        AuthRequirement        `json:"auth" yaml:"auth"`
	Timeout     time.Duration          `json:"timeout" yaml:"timeout"`
	Tags        []string               `json:"tags" yaml:"tags"`
	Metadata    map[string]interface{} `json:"metadata" yaml:"metadata"`
}

// ArbitrarySQLRequest defines a request for arbitrary SQL execution
type ArbitrarySQLRequest struct {
	SourceID    string                 `json:"sourceId" yaml:"sourceId"`
	Query       string                 `json:"query" yaml:"query"`
	Parameters  map[string]interface{} `json:"parameters" yaml:"parameters"`
	Timeout     time.Duration          `json:"timeout" yaml:"timeout"`
	MaxRows     int                    `json:"maxRows" yaml:"maxRows"`
	DryRun      bool                   `json:"dryRun" yaml:"dryRun"`
	Context     map[string]interface{} `json:"context" yaml:"context"`
}

// ParameterSpec defines a parameter for dynamic tools
type ParameterSpec struct {
	Name        string      `json:"name" yaml:"name"`
	Type        string      `json:"type" yaml:"type"`
	Description string      `json:"description" yaml:"description"`
	Required    bool        `json:"required" yaml:"required"`
	Default     interface{} `json:"default" yaml:"default"`
	Validation  interface{} `json:"validation" yaml:"validation"`
}

// AuthRequirement defines authentication requirements for dynamic tools
type AuthRequirement struct {
	Required     bool     `json:"required" yaml:"required"`
	Services     []string `json:"services" yaml:"services"`
	Permissions  []string `json:"permissions" yaml:"permissions"`
	PolicyChecks []string `json:"policyChecks" yaml:"policyChecks"`
}

// ToolManifest provides metadata about a dynamic tool
type ToolManifest struct {
	ID          string                 `json:"id" yaml:"id"`
	Name        string                 `json:"name" yaml:"name"`
	Description string                 `json:"description" yaml:"description"`
	SourceID    string                 `json:"sourceId" yaml:"sourceId"`
	Kind        string                 `json:"kind" yaml:"kind"`
	CreatedAt   time.Time              `json:"createdAt" yaml:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt" yaml:"updatedAt"`
	LastUsed    time.Time              `json:"lastUsed" yaml:"lastUsed"`
	UsageCount  int64                  `json:"usageCount" yaml:"usageCount"`
	Tags        []string               `json:"tags" yaml:"tags"`
	Metadata    map[string]interface{} `json:"metadata" yaml:"metadata"`
	Status      string                 `json:"status" yaml:"status"`
}

// SourceManifest provides metadata about a source
type SourceManifest struct {
	ID          string                 `json:"id" yaml:"id"`
	Name        string                 `json:"name" yaml:"name"`
	Kind        string                 `json:"kind" yaml:"kind"`
	Description string                 `json:"description" yaml:"description"`
	CreatedAt   time.Time              `json:"createdAt" yaml:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt" yaml:"updatedAt"`
	LastUsed    time.Time              `json:"lastUsed" yaml:"lastUsed"`
	Status      string                 `json:"status" yaml:"status"`
	Metadata    map[string]interface{} `json:"metadata" yaml:"metadata"`
	Static      bool                   `json:"static" yaml:"static"`
}

// DynamicConfig represents the complete dynamic configuration
type DynamicConfig struct {
	Version       string                            `json:"version" yaml:"version"`
	UpdatedAt     time.Time                         `json:"updatedAt" yaml:"updatedAt"`
	Sources       map[string]DynamicSourceConfig    `json:"sources" yaml:"sources"`
	Tools         map[string]DynamicToolConfig      `json:"tools" yaml:"tools"`
	Policies      map[string]DynamicPolicyConfig    `json:"policies" yaml:"policies"`
	Settings      DynamicSettingsConfig             `json:"settings" yaml:"settings"`
	Metadata      map[string]interface{}            `json:"metadata" yaml:"metadata"`
}

// DynamicSourceConfig represents a dynamic source configuration
type DynamicSourceConfig struct {
	Kind        string                 `json:"kind" yaml:"kind"`
	Config      map[string]interface{} `json:"config" yaml:"config"`
	CreatedAt   time.Time              `json:"createdAt" yaml:"createdAt"`
	Tags        []string               `json:"tags" yaml:"tags"`
	Metadata    map[string]interface{} `json:"metadata" yaml:"metadata"`
}

// DynamicToolConfig represents a dynamic tool configuration
type DynamicToolConfig struct {
	Spec        DynamicToolSpec        `json:"spec" yaml:"spec"`
	CreatedAt   time.Time              `json:"createdAt" yaml:"createdAt"`
	Tags        []string               `json:"tags" yaml:"tags"`
	Metadata    map[string]interface{} `json:"metadata" yaml:"metadata"`
}

// DynamicPolicyConfig represents dynamic authorization policies
type DynamicPolicyConfig struct {
	Rules       []PolicyRule           `json:"rules" yaml:"rules"`
	CreatedAt   time.Time              `json:"createdAt" yaml:"createdAt"`
	Metadata    map[string]interface{} `json:"metadata" yaml:"metadata"`
}

// DynamicSettingsConfig represents dynamic system settings
type DynamicSettingsConfig struct {
	MaxDynamicTools    int           `json:"maxDynamicTools" yaml:"maxDynamicTools"`
	MaxDynamicSources  int           `json:"maxDynamicSources" yaml:"maxDynamicSources"`
	DefaultTimeout     time.Duration `json:"defaultTimeout" yaml:"defaultTimeout"`
	CleanupInterval    time.Duration `json:"cleanupInterval" yaml:"cleanupInterval"`
	MaxQueryComplexity int           `json:"maxQueryComplexity" yaml:"maxQueryComplexity"`
	EnableAuditLog     bool          `json:"enableAuditLog" yaml:"enableAuditLog"`
}

// PolicyRule defines authorization rules for dynamic operations
type PolicyRule struct {
	ID          string                 `json:"id" yaml:"id"`
	Effect      string                 `json:"effect" yaml:"effect"` // "allow" or "deny"
	Actions     []string               `json:"actions" yaml:"actions"`
	Resources   []string               `json:"resources" yaml:"resources"`
	Conditions  map[string]interface{} `json:"conditions" yaml:"conditions"`
	Priority    int                    `json:"priority" yaml:"priority"`
}

// HybridToolRegistry manages both static and dynamic tools
type HybridToolRegistry struct {
	mu           sync.RWMutex
	staticTools  map[string]tools.Tool
	dynamicTools map[string]DynamicTool
	manager      DynamicToolManager
	tracer       trace.Tracer
}

// DynamicTool wraps a dynamically created tool with additional metadata
type DynamicTool struct {
	Tool        tools.Tool
	Manifest    ToolManifest
	mu          sync.RWMutex
	refCount    int64
	lastAccess  time.Time
}

// NewHybridToolRegistry creates a new hybrid tool registry
func NewHybridToolRegistry(staticTools map[string]tools.Tool, manager DynamicToolManager, tracer trace.Tracer) *HybridToolRegistry {
	return &HybridToolRegistry{
		staticTools:  staticTools,
		dynamicTools: make(map[string]DynamicTool),
		manager:      manager,
		tracer:       tracer,
	}
}

// GetTool retrieves a tool by name, checking dynamic tools first, then static
func (r *HybridToolRegistry) GetTool(name string) (tools.Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	// Check dynamic tools first (allows overriding static tools)
	if dynamicTool, exists := r.dynamicTools[name]; exists {
		dynamicTool.updateAccess()
		return dynamicTool.Tool, true
	}
	
	// Fall back to static tools
	if staticTool, exists := r.staticTools[name]; exists {
		return staticTool, true
	}
	
	return nil, false
}

// ListTools returns all available tools (static + dynamic)
func (r *HybridToolRegistry) ListTools() map[string]tools.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	result := make(map[string]tools.Tool)
	
	// Add static tools
	for name, tool := range r.staticTools {
		result[name] = tool
	}
	
	// Add dynamic tools (can override static tools)
	for name, dynamicTool := range r.dynamicTools {
		result[name] = dynamicTool.Tool
	}
	
	return result
}

// AddDynamicTool adds a dynamically created tool to the registry
func (r *HybridToolRegistry) AddDynamicTool(name string, tool tools.Tool, manifest ToolManifest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	dynamicTool := DynamicTool{
		Tool:       tool,
		Manifest:   manifest,
		refCount:   1,
		lastAccess: time.Now(),
	}
	
	r.dynamicTools[name] = dynamicTool
	return nil
}

// RemoveDynamicTool removes a dynamically created tool from the registry
func (r *HybridToolRegistry) RemoveDynamicTool(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if _, exists := r.dynamicTools[name]; !exists {
		return fmt.Errorf("dynamic tool %q not found", name)
	}
	
	delete(r.dynamicTools, name)
	return nil
}

// ListDynamicTools returns all dynamic tools with their manifests
func (r *HybridToolRegistry) ListDynamicTools() map[string]ToolManifest {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	result := make(map[string]ToolManifest)
	for name, dynamicTool := range r.dynamicTools {
		result[name] = dynamicTool.Manifest
	}
	
	return result
}

// CleanupUnusedTools removes dynamic tools that haven't been accessed recently
func (r *HybridToolRegistry) CleanupUnusedTools(maxAge time.Duration) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	cutoff := time.Now().Add(-maxAge)
	removed := 0
	
	for name, dynamicTool := range r.dynamicTools {
		dynamicTool.mu.RLock()
		shouldRemove := dynamicTool.refCount == 0 && dynamicTool.lastAccess.Before(cutoff)
		dynamicTool.mu.RUnlock()
		
		if shouldRemove {
			delete(r.dynamicTools, name)
			removed++
		}
	}
	
	return removed
}

// updateAccess updates the last access time and increments reference count
func (dt *DynamicTool) updateAccess() {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	
	dt.lastAccess = time.Now()
	dt.refCount++
}

// decrementRef decrements the reference count
func (dt *DynamicTool) decrementRef() {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	
	if dt.refCount > 0 {
		dt.refCount--
	}
}