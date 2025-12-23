// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package cloudgda

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/util"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const SourceKind string = "cloud-gemini-data-analytics"
const Endpoint string = "https://geminidataanalytics.googleapis.com"

// validate interface
var _ sources.SourceConfig = Config{}

func init() {
	if !sources.Register(SourceKind, newConfig) {
		panic(fmt.Sprintf("source kind %q already registered", SourceKind))
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
	Name           string `yaml:"name" validate:"required"`
	Kind           string `yaml:"kind" validate:"required"`
	ProjectID      string `yaml:"projectId" validate:"required"`
	UseClientOAuth bool   `yaml:"useClientOAuth"`
}

func (r Config) SourceConfigKind() string {
	return SourceKind
}

// Initialize initializes a Gemini Data Analytics Source instance.
func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	ua, err := util.UserAgentFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("error in User Agent retrieval: %s", err)
	}

	var client *http.Client
	if r.UseClientOAuth {
		client = &http.Client{
			Transport: util.NewUserAgentRoundTripper(ua, http.DefaultTransport),
		}
	} else {
		// Use Application Default Credentials
		// Scope: "https://www.googleapis.com/auth/cloud-platform" is generally sufficient for GDA
		creds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
		if err != nil {
			return nil, fmt.Errorf("failed to find default credentials: %w", err)
		}
		baseClient := oauth2.NewClient(ctx, creds.TokenSource)
		baseClient.Transport = util.NewUserAgentRoundTripper(ua, baseClient.Transport)
		client = baseClient
	}

	s := &Source{
		Config:    r,
		Client:    client,
		BaseURL:   Endpoint,
		userAgent: ua,
	}
	return s, nil
}

var _ sources.Source = &Source{}

type Source struct {
	Config
	Client    *http.Client
	BaseURL   string
	userAgent string
}

func (s *Source) SourceKind() string {
	return SourceKind
}

func (s *Source) ToConfig() sources.SourceConfig {
	return s.Config
}

func (s *Source) GetProjectID() string {
	return s.ProjectID
}

func (s *Source) GetBaseURL() string {
	return s.BaseURL
}

func (s *Source) GetClient(ctx context.Context, accessToken string) (*http.Client, error) {
	if s.UseClientOAuth {
		if accessToken == "" {
			return nil, fmt.Errorf("client-side OAuth is enabled but no access token was provided")
		}
		token := &oauth2.Token{AccessToken: accessToken}
		baseClient := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))
		baseClient.Transport = util.NewUserAgentRoundTripper(s.userAgent, baseClient.Transport)
		return baseClient, nil
	}
	return s.Client, nil
}

func (s *Source) UseClientAuthorization() bool {
	return s.UseClientOAuth
}

func (s *Source) RunQuery(ctx context.Context, tokenStr string, bodyBytes []byte) (any, error) {
	// The API endpoint itself always uses the "global" location.
	apiLocation := "global"
	apiParent := fmt.Sprintf("projects/%s/locations/%s", s.GetProjectID(), apiLocation)
	apiURL := fmt.Sprintf("%s/v1beta/%s:queryData", s.GetBaseURL(), apiParent)

	client, err := s.GetClient(ctx, tokenStr)
	if err != nil {
		return nil, fmt.Errorf("failed to get HTTP client: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return result, nil
}
