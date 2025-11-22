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

package firestorequery

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	firestoreapi "cloud.google.com/go/firestore"
	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	firestoreds "github.com/googleapis/genai-toolbox/internal/sources/firestore"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/tools/firestore/util"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
)

// Constants for tool configuration
const (
	kind         = "firestore-query"
	defaultLimit = 100
)

// Firestore operators
var validOperators = map[string]bool{
	"<":                  true,
	"<=":                 true,
	">":                  true,
	">=":                 true,
	"==":                 true,
	"!=":                 true,
	"array-contains":     true,
	"array-contains-any": true,
	"in":                 true,
	"not-in":             true,
}

// Error messages
const (
	errFilterParseFailed      = "failed to parse filters: %w"
	errQueryExecutionFailed   = "failed to execute query: %w"
	errTemplateParseFailed    = "failed to parse template: %w"
	errTemplateExecFailed     = "failed to execute template: %w"
	errLimitParseFailed       = "failed to parse limit value '%s': %w"
	errSelectFieldParseFailed = "failed to parse select field: %w"
)

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

// compatibleSource defines the interface for sources that can provide a Firestore client
type compatibleSource interface {
	FirestoreClient() *firestoreapi.Client
}

// validate compatible sources are still compatible
var _ compatibleSource = &firestoreds.Source{}

var compatibleSources = [...]string{firestoreds.SourceKind}

// Config represents the configuration for the Firestore query tool
type Config struct {
	Name         string   `yaml:"name" validate:"required"`
	Kind         string   `yaml:"kind" validate:"required"`
	Source       string   `yaml:"source" validate:"required"`
	Description  string   `yaml:"description" validate:"required"`
	AuthRequired []string `yaml:"authRequired"`

	// Template fields
	CollectionPath string         `yaml:"collectionPath" validate:"required"`
	Filters        string         `yaml:"filters"`      // JSON string template
	Select         []string       `yaml:"select"`       // Fields to select
	OrderBy        map[string]any `yaml:"orderBy"`      // Order by configuration
	Limit          string         `yaml:"limit"`        // Limit template (can be a number or template)
	AnalyzeQuery   bool           `yaml:"analyzeQuery"` // Analyze query (boolean, not parameterizable)

	// Parameters for template substitution
	Parameters parameters.Parameters `yaml:"parameters"`
}

// validate interface
var _ tools.ToolConfig = Config{}

// ToolConfigKind returns the kind of tool configuration
func (cfg Config) ToolConfigKind() string {
	return kind
}

// Initialize creates a new Tool instance from the configuration
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

	// Set default limit if not specified
	if cfg.Limit == "" {
		cfg.Limit = fmt.Sprintf("%d", defaultLimit)
	}

	// Create MCP manifest
	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, cfg.Parameters, nil)

	// finish tool setup
	t := Tool{
		Config:      cfg,
		Client:      s.FirestoreClient(),
		manifest:    tools.Manifest{Description: cfg.Description, Parameters: cfg.Parameters.Manifest(), AuthRequired: cfg.AuthRequired},
		mcpManifest: mcpManifest,
	}
	return t, nil
}

// validate interface
var _ tools.Tool = Tool{}

