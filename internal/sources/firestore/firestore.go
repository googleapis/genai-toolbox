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

package firestore

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/util"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/api/firebaserules/v1"
	"google.golang.org/api/option"
	"google.golang.org/genproto/googleapis/type/latlng"
)

const SourceType string = "firestore"

// validate interface
var _ sources.SourceConfig = Config{}

func init() {
	if !sources.Register(SourceType, newConfig) {
		panic(fmt.Sprintf("source type %q already registered", SourceType))
	}
}

func newConfig(ctx context.Context, name string, decoder *yaml.Decoder) (sources.SourceConfig, error) {
	actual := Config{Name: name}
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return nil, err
	}
	return actual, nil
}

type Config struct {
	// Firestore configs
	Name     string `yaml:"name" validate:"required"`
	Type     string `yaml:"type" validate:"required"`
	Project  string `yaml:"project" validate:"required"`
	Database string `yaml:"database"` // Optional, defaults to "(default)"
}

func (r Config) SourceConfigType() string {
	// Returns Firestore source type
	return SourceType
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	// Initializes a Firestore source
	client, err := initFirestoreConnection(ctx, tracer, r.Name, r.Project, r.Database)
	if err != nil {
		return nil, err
	}

	// Initialize Firebase Rules client
	rulesClient, err := initFirebaseRulesConnection(ctx, r.Project)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Firebase Rules client: %w", err)
	}

	s := &Source{
		Config:      r,
		Client:      client,
		RulesClient: rulesClient,
	}
	return s, nil
}

var _ sources.Source = &Source{}

type Source struct {
	Config
	Client      *firestore.Client
	RulesClient *firebaserules.Service
}

func (s *Source) SourceType() string {
	// Returns Firestore source type
	return SourceType
}

func (s *Source) ToConfig() sources.SourceConfig {
	return s.Config
}

func (s *Source) FirestoreClient() *firestore.Client {
	return s.Client
}

func (s *Source) FirebaseRulesClient() *firebaserules.Service {
	return s.RulesClient
}

func (s *Source) GetProjectId() string {
	return s.Project
}

func (s *Source) GetDatabaseId() string {
	return s.Database
}

// FirestoreValueToJSON converts a Firestore value to a simplified JSON representation
// This removes type information and returns plain values
func FirestoreValueToJSON(value any) any {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case time.Time:
		return v.Format(time.RFC3339Nano)
	case *latlng.LatLng:
		return map[string]any{
			"latitude":  v.Latitude,
			"longitude": v.Longitude,
		}
	case []byte:
		return base64.StdEncoding.EncodeToString(v)
	case []any:
		result := make([]any, len(v))
		for i, item := range v {
			result[i] = FirestoreValueToJSON(item)
		}
		return result
	case map[string]any:
		result := make(map[string]any)
		for k, val := range v {
			result[k] = FirestoreValueToJSON(val)
		}
		return result
	case *firestore.DocumentRef:
		return v.Path
	default:
		return value
	}
}

// BuildQuery constructs the Firestore query from parameters
func (s *Source) BuildQuery(collectionPath string, filter firestore.EntityFilter, selectFields []string, field string, direction firestore.Direction, limit int, analyzeQuery bool) (*firestore.Query, error) {
	collection := s.FirestoreClient().Collection(collectionPath)
	query := collection.Query

	// Process and apply filters if template is provided
	if filter != nil {
		query = query.WhereEntity(filter)
	}
	if len(selectFields) > 0 {
		query = query.Select(selectFields...)
	}
	if field != "" {
		query = query.OrderBy(field, direction)
	}
	query = query.Limit(limit)

	// Apply analyze options if enabled
	if analyzeQuery {
		query = query.WithRunOptions(firestore.ExplainOptions{
			Analyze: true,
		})
	}

	return &query, nil
}

// QueryResult represents a document result from the query
type QueryResult struct {
	ID         string         `json:"id"`
	Path       string         `json:"path"`
	Data       map[string]any `json:"data"`
	CreateTime any            `json:"createTime,omitempty"`
	UpdateTime any            `json:"updateTime,omitempty"`
	ReadTime   any            `json:"readTime,omitempty"`
}

