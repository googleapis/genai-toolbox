// Copyright 2026 Google LLC
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

package databaseinsights

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const SourceKind string = "databaseinsights"

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
	Name     string `yaml:"name" validate:"required"`
	Type     string `yaml:"type" validate:"required"`
	Project  string `yaml:"project"`  // Optional: fallback billing project
	Endpoint string `yaml:"endpoint"` // Optional: override endpoint for staging/testing
}

func (cfg Config) SourceConfigType() string {
	return SourceKind
}

func (cfg Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	httpClient, endpoint, err := initConnection(ctx, tracer, cfg.Name, cfg.Project, cfg.Endpoint)
	if err != nil {
		return nil, err
	}

	s := &Source{
		Config:     cfg,
		httpClient: httpClient,
		endpoint:   endpoint,
	}
	return s, nil
}

var _ sources.Source = &Source{}

type Source struct {
	Config
	httpClient *http.Client
	endpoint   string
}

func (s *Source) SourceType() string {
	return SourceKind
}

func (s *Source) ToConfig() sources.SourceConfig {
	return s.Config
}

func (s *Source) HTTPClient() *http.Client {
	return s.httpClient
}

func (s *Source) APIEndpoint() string {
	return s.endpoint
}

func (s *Source) ProjectID() string {
	return s.Project
}

func initConnection(
	ctx context.Context,
	tracer trace.Tracer,
	name string,
	project string,
	endpoint string,
) (*http.Client, string, error) {
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, name)
	defer span.End()

	cred, err := google.FindDefaultCredentials(ctx, sources.CloudPlatformScope)
	if err != nil {
		return nil, "", fmt.Errorf("failed to find default Google Cloud credentials with scope %q: %w", sources.CloudPlatformScope, err)
	}

	userAgent, err := util.UserAgentFromContext(ctx)
	if err != nil {
		return nil, "", err
	}

	// Create authenticated HTTP client using the credentials token source
	httpClient := oauth2.NewClient(ctx, cred.TokenSource)
	httpClient.Transport = &authHeadersRoundTripper{
		configProject: project,
		adcProject:    cred.ProjectID,
		userAgent:     userAgent,
		next:          httpClient.Transport,
	}

	if endpoint == "" {
		endpoint = "https://databaseinsights.googleapis.com"
	}

	return httpClient, endpoint, nil
}

type authHeadersRoundTripper struct {
	configProject string
	adcProject    string
	userAgent     string
	next          http.RoundTripper
}

func (rt *authHeadersRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	newReq := req.Clone(req.Context())

	// Inject User-Agent
	ua := newReq.Header.Get("User-Agent")
	if ua == "" {
		newReq.Header.Set("User-Agent", rt.userAgent)
	} else {
		newReq.Header.Set("User-Agent", ua+" "+rt.userAgent)
	}

	// Determine the billing/quota project to use with correct precedence:
	// 1. YAML Config (rt.configProject)
	// 2. Extracted URL Path
	// 3. ADC credentials (rt.adcProject)
	var quotaProject string
	if rt.configProject != "" {
		quotaProject = rt.configProject
	} else if extracted := extractProjectFromPath(newReq.URL.Path); extracted != "" {
		quotaProject = extracted
	} else {
		quotaProject = rt.adcProject
	}

	// Inject billing/quota project header required for ADC authentication if not already set
	if quotaProject != "" && newReq.Header.Get("X-Goog-User-Project") == "" {
		newReq.Header.Set("X-Goog-User-Project", quotaProject)
	}

	return rt.next.RoundTrip(newReq)
}

func extractProjectFromPath(path string) string {
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	for i := 0; i < len(parts)-1; i++ {
		if parts[i] == "projects" {
			return parts[i+1]
		}
	}
	return ""
}
