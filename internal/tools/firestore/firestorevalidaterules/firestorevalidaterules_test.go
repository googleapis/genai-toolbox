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
	"testing"

	"github.com/googleapis/genai-toolbox/internal/sources"
	firestoreds "github.com/googleapis/genai-toolbox/internal/sources/firestore"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/firebaserules/v1"
)

func TestConfig_ToolConfigKind(t *testing.T) {
	cfg := Config{}
	assert.Equal(t, kind, cfg.ToolConfigKind())
}

func TestConfig_Initialize(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		srcs    map[string]sources.Source
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid configuration",
			cfg: Config{
				Name:        "test-validate-rules",
				Kind:        kind,
				Source:      "firestore",
				Description: "Test validate rules tool",
			},
			srcs: map[string]sources.Source{
				"firestore": &firestoreds.Source{
					Name: "firestore",
					Kind: firestoreds.SourceKind,
				},
			},
			wantErr: false,
		},
		{
			name: "missing source",
			cfg: Config{
				Name:        "test-validate-rules",
				Kind:        kind,
				Source:      "nonexistent",
				Description: "Test validate rules tool",
			},
			srcs:    map[string]sources.Source{},
			wantErr: true,
			errMsg:  "no source named \"nonexistent\" configured",
		},
		{
			name: "incompatible source",
			cfg: Config{
				Name:        "test-validate-rules",
				Kind:        kind,
				Source:      "incompatible",
				Description: "Test validate rules tool",
			},
			srcs: map[string]sources.Source{
				"incompatible": &mockIncompatibleSource{},
			},
			wantErr: true,
			errMsg:  "invalid source for \"firestore-validate-rules\" tool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool, err := tt.cfg.Initialize(tt.srcs)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, tool)
			}
		})
	}
}

func TestTool_ParseParams(t *testing.T) {
	tool := Tool{
		Parameters: createParameters(),
	}

	tests := []struct {
		name    string
		data    map[string]any
		wantErr bool
	}{
		{
			name: "valid parameters",
			data: map[string]any{
				"source": "rules_version = '2';",
			},
			wantErr: false,
		},
		{
			name: "empty source",
			data: map[string]any{
				"source": "",
			},
			wantErr: false, // ParseParams doesn't validate emptiness
		},
		{
			name:    "missing source",
			data:    map[string]any{},
			wantErr: false, // ParseParams doesn't validate required
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params, err := tool.ParseParams(tt.data, nil)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, params)
			}
		})
	}
}

func TestTool_Manifest(t *testing.T) {
	tool := Tool{
		manifest: tools.Manifest{
			Description: "Test description",
			Parameters: []tools.ManifestParameter{
				{
					Name:        "source",
					Type:        "string",
					Description: "The Firestore Rules source code to validate",
					Required:    true,
				},
			},
		},
	}

	manifest := tool.Manifest()
	assert.Equal(t, "Test description", manifest.Description)
	assert.Len(t, manifest.Parameters, 1)
	assert.Equal(t, "source", manifest.Parameters[0].Name)
}

func TestTool_McpManifest(t *testing.T) {
	tool := Tool{
		mcpManifest: tools.McpManifest{
			Name:        "test-validate-rules",
			Description: "Test description",
			InputSchema: tools.McpInputSchema{
				Type: "object",
				Properties: map[string]tools.McpProperty{
					"source": {
						Type:        "string",
						Description: "The Firestore Rules source code to validate",
					},
				},
				Required: []string{"source"},
			},
		},
	}

	manifest := tool.McpManifest()
	assert.Equal(t, "test-validate-rules", manifest.Name)
	assert.Equal(t, "Test description", manifest.Description)
	assert.Contains(t, manifest.InputSchema.Properties, "source")
}

func TestTool_Authorized(t *testing.T) {
	tests := []struct {
		name                 string
		authRequired         []string
		verifiedAuthServices []string
		expected             bool
	}{
		{
			name:                 "no auth required",
			authRequired:         []string{},
			verifiedAuthServices: []string{},
			expected:             true,
		},
		{
			name:                 "auth required and provided",
			authRequired:         []string{"google"},
			verifiedAuthServices: []string{"google"},
			expected:             true,
		},
		{
			name:                 "auth required but not provided",
			authRequired:         []string{"google"},
			verifiedAuthServices: []string{},
			expected:             false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool := Tool{
				AuthRequired: tt.authRequired,
			}
			result := tool.Authorized(tt.verifiedAuthServices)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTool_formatRulesetIssues(t *testing.T) {
	tool := Tool{}
	
	source := `rules_version = '2';
service cloud.firestore {
  match /databases/{database}/documents {
    allow read, write: if true;
  }
}`

	issues := []Issue{
		{
			Description: "Missing semicolon",
			Severity:    "ERROR",
			SourcePosition: SourcePosition{
				Line:          4,
				Column:        31,
				CurrentOffset: 95,
				EndOffset:     99,
			},
		},
	}

	result := tool.formatRulesetIssues(issues, source)
	
	assert.Contains(t, result, "Found 1 issue(s)")
	assert.Contains(t, result, "ERROR: Missing semicolon")
	assert.Contains(t, result, "[Ln 4, Col 31]")
	assert.Contains(t, result, "allow read, write: if true;")
}

func TestTool_processValidationResponse(t *testing.T) {
	tool := Tool{}
	
	tests := []struct {
		name     string
		response *firebaserules.TestRulesetResponse
		source   string
		wantValid bool
		wantCount int
	}{
		{
			name: "no issues",
			response: &firebaserules.TestRulesetResponse{
				Issues: []*firebaserules.Issue{},
			},
			source:    "test source",
			wantValid: true,
			wantCount: 0,
		},
		{
			name: "with issues",
			response: &firebaserules.TestRulesetResponse{
				Issues: []*firebaserules.Issue{
					{
						Description: "Test issue",
						Severity:    "ERROR",
						SourcePosition: &firebaserules.SourcePosition{
							Line:          1,
							Column:        1,
							CurrentOffset: 0,
							EndOffset:     4,
						},
					},
				},
			},
			source:    "test source",
			wantValid: false,
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tool.processValidationResponse(tt.response, tt.source)
			assert.Equal(t, tt.wantValid, result.Valid)
			assert.Equal(t, tt.wantCount, result.IssueCount)
			if tt.wantValid {
				assert.Contains(t, result.FormattedIssues, "No errors detected")
			} else {
				assert.Contains(t, result.FormattedIssues, "issue(s)")
			}
		})
	}
}

// mockIncompatibleSource is a mock source that doesn't implement compatibleSource
type mockIncompatibleSource struct{}

func (m *mockIncompatibleSource) SourceKind() string {
	return "mock"
}
