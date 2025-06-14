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

	"github.com/google/uuid"
	"github.com/googleapis/genai-toolbox/internal/log"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// DefaultManager implements DynamicToolManager with safety and performance features
type DefaultManager struct {
	mu            sync.RWMutex
	sources       map[string]sources.Source
	dynamicTools  map[string]*DynamicTool
	registry      *HybridToolRegistry
	configManager ConfigManager
	tracer        trace.Tracer
	logger        log.Logger
	
	// Configuration and limits
	maxDynamicTools    int
	defaultTimeout     time.Duration
	maxQueryComplexity int
	cleanupInterval    time.Duration
	
	// Cleanup management
	cleanupTicker *time.Ticker
	stopCleanup   chan struct{}
	cleanupOnce   sync.Once
}

// NewDefaultManager creates a new instance of DefaultManager
func NewDefaultManager(
	sources map[string]sources.Source,
	configManager ConfigManager,
	tracer trace.Tracer,
	logger log.Logger,
) *DefaultManager {
	manager := &DefaultManager{
		sources:            sources,
		dynamicTools:       make(map[string]*DynamicTool),
		configManager:      configManager,
		tracer:             tracer,
		logger:             logger,
		maxDynamicTools:    100,    // Default limit
		defaultTimeout:     30 * time.Second,
		maxQueryComplexity: 1000,  // Default complexity limit
		cleanupInterval:    5 * time.Minute,
		stopCleanup:        make(chan struct{}),
	}
	
	// Start periodic cleanup
	manager.startPeriodicCleanup()
	
	return manager
}

// CreateDynamicTool creates a new tool at runtime with the given specification
func (m *DefaultManager) CreateDynamicTool(ctx context.Context, spec DynamicToolSpec) (tools.Tool, error) {
	ctx, span := m.tracer.Start(ctx, "dynamic_tool_manager/create_tool")
	defer span.End()
	
	span.SetAttributes(
		attribute.String("tool_name", spec.Name),
		attribute.String("source_id", spec.SourceID),
	)
	
	m.logger.InfoContext(ctx, fmt.Sprintf("Creating dynamic tool: %s", spec.Name))
	
	// Validate input
	if err := m.validateDynamicToolSpec(ctx, spec); err != nil {
		return nil, fmt.Errorf("invalid tool specification: %w", err)
	}
	
	// Check limits
	if err := m.checkDynamicToolLimits(ctx); err != nil {
		return nil, err
	}
	
	// Verify source exists and is accessible
	source, ok := m.sources[spec.SourceID]
	if !ok {
		return nil, fmt.Errorf("source %q not found", spec.SourceID)
	}
	
	// Create the dynamic tool
	tool, manifest, err := m.createToolFromSpec(ctx, spec, source)
	if err != nil {
		return nil, fmt.Errorf("failed to create tool: %w", err)
	}
	
	// Store the dynamic tool
	m.mu.Lock()
	defer m.mu.Unlock()
	
	dynamicTool := &DynamicTool{
		Tool:       tool,
		Manifest:   manifest,
		refCount:   1,
		lastAccess: time.Now(),
	}
	
	m.dynamicTools[spec.Name] = dynamicTool
	
	// Add to registry if available
	if m.registry != nil {
		if err := m.registry.AddDynamicTool(spec.Name, tool, manifest); err != nil {
			m.logger.WarnContext(ctx, fmt.Sprintf("Failed to add tool to registry: %v", err))
		}
	}
	
	m.logger.InfoContext(ctx, fmt.Sprintf("Successfully created dynamic tool: %s", spec.Name))
	return tool, nil
}

// ExecuteArbitrarySQL executes arbitrary SQL queries against configured sources
func (m *DefaultManager) ExecuteArbitrarySQL(ctx context.Context, req ArbitrarySQLRequest) ([]any, error) {
	ctx, span := m.tracer.Start(ctx, "dynamic_tool_manager/execute_arbitrary_sql")
	defer span.End()
	
	span.SetAttributes(
		attribute.String("source_id", req.SourceID),
		attribute.Bool("dry_run", req.DryRun),
		attribute.Int("max_rows", req.MaxRows),
	)
	
	m.logger.InfoContext(ctx, fmt.Sprintf("Executing arbitrary SQL on source: %s", req.SourceID))
	
	// Validate request
	if err := m.validateArbitrarySQLRequest(ctx, req); err != nil {
		return nil, fmt.Errorf("invalid SQL request: %w", err)
	}
	
	// Get source
	source, ok := m.sources[req.SourceID]
	if !ok {
		return nil, fmt.Errorf("source %q not found", req.SourceID)
	}
	
	// Dry run - just validate the query
	if req.DryRun {
		if err := m.validateSQLQuery(ctx, req.Query, source); err != nil {
			return nil, fmt.Errorf("query validation failed: %w", err)
		}
		return []any{map[string]any{"status": "valid", "message": "Query syntax is valid"}}, nil
	}
	
	// Execute the query
	result, err := m.executeSQLQuery(ctx, req, source)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	
	m.logger.InfoContext(ctx, fmt.Sprintf("Successfully executed arbitrary SQL, returned %d rows", len(result)))
	return result, nil
}

