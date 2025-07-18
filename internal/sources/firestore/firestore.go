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

package firestore

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/util"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/api/firebaserules/v1"
	"google.golang.org/api/option"
)

const SourceKind string = "firestore"

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
	// Firestore configs
	Name     string `yaml:"name" validate:"required"`
	Kind     string `yaml:"kind" validate:"required"`
	Project  string `yaml:"project" validate:"required"`
	Database string `yaml:"database"` // Optional, defaults to "(default)"
}

func (r Config) SourceConfigKind() string {
	// Returns Firestore source kind
	return SourceKind
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	// Initializes a Firestore source
	client, err := initFirestoreConnection(ctx, tracer, r.Name, r.Project, r.Database)
	if err != nil {
		return nil, err
	}

	// Initialize Firebase Rules client
	rulesClient, err := initFirebaseRulesConnection(ctx, r.Project)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Firebase Rules client: %w", err)
	}

	s := &Source{
		Name:        r.Name,
		Kind:        SourceKind,
		Client:      client,
		RulesClient: rulesClient,
		ProjectId:   r.Project,
	}
	return s, nil
}

var _ sources.Source = &Source{}

type Source struct {
	// Firestore struct with client
	Name        string `yaml:"name"`
	Kind        string `yaml:"kind"`
	Client      *firestore.Client
	RulesClient *firebaserules.Service
	ProjectId   string `yaml:"projectId"`
}

func (s *Source) SourceKind() string {
	// Returns Firestore source kind
	return SourceKind
}

func (s *Source) FirestoreClient() *firestore.Client {
	return s.Client
}

func (s *Source) FirebaseRulesClient() *firebaserules.Service {
	return s.RulesClient
}

func (s *Source) GetProjectId() string {
	return s.ProjectId
}

func initFirestoreConnection(
	ctx context.Context,
	tracer trace.Tracer,
	name string,
	project string,
	database string,
) (*firestore.Client, error) {
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, name)
	defer span.End()

	userAgent, err := util.UserAgentFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// If database is not specified, use the default database
	if database == "" {
		database = "(default)"
	}

	// Create the Firestore client
	client, err := firestore.NewClientWithDatabase(ctx, project, database, option.WithUserAgent(userAgent))
	if err != nil {
		return nil, fmt.Errorf("failed to create Firestore client for project %q and database %q: %w", project, database, err)
	}

	return client, nil
}

func initFirebaseRulesConnection(
	ctx context.Context,
	project string,
) (*firebaserules.Service, error) {
	// Create the Firebase Rules client
	rulesClient, err := firebaserules.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create Firebase Rules client for project %q: %w", project, err)
	}

	return rulesClient, nil
}
