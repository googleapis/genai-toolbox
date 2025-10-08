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
	"net/http"
	"os"
	"strings"
	"sync"

	bigqueryapi "cloud.google.com/go/bigquery"
	dataplexapi "cloud.google.com/go/dataplex/apiv1"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/util"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	bigqueryrestapi "google.golang.org/api/bigquery/v2"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

const SourceKind string = "bigquery"

// validate interface
var _ sources.SourceConfig = Config{}

type BigqueryClientCreator func(tokenString string, wantRestService bool) (*bigqueryapi.Client, *bigqueryrestapi.Service, error)

type DataplexClientCreator func(tokenString string) (*dataplexapi.CatalogClient, error)

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
	Name            string   `yaml:"name" validate:"required"`
	Kind            string   `yaml:"kind" validate:"required"`
	Project         string   `yaml:"project" validate:"required"`
	Location        string   `yaml:"location"`
	AllowedDatasets []string `yaml:"allowedDatasets"`
	UseClientOAuth  bool     `yaml:"useClientOAuth"`
	CredentialsJSON string   `yaml:"credentialsJson"`
	CredentialsPath string   `yaml:"credentialsPath"`
}

// validateCredentials ensures that only one credential method is provided.
func (r Config) validateCredentials() error {
	provided := 0
	if r.UseClientOAuth {
		provided++
	}
	if r.CredentialsJSON != "" {
		provided++
	}
	if r.CredentialsPath != "" {
		provided++
	}
	if provided > 1 {
		return fmt.Errorf("for source %q, you can only set one of useClientOAuth, credentialsJson, or credentialsPath", r.Name)
	}
	return nil
}

func (r Config) SourceConfigKind() string {
	// Returns BigQuery source kind
	return SourceKind
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	if err := r.validateCredentials(); err != nil {
		return nil, err
	}

	var client *bigqueryapi.Client
	var restService *bigqueryrestapi.Service
	var cloudPlatformTokenSource oauth2.TokenSource
	var clientCreator BigqueryClientCreator
	var err error
	var credsJSON []byte

	if r.UseClientOAuth {
		clientCreator, err = newBigQueryClientCreator(ctx, tracer, r.Project, r.Location, r.Name)
		if err != nil {
			return nil, fmt.Errorf("error constructing client creator: %w", err)
		}
	} else { // Use service account (JSON/path) or ADC
		if r.CredentialsPath != "" {
			credsJSON, err = os.ReadFile(r.CredentialsPath)
			if err != nil {
				return nil, fmt.Errorf("failed to read credentials from path %q: %w", r.CredentialsPath, err)
			}
		} else if r.CredentialsJSON != "" {
			credsJSON = []byte(r.CredentialsJSON)
		}

		opts := []option.ClientOption{}
		if len(credsJSON) > 0 {
			// Use the provided credentials for both BQ client and the separate token source.
			client, restService, cloudPlatformTokenSource, err = initBigQueryConnection(ctx, tracer, r.Name, r.Project, r.Location, false, credsJSON, opts...)
			if err != nil {
				return nil, fmt.Errorf("error creating client from credentials: %w", err)
			}
		} else {
			client, restService, cloudPlatformTokenSource, err = initBigQueryConnection(ctx, tracer, r.Name, r.Project, r.Location, true, nil)
			if err != nil {
				return nil, fmt.Errorf("error creating client from ADC: %w", err)
			}
		}
	}

	allowedDatasets := make(map[string]struct{})
	// Get full id of allowed datasets and verify they exist.
	if len(r.AllowedDatasets) > 0 {
		for _, allowed := range r.AllowedDatasets {
			var projectID, datasetID, allowedFullID string
			if strings.Contains(allowed, ".") {
				parts := strings.Split(allowed, ".")
				if len(parts) != 2 {
					return nil, fmt.Errorf("invalid allowedDataset format: %q, expected 'project.dataset' or 'dataset'", allowed)
				}
				projectID = parts[0]
				datasetID = parts[1]
				allowedFullID = allowed
			} else {
				projectID = client.Project()
				datasetID = allowed
				allowedFullID = fmt.Sprintf("%s.%s", projectID, datasetID)
			}

			dataset := client.DatasetInProject(projectID, datasetID)
			_, err := dataset.Metadata(ctx)
			if err != nil {
				if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == http.StatusNotFound {
					return nil, fmt.Errorf("allowedDataset '%s' not found in project '%s'", datasetID, projectID)
				}
				return nil, fmt.Errorf("failed to verify allowedDataset '%s' in project '%s': %w", datasetID, projectID, err)
			}
			allowedDatasets[allowedFullID] = struct{}{}
		}
	}

	s := &Source{
		Name:                     r.Name,
		Kind:                     SourceKind,
		Project:                  r.Project,
		Location:                 r.Location,
		CredentialsJSON:          string(credsJSON),
		CredentialsPath:          r.CredentialsPath,
		Client:                   client,
		RestService:              restService,
		CloudPlatformTokenSource: cloudPlatformTokenSource,
		MaxQueryResultRows:       50,
		ClientCreator:            clientCreator,
		AllowedDatasets:          allowedDatasets,
		UseClientOAuth:           r.UseClientOAuth,
	}
	s.makeDataplexCatalogClient = s.lazyInitDataplexClient(ctx, tracer)
	return s, nil

}

var _ sources.Source = &Source{}

type Source struct {
	// BigQuery Google SQL struct with client
	Name                      string `yaml:"name"`
	Kind                      string `yaml:"kind"`
	Project                   string
	Location                  string
	CredentialsJSON           string
	CredentialsPath           string
	Client                    *bigqueryapi.Client
	RestService               *bigqueryrestapi.Service
	CloudPlatformTokenSource  oauth2.TokenSource
	MaxQueryResultRows        int
	ClientCreator             BigqueryClientCreator
	AllowedDatasets           map[string]struct{}
	UseClientOAuth            bool
	makeDataplexCatalogClient func() (*dataplexapi.CatalogClient, DataplexClientCreator, error)
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

func (s *Source) BigQueryCloudPlatformTokenSource() oauth2.TokenSource {
	return s.CloudPlatformTokenSource
}

func (s *Source) GetMaxQueryResultRows() int {
	return s.MaxQueryResultRows
}

func (s *Source) BigQueryClientCreator() BigqueryClientCreator {
	return s.ClientCreator
}

func (s *Source) BigQueryAllowedDatasets() []string {
	if len(s.AllowedDatasets) == 0 {
		return nil
	}
	datasets := make([]string, 0, len(s.AllowedDatasets))
	for d := range s.AllowedDatasets {
		datasets = append(datasets, d)
	}
	return datasets
}

// IsDatasetAllowed checks if a given dataset is accessible based on the source's configuration.
func (s *Source) IsDatasetAllowed(projectID, datasetID string) bool {
	// If the normalized map is empty, it means no restrictions were configured.
	if len(s.AllowedDatasets) == 0 {
		return true
	}

	targetDataset := fmt.Sprintf("%s.%s", projectID, datasetID)
	_, ok := s.AllowedDatasets[targetDataset]
	return ok
}

func (s *Source) MakeDataplexCatalogClient() func() (*dataplexapi.CatalogClient, DataplexClientCreator, error) {
	return s.makeDataplexCatalogClient
}

func (s *Source) lazyInitDataplexClient(ctx context.Context, tracer trace.Tracer) func() (*dataplexapi.CatalogClient, DataplexClientCreator, error) {
	var once sync.Once
	var client *dataplexapi.CatalogClient
	var clientCreator DataplexClientCreator
	var err error

	return func() (*dataplexapi.CatalogClient, DataplexClientCreator, error) {
		once.Do(func() {
			c, cc, e := initDataplexConnection(ctx, tracer, s.Name, s.Project, s.UseClientOAuth, []byte(s.CredentialsJSON))
			if e != nil {
				err = fmt.Errorf("failed to initialize dataplex client: %w", e)
				return
			}
			client = c
			clientCreator = cc
		})
		return client, clientCreator, err
	}
}

func initBigQueryConnection(
	ctx context.Context,
	tracer trace.Tracer,
	name string,
	project string,
	location string,
	useADC bool,
	credsJSON []byte,
	opts ...option.ClientOption,
) (*bigqueryapi.Client, *bigqueryrestapi.Service, oauth2.TokenSource, error) {
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, name)
	defer span.End()

	userAgent, err := util.UserAgentFromContext(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	opts = append(opts, option.WithUserAgent(userAgent))

	var cred *google.Credentials
	var cloudPlatformTokenSource oauth2.TokenSource
	if useADC {
		cred, err = google.FindDefaultCredentials(ctx, bigqueryapi.Scope)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to find default Google Cloud credentials with scope %q: %w", bigqueryapi.Scope, err)
		}
		cloudPlatformCreds, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to find default Google Cloud credentials with cloud-platform scope: %w", err)
		}
		if cloudPlatformCreds != nil {
			cloudPlatformTokenSource = cloudPlatformCreds.TokenSource
		}
	} else {
		// When not using ADC, we expect credentials to be in opts.
		// We need to extract them to create a separate token source with a different scope.
		creds, err := google.CredentialsFromJSON(ctx, credsJSON, "https://www.googleapis.com/auth/cloud-platform")
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to create credentials with cloud-platform scope from JSON: %w", err)
		}
		cloudPlatformTokenSource = creds.TokenSource

		// For the BQ client itself, we create another credential object with the more limited BQ scope.
		bqCreds, err := google.CredentialsFromJSON(ctx, credsJSON, bigqueryapi.Scope)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to create credentials with bigquery scope from JSON: %w", err)
		}
		cred = bqCreds
	}

	// The credentials from FindDefaultCredentials should be passed as an option
	// to the client, along with any other options.
	if cred != nil {
		// If opts already contains credentials (from SA), this will overwrite it with the correctly-scoped one.
		// If using ADC, this will add the found credentials.
		opts = append(opts, option.WithCredentials(cred))
	}

	// Initialize the high-level BigQuery client
	client, err := bigqueryapi.NewClient(ctx, project, opts...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create BigQuery client for project %q: %w", project, err)
	}
	client.Location = location

	// Initialize the low-level BigQuery REST service using the same credentials
	restService, err := bigqueryrestapi.NewService(ctx, opts...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create BigQuery v2 service: %w", err)
	}

	return client, restService, cloudPlatformTokenSource, nil
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