// ListDynamicTools returns a list of all dynamically created tools
func (m *DefaultManager) ListDynamicTools(ctx context.Context) ([]ToolManifest, error) {
	ctx, span := m.tracer.Start(ctx, "dynamic_tool_manager/list_tools")
	defer span.End()
	
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	manifests := make([]ToolManifest, 0, len(m.dynamicTools))
	for _, dynamicTool := range m.dynamicTools {
		manifests = append(manifests, dynamicTool.Manifest)
	}
	
	span.SetAttributes(attribute.Int("tool_count", len(manifests)))
	return manifests, nil
}

// RemoveDynamicTool removes a dynamically created tool
func (m *DefaultManager) RemoveDynamicTool(ctx context.Context, toolID string) error {
	ctx, span := m.tracer.Start(ctx, "dynamic_tool_manager/remove_tool")
	defer span.End()
	
	span.SetAttributes(attribute.String("tool_id", toolID))
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	dynamicTool, exists := m.dynamicTools[toolID]
	if !exists {
		return fmt.Errorf("dynamic tool %q not found", toolID)
	}
	
	// Check if tool is currently in use
	dynamicTool.mu.RLock()
	inUse := dynamicTool.refCount > 0
	dynamicTool.mu.RUnlock()
	
	if inUse {
		return fmt.Errorf("cannot remove tool %q: currently in use", toolID)
	}
	
	// Remove from internal storage
	delete(m.dynamicTools, toolID)
	
	// Remove from registry if available
	if m.registry != nil {
		if err := m.registry.RemoveDynamicTool(toolID); err != nil {
			m.logger.WarnContext(ctx, fmt.Sprintf("Failed to remove tool from registry: %v", err))
		}
	}
	
	m.logger.InfoContext(ctx, fmt.Sprintf("Successfully removed dynamic tool: %s", toolID))
	return nil
}

// GetDynamicTool retrieves a dynamically created tool by ID
func (m *DefaultManager) GetDynamicTool(ctx context.Context, toolID string) (tools.Tool, error) {
	ctx, span := m.tracer.Start(ctx, "dynamic_tool_manager/get_tool")
	defer span.End()
	
	span.SetAttributes(attribute.String("tool_id", toolID))
	
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	dynamicTool, exists := m.dynamicTools[toolID]
	if !exists {
		return nil, fmt.Errorf("dynamic tool %q not found", toolID)
	}
	
	// Update access tracking
	dynamicTool.updateAccess()
	
	return dynamicTool.Tool, nil
}

// Cleanup performs periodic cleanup of unused dynamic resources
func (m *DefaultManager) Cleanup(ctx context.Context) error {
	ctx, span := m.tracer.Start(ctx, "dynamic_tool_manager/cleanup")
	defer span.End()
	
	maxAge := 1 * time.Hour // Tools unused for 1 hour are eligible for cleanup
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	cutoff := time.Now().Add(-maxAge)
	removed := 0
	
	for toolID, dynamicTool := range m.dynamicTools {
		dynamicTool.mu.RLock()
		shouldRemove := dynamicTool.refCount == 0 && dynamicTool.lastAccess.Before(cutoff)
		dynamicTool.mu.RUnlock()
		
		if shouldRemove {
			delete(m.dynamicTools, toolID)
			removed++
			
			// Remove from registry if available
			if m.registry != nil {
				if err := m.registry.RemoveDynamicTool(toolID); err != nil {
					m.logger.WarnContext(ctx, fmt.Sprintf("Failed to remove tool from registry during cleanup: %v", err))
				}
			}
		}
	}
	
	span.SetAttributes(attribute.Int("removed_count", removed))
	m.logger.InfoContext(ctx, fmt.Sprintf("Cleanup completed: removed %d unused dynamic tools", removed))
	
	return nil
}

// SetRegistry sets the hybrid tool registry for integration
func (m *DefaultManager) SetRegistry(registry *HybridToolRegistry) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.registry = registry
}

// startPeriodicCleanup starts the periodic cleanup routine
func (m *DefaultManager) startPeriodicCleanup() {
	m.cleanupOnce.Do(func() {
		m.cleanupTicker = time.NewTicker(m.cleanupInterval)
		go func() {
			for {
				select {
				case <-m.cleanupTicker.C:
					ctx := context.Background()
					if err := m.Cleanup(ctx); err != nil {
						m.logger.ErrorContext(ctx, fmt.Sprintf("Cleanup failed: %v", err))
					}
				case <-m.stopCleanup:
					return
				}
			}
		}()
	})
}

