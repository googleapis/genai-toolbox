// Copyright 2025 Google LLC
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

package trinoanalyze

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/trino"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util"
)

const kind string = "trino-analyze"

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

type compatibleSource interface {
	TrinoDB() *sql.DB
}

// validate compatible sources are still compatible
var _ compatibleSource = &trino.Source{}

var compatibleSources = [...]string{trino.SourceKind}

type Config struct {
	Name         string   `yaml:"name" validate:"required"`
	Kind         string   `yaml:"kind" validate:"required"`
	Source       string   `yaml:"source" validate:"required"`
	Description  string   `yaml:"description" validate:"required"`
	AuthRequired []string `yaml:"authRequired"`
}

// validate interface
var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigKind() string {
	return kind
}

func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	// verify source exists
	rawS, ok := srcs[cfg.Source]
	if !ok {
		return nil, fmt.Errorf("no source named %q configured", cfg.Source)
	}

	// verify the source is compatible
	s, ok := rawS.(compatibleSource)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be one of %q", kind, compatibleSources)
	}

	// Define parameters for the tool
	parameters := tools.Parameters{
		tools.NewStringParameter("query", "The SQL query to analyze. This should be a SELECT, INSERT, UPDATE, or DELETE query."),
		tools.NewStringParameterWithDefault("format", "text", "The output format for the query plan. Options: 'text' (default), 'json', 'graphviz', 'summary'. 'summary' provides a simplified explanation."),
		tools.NewBooleanParameterWithDefault("analyze", false, "If true, runs ANALYZE to get actual execution statistics (may execute the query). Default is false for EXPLAIN only."),
		tools.NewBooleanParameterWithDefault("distributed", true, "If true, shows the distributed execution plan. Default is true."),
		tools.NewBooleanParameterWithDefault("validate", false, "If true, only validates the query syntax without generating a plan. Default is false."),
	}

	mcpManifest := tools.McpManifest{
		Name:        cfg.Name,
		Description: cfg.Description,
		InputSchema: parameters.McpManifest(),
	}

	// finish tool setup
	t := Tool{
		Name:         cfg.Name,
		Kind:         kind,
		Parameters:   parameters,
		AuthRequired: cfg.AuthRequired,
		Db:           s.TrinoDB(),
		manifest:     tools.Manifest{Description: cfg.Description, Parameters: parameters.Manifest(), AuthRequired: cfg.AuthRequired},
		mcpManifest:  mcpManifest,
	}
	return t, nil
}

// QueryPlan represents the analyzed query plan
type QueryPlan struct {
	Query            string                 `json:"query"`
	Plan             string                 `json:"plan,omitempty"`
	PlanJSON         map[string]interface{} `json:"planJson,omitempty"`
	Statistics       *PlanStatistics        `json:"statistics,omitempty"`
	Recommendations  []string               `json:"recommendations,omitempty"`
	Warnings         []string               `json:"warnings,omitempty"`
	EstimatedCost    *float64               `json:"estimatedCost,omitempty"`
	EstimatedRows    *int64                 `json:"estimatedRows,omitempty"`
	IsValid          bool                   `json:"isValid"`
	ValidationErrors []string               `json:"validationErrors,omitempty"`
}

// PlanStatistics contains execution statistics
type PlanStatistics struct {
	TotalCPUTime       string   `json:"totalCpuTime,omitempty"`
	TotalScheduledTime string   `json:"totalScheduledTime,omitempty"`
	TotalBlockedTime   string   `json:"totalBlockedTime,omitempty"`
	RawInputRows       *int64   `json:"rawInputRows,omitempty"`
	RawInputBytes      *int64   `json:"rawInputBytes,omitempty"`
	ProcessedRows      *int64   `json:"processedRows,omitempty"`
	ProcessedBytes     *int64   `json:"processedBytes,omitempty"`
	OutputRows         *int64   `json:"outputRows,omitempty"`
	OutputBytes        *int64   `json:"outputBytes,omitempty"`
	WrittenRows        *int64   `json:"writtenRows,omitempty"`
	WrittenBytes       *int64   `json:"writtenBytes,omitempty"`
	Stages             []string `json:"stages,omitempty"`
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Name         string           `yaml:"name"`
	Kind         string           `yaml:"kind"`
	AuthRequired []string         `yaml:"authRequired"`
	Parameters   tools.Parameters `yaml:"parameters"`

	Db          *sql.DB
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) Invoke(ctx context.Context, params tools.ParamValues, accessToken tools.AccessToken) (any, error) {
	paramsMap := params.AsMap()

	query, ok := paramsMap["query"].(string)
	if !ok || query == "" {
		return nil, fmt.Errorf("'query' parameter is required and must be a non-empty string")
	}

	format, _ := paramsMap["format"].(string)
	if format == "" {
		format = "text"
	}

	analyze, _ := paramsMap["analyze"].(bool)
	distributed, _ := paramsMap["distributed"].(bool)
	if _, ok := paramsMap["distributed"]; !ok {
		distributed = true // Default to true
	}
	validate, _ := paramsMap["validate"].(bool)

	// Log the analysis request
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting logger: %s", err)
	}
	logger.DebugContext(ctx, "analyzing query with format=%s, analyze=%v, distributed=%v, validate=%v",
		format, analyze, distributed, validate)

	// If only validating, check syntax
	if validate {
		return t.validateQuery(ctx, query)
	}

	// Build the EXPLAIN command
	explainCmd := t.buildExplainCommand(query, format, analyze, distributed)

	// Execute the EXPLAIN command
	result, err := t.executeExplain(ctx, explainCmd, query, format)
	if err != nil {
		return nil, err
	}

	// Add recommendations based on the plan
	t.addRecommendations(result)

	return result, nil
}

