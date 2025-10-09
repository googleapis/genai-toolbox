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
	"strings"
	"sync"
	"time"

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

const (
	// No write operations are allowed.
	WriteModeBlocked string = "blocked"
	// Only protected write operations are allowed in a BigQuery session.
	WriteModeProtected string = "protected"
	// All write operations are allowed.
	WriteModeAllowed string = "allowed"
)

// validate interface
var _ sources.SourceConfig = Config{}

type BigqueryClientCreator func(tokenString string, wantRestService bool) (*bigqueryapi.Client, *bigqueryrestapi.Service, error)

type BigQuerySessionProvider func(ctx context.Context) (*Session, error)

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
	WriteMode       string   `yaml:"writeMode"`
	AllowedDatasets []string `yaml:"allowedDatasets"`
	UseClientOAuth  bool     `yaml:"useClientOAuth"`
}

func (r Config) SourceConfigKind() string {
	// Returns BigQuery source kind
	return SourceKind
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	if r.WriteMode == "" {
		r.WriteMode = WriteModeAllowed
	}

	if r.WriteMode == WriteModeProtected && r.UseClientOAuth {
		return nil, fmt.Errorf("writeMode 'protected' cannot be used with useClientOAuth 'true'")
	}

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
				projectID = r.Project
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
		Name:               r.Name,
		Kind:               SourceKind,
		Project:            r.Project,
		Location:           r.Location,
		Client:             client,
		RestService:        restService,
		TokenSource:        tokenSource,
		MaxQueryResultRows: 50,
		ClientCreator:      clientCreator,
		WriteMode:          r.WriteMode,
		AllowedDatasets:    allowedDatasets,
		UseClientOAuth:     r.UseClientOAuth,
	}
	s.SessionProvider = s.newBigQuerySessionProvider()

	if r.WriteMode != WriteModeAllowed && r.WriteMode != WriteModeBlocked && r.WriteMode != WriteModeProtected {
		return nil, fmt.Errorf("invalid writeMode %q: must be one of %q, %q, or %q", r.WriteMode, WriteModeAllowed, WriteModeProtected, WriteModeBlocked)
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
	Client                    *bigqueryapi.Client
	RestService               *bigqueryrestapi.Service
	TokenSource               oauth2.TokenSource
	MaxQueryResultRows        int
	ClientCreator             BigqueryClientCreator
	AllowedDatasets           map[string]struct{}
	UseClientOAuth            bool
	WriteMode                 string
	sessionMutex              sync.Mutex
	makeDataplexCatalogClient func() (*dataplexapi.CatalogClient, DataplexClientCreator, error)
	SessionProvider           BigQuerySessionProvider
	Session                   *Session
}

type Session struct {
	ID           string
	ProjectID    string
	DatasetID    string
	CreationTime time.Time
	LastUsed     time.Time
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

func (s *Source) BigQueryWriteMode() string {
	return s.WriteMode
}

func (s *Source) BigQuerySession() BigQuerySessionProvider {
	return s.SessionProvider
}

func (s *Source) newBigQuerySessionProvider() BigQuerySessionProvider {
	return func(ctx context.Context) (*Session, error) {
		if s.WriteMode != WriteModeProtected {
			return nil, nil
		}

		s.sessionMutex.Lock()
		defer s.sessionMutex.Unlock()

		logger, err := util.LoggerFromContext(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get logger from context: %w", err)
		}

		if s.Session != nil {
			// Absolute 7-day lifetime check.
			const sessionMaxLifetime = 7 * 24 * time.Hour
			// This assumes a single task will not exceed 30 minutes, preventing it from failing mid-execution.
			const refreshThreshold = 30 * time.Minute
			if time.Since(s.Session.CreationTime) > (sessionMaxLifetime - refreshThreshold) {
				logger.DebugContext(ctx, "Session is approaching its 7-day maximum lifetime. Creating a new one.")
			} else {
				job := &bigqueryrestapi.Job{
					Configuration: &bigqueryrestapi.JobConfiguration{
						DryRun: true,
						Query: &bigqueryrestapi.JobConfigurationQuery{
							Query:                "SELECT 1",
							UseLegacySql:         new(bool),
							ConnectionProperties: []*bigqueryrestapi.ConnectionProperty{{Key: "session_id", Value: s.Session.ID}},
						},
					},
				}
				_, err := s.RestService.Jobs.Insert(s.Project, job).Do()
				if err == nil {
					s.Session.LastUsed = time.Now()
					return s.Session, nil
				}
				logger.DebugContext(ctx, "Session validation failed (likely expired), creating a new one.", "error", err)
			}
		}

		// Create a new session if one doesn't exist, it has passed its 7-day lifetime,
		// or it failed the validation dry run.

		creationTime := time.Now()
		job := &bigqueryrestapi.Job{
			JobReference: &bigqueryrestapi.JobReference{
				ProjectId: s.Project,
				Location:  s.Location,
			},
			Configuration: &bigqueryrestapi.JobConfiguration{
				DryRun: true,
				Query: &bigqueryrestapi.JobConfigurationQuery{
					Query:         "SELECT 1",
					CreateSession: true,
				},
			},
		}

		createdJob, err := s.RestService.Jobs.Insert(s.Project, job).Do()
		if err != nil {
			return nil, fmt.Errorf("failed to create new session: %w", err)
		}

		var sessionID, sessionDatasetID, projectID string
		if createdJob.Status != nil && createdJob.Statistics.SessionInfo != nil {
			sessionID = createdJob.Statistics.SessionInfo.SessionId
		} else {
			return nil, fmt.Errorf("failed to get session ID from new session job")
		}

		if createdJob.Configuration != nil && createdJob.Configuration.Query != nil && createdJob.Configuration.Query.DestinationTable != nil {
			sessionDatasetID = createdJob.Configuration.Query.DestinationTable.DatasetId
			projectID = createdJob.Configuration.Query.DestinationTable.ProjectId
		} else {
			return nil, fmt.Errorf("failed to get session dataset ID from new session job")
		}

		s.Session = &Session{
			ID:           sessionID,
			ProjectID:    projectID,
			DatasetID:    sessionDatasetID,
			CreationTime: creationTime,
			LastUsed:     creationTime,
		}
		return s.Session, nil
	}
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

func (s *Source) BigQueryTokenSourceWithScope(ctx context.Context, scope string) (oauth2.TokenSource, error) {
	return google.DefaultTokenSource(ctx, scope)
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
			c, cc, e := initDataplexConnection(ctx, tracer, s.Name, s.Project, s.UseClientOAuth)
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
) (*bigqueryapi.Client, *bigqueryrestapi.Service, oauth2.TokenSource, error) {
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, name)
	defer span.End()

	cred, err := google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
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

func initDataplexConnection(
	ctx context.Context,
	tracer trace.Tracer,
	name string,
	project string,
	useClientOAuth bool,
) (*dataplexapi.CatalogClient, DataplexClientCreator, error) {
	var client *dataplexapi.CatalogClient
	var clientCreator DataplexClientCreator
	var err error

	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, name)
	defer span.End()

	cred, err := google.FindDefaultCredentials(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find default Google Cloud credentials: %w", err)
	}

	userAgent, err := util.UserAgentFromContext(ctx)
	if err != nil {
		return nil, nil, err
	}

	if useClientOAuth {
		clientCreator = newDataplexClientCreator(ctx, project, userAgent)
	} else {
		client, err = dataplexapi.NewCatalogClient(ctx, option.WithUserAgent(userAgent), option.WithCredentials(cred))
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create Dataplex client for project %q: %w", project, err)
		}
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
