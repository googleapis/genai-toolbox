// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cloudgda_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	yaml "github.com/goccy/go-yaml"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/sources"
	cloudgdasrc "github.com/googleapis/genai-toolbox/internal/sources/cloudgda"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools"
	cloudgdatool "github.com/googleapis/genai-toolbox/internal/tools/cloudgda"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
)

func TestParseFromYaml(t *testing.T) {
	t.Parallel()
	tcs := []struct {
		desc string
		in   string
		want server.ToolConfigs
	}{
		{
			desc: "basic example",
			in: `
			tools:
				my-gda-query-tool:
					kind: cloud-gemini-data-analytics-query
					source: gda-api-source
					description: Test Description
					location: us-central1
					context:
						datasourceReferences:
							spannerReference:
								databaseReference:
									projectId:  "cloud-db-nl2sql"
									region:     "us-central1"
									instanceId: "evalbench"
									databaseId: "financial"
									engine:     "GOOGLE_SQL"
								agentContextReference:
									contextSetId: "projects/cloud-db-nl2sql/locations/us-east1/contextSets/bdf_gsql_gemini_all_templates"
					generationOptions:
						generateQueryResult: true
			`,
			want: map[string]tools.ToolConfig{
				"my-gda-query-tool": cloudgdatool.Config{
					Name:         "my-gda-query-tool",
					Kind:         cloudgdatool.Kind,
					Source:       "gda-api-source",
					Description:  "Test Description",
					Location:     "us-central1",
					AuthRequired: []string{},
					Context: &cloudgdatool.QueryDataContext{
						DatasourceReferences: &cloudgdatool.DatasourceReferences{
							SpannerReference: &cloudgdatool.SpannerReference{
								DatabaseReference: &cloudgdatool.SpannerDatabaseReference{
									ProjectID:  "cloud-db-nl2sql",
									Region:     "us-central1",
									InstanceID: "evalbench",
									DatabaseID: "financial",
									Engine:     cloudgdatool.SpannerEngineGoogleSQL,
								},
								AgentContextReference: &cloudgdatool.AgentContextReference{
									ContextSetID: "projects/cloud-db-nl2sql/locations/us-east1/contextSets/bdf_gsql_gemini_all_templates",
								},
							},
						},
					},
					GenerationOptions: &cloudgdatool.GenerationOptions{
						GenerateQueryResult: true,
					},
				},
			},
		},
	}
	for _, tc := range tcs {
		tc := tc
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()
			got := struct {
				Tools server.ToolConfigs `yaml:"tools"`
			}{}
			// Parse contents
			err := yaml.Unmarshal(testutils.FormatYaml(tc.in), &got)
			if err != nil {
				t.Fatalf("unable to unmarshal: %s", err)
			}
			if !cmp.Equal(tc.want, got.Tools) {
				t.Fatalf("incorrect parse: want %v, got %v", tc.want, got.Tools)
			}
		})
	}
}

// authRoundTripper is a mock http.RoundTripper that adds a dummy Authorization header.
type authRoundTripper struct {
	Token string
	Next  http.RoundTripper
}

func (rt *authRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	newReq := *req
	newReq.Header = make(http.Header)
	for k, v := range req.Header {
		newReq.Header[k] = v
	}
	newReq.Header.Set("Authorization", rt.Token)
	if rt.Next == nil {
		return http.DefaultTransport.RoundTrip(&newReq)
	}
	return rt.Next.RoundTrip(&newReq)
}

type mockSource struct {
	kind    string
	client  *http.Client       // Can be used to inject a specific client
	baseURL string             // BaseURL is needed to implement sources.Source.BaseURL
	config  cloudgdasrc.Config // to return from ToConfig
}

