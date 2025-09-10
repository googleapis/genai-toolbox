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

package bigquery

import (
	"context"
	"fmt"

	bigqueryapi "cloud.google.com/go/bigquery"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/util"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	bigqueryrestapi "google.golang.org/api/bigquery/v2"
	"google.golang.org/api/option"
)

const SourceKind string = "bigquery"

// validate interface
var _ sources.SourceConfig = Config{}

type BigqueryClientCreator func(tokenString string, wantRestService bool) (*bigqueryapi.Client, *bigqueryrestapi.Service, error)

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
	// BigQuery configs
	Name           string `yaml:"name" validate:"required"`
	Kind           string `yaml:"kind" validate:"required"`
	Project        string `yaml:"project" validate:"required"`
	Location       string `yaml:"location"`
	UseClientOAuth bool   `yaml:"useClientOAuth"`
}

func (r Config) SourceConfigKind() string {
	// Returns BigQuery source kind
	return SourceKind
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	var client *bigqueryapi.Client
	var restService *bigqueryrestapi.Service
	var tokenSource oauth2.TokenSource
	var clientCreator BigqueryClientCreator
	var err error

	if r.UseClientOAuth {
		clientCreator, err = newBigQueryClientCreator(ctx, tracer, r.Project, r.Location, r.Name)
		if err != nil {
			return nil, fmt.Errorf("error constructing client creator: %w", err)
		}
	} else {
		// Initializes a BigQuery Google SQL source
		client, restService, tokenSource, err = initBigQueryConnection(ctx, tracer, r.Name, r.Project, r.Location)
		if err != nil {
			return nil, fmt.Errorf("error creating client from ADC: %w", err)
		}
	}

	s := &Source{
		Name:               r.Name,
		Kind:               SourceKind,
		Project:            r.Project,
		Location:           r.Location,
		Client:             client,
		RestService:        restService,
		TokenSource:        tokenSource,
		MaxQueryResultRows: 50,
		ClientCreator:      clientCreator,
		UseClientOAuth:     r.UseClientOAuth,
	}
	return s, nil

}

var _ sources.Source = &Source{}

type Source struct {
	// BigQuery Google SQL struct with client
	Name               string `yaml:"name"`
	Kind               string `yaml:"kind"`
	Project            string
	Location           string
	Client             *bigqueryapi.Client
	RestService        *bigqueryrestapi.Service
	TokenSource        oauth2.TokenSource
	MaxQueryResultRows int
	ClientCreator      BigqueryClientCreator
	UseClientOAuth     bool
}

func (s *Source) SourceKind() string {
	// Returns BigQuery Google SQL source kind
	return SourceKind
}

func (s *Source) BigQueryClient() *bigqueryapi.Client {
	return s.Client
}

func (s *Source) BigQueryRestService() *bigqueryrestapi.Service {
	return s.RestService
}

func (s *Source) UseClientAuthorization() bool {
	return s.UseClientOAuth
}

func (s *Source) BigQueryProject() string {
	return s.Project
}

func (s *Source) BigQueryLocation() string {
	return s.Location
}

func (s *Source) BigQueryTokenSource() oauth2.TokenSource {
	return s.TokenSource
}

func (s *Source) GetMaxQueryResultRows() int {
	return s.MaxQueryResultRows
}

func (s *Source) BigQueryClientCreator() BigqueryClientCreator {
	return s.ClientCreator
}

func initBigQueryConnection(
	ctx context.Context,
	tracer trace.Tracer,
	name string,
	project string,
	location string,
) (*bigqueryapi.Client, *bigqueryrestapi.Service, oauth2.TokenSource, error) {
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, name)
	defer span.End()

	cred, err := google.FindDefaultCredentials(ctx, bigqueryapi.Scope)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to find default Google Cloud credentials with scope %q: %w", bigqueryapi.Scope, err)
	}

	userAgent, err := util.UserAgentFromContext(ctx)
	if err != nil {
		return nil, nil, nil, err
	}

	// Initialize the high-level BigQuery client
	client, err := bigqueryapi.NewClient(ctx, project, option.WithUserAgent(userAgent), option.WithCredentials(cred))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create BigQuery client for project %q: %w", project, err)
	}
	client.Location = location

	// Initialize the low-level BigQuery REST service using the same credentials
	restService, err := bigqueryrestapi.NewService(ctx, option.WithUserAgent(userAgent), option.WithCredentials(cred))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create BigQuery v2 service: %w", err)
	}

	return client, restService, cred.TokenSource, nil
}

// initBigQueryConnectionWithOAuthToken initialize a BigQuery client with an
// OAuth access token.
func initBigQueryConnectionWithOAuthToken(
	ctx context.Context,
	tracer trace.Tracer,
	project string,
	location string,
	name string,
	userAgent string,
	tokenString string,
	wantRestService bool,
) (*bigqueryapi.Client, *bigqueryrestapi.Service, error) {
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, name)
	defer span.End()
	// Construct token source
	token := &oauth2.Token{
		AccessToken: string(tokenString),
	}
	ts := oauth2.StaticTokenSource(token)

	// Initialize the BigQuery client with tokenSource
	client, err := bigqueryapi.NewClient(ctx, project, option.WithUserAgent(userAgent), option.WithTokenSource(ts))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create BigQuery client for project %q: %w", project, err)
	}
	client.Location = location

	if wantRestService {
		// Initialize the low-level BigQuery REST service using the same credentials
		restService, err := bigqueryrestapi.NewService(ctx, option.WithUserAgent(userAgent), option.WithTokenSource(ts))
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create BigQuery v2 service: %w", err)
		}
		return client, restService, nil
	}

	return client, nil, nil
}

// newBigQueryClientCreator sets the project parameters for the init helper
// function. The returned function takes in an OAuth access token and uses it to
// create a BQ client.
func newBigQueryClientCreator(
	ctx context.Context,
	tracer trace.Tracer,
	project string,
	location string,
	name string,
) (func(string, bool) (*bigqueryapi.Client, *bigqueryrestapi.Service, error), error) {
	userAgent, err := util.UserAgentFromContext(ctx)
	if err != nil {
		return nil, err
	}

	return func(tokenString string, wantRestService bool) (*bigqueryapi.Client, *bigqueryrestapi.Service, error) {
		return initBigQueryConnectionWithOAuthToken(ctx, tracer, project, location, name, userAgent, tokenString, wantRestService)
	}, nil
}
