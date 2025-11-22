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

package firestorevalidaterules

import (
	"context"
	"fmt"
	"strings"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	firestoreds "github.com/googleapis/genai-toolbox/internal/sources/firestore"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
	"google.golang.org/api/firebaserules/v1"
)

const kind string = "firestore-validate-rules"

// Parameter keys
const (
	sourceKey = "source"
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

type compatibleSource interface {
	FirebaseRulesClient() *firebaserules.Service
	GetProjectId() string
}

// validate compatible sources are still compatible
var _ compatibleSource = &firestoreds.Source{}

var compatibleSources = [...]string{firestoreds.SourceKind}

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

	// Create parameters
	params := createParameters()
	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, params, nil)

	// finish tool setup
	t := Tool{
		Config:      cfg,
		Parameters:  params,
		RulesClient: s.FirebaseRulesClient(),
		ProjectId:   s.GetProjectId(),
		manifest:    tools.Manifest{Description: cfg.Description, Parameters: params.Manifest(), AuthRequired: cfg.AuthRequired},
		mcpManifest: mcpManifest,
	}
	return t, nil
}

// createParameters creates the parameter definitions for the tool
func createParameters() parameters.Parameters {
	sourceParameter := parameters.NewStringParameter(
		sourceKey,
		"The Firestore Rules source code to validate",
	)

	return parameters.Parameters{sourceParameter}
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Config
	Parameters parameters.Parameters `yaml:"parameters"`

	RulesClient *firebaserules.Service
	ProjectId   string
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

// Issue represents a validation issue in the rules
type Issue struct {
	SourcePosition SourcePosition `json:"sourcePosition"`
	Description    string         `json:"description"`
	Severity       string         `json:"severity"`
}

// SourcePosition represents the location of an issue in the source
type SourcePosition struct {
	FileName      string `json:"fileName,omitempty"`
	Line          int64  `json:"line"`          // 1-based
	Column        int64  `json:"column"`        // 1-based
	CurrentOffset int64  `json:"currentOffset"` // 0-based, inclusive start
	EndOffset     int64  `json:"endOffset"`     // 0-based, exclusive end
}

// ValidationResult represents the result of rules validation
type ValidationResult struct {
	Valid           bool    `json:"valid"`
	IssueCount      int     `json:"issueCount"`
	FormattedIssues string  `json:"formattedIssues,omitempty"`
	RawIssues       []Issue `json:"rawIssues,omitempty"`
}

func (t Tool) Invoke(ctx context.Context, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	mapParams := params.AsMap()

	// Get source parameter
	source, ok := mapParams[sourceKey].(string)
	if !ok || source == "" {
		return nil, fmt.Errorf("invalid or missing '%s' parameter", sourceKey)
	}

	// Create test request
	testRequest := &firebaserules.TestRulesetRequest{
		Source: &firebaserules.Source{
			Files: []*firebaserules.File{
				{
					Name:    "firestore.rules",
					Content: source,
				},
			},
		},
		// We don't need test cases for validation only
		TestSuite: &firebaserules.TestSuite{
			TestCases: []*firebaserules.TestCase{},
		},
	}

	// Call the test API
	projectName := fmt.Sprintf("projects/%s", t.ProjectId)
	response, err := t.RulesClient.Projects.Test(projectName, testRequest).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to validate rules: %w", err)
	}

	// Process the response
	result := t.processValidationResponse(response, source)

	return result, nil
}

func (t Tool) processValidationResponse(response *firebaserules.TestRulesetResponse, source string) ValidationResult {
	if len(response.Issues) == 0 {
		return ValidationResult{
			Valid:           true,
			IssueCount:      0,
			FormattedIssues: "✓ No errors detected. Rules are valid.",
		}
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
	formattedIssues := t.formatRulesetIssues(issues, source)

	return ValidationResult{
		Valid:           false,
		IssueCount:      len(issues),
		FormattedIssues: formattedIssues,
		RawIssues:       issues,
	}
}

// formatRulesetIssues formats validation issues into a human-readable string with code snippets
func (t Tool) formatRulesetIssues(issues []Issue, rulesSource string) string {
	sourceLines := strings.Split(rulesSource, "\n")
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

	return strings.Join(formattedOutput, "\n\n")
}

func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (parameters.ParamValues, error) {
	return parameters.ParseParams(t.Parameters, data, claims)
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

func (t Tool) RequiresClientAuthorization() bool {
	return false
}

func (t Tool) GetAuthTokenHeaderName() string {
	return "Authorization"
}