func (m *mockSource) SourceKind() string             { return m.kind }
func (m *mockSource) ToConfig() sources.SourceConfig { return m.config }
func (m *mockSource) GetClient(ctx context.Context, token string) (*http.Client, error) {
	if m.client != nil {
		return m.client, nil
	}
	// Default client for testing if not explicitly set
	transport := &http.Transport{}
	authTransport := &authRoundTripper{
		Token: "Bearer test-access-token", // Dummy token
		Next:  transport,
	}
	return &http.Client{Transport: authTransport}, nil
}
func (m *mockSource) UseClientAuthorization() bool { return false }
func (m *mockSource) Initialize(ctx context.Context, tracer interface{}) (sources.Source, error) {
	return m, nil
}
func (m *mockSource) BaseURL() string { return m.baseURL }

func TestInitialize(t *testing.T) {
	t.Parallel()

	srcs := map[string]sources.Source{
		"gda-api-source": &cloudgdasrc.Source{
			Config:  cloudgdasrc.Config{Name: "gda-api-source", Kind: cloudgdasrc.SourceKind, ProjectID: "test-project"},
			Client:  &http.Client{},
			BaseURL: cloudgdasrc.Endpoint,
		},
	}

	tcs := []struct {
		desc      string
		cfg       cloudgdatool.Config
		expectErr bool
	}{
		{
			desc: "successful initialization",
			cfg: cloudgdatool.Config{
				Name:        "my-gda-query-tool",
				Kind:        cloudgdatool.Kind,
				Source:      "gda-api-source",
				Description: "Test Description",
				Location:    "us-central1",
			},
			expectErr: false,
		},
		{
			desc: "missing source",
			cfg: cloudgdatool.Config{
				Name:        "my-gda-query-tool",
				Kind:        cloudgdatool.Kind,
				Source:      "non-existent-source",
				Description: "Test Description",
				Location:    "us-central1",
			},
			expectErr: true,
		},
		{
			desc: "incompatible source kind",
			cfg: cloudgdatool.Config{
				Name:        "my-gda-query-tool",
				Kind:        cloudgdatool.Kind,
				Source:      "incompatible-source",
				Description: "Test Description",
				Location:    "us-central1",
			},
			expectErr: true,
		},
	}

	// Add an incompatible source for testing
	srcs["incompatible-source"] = &mockSource{kind: "another-kind"}

	for _, tc := range tcs {
		tc := tc
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()
			tool, err := tc.cfg.Initialize(srcs)
			if tc.expectErr && err == nil {
				t.Fatalf("expected an error but got none")
			}
			if !tc.expectErr && err != nil {
				t.Fatalf("did not expect an error but got: %v", err)
			}
			if !tc.expectErr {
				// Basic sanity check on the returned tool
				_ = tool // Avoid unused variable error
			}
		})
	}
}