// QueryResponse represents the full response including optional metrics
type QueryResponse struct {
	Documents      []QueryResult  `json:"documents"`
	ExplainMetrics map[string]any `json:"explainMetrics,omitempty"`
}

// ExecuteQuery runs the query and formats the results
func (s *Source) ExecuteQuery(ctx context.Context, query *firestore.Query, analyzeQuery bool) (any, error) {
	docIterator := query.Documents(ctx)
	docs, err := docIterator.GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
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
		explainMetrics, err := getExplainMetrics(docIterator)
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
func getExplainMetrics(docIterator *firestore.DocumentIterator) (map[string]any, error) {
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

func (s *Source) GetDocuments(ctx context.Context, documentPaths []string) ([]any, error) {
	// Create document references from paths
	docRefs := make([]*firestore.DocumentRef, len(documentPaths))
	for i, path := range documentPaths {
		docRefs[i] = s.FirestoreClient().Doc(path)
	}

	// Get all documents
	snapshots, err := s.FirestoreClient().GetAll(ctx, docRefs)
	if err != nil {
		return nil, fmt.Errorf("failed to get documents: %w", err)
	}

	// Convert snapshots to response data
	results := make([]any, len(snapshots))
	for i, snapshot := range snapshots {
		docData := make(map[string]any)
		docData["path"] = documentPaths[i]
		docData["exists"] = snapshot.Exists()

		if snapshot.Exists() {
			docData["data"] = snapshot.Data()
			docData["createTime"] = snapshot.CreateTime
			docData["updateTime"] = snapshot.UpdateTime
			docData["readTime"] = snapshot.ReadTime
		}

		results[i] = docData
	}

	return results, nil
}

func (s *Source) AddDocuments(ctx context.Context, collectionPath string, documentData any, returnData bool) (map[string]any, error) {
	// Get the collection reference
	collection := s.FirestoreClient().Collection(collectionPath)

	// Add the document to the collection
	docRef, writeResult, err := collection.Add(ctx, documentData)
	if err != nil {
		return nil, fmt.Errorf("failed to add document: %w", err)
	}
	// Build the response
	response := map[string]any{
		"documentPath": docRef.Path,
		"createTime":   writeResult.UpdateTime.Format("2006-01-02T15:04:05.999999999Z"),
	}
	// Add document data if requested
	if returnData {
		// Fetch the updated document to return the current state
		snapshot, err := docRef.Get(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve updated document: %w", err)
		}
		// Convert the document data back to simple JSON format
		simplifiedData := FirestoreValueToJSON(snapshot.Data())
		response["documentData"] = simplifiedData
	}
	return response, nil
}

func (s *Source) UpdateDocument(ctx context.Context, documentPath string, updates []firestore.Update, documentData any, returnData bool) (map[string]any, error) {
	// Get the document reference
	docRef := s.FirestoreClient().Doc(documentPath)

	// Prepare update data
	var writeResult *firestore.WriteResult
	var writeErr error

	if len(updates) > 0 {
		writeResult, writeErr = docRef.Update(ctx, updates)
	} else {
		writeResult, writeErr = docRef.Set(ctx, documentData, firestore.MergeAll)
	}

	if writeErr != nil {
		return nil, fmt.Errorf("failed to update document: %w", writeErr)
	}

	// Build the response
	response := map[string]any{
		"documentPath": docRef.Path,
		"updateTime":   writeResult.UpdateTime.Format("2006-01-02T15:04:05.999999999Z"),
	}

	// Add document data if requested
	if returnData {
		// Fetch the updated document to return the current state
		snapshot, err := docRef.Get(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve updated document: %w", err)
		}
		// Convert the document data to simple JSON format
		simplifiedData := FirestoreValueToJSON(snapshot.Data())
		response["documentData"] = simplifiedData
	}

	return response, nil
}

func (s *Source) DeleteDocuments(ctx context.Context, documentPaths []string) ([]any, error) {
	// Create a BulkWriter to handle multiple deletions efficiently
	bulkWriter := s.FirestoreClient().BulkWriter(ctx)

	// Keep track of jobs for each document
	jobs := make([]*firestore.BulkWriterJob, len(documentPaths))

	// Add all delete operations to the BulkWriter
	for i, path := range documentPaths {
		docRef := s.FirestoreClient().Doc(path)
		job, err := bulkWriter.Delete(docRef)
		if err != nil {
			return nil, fmt.Errorf("failed to add delete operation for document %q: %w", path, err)
		}
		jobs[i] = job
	}

	// End the BulkWriter to execute all operations
	bulkWriter.End()

	// Collect results
	results := make([]any, len(documentPaths))
	for i, job := range jobs {
		docData := make(map[string]any)
		docData["path"] = documentPaths[i]

		// Wait for the job to complete and get the result
		_, err := job.Results()
		if err != nil {
			docData["success"] = false
			docData["error"] = err.Error()
		} else {
			docData["success"] = true
		}

		results[i] = docData
	}
	return results, nil
}

func (s *Source) ListCollections(ctx context.Context, parentPath string) ([]any, error) {
	var collectionRefs []*firestore.CollectionRef
	var err error
	if parentPath != "" {
		// List subcollections of the specified document
		docRef := s.FirestoreClient().Doc(parentPath)
		collectionRefs, err = docRef.Collections(ctx).GetAll()
		if err != nil {
			return nil, fmt.Errorf("failed to list subcollections of document %q: %w", parentPath, err)
		}
	} else {
		// List root collections
		collectionRefs, err = s.FirestoreClient().Collections(ctx).GetAll()
		if err != nil {
			return nil, fmt.Errorf("failed to list root collections: %w", err)
		}
	}

	// Convert collection references to response data
	results := make([]any, len(collectionRefs))
	for i, collRef := range collectionRefs {
		collData := make(map[string]any)
		collData["id"] = collRef.ID
		collData["path"] = collRef.Path

		// If this is a subcollection, include parent information
		if collRef.Parent != nil {
			collData["parent"] = collRef.Parent.Path
		}
		results[i] = collData
	}
	return results, nil
}

func (s *Source) GetRules(ctx context.Context) (any, error) {
	// Get the latest release for Firestore
	releaseName := fmt.Sprintf("projects/%s/releases/cloud.firestore/%s", s.GetProjectId(), s.GetDatabaseId())
	release, err := s.FirebaseRulesClient().Projects.Releases.Get(releaseName).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get latest Firestore release: %w", err)
	}

	if release.RulesetName == "" {
		return nil, fmt.Errorf("no active Firestore rules were found in project '%s' and database '%s'", s.GetProjectId(), s.GetDatabaseId())
	}

	// Get the ruleset content
	ruleset, err := s.FirebaseRulesClient().Projects.Rulesets.Get(release.RulesetName).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get ruleset content: %w", err)
	}

	if ruleset.Source == nil || len(ruleset.Source.Files) == 0 {
		return nil, fmt.Errorf("no rules files found in ruleset")
	}

	return ruleset, nil
}

