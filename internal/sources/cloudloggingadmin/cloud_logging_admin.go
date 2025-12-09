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
package cloudloggingadmin

import (
	"context"
	"fmt"

	"cloud.google.com/go/logging"
	"cloud.google.com/go/logging/logadmin"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/util"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/impersonate"
	"google.golang.org/api/option"
)

const SourceKind string = "cloud-logging-admin"

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
	Name                      string `yaml:"name" validate:"required"`
	Kind                      string `yaml:"kind" validate:"required"`
	Project                   string `yaml:"project" validate:"required"`
	UseClientOAuth            bool   `yaml:"useClientOAuth"`
	ImpersonateServiceAccount string `yaml:"impersonateServiceAccount"`
}

func (r Config) SourceConfigKind() string {
	return SourceKind
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {

	if r.UseClientOAuth && r.ImpersonateServiceAccount != "" {
		return nil, fmt.Errorf("useClientOAuth cannot be used with impersonateServiceAccount")
	}

	var client *logadmin.Client
	var tokenSource oauth2.TokenSource
	var clientCreator LogAdminClientCreator
	var err error

	s := &Source{
		Config:        r,
		Client:        client,
		TokenSource:   tokenSource,
		ClientCreator: clientCreator,
	}

	if r.UseClientOAuth {
		// use client OAuth
		baseClientCreator, err := newLogAdminClientCreator(ctx, tracer, r.Project, r.Name)
		if err != nil {
			return nil, fmt.Errorf("error constructing client creator: %w", err)
		}
		setupClientCaching(s, baseClientCreator)
	} else {
		client, tokenSource, err = initLogAdminConnection(ctx, tracer, r.Name, r.Project, r.ImpersonateServiceAccount)
		if err != nil {
			return nil, fmt.Errorf("error creating client from ADC %w", err)
		}
		s.Client = client
		s.TokenSource = tokenSource
	}
	return s, nil
}

var _ sources.Source = &Source{}

type LogAdminClientCreator func(tokenString string) (*logadmin.Client, error)

type Source struct {
	Config
	Client        *logadmin.Client
	TokenSource   oauth2.TokenSource
	ClientCreator LogAdminClientCreator

	// Caches for OAuth clients
	logadminClientCache *sources.Cache
}

func (s *Source) SourceKind() string {
	// Returns logadmin source kind
	return SourceKind
}

func (s *Source) ToConfig() sources.SourceConfig {
	return s.Config
}

func (s *Source) UseClientAuthorization() bool {
	return s.UseClientOAuth
}

func (s *Source) LogAdminClient() *logadmin.Client {
	return s.Client
}

func (s *Source) LogAdminTokenSource() oauth2.TokenSource {
	return s.TokenSource
}

func (s *Source) LogAdminClientCreator() LogAdminClientCreator {
	return s.ClientCreator
}

func setupClientCaching(s *Source, baseCreator LogAdminClientCreator) {
	onEvict := func(key string, value interface{}) {
		if client, ok := value.(*logadmin.Client); ok && client != nil {
			client.Close()
		}
	}

	s.logadminClientCache = sources.NewCache(onEvict)

	s.ClientCreator = func(tokenString string) (*logadmin.Client, error) {
		if val, found := s.logadminClientCache.Get(tokenString); found {
			return val.(*logadmin.Client), nil
		}

		client, err := baseCreator(tokenString)
		if err != nil {
			return nil, err
		}
		s.logadminClientCache.Set(tokenString, client)
		return client, nil
	}
}

func initLogAdminConnection(
	ctx context.Context,
	tracer trace.Tracer,
	name string,
	project string,
	impersonateServiceAccount string,
) (*logadmin.Client, oauth2.TokenSource, error) {
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, name)
	defer span.End()

	userAgent, err := util.UserAgentFromContext(ctx)
	if err != nil {
		return nil, nil, err
	}

	var tokenSource oauth2.TokenSource
	var opts []option.ClientOption

	if impersonateServiceAccount != "" {
		// Create impersonated credentials token source with cloud-platform scope
		// This broader scope is needed for tools like conversational analytics
		cloudPlatformTokenSource, err := impersonate.CredentialsTokenSource(ctx, impersonate.CredentialsConfig{
			TargetPrincipal: impersonateServiceAccount,
			Scopes:          []string{"https://www.googleapis.com/auth/cloud-platform"},
		})

		if err != nil {
			return nil, nil, fmt.Errorf("failed to create impersonated credentials for %q: %w", impersonateServiceAccount, err)
		}

		tokenSource = cloudPlatformTokenSource
		opts = []option.ClientOption{
			option.WithUserAgent(userAgent),
			option.WithTokenSource(cloudPlatformTokenSource),
		}
	} else {
		// Use default credentials
		cred, err := google.FindDefaultCredentials(ctx, logging.AdminScope)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to find default Google Cloud credentials with scope %q: %w", logging.AdminScope, err)
		}
		tokenSource = cred.TokenSource
		opts = []option.ClientOption{
			option.WithUserAgent(userAgent),
			option.WithCredentials(cred),
		}
	}

	client, err := logadmin.NewClient(ctx, project, opts...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Cloud Logging Admin client for project %q: %w", project, err)
	}
	return client, tokenSource, nil
}

func initLogAdminConnectionWithOAuthToken(
	ctx context.Context,
	tracer trace.Tracer,
	project, name, userAgent, tokenString string,
) (*logadmin.Client, error) {
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, name)
	defer span.End()

	token := &oauth2.Token{
		AccessToken: string(tokenString),
	}
	ts := oauth2.StaticTokenSource(token)

	// Initialize the logadmin client with tokenSource
	client, err := logadmin.NewClient(ctx, project, option.WithUserAgent(userAgent), option.WithTokenSource(ts))
	if err != nil {
		return nil, fmt.Errorf("failed to create logadmin client for project %q: %w", project, err)
	}
	return client, nil
}

func newLogAdminClientCreator(
	ctx context.Context,
	tracer trace.Tracer,
	project, name string,
) (LogAdminClientCreator, error) {
	userAgent, err := util.UserAgentFromContext(ctx)
	if err != nil {
		return nil, err
	}

	return func(tokenString string) (*logadmin.Client, error) {
		return initLogAdminConnectionWithOAuthToken(ctx, tracer, project, name, userAgent, tokenString)
	}, nil
}