func TestInvoke(t *testing.T) {
	t.Parallel()
	// Mock the HTTP client and server for Invoke testing
	serverMux := http.NewServeMux()
	// Update expected URL path to include the location "us-central1"
	serverMux.HandleFunc("/v1beta/projects/test-project/locations/global:queryData", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST method, got %s", r.Method)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		// Read and unmarshal the request body
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read request body: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		var reqPayload cloudgdatool.QueryDataRequest
		if err := json.Unmarshal(bodyBytes, &reqPayload); err != nil {
			t.Errorf("failed to unmarshal request payload: %v", err)
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		// Verify expected fields
		if r.Header.Get("Authorization") == "" {
			t.Errorf("expected Authorization header, got empty")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if reqPayload.Prompt != "How many accounts who have region in Prague are eligible for loans?" {
			t.Errorf("unexpected prompt: %s", reqPayload.Prompt)
		}

		// Verify payload's parent uses the tool's configured location
		if reqPayload.Parent != fmt.Sprintf("projects/%s/locations/%s", "test-project", "us-central1") {
			t.Errorf("unexpected payload parent: got %q, want %q", reqPayload.Parent, fmt.Sprintf("projects/%s/locations/%s", "test-project", "us-central1"))
		}

		// Verify context from config
		if reqPayload.Context == nil ||
			reqPayload.Context.DatasourceReferences == nil ||
			reqPayload.Context.DatasourceReferences.SpannerReference == nil ||
			reqPayload.Context.DatasourceReferences.SpannerReference.DatabaseReference == nil ||
			reqPayload.Context.DatasourceReferences.SpannerReference.DatabaseReference.ProjectID != "cloud-db-nl2sql" {
			t.Errorf("unexpected context: %v", reqPayload.Context)
		}

		// Verify generation options from config
		if reqPayload.GenerationOptions == nil || !reqPayload.GenerationOptions.GenerateQueryResult {
			t.Errorf("unexpected generation options: %v", reqPayload.GenerationOptions)
		}

		// Simulate a successful response
		resp := map[string]any{
			"queryResult":           "SELECT count(*) FROM accounts WHERE region = 'Prague' AND eligible_for_loans = true;",
			"naturalLanguageAnswer": "There are 5 accounts in Prague eligible for loans.",
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	mockServer := httptest.NewServer(serverMux)
	defer mockServer.Close()

	ctx := testutils.ContextWithUserAgent(context.Background(), "test-user-agent")

	// Create an authenticated client that uses the mock server
	authTransport := &authRoundTripper{
		Token: "Bearer test-access-token",
		Next:  mockServer.Client().Transport,
	}
	authClient := &http.Client{Transport: authTransport}

	// Create a real cloudgdasrc.Source but inject the authenticated client
	mockGdaSource := &cloudgdasrc.Source{
		Config:  cloudgdasrc.Config{Name: "mock-gda-source", Kind: cloudgdasrc.SourceKind, ProjectID: "test-project"},
		Client:  authClient,
		BaseURL: mockServer.URL,
	}
	srcs := map[string]sources.Source{
		"mock-gda-source": mockGdaSource,
	}

	// Initialize the tool config with context
	toolCfg := cloudgdatool.Config{
		Name:        "query-data-tool",
		Kind:        "cloud-gemini-data-analytics-query",
		Source:      "mock-gda-source",
		Description: "Query Gemini Data Analytics",
		Location:    "us-central1", // Set location for the test
		Context: &cloudgdatool.QueryDataContext{
			DatasourceReferences: &cloudgdatool.DatasourceReferences{
				SpannerReference: &cloudgdatool.SpannerReference{
					DatabaseReference: &cloudgdatool.SpannerDatabaseReference{
						ProjectID:  "cloud-db-nl2sql",
						Region:     "us-central1",
						InstanceID: "evalbench",
						DatabaseID: "financial",
						Engine:     cloudgdatool.SpannerEngineGoogleSQL,
					},
					AgentContextReference: &cloudgdatool.AgentContextReference{
						ContextSetID: "projects/cloud-db-nl2sql/locations/us-east1/contextSets/bdf_gsql_gemini_all_templates",
					},
				},
			},
		},
		GenerationOptions: &cloudgdatool.GenerationOptions{
			GenerateQueryResult: true,
		},
	}

	tool, err := toolCfg.Initialize(srcs)
	if err != nil {
		t.Fatalf("failed to initialize tool: %v", err)
	}

	// Prepare parameters for invocation - ONLY prompt
	params := parameters.ParamValues{
		{Name: "prompt", Value: "How many accounts who have region in Prague are eligible for loans?"},
	}

	// Invoke the tool
	result, err := tool.Invoke(ctx, nil, params, "") // No accessToken needed for ADC client
	if err != nil {
		t.Fatalf("tool invocation failed: %v", err)
	}

	// Validate the result
	expectedResult := map[string]any{
		"queryResult":           "SELECT count(*) FROM accounts WHERE region = 'Prague' AND eligible_for_loans = true;",
		"naturalLanguageAnswer": "There are 5 accounts in Prague eligible for loans.",
	}

	if !cmp.Equal(expectedResult, result) {
		t.Errorf("unexpected result: got %v, want %v", result, expectedResult)
	}
}