func (t Tool) buildExplainCommand(query, format string, analyze, distributed bool) string {
	var parts []string
	parts = append(parts, "EXPLAIN")

	if analyze {
		parts = append(parts, "(TYPE ANALYZE)")
	} else if distributed {
		parts = append(parts, "(TYPE DISTRIBUTED)")
	}

	// Handle format options
	switch format {
	case "json":
		parts = append(parts, "(FORMAT JSON)")
	case "graphviz":
		parts = append(parts, "(FORMAT GRAPHVIZ)")
	case "text", "summary":
		// Default format, no need to specify
	}

	// Combine all parts
	explainParts := []string{}
	for _, part := range parts {
		if strings.Contains(part, "(") {
			explainParts = append(explainParts, part)
		}
	}

	if len(explainParts) > 1 {
		// Combine multiple options
		options := []string{}
		for _, part := range explainParts {
			option := strings.TrimPrefix(strings.TrimSuffix(part, ")"), "(")
			options = append(options, option)
		}
		return fmt.Sprintf("EXPLAIN (%s) %s", strings.Join(options, ", "), query)
	} else if len(explainParts) == 1 {
		return fmt.Sprintf("EXPLAIN %s %s", explainParts[0], query)
	} else {
		return fmt.Sprintf("EXPLAIN %s", query)
	}
}

func (t Tool) executeExplain(ctx context.Context, explainCmd, originalQuery, format string) (*QueryPlan, error) {
	rows, err := t.Db.QueryContext(ctx, explainCmd)
	if err != nil {
		// Query might have syntax errors
		return &QueryPlan{
			Query:            originalQuery,
			IsValid:          false,
			ValidationErrors: []string{err.Error()},
		}, nil
	}
	defer rows.Close()

	result := &QueryPlan{
		Query:   originalQuery,
		IsValid: true,
	}

	// Collect the plan output
	var planLines []string
	var jsonOutput string

	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			return nil, fmt.Errorf("failed to scan explain result: %w", err)
		}

		if format == "json" {
			jsonOutput += line
		} else {
			planLines = append(planLines, line)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading explain results: %w", err)
	}

	// Process based on format
	switch format {
	case "json":
		var planJSON map[string]interface{}
		if err := json.Unmarshal([]byte(jsonOutput), &planJSON); err == nil {
			result.PlanJSON = planJSON
			// Extract statistics from JSON if available
			t.extractJSONStatistics(planJSON, result)
		} else {
			result.Plan = jsonOutput // Fallback to raw string
		}
	case "summary":
		result.Plan = t.generateSummary(planLines)
	default:
		result.Plan = strings.Join(planLines, "\n")
		// Extract basic statistics from text plan
		t.extractTextStatistics(planLines, result)
	}

	return result, nil
}

func (t Tool) validateQuery(ctx context.Context, query string) (*QueryPlan, error) {
	// Try to prepare the query to validate syntax
	// Use EXPLAIN with VALIDATE type if available, otherwise just EXPLAIN
	validateCmd := fmt.Sprintf("EXPLAIN (TYPE VALIDATE) %s", query)

	_, err := t.Db.ExecContext(ctx, validateCmd)

	result := &QueryPlan{
		Query:   query,
		IsValid: err == nil,
	}

	if err != nil {
		result.ValidationErrors = []string{err.Error()}
	}

	return result, nil
}

