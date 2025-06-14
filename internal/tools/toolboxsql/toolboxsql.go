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

package toolboxsql

import (
	"context"
	"fmt"
	"time"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/runtime"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

const kind string = "toolbox-execute-arbitrary-sql"

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
		cfg.Description = "Execute arbitrary SQL queries against configured database sources"
	}
	
	// Parameters for the MCP tool
	parameters := tools.Parameters{
		tools.NewStringParameter("sourceId", "Database source identifier to execute the query against"),
		tools.NewStringParameter("query", "SQL query to execute"),
		tools.NewStringParameter("parameters", "Query parameters for parameterized queries (JSON object)"),
		tools.NewIntParameter("timeout", "Query timeout in seconds (max 300)"),
		tools.NewIntParameter("maxRows", "Maximum number of rows to return (max 10000)"),
		tools.NewBooleanParameter("dryRun", "Validate query syntax without executing"),
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
	sourceID, ok := paramsMap["sourceId"].(string)
	if !ok || sourceID == "" {
		return nil, fmt.Errorf("sourceId is required and must be a string")
	}
	
	query, ok := paramsMap["query"].(string)
	if !ok || query == "" {
		return nil, fmt.Errorf("query is required and must be a string")
	}
	
	// Extract optional parameters with defaults
	timeout := 30 * time.Second
	if timeoutSec, ok := paramsMap["timeout"].(int); ok {
		if timeoutSec <= 0 || timeoutSec > 300 {
			return nil, fmt.Errorf("timeout must be between 1 and 300 seconds")
		}
		timeout = time.Duration(timeoutSec) * time.Second
	}
	
	maxRows := 1000
	if mr, ok := paramsMap["maxRows"].(int); ok {
		if mr <= 0 || mr > 10000 {
			return nil, fmt.Errorf("maxRows must be between 1 and 10000")
		}
		maxRows = mr
	}
	
	dryRun := false
	if dr, ok := paramsMap["dryRun"].(bool); ok {
		dryRun = dr
	}
	
	// Extract query parameters
	queryParams := make(map[string]interface{})
	if qp, ok := paramsMap["parameters"].(map[string]interface{}); ok {
		queryParams = qp
	}
	
	// Verify source exists
	if _, exists := t.Sources[sourceID]; !exists {
		return nil, fmt.Errorf("source %q not found", sourceID)
	}
	
	// Check if manager is available
	if t.Manager == nil {
		return nil, fmt.Errorf("dynamic tool manager not available")
	}
	
	// Create arbitrary SQL request
	req := runtime.ArbitrarySQLRequest{
		SourceID:   sourceID,
		Query:      query,
		Parameters: queryParams,
		Timeout:    timeout,
		MaxRows:    maxRows,
		DryRun:     dryRun,
		Context: map[string]interface{}{
			"tool_name": t.Name,
			"tool_kind": t.Kind,
		},
	}
	
	// Execute the query
	result, err := t.Manager.ExecuteArbitrarySQL(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute arbitrary SQL: %w", err)
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