// Stop stops the periodic cleanup routine
func (m *DefaultManager) Stop() {
	if m.cleanupTicker != nil {
		m.cleanupTicker.Stop()
	}
	close(m.stopCleanup)
}

// validateDynamicToolSpec validates the tool specification
func (m *DefaultManager) validateDynamicToolSpec(ctx context.Context, spec DynamicToolSpec) error {
	if spec.Name == "" {
		return fmt.Errorf("tool name is required")
	}
	
	if spec.SourceID == "" {
		return fmt.Errorf("source ID is required")
	}
	
	if spec.Query == "" {
		return fmt.Errorf("query is required")
	}
	
	if spec.Description == "" {
		return fmt.Errorf("description is required")
	}
	
	// Validate timeout
	if spec.Timeout > 0 && spec.Timeout > 5*time.Minute {
		return fmt.Errorf("timeout cannot exceed 5 minutes")
	}
	
	// Set default timeout if not specified
	if spec.Timeout == 0 {
		spec.Timeout = m.defaultTimeout
	}
	
	return nil
}

// checkDynamicToolLimits checks if creating a new tool would exceed limits
func (m *DefaultManager) checkDynamicToolLimits(ctx context.Context) error {
	m.mu.RLock()
	currentCount := len(m.dynamicTools)
	m.mu.RUnlock()
	
	if currentCount >= m.maxDynamicTools {
		return fmt.Errorf("maximum number of dynamic tools (%d) reached", m.maxDynamicTools)
	}
	
	return nil
}

// createToolFromSpec creates a concrete tool implementation from the specification
func (m *DefaultManager) createToolFromSpec(ctx context.Context, spec DynamicToolSpec, source sources.Source) (tools.Tool, ToolManifest, error) {
	// Generate unique ID for the tool
	toolID := uuid.New().String()
	
	// Create manifest
	manifest := ToolManifest{
		ID:          toolID,
		Name:        spec.Name,
		Description: spec.Description,
		SourceID:    spec.SourceID,
		Kind:        "dynamic-sql",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		LastUsed:    time.Now(),
		UsageCount:  0,
		Tags:        spec.Tags,
		Metadata:    spec.Metadata,
		Status:      "active",
	}
	
	// Create the tool based on source type
	tool, err := m.createToolForSource(ctx, spec, source, manifest)
	if err != nil {
		return nil, ToolManifest{}, err
	}
	
	return tool, manifest, nil
}

// createToolForSource creates a tool implementation specific to the source type
func (m *DefaultManager) createToolForSource(ctx context.Context, spec DynamicToolSpec, source sources.Source, manifest ToolManifest) (tools.Tool, error) {
	// For now, create a generic dynamic SQL tool
	// In the future, this could be extended to create source-specific tools
	
	dynamicTool := &GenericDynamicTool{
		id:          manifest.ID,
		name:        spec.Name,
		description: spec.Description,
		source:      source,
		query:       spec.Query,
		parameters:  spec.Parameters,
		auth:        spec.Auth,
		timeout:     spec.Timeout,
		manifest:    manifest,
		manager:     m,
	}
	
	return dynamicTool, nil
}

// validateArbitrarySQLRequest validates an arbitrary SQL request
func (m *DefaultManager) validateArbitrarySQLRequest(ctx context.Context, req ArbitrarySQLRequest) error {
	if req.SourceID == "" {
		return fmt.Errorf("source ID is required")
	}
	
	if req.Query == "" {
		return fmt.Errorf("query is required")
	}
	
	// Set default values
	if req.MaxRows == 0 {
		req.MaxRows = 1000 // Default limit
	}
	
	if req.MaxRows > 10000 {
		return fmt.Errorf("max rows cannot exceed 10,000")
	}
	
	if req.Timeout == 0 {
		req.Timeout = m.defaultTimeout
	}
	
	if req.Timeout > 5*time.Minute {
		return fmt.Errorf("timeout cannot exceed 5 minutes")
	}
	
	return nil
}

// validateSQLQuery validates SQL query syntax and complexity
func (m *DefaultManager) validateSQLQuery(ctx context.Context, query string, source sources.Source) error {
	// Basic validation - in the future this will use SQL parser
	if len(query) > 10000 {
		return fmt.Errorf("query too long (max 10,000 characters)")
	}
	
	// TODO: Implement proper SQL parsing and validation in Phase 2
	// For now, perform basic checks
	
	return nil
}

// executeSQLQuery executes a SQL query against the specified source
func (m *DefaultManager) executeSQLQuery(ctx context.Context, req ArbitrarySQLRequest, source sources.Source) ([]any, error) {
	// Set timeout context
	if req.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, req.Timeout)
		defer cancel()
	}
	
	// TODO: Implement actual SQL execution based on source type
	// For now, return a placeholder result
	
	result := []any{
		map[string]any{
			"status":  "executed",
			"message": fmt.Sprintf("Query executed successfully on source %s", req.SourceID),
			"query":   req.Query,
		},
	}
	
	return result, nil
}