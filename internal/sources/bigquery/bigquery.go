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

	"strings"

	bigqueryapi "cloud.google.com/go/bigquery"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/util"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/oauth2/google"
	bigqueryrestapi "google.golang.org/api/bigquery/v2"
	"google.golang.org/api/option"
)

const SourceKind string = "bigquery"

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
	// BigQuery configs
	Name     string   `yaml:"name" validate:"required"`
	Kind     string   `yaml:"kind" validate:"required"`
	Project  string   `yaml:"project" validate:"required"`
	Location string   `yaml:"location"`
	Datasets []string `yaml:"datasets"`
}

func (r Config) SourceConfigKind() string {
	// Returns BigQuery source kind
	return SourceKind
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	// Initializes a BigQuery Google SQL source
	client, restService, err := initBigQueryConnection(ctx, tracer, r.Name, r.Project, r.Location)
	if err != nil {
		return nil, err
	}

	allowedDatasets := make(map[string]struct{})
	// Get full id of allowed datasets
	if len(r.Datasets) > 0 {
		for _, allowed := range r.Datasets {
			var allowedFullID string
			if strings.Contains(allowed, ".") {
				allowedFullID = allowed
			} else {
				allowedFullID = fmt.Sprintf("%s.%s", client.Project(), allowed)
			}
			allowedDatasets[allowedFullID] = struct{}{}
		}
	}

	s := &Source{
		Name:            r.Name,
		Kind:            SourceKind,
		Client:          client,
		RestService:     restService,
		Location:        r.Location,
		allowedDatasets: allowedDatasets,
	}
	return s, nil

}

var _ sources.Source = &Source{}

type Source struct {
	// BigQuery Google SQL struct with client
	Name            string `yaml:"name"`
	Kind            string `yaml:"kind"`
	Client          *bigqueryapi.Client
	RestService     *bigqueryrestapi.Service
	Location        string `yaml:"location"`
	allowedDatasets map[string]struct{}
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

// AreDatasetsRestricted returns true if the source is configured with a list of allowed datasets.
func (s *Source) AreDatasetsRestricted() bool {
	return len(s.allowedDatasets) > 0
}

// IsDatasetAllowed checks if a given dataset is accessible based on the source's configuration.
func (s *Source) IsDatasetAllowed(projectID, datasetID string) bool {
	// If the normalized map is empty, it means no restrictions were configured.
	if !s.AreDatasetsRestricted() {
		return true
	}

	targetDataset := fmt.Sprintf("%s.%s", projectID, datasetID)
	_, ok := s.allowedDatasets[targetDataset]
	return ok
}

func initBigQueryConnection(
	ctx context.Context,
	tracer trace.Tracer,
	name string,
	project string,
	location string,
) (*bigqueryapi.Client, *bigqueryrestapi.Service, error) {
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, name)
	defer span.End()

	cred, err := google.FindDefaultCredentials(ctx, bigqueryapi.Scope)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find default Google Cloud credentials with scope %q: %w", bigqueryapi.Scope, err)
	}

	userAgent, err := util.UserAgentFromContext(ctx)
	if err != nil {
		return nil, nil, err
	}

	// Initialize the high-level BigQuery client
	client, err := bigqueryapi.NewClient(ctx, project, option.WithUserAgent(userAgent), option.WithCredentials(cred))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create BigQuery client for project %q: %w", project, err)
	}
	client.Location = location

	// Initialize the low-level BigQuery REST service using the same credentials
	restService, err := bigqueryrestapi.NewService(ctx, option.WithUserAgent(userAgent), option.WithCredentials(cred))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create BigQuery v2 service: %w", err)
	}

	return client, restService, nil
}
