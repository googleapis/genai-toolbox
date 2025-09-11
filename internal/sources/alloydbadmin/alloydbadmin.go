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
)

const SourceKind string = "alloydb-admin"

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
	Name                   string            `yaml:"name" validate:"required"`
	Kind                   string            `yaml:"kind" validate:"required"`
	UseClientOAuth         bool              `yaml:"useClientOAuth"`
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
		client = nil
	} else {
		// Use Application Default Credentials
		creds, err := google.FindDefaultCredentials(ctx, alloydbrestapi.CloudPlatformScope)
		if err != nil {
			return nil, fmt.Errorf("failed to find default credentials: %w", err)
		}
		client = oauth2.NewClient(ctx, creds.TokenSource)
	}

	s := &Source{
		Name:           r.Name,
		Kind:           SourceKind,
		BaseURL:        "https://alloydb.googleapis.com",
		Client:         client,
		UserAgent:      ua,
		UseClientOAuth: r.UseClientOAuth,
	}

	return s, nil
}

var _ sources.Source = &Source{}

type Source struct {
	Name           string `yaml:"name"`
	Kind           string `yaml:"kind"`
	BaseURL        string
	Client         *http.Client
	UserAgent      string
	UseClientOAuth bool
}

func (s *Source) SourceKind() string {
	return SourceKind
}

func (s *Source) GetClient(ctx context.Context, accessToken string) (*http.Client, error) {
	if s.UseClientOAuth {
		if accessToken == "" {
			return nil, fmt.Errorf("client-side OAuth is enabled but no access token was provided")
		}
		token := &oauth2.Token{AccessToken: accessToken}
		return oauth2.NewClient(ctx, oauth2.StaticTokenSource(token)), nil
	}
	return s.Client, nil
}

func (s *Source) UseClientAuthorization() bool {
	return s.UseClientOAuth
}
