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
package alloydbadmin

import (
	"context"
	"fmt"
	"net/http"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/util"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	alloydbrestapi "google.golang.org/api/alloydb/v1"
	"google.golang.org/api/option"
)

const SourceKind string = "alloydb-admin"

type userAgentRoundTripper struct {
	userAgent string
	next      http.RoundTripper
}

func (rt *userAgentRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	newReq := *req
	newReq.Header = make(http.Header)
	for k, v := range req.Header {
		newReq.Header[k] = v
	}
	ua := newReq.Header.Get("User-Agent")
	if ua == "" {
		newReq.Header.Set("User-Agent", rt.userAgent)
	} else {
		newReq.Header.Set("User-Agent", ua+" "+rt.userAgent)
	}
	return rt.next.RoundTrip(&newReq)
}

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
	DefaultProject string `yaml:"defaultProject"`
	UseClientOAuth bool   `yaml:"useClientOAuth"`
}

func (r Config) SourceConfigKind() string {
	return SourceKind
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	ua, err := util.UserAgentFromContext(ctx)
	if err != nil {
		fmt.Printf("Error in User Agent retrieval: %s", err)
	}

	var client *http.Client
	if r.UseClientOAuth {
		client = &http.Client{
			Transport: &userAgentRoundTripper{
				userAgent: ua,
				next:      http.DefaultTransport,
			},
		}
	} else {
		// Use Application Default Credentials
		creds, err := google.FindDefaultCredentials(ctx, alloydbrestapi.CloudPlatformScope)
		if err != nil {
			return nil, fmt.Errorf("failed to find default credentials: %w", err)
		}
		baseClient := oauth2.NewClient(ctx, creds.TokenSource)
		baseClient.Transport = &userAgentRoundTripper{
			userAgent: ua,
			next:      baseClient.Transport,
		}
		client = baseClient
	}

	service, err := alloydbrestapi.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("error creating new alloydb service: %w", err)
	}

	defaultProject := util.ExpandEnv(r.DefaultProject)

	s := &Source{
		Name:           r.Name,
		Kind:           SourceKind,
		BaseURL:        "https://alloydb.googleapis.com",
		Service:        service,
		DefaultProject: defaultProject,
		UseClientOAuth: r.UseClientOAuth,
	}

	return s, nil
}

var _ sources.Source = &Source{}

type Source struct {
	Name           string `yaml:"name"`
	Kind           string `yaml:"kind"`
	BaseURL        string
	Service        *alloydbrestapi.Service
	DefaultProject string
	UseClientOAuth bool
}

func (s *Source) SourceKind() string {
	return SourceKind
}

func (s *Source) GetService(ctx context.Context, accessToken string) (*alloydbrestapi.Service, error) {
	if s.UseClientOAuth {
		token := &oauth2.Token{AccessToken: accessToken}
		client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))
		service, err := alloydbrestapi.NewService(ctx, option.WithHTTPClient(client))
		if err != nil {
			return nil, fmt.Errorf("error creating new alloydb service: %w", err)
		}
		return service, nil
	}
	return s.Service, nil
}

func (s *Source) UseClientAuthorization() bool {
	return s.UseClientOAuth
}
