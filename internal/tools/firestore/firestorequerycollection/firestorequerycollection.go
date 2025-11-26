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

package firestorequerycollection

import (
	"context"
	"encoding/json"
	"fmt"
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
	kind            = "firestore-query-collection"
	defaultLimit    = 100
	defaultAnalyze  = false
	maxFilterLength = 100 // Maximum filters to prevent abuse
)

// Parameter keys
const (
	collectionPathKey = "collectionPath"
	filtersKey        = "filters"
	orderByKey        = "orderBy"
	limitKey          = "limit"
	analyzeQueryKey   = "analyzeQuery"
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
	errMissingCollectionPath = "invalid or missing '%s' parameter"
	errInvalidFilters        = "invalid '%s' parameter; expected an array"
	errFilterNotString       = "filter at index %d is not a string"
	errFilterParseFailed     = "failed to parse filter at index %d: %w"
	errInvalidOperator       = "unsupported operator: %s. Valid operators are: %v"
	errMissingFilterValue    = "no value specified for filter on field '%s'"
	errOrderByParseFailed    = "failed to parse orderBy: %w"
	errQueryExecutionFailed  = "failed to execute query: %w"
	errTooManyFilters        = "too many filters provided: %d (maximum: %d)"
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

// Config represents the configuration for the Firestore query collection tool
type Config struct {
	Name         string   `yaml:"name" validate:"required"`
	Kind         string   `yaml:"kind" validate:"required"`
	Source       string   `yaml:"source" validate:"required"`
	Description  string   `yaml:"description" validate:"required"`
	AuthRequired []string `yaml:"authRequired"`
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

	// Create parameters
	params := createParameters()

	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, params, nil)

	// finish tool setup
	t := Tool{
		Config:      cfg,
		Parameters:  params,
		Client:      s.FirestoreClient(),
		manifest:    tools.Manifest{Description: cfg.Description, Parameters: params.Manifest(), AuthRequired: cfg.AuthRequired},
		mcpManifest: mcpManifest,
	}
	return t, nil
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

// createParameters creates the parameter definitions for the tool
func createParameters() parameters.Parameters {
	collectionPathParameter := parameters.NewStringParameter(
		collectionPathKey,
		"The relative path to the Firestore collection to query (e.g., 'users' or 'users/userId/posts'). Note: This is a relative path, NOT an absolute path like 'projects/{project_id}/databases/{database_id}/documents/...'",
	)

	filtersDescription := `Array of filter objects to apply to the query. Each filter is a JSON string with:
- field: The field name to filter on
- op: The operator to use ("<", "<=", ">", ">=", "==", "!=", "array-contains", "array-contains-any", "in", "not-in")
- value: The value to compare against (can be string, number, boolean, or array)
Example: {"field": "age", "op": ">", "value": 18}`

	filtersParameter := parameters.NewArrayParameter(
		filtersKey,
		filtersDescription,
		parameters.NewStringParameter("item", "JSON string representation of a filter object"),
	)

	orderByParameter := parameters.NewStringParameter(
		orderByKey,
		"JSON string specifying the field and direction to order by (e.g., {\"field\": \"name\", \"direction\": \"ASCENDING\"}). Leave empty if not specified",
	)

	limitParameter := parameters.NewIntParameterWithDefault(
		limitKey,
		defaultLimit,
		"The maximum number of documents to return",
	)

	analyzeQueryParameter := parameters.NewBooleanParameterWithDefault(
		analyzeQueryKey,
		defaultAnalyze,
		"If true, returns query explain metrics including execution statistics",
	)

	return parameters.Parameters{
		collectionPathParameter,
		filtersParameter,
		orderByParameter,
		limitParameter,
		analyzeQueryParameter,
	}
}

// validate interface
var _ tools.Tool = Tool{}

// Tool represents the Firestore query collection tool
type Tool struct {
	Config
	Parameters parameters.Parameters `yaml:"parameters"`

	Client      *firestoreapi.Client
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

// FilterConfig represents a filter for the query
type FilterConfig struct {
	Field string      `json:"field"`
	Op    string      `json:"op"`
	Value interface{} `json:"value"`
}

// Validate checks if the filter configuration is valid
func (f *FilterConfig) Validate() error {
	if f.Field == "" {
		return fmt.Errorf("filter field cannot be empty")
	}

	if !validOperators[f.Op] {
		ops := make([]string, 0, len(validOperators))
		for op := range validOperators {
			ops = append(ops, op)
		}
		return fmt.Errorf(errInvalidOperator, f.Op, ops)
	}

	if f.Value == nil {
		return fmt.Errorf(errMissingFilterValue, f.Field)
	}

	return nil
}

// OrderByConfig represents ordering configuration
type OrderByConfig struct {
	Field     string `json:"field"`
	Direction string `json:"direction"`
}

// GetDirection returns the Firestore direction constant
func (o *OrderByConfig) GetDirection() firestoreapi.Direction {
	if strings.EqualFold(o.Direction, "DESCENDING") {
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
	// Parse parameters
	queryParams, err := t.parseQueryParameters(params)
	if err != nil {
		return nil, err
	}

	// Build the query
	query, err := t.buildQuery(queryParams)
	if err != nil {
		return nil, err
	}

	// Execute the query and return results
	return t.executeQuery(ctx, query, queryParams.AnalyzeQuery)
}

// queryParameters holds all parsed query parameters
type queryParameters struct {
	CollectionPath string
	Filters        []FilterConfig
	OrderBy        *OrderByConfig
	Limit          int
	AnalyzeQuery   bool
}

// parseQueryParameters extracts and validates parameters from the input
func (t Tool) parseQueryParameters(params parameters.ParamValues) (*queryParameters, error) {
	mapParams := params.AsMap()

	// Get collection path
	collectionPath, ok := mapParams[collectionPathKey].(string)
	if !ok || collectionPath == "" {
		return nil, fmt.Errorf(errMissingCollectionPath, collectionPathKey)
	}

	// Validate collection path
	if err := util.ValidateCollectionPath(collectionPath); err != nil {
		return nil, fmt.Errorf("invalid collection path: %w", err)
	}

	result := &queryParameters{
		CollectionPath: collectionPath,
		Limit:          defaultLimit,
		AnalyzeQuery:   defaultAnalyze,
	}

	// Parse filters
	if filtersRaw, ok := mapParams[filtersKey]; ok && filtersRaw != nil {
		filters, err := t.parseFilters(filtersRaw)
		if err != nil {
			return nil, err
		}
		result.Filters = filters
	}

	// Parse orderBy
	if orderByRaw, ok := mapParams[orderByKey]; ok && orderByRaw != nil {
		orderBy, err := t.parseOrderBy(orderByRaw)
		if err != nil {
			return nil, err
		}
		result.OrderBy = orderBy
	}

	// Parse limit
	if limit, ok := mapParams[limitKey].(int); ok {
		result.Limit = limit
	}

	// Parse analyze
	if analyze, ok := mapParams[analyzeQueryKey].(bool); ok {
		result.AnalyzeQuery = analyze
	}

	return result, nil
}

// parseFilters parses and validates filter configurations
func (t Tool) parseFilters(filtersRaw interface{}) ([]FilterConfig, error) {
	filters, ok := filtersRaw.([]any)
	if !ok {
		return nil, fmt.Errorf(errInvalidFilters, filtersKey)
	}

	if len(filters) > maxFilterLength {
		return nil, fmt.Errorf(errTooManyFilters, len(filters), maxFilterLength)
	}

	result := make([]FilterConfig, 0, len(filters))
	for i, filterRaw := range filters {
		filterJSON, ok := filterRaw.(string)
		if !ok {
			return nil, fmt.Errorf(errFilterNotString, i)
		}

		var filter FilterConfig
		if err := json.Unmarshal([]byte(filterJSON), &filter); err != nil {
			return nil, fmt.Errorf(errFilterParseFailed, i, err)
		}

		if err := filter.Validate(); err != nil {
			return nil, fmt.Errorf("filter at index %d is invalid: %w", i, err)
		}

		result = append(result, filter)
	}

	return result, nil
}

// parseOrderBy parses the orderBy configuration
func (t Tool) parseOrderBy(orderByRaw interface{}) (*OrderByConfig, error) {
	orderByJSON, ok := orderByRaw.(string)
	if !ok || orderByJSON == "" {
		return nil, nil
	}

	var orderBy OrderByConfig
	if err := json.Unmarshal([]byte(orderByJSON), &orderBy); err != nil {
		return nil, fmt.Errorf(errOrderByParseFailed, err)
	}

	if orderBy.Field == "" {
		return nil, nil
	}

	return &orderBy, nil
}

// buildQuery constructs the Firestore query from parameters
func (t Tool) buildQuery(params *queryParameters) (*firestoreapi.Query, error) {
	collection := t.Client.Collection(params.CollectionPath)
	query := collection.Query

	// Apply filters
	if len(params.Filters) > 0 {
		filterConditions := make([]firestoreapi.EntityFilter, 0, len(params.Filters))
		for _, filter := range params.Filters {
			filterConditions = append(filterConditions, firestoreapi.PropertyFilter{
				Path:     filter.Field,
				Operator: filter.Op,
				Value:    filter.Value,
			})
		}

		query = query.WhereEntity(firestoreapi.AndFilter{
			Filters: filterConditions,
		})
	}

	// Apply ordering
	if params.OrderBy != nil {
		query = query.OrderBy(params.OrderBy.Field, params.OrderBy.GetDirection())
	}

	// Apply limit
	query = query.Limit(params.Limit)

	// Apply analyze options
	if params.AnalyzeQuery {
		query = query.WithRunOptions(firestoreapi.ExplainOptions{
			Analyze: true,
		})
	}

	return &query, nil
}

// executeQuery runs the query and formats the results
func (t Tool) executeQuery(ctx context.Context, query *firestoreapi.Query, analyzeQuery bool) (any, error) {
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
	if analyzeQuery {
		explainMetrics, err := t.getExplainMetrics(docIterator)
		if err == nil && explainMetrics != nil {
			response := QueryResponse{
				Documents:      results,
				ExplainMetrics: explainMetrics,
			}
			return response, nil
		}
	}

	// Return just the documents
	resultsAny := make([]any, len(results))
	for i, r := range results {
		resultsAny[i] = r
	}
	return resultsAny, nil
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