// SourcePosition represents the location of an issue in the source
type SourcePosition struct {
	FileName      string `json:"fileName,omitempty"`
	Line          int64  `json:"line"`          // 1-based
	Column        int64  `json:"column"`        // 1-based
	CurrentOffset int64  `json:"currentOffset"` // 0-based, inclusive start
	EndOffset     int64  `json:"endOffset"`     // 0-based, exclusive end
}

// Issue represents a validation issue in the rules
type Issue struct {
	SourcePosition SourcePosition `json:"sourcePosition"`
	Description    string         `json:"description"`
	Severity       string         `json:"severity"`
}

// ValidationResult represents the result of rules validation
type ValidationResult struct {
	Valid           bool    `json:"valid"`
	IssueCount      int     `json:"issueCount"`
	FormattedIssues string  `json:"formattedIssues,omitempty"`
	RawIssues       []Issue `json:"rawIssues,omitempty"`
}

func (s *Source) ValidateRules(ctx context.Context, sourceParam string) (any, error) {
	// Create test request
	testRequest := &firebaserules.TestRulesetRequest{
		Source: &firebaserules.Source{
			Files: []*firebaserules.File{
				{
					Name:    "firestore.rules",
					Content: sourceParam,
				},
			},
		},
		// We don't need test cases for validation only
		TestSuite: &firebaserules.TestSuite{
			TestCases: []*firebaserules.TestCase{},
		},
	}
	// Call the test API
	projectName := fmt.Sprintf("projects/%s", s.GetProjectId())
	response, err := s.FirebaseRulesClient().Projects.Test(projectName, testRequest).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to validate rules: %w", err)
	}

	// Process the response
	if len(response.Issues) == 0 {
		return ValidationResult{
			Valid:           true,
			IssueCount:      0,
			FormattedIssues: "✓ No errors detected. Rules are valid.",
		}, nil
	}

	// Convert issues to our format
	issues := make([]Issue, len(response.Issues))
	for i, issue := range response.Issues {
		issues[i] = Issue{
			Description: issue.Description,
			Severity:    issue.Severity,
			SourcePosition: SourcePosition{
				FileName:      issue.SourcePosition.FileName,
				Line:          issue.SourcePosition.Line,
				Column:        issue.SourcePosition.Column,
				CurrentOffset: issue.SourcePosition.CurrentOffset,
				EndOffset:     issue.SourcePosition.EndOffset,
			},
		}
	}

	// Format issues
	sourceLines := strings.Split(sourceParam, "\n")
	var formattedOutput []string

	formattedOutput = append(formattedOutput, fmt.Sprintf("Found %d issue(s) in rules source:\n", len(issues)))

	for _, issue := range issues {
		issueString := fmt.Sprintf("%s: %s [Ln %d, Col %d]",
			issue.Severity,
			issue.Description,
			issue.SourcePosition.Line,
			issue.SourcePosition.Column)

		if issue.SourcePosition.Line > 0 {
			lineIndex := int(issue.SourcePosition.Line - 1) // 0-based index
			if lineIndex >= 0 && lineIndex < len(sourceLines) {
				errorLine := sourceLines[lineIndex]
				issueString += fmt.Sprintf("\n```\n%s", errorLine)

				// Add carets if we have column and offset information
				if issue.SourcePosition.Column > 0 &&
					issue.SourcePosition.CurrentOffset >= 0 &&
					issue.SourcePosition.EndOffset > issue.SourcePosition.CurrentOffset {

					startColumn := int(issue.SourcePosition.Column - 1) // 0-based
					errorTokenLength := int(issue.SourcePosition.EndOffset - issue.SourcePosition.CurrentOffset)

					if startColumn >= 0 && errorTokenLength > 0 && startColumn <= len(errorLine) {
						padding := strings.Repeat(" ", startColumn)
						carets := strings.Repeat("^", errorTokenLength)
						issueString += fmt.Sprintf("\n%s%s", padding, carets)
					}
				}
				issueString += "\n```"
			}
		}

		formattedOutput = append(formattedOutput, issueString)
	}

	formattedIssues := strings.Join(formattedOutput, "\n\n")

	return ValidationResult{
		Valid:           false,
		IssueCount:      len(issues),
		FormattedIssues: formattedIssues,
		RawIssues:       issues,
	}, nil
}

func initFirestoreConnection(
	ctx context.Context,
	tracer trace.Tracer,
	name string,
	project string,
	database string,
) (*firestore.Client, error) {
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceType, name)
	defer span.End()

	userAgent, err := util.UserAgentFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// If database is not specified, use the default database
	if database == "" {
		database = "(default)"
	}

	// Create the Firestore client
	client, err := firestore.NewClientWithDatabase(ctx, project, database, option.WithUserAgent(userAgent))
	if err != nil {
		return nil, fmt.Errorf("failed to create Firestore client for project %q and database %q: %w", project, database, err)
	}

	return client, nil
}

func initFirebaseRulesConnection(
	ctx context.Context,
	project string,
) (*firebaserules.Service, error) {
	// Create the Firebase Rules client
	rulesClient, err := firebaserules.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create Firebase Rules client for project %q: %w", project, err)
	}

	return rulesClient, nil
}