// Tool represents the Firestore query tool
type Tool struct {
	Config
	Client *firestoreapi.Client

	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

// SimplifiedFilter represents the simplified filter format
type SimplifiedFilter struct {
	And   []SimplifiedFilter `json:"and,omitempty"`
	Or    []SimplifiedFilter `json:"or,omitempty"`
	Field string             `json:"field,omitempty"`
	Op    string             `json:"op,omitempty"`
	Value interface{}        `json:"value,omitempty"`
}

// OrderByConfig represents ordering configuration
type OrderByConfig struct {
	Field     string `json:"field"`
	Direction string `json:"direction"`
}

// GetDirection returns the Firestore direction constant
func (o *OrderByConfig) GetDirection() firestoreapi.Direction {
	if strings.EqualFold(o.Direction, "DESCENDING") || strings.EqualFold(o.Direction, "DESC") {
		return firestoreapi.Desc
	}
	return firestoreapi.Asc
}

// QueryResult represents a document result from the query
type QueryResult struct {
	ID         string         `json:"id"`
	Path       string         `json:"path"`
	Data       map[string]any `json:"data"`
	CreateTime interface{}    `json:"createTime,omitempty"`
	UpdateTime interface{}    `json:"updateTime,omitempty"`
	ReadTime   interface{}    `json:"readTime,omitempty"`
}

// QueryResponse represents the full response including optional metrics
type QueryResponse struct {
	Documents      []QueryResult  `json:"documents"`
	ExplainMetrics map[string]any `json:"explainMetrics,omitempty"`
}

// Invoke executes the Firestore query based on the provided parameters
func (t Tool) Invoke(ctx context.Context, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	paramsMap := params.AsMap()

	// Process collection path with template substitution
	collectionPath, err := parameters.PopulateTemplate("collectionPath", t.CollectionPath, paramsMap)
	if err != nil {
		return nil, fmt.Errorf("failed to process collection path: %w", err)
	}

	// Build the query
	query, err := t.buildQuery(collectionPath, paramsMap)
	if err != nil {
		return nil, err
	}

	// Execute the query and return results
	return t.executeQuery(ctx, query)
}

// buildQuery constructs the Firestore query from parameters
func (t Tool) buildQuery(collectionPath string, params map[string]any) (*firestoreapi.Query, error) {
	collection := t.Client.Collection(collectionPath)
	query := collection.Query

	// Process and apply filters if template is provided
	if t.Filters != "" {
		// Apply template substitution to filters
		filtersJSON, err := parameters.PopulateTemplateWithJSON("filters", t.Filters, params)
		if err != nil {
			return nil, fmt.Errorf("failed to process filters template: %w", err)
		}

		// Parse the simplified filter format
		var simplifiedFilter SimplifiedFilter
		if err := json.Unmarshal([]byte(filtersJSON), &simplifiedFilter); err != nil {
			return nil, fmt.Errorf(errFilterParseFailed, err)
		}

		// Convert simplified filter to Firestore filter
		if filter := t.convertToFirestoreFilter(simplifiedFilter); filter != nil {
			query = query.WhereEntity(filter)
		}
	}

	// Process select fields
	selectFields, err := t.processSelectFields(params)
	if err != nil {
		return nil, err
	}
	if len(selectFields) > 0 {
		query = query.Select(selectFields...)
	}

	// Process and apply ordering
	orderBy, err := t.getOrderBy(params)
	if err != nil {
		return nil, err
	}
	if orderBy != nil {
		query = query.OrderBy(orderBy.Field, orderBy.GetDirection())
	}

	// Process and apply limit
	limit, err := t.getLimit(params)
	if err != nil {
		return nil, err
	}
	query = query.Limit(limit)

	// Apply analyze options if enabled
	if t.AnalyzeQuery {
		query = query.WithRunOptions(firestoreapi.ExplainOptions{
			Analyze: true,
		})
	}

	return &query, nil
}

// convertToFirestoreFilter converts simplified filter format to Firestore EntityFilter
func (t Tool) convertToFirestoreFilter(filter SimplifiedFilter) firestoreapi.EntityFilter {
	// Handle AND filters
	if len(filter.And) > 0 {
		filters := make([]firestoreapi.EntityFilter, 0, len(filter.And))
		for _, f := range filter.And {
			if converted := t.convertToFirestoreFilter(f); converted != nil {
				filters = append(filters, converted)
			}
		}
		if len(filters) > 0 {
			return firestoreapi.AndFilter{Filters: filters}
		}
		return nil
	}

	// Handle OR filters
	if len(filter.Or) > 0 {
		filters := make([]firestoreapi.EntityFilter, 0, len(filter.Or))
		for _, f := range filter.Or {
			if converted := t.convertToFirestoreFilter(f); converted != nil {
				filters = append(filters, converted)
			}
		}
		if len(filters) > 0 {
			return firestoreapi.OrFilter{Filters: filters}
		}
		return nil
	}

	// Handle simple property filter
	if filter.Field != "" && filter.Op != "" && filter.Value != nil {
		if validOperators[filter.Op] {
			// Convert the value using the Firestore native JSON converter
			convertedValue, err := util.JSONToFirestoreValue(filter.Value, t.Client)
			if err != nil {
				// If conversion fails, use the original value
				convertedValue = filter.Value
			}

			return firestoreapi.PropertyFilter{
				Path:     filter.Field,
				Operator: filter.Op,
				Value:    convertedValue,
			}
		}
	}

	return nil
}

// processSelectFields processes the select fields with parameter substitution
func (t Tool) processSelectFields(params map[string]any) ([]string, error) {
	var selectFields []string

	// Process configured select fields with template substitution
	for _, field := range t.Select {
		// Check if it's a template
		if strings.Contains(field, "{{") {
			processed, err := parameters.PopulateTemplate("selectField", field, params)
			if err != nil {
				return nil, err
			}
			if processed != "" {
				// The processed field might be an array format [a b c] or a single value
				trimmedProcessed := strings.TrimSpace(processed)

				// Check if it's in array format [a b c]
				if strings.HasPrefix(trimmedProcessed, "[") && strings.HasSuffix(trimmedProcessed, "]") {
					// Remove brackets and split by spaces
					arrayContent := strings.TrimPrefix(trimmedProcessed, "[")
					arrayContent = strings.TrimSuffix(arrayContent, "]")
					fields := strings.Fields(arrayContent) // Fields splits by any whitespace
					for _, f := range fields {
						if f != "" {
							selectFields = append(selectFields, f)
						}
					}
				} else {
					selectFields = append(selectFields, processed)
				}
			}
		} else {
			selectFields = append(selectFields, field)
		}
	}

	return selectFields, nil
}

// getOrderBy processes the orderBy configuration with parameter substitution
func (t Tool) getOrderBy(params map[string]any) (*OrderByConfig, error) {
	if t.OrderBy == nil {
		return nil, nil
	}

	orderBy := &OrderByConfig{}

	// Process field
	field, err := t.getOrderByForKey("field", params)
	if err != nil {
		return nil, err
	}
	orderBy.Field = field

	// Process direction
	direction, err := t.getOrderByForKey("direction", params)
	if err != nil {
		return nil, err
	}
	orderBy.Direction = direction

	if orderBy.Field == "" {
		return nil, nil
	}

	return orderBy, nil
}

func (t Tool) getOrderByForKey(key string, params map[string]any) (string, error) {
	value, ok := t.OrderBy[key].(string)
	if !ok {
		return "", nil
	}

	processedValue, err := parameters.PopulateTemplate(fmt.Sprintf("orderBy%s", key), value, params)
	if err != nil {
		return "", err
	}

	return processedValue, nil
}

// processLimit processes the limit field with parameter substitution
func (t Tool) getLimit(params map[string]any) (int, error) {
	limit := defaultLimit
	if t.Limit != "" {
		processedValue, err := parameters.PopulateTemplate("limit", t.Limit, params)
		if err != nil {
			return 0, err
		}

		// Try to parse as integer
		if processedValue != "" {
			parsedLimit, err := strconv.Atoi(processedValue)
			if err != nil {
				return 0, fmt.Errorf(errLimitParseFailed, processedValue, err)
			}
			limit = parsedLimit
		}
	}
	return limit, nil
}

// executeQuery runs the query and formats the results
func (t Tool) executeQuery(ctx context.Context, query *firestoreapi.Query) (any, error) {
	docIterator := query.Documents(ctx)
	docs, err := docIterator.GetAll()
	if err != nil {
		return nil, fmt.Errorf(errQueryExecutionFailed, err)
	}

	// Convert results to structured format
	results := make([]QueryResult, len(docs))
	for i, doc := range docs {
		results[i] = QueryResult{
			ID:         doc.Ref.ID,
			Path:       doc.Ref.Path,
			Data:       doc.Data(),
			CreateTime: doc.CreateTime,
			UpdateTime: doc.UpdateTime,
			ReadTime:   doc.ReadTime,
		}
	}

	// Return with explain metrics if requested
	if t.AnalyzeQuery {
		explainMetrics, err := t.getExplainMetrics(docIterator)
		if err == nil && explainMetrics != nil {
			response := QueryResponse{
				Documents:      results,
				ExplainMetrics: explainMetrics,
			}
			return response, nil
		}
	}

	return results, nil
}

// getExplainMetrics extracts explain metrics from the query iterator
func (t Tool) getExplainMetrics(docIterator *firestoreapi.DocumentIterator) (map[string]any, error) {
	explainMetrics, err := docIterator.ExplainMetrics()
	if err != nil || explainMetrics == nil {
		return nil, err
	}

	metricsData := make(map[string]any)

	// Add plan summary if available
	if explainMetrics.PlanSummary != nil {
		planSummary := make(map[string]any)
		planSummary["indexesUsed"] = explainMetrics.PlanSummary.IndexesUsed
		metricsData["planSummary"] = planSummary
	}

	// Add execution stats if available
	if explainMetrics.ExecutionStats != nil {
		executionStats := make(map[string]any)
		executionStats["resultsReturned"] = explainMetrics.ExecutionStats.ResultsReturned
		executionStats["readOperations"] = explainMetrics.ExecutionStats.ReadOperations

		if explainMetrics.ExecutionStats.ExecutionDuration != nil {
			executionStats["executionDuration"] = explainMetrics.ExecutionStats.ExecutionDuration.String()
		}

		if explainMetrics.ExecutionStats.DebugStats != nil {
			executionStats["debugStats"] = *explainMetrics.ExecutionStats.DebugStats
		}

		metricsData["executionStats"] = executionStats
	}

	return metricsData, nil
}

// ParseParams parses and validates input parameters
func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (parameters.ParamValues, error) {
	return parameters.ParseParams(t.Parameters, data, claims)
}

// Manifest returns the tool manifest
func (t Tool) Manifest() tools.Manifest {
	return t.manifest
}

// McpManifest returns the MCP manifest
func (t Tool) McpManifest() tools.McpManifest {
	return t.mcpManifest
}

// Authorized checks if the tool is authorized based on verified auth services
func (t Tool) Authorized(verifiedAuthServices []string) bool {
	return tools.IsAuthorized(t.AuthRequired, verifiedAuthServices)
}

func (t Tool) RequiresClientAuthorization() bool {
	return false
}

func (t Tool) GetAuthTokenHeaderName() string {
	return "Authorization"
}
