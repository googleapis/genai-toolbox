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
	Name          string `yaml:"name" validate:"required"`
	Kind          string `yaml:"kind" validate:"required"`
	Project       string `yaml:"project" validate:"required"`
	Location      string `yaml:"location"`
	AllowedDataset string `yaml:"allowedDataset"` // Dataset access restriction: empty=no access, "*"=all BillingReport_ datasets, specific name=only that dataset
}

func (r Config) SourceConfigKind() string {
	// Returns BigQuery source kind
	return SourceKind
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	// Initializes a BigQuery Google SQL source
	client, err := initBigQueryConnection(ctx, tracer, r.Name, r.Project, r.Location)
	if err != nil {
		return nil, err
	}
	s := &Source{
		Name:           r.Name,
		Kind:           SourceKind,
		Client:         client,
		Location:       r.Location,
		AllowedDataset: r.AllowedDataset,
	}
	return s, nil

}

var _ sources.Source = &Source{}

type Source struct {
	// BigQuery Google SQL struct with client
	Name           string `yaml:"name"`
	Kind           string `yaml:"kind"`
	Client         *bigqueryapi.Client
	Location       string `yaml:"location"`
	AllowedDataset string `yaml:"allowedDataset"` // Dataset access restriction from source config
}

func (s *Source) SourceKind() string {
	// Returns BigQuery Google SQL source kind
	return SourceKind
}

func (s *Source) BigQueryClient() *bigqueryapi.Client {
	return s.Client
}

// ValidateDatasetAccess validates if access to the specified dataset is allowed
// based on the source's allowedDataset configuration
func (s *Source) ValidateDatasetAccess(dataset string) error {
	// If no dataset access is configured, deny all queries
	if s.AllowedDataset == "" {
		return fmt.Errorf("no dataset access configured - all queries are denied")
	}

	// Always check if dataset has required "BillingReport_" prefix
	if !strings.HasPrefix(dataset, "BillingReport_") {
		return fmt.Errorf("dataset '%s' does not have required 'BillingReport_' prefix", dataset)
	}

	// If wildcard "*" is specified, allow access to all BillingReport_ datasets
	if s.AllowedDataset == "*" {
		return nil
	}

	// Check if the dataset matches the specifically allowed dataset
	if dataset != s.AllowedDataset {
		return fmt.Errorf("access denied to dataset '%s', only '%s' is allowed", dataset, s.AllowedDataset)
	}

	return nil
}

func initBigQueryConnection(
	ctx context.Context,
	tracer trace.Tracer,
	name string,
	project string,
	location string,
) (*bigqueryapi.Client, error) {
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, name)
	defer span.End()

	cred, err := google.FindDefaultCredentials(ctx, bigqueryapi.Scope)
	if err != nil {
		return nil, fmt.Errorf("failed to find default Google Cloud credentials with scope %q: %w", bigqueryapi.Scope, err)
	}

	userAgent, err := util.UserAgentFromContext(ctx)
	if err != nil {
		return nil, err
	}

	client, err := bigqueryapi.NewClient(ctx, project, option.WithUserAgent(userAgent), option.WithCredentials(cred))
	client.Location = location
	if err != nil {
		return nil, fmt.Errorf("failed to create BigQuery client for project %q: %w", project, err)
	}
	return client, nil
}