func initDataplexConnection(
	ctx context.Context,
	tracer trace.Tracer,
	name string,
	project string,
	useClientOAuth bool,
	credsJSON []byte,
) (*dataplexapi.CatalogClient, DataplexClientCreator, error) {
	var client *dataplexapi.CatalogClient
	var clientCreator DataplexClientCreator
	var err error
	var opts []option.ClientOption

	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, name)
	defer span.End()

	userAgent, err := util.UserAgentFromContext(ctx)
	if err != nil {
		return nil, nil, err
	}
	opts = append(opts, option.WithUserAgent(userAgent))

	if useClientOAuth {
		clientCreator = newDataplexClientCreator(ctx, project, userAgent)
	} else if len(credsJSON) > 0 {
		opts = append(opts, option.WithCredentialsJSON(credsJSON))
		client, err = dataplexapi.NewCatalogClient(ctx, opts...)
	} else {
		// Use Application Default Credentials
		client, err = dataplexapi.NewCatalogClient(ctx, opts...)
	}

	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Dataplex client for project %q: %w", project, err)
	}

	return client, clientCreator, nil
}

func initDataplexConnectionWithOAuthToken(
	ctx context.Context,
	project string,
	userAgent string,
	tokenString string,
) (*dataplexapi.CatalogClient, error) {
	// Construct token source
	token := &oauth2.Token{
		AccessToken: string(tokenString),
	}
	ts := oauth2.StaticTokenSource(token)

	client, err := dataplexapi.NewCatalogClient(ctx, option.WithUserAgent(userAgent), option.WithTokenSource(ts))
	if err != nil {
		return nil, fmt.Errorf("failed to create Dataplex client for project %q: %w", project, err)
	}
	return client, nil
}

func newDataplexClientCreator(
	ctx context.Context,
	project string,
	userAgent string,
) func(string) (*dataplexapi.CatalogClient, error) {
	return func(tokenString string) (*dataplexapi.CatalogClient, error) {
		return initDataplexConnectionWithOAuthToken(ctx, project, userAgent, tokenString)
	}
}