func (t Tool) extractJSONStatistics(planJSON map[string]interface{}, result *QueryPlan) {
	stats := &PlanStatistics{}

	// Try to extract common statistics from JSON plan
	if root, ok := planJSON["root"].(map[string]interface{}); ok {
		if estimates, ok := root["estimates"].(map[string]interface{}); ok {
			if rows, ok := estimates["outputRowCount"].(float64); ok {
				rowsInt := int64(rows)
				result.EstimatedRows = &rowsInt
			}
			if cost, ok := estimates["cpuCost"].(float64); ok {
				result.EstimatedCost = &cost
			}
		}
	}

	// Extract stage information if available
	if stages, ok := planJSON["stages"].([]interface{}); ok {
		for _, stage := range stages {
			if stageMap, ok := stage.(map[string]interface{}); ok {
				if stageId, ok := stageMap["stageId"].(string); ok {
					stats.Stages = append(stats.Stages, stageId)
				}
			}
		}
	}

	if len(stats.Stages) > 0 {
		result.Statistics = stats
	}
}

func (t Tool) extractTextStatistics(planLines []string, result *QueryPlan) {
	// Extract basic information from text plan
	for _, line := range planLines {
		line = strings.TrimSpace(line)

		// Look for cost estimates
		if strings.Contains(line, "Cost:") {
			// Extract cost value if possible
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				costStr := strings.TrimSpace(parts[1])
				// Try to parse numeric cost
				var cost float64
				if _, err := fmt.Sscanf(costStr, "%f", &cost); err == nil {
					result.EstimatedCost = &cost
				}
			}
		}

		// Look for row estimates
		if strings.Contains(line, "rows=") || strings.Contains(line, "Rows:") {
			// Extract row count
			var rows int64
			if _, err := fmt.Sscanf(line, "%*[^0-9]%d", &rows); err == nil {
				result.EstimatedRows = &rows
			}
		}
	}
}

func (t Tool) generateSummary(planLines []string) string {
	// Generate a simplified summary of the query plan
	summary := []string{"Query Plan Summary:"}

	// Extract key operations
	operations := []string{}
	for _, line := range planLines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Look for main operations (simplified)
		if strings.Contains(line, "TableScan") {
			operations = append(operations, "- Table scan detected")
		} else if strings.Contains(line, "IndexScan") {
			operations = append(operations, "- Index scan detected")
		} else if strings.Contains(line, "Join") {
			operations = append(operations, "- Join operation detected")
		} else if strings.Contains(line, "Sort") || strings.Contains(line, "Order") {
			operations = append(operations, "- Sort operation detected")
		} else if strings.Contains(line, "Group") || strings.Contains(line, "Aggregate") {
			operations = append(operations, "- Aggregation detected")
		} else if strings.Contains(line, "Filter") || strings.Contains(line, "Where") {
			operations = append(operations, "- Filter condition detected")
		}
	}

	if len(operations) > 0 {
		summary = append(summary, operations...)
	} else {
		summary = append(summary, "- Query plan generated successfully")
	}

	return strings.Join(summary, "\n")
}

func (t Tool) addRecommendations(result *QueryPlan) {
	recommendations := []string{}
	warnings := []string{}

	// Analyze the plan for potential improvements
	planText := result.Plan
	if result.PlanJSON != nil {
		// Convert JSON to string for analysis
		jsonBytes, _ := json.Marshal(result.PlanJSON)
		planText = string(jsonBytes)
	}

	// Check for common performance issues
	if strings.Contains(strings.ToLower(planText), "tablescan") && !strings.Contains(strings.ToLower(planText), "indexscan") {
		recommendations = append(recommendations, "Consider adding indexes to avoid full table scans")
	}

	if strings.Contains(strings.ToLower(planText), "cross join") {
		warnings = append(warnings, "Cross join detected - this can be very expensive for large tables")
	}

	if strings.Contains(strings.ToLower(planText), "distinct") {
		recommendations = append(recommendations, "DISTINCT operations can be expensive - ensure they are necessary")
	}

	if strings.Contains(strings.ToLower(planText), "sort") && !strings.Contains(strings.ToLower(planText), "index") {
		recommendations = append(recommendations, "Sort operation detected - consider if an index could help with ordering")
	}

	// Check for large estimated rows
	if result.EstimatedRows != nil && *result.EstimatedRows > 1000000 {
		warnings = append(warnings, fmt.Sprintf("Query estimated to process %d rows - consider adding filters or limits", *result.EstimatedRows))
	}

	// Add memory-intensive operation warnings
	if strings.Contains(strings.ToLower(planText), "hash join") {
		recommendations = append(recommendations, "Hash join detected - ensure sufficient memory is available")
	}

	if strings.Contains(strings.ToLower(planText), "broadcast") {
		recommendations = append(recommendations, "Broadcast join detected - consider if the broadcasted table is small enough")
	}

	if len(recommendations) > 0 {
		result.Recommendations = recommendations
	}
	if len(warnings) > 0 {
		result.Warnings = warnings
	}
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
