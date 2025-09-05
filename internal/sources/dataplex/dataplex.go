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

package dataplex

import (
	"context"
	"fmt"

	dataplexapi "cloud.google.com/go/dataplex/apiv1"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/util"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

const SourceType string = "dataplex"

// validate interface
var _ sources.SourceConfig = Config{}

func init() {
	if !sources.Register(SourceType, newConfig) {
		panic(fmt.Sprintf("source type %q already registered", SourceType))
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
	// Dataplex configs
	Name    string `yaml:"name" validate:"required"`
	Type    string `yaml:"kind" validate:"required"`
	Project string `yaml:"project" validate:"required"`
}

func (r Config) SourceConfigType() string {
	// Returns Dataplex source type
	return SourceType
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	// Initializes a Dataplex source
	client, err := initDataplexConnection(ctx, tracer, r.Name, r.Project)
	if err != nil {
		return nil, err
	}
	s := &Source{
		Name:    r.Name,
		Type:    SourceType,
		Client:  client,
		Project: r.Project,
	}

	return s, nil
}

var _ sources.Source = &Source{}

type Source struct {
	// Source struct with Dataplex client
	Name     string `yaml:"name"`
	Type     string `yaml:"type"`
	Client   *dataplexapi.CatalogClient
	Project  string `yaml:"project"`
	Location string `yaml:"location"`
}

func (s *Source) SourceType() string {
	// Returns Dataplex source type
	return SourceType
}

func (s *Source) ProjectID() string {
	return s.Project
}

func (s *Source) CatalogClient() *dataplexapi.CatalogClient {
	return s.Client
}

func initDataplexConnection(
	ctx context.Context,
	tracer trace.Tracer,
	name string,
	project string,
) (*dataplexapi.CatalogClient, error) {
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceType, name)
	defer span.End()

	cred, err := google.FindDefaultCredentials(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find default Google Cloud credentials: %w", err)
	}

	userAgent, err := util.UserAgentFromContext(ctx)
	if err != nil {
		return nil, err
	}

	client, err := dataplexapi.NewCatalogClient(ctx, option.WithUserAgent(userAgent), option.WithCredentials(cred))
	if err != nil {
		return nil, fmt.Errorf("failed to create Dataplex client for project %q: %w", project, err)
	}
	return client, nil
}
