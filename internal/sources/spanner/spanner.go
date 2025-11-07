// Copyright 2024 Google LLC
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

package spanner

import (
	"context"
	"fmt"

	"cloud.google.com/go/spanner"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/util"
	"go.opentelemetry.io/otel/trace"
)

const SourceKind string = "spanner"

// validate interface
var _ sources.SourceConfig = Config{}

func init() {
	if !sources.Register(SourceKind, newConfig) {
		panic(fmt.Sprintf("source kind %q already registered", SourceKind))
	}
}

func newConfig(ctx context.Context, name string, decoder *yaml.Decoder) (sources.SourceConfig, error) {
	actual := Config{Name: name, Dialect: "googlesql"} // Default dialect
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return nil, err
	}
	return actual, nil
}

type Config struct {
	Name     string          `yaml:"name" validate:"required"`
	Kind     string          `yaml:"kind" validate:"required"`
	Project  string          `yaml:"project" validate:"required"`
	Instance string          `yaml:"instance" validate:"required"`
	Dialect  sources.Dialect `yaml:"dialect" validate:"required"`
	Database string          `yaml:"database" validate:"required"`
}

func (r Config) SourceConfigKind() string {
	return SourceKind
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	client, err := initSpannerClient(ctx, tracer, r.Name, r.Project, r.Instance, r.Database)
	if err != nil {
		return nil, fmt.Errorf("unable to create client: %w", err)
	}

	s := &Source{
		Name:       r.Name,
		Kind:       SourceKind,
		Client:     client,
		Dialect:    r.Dialect.String(),
		ProjectID:  r.Project,
		InstanceID: r.Instance,
		DatabaseID: r.Database,
	}
	return s, nil
}

var _ sources.Source = &Source{}

type Source struct {
	Name       string `yaml:"name"`
	Kind       string `yaml:"kind"`
	Client     *spanner.Client
	Dialect    string
	ProjectID  string
	InstanceID string
	DatabaseID string
}

func (s *Source) SourceKind() string {
	return SourceKind
}

func (s *Source) SpannerClient() *spanner.Client {
	return s.Client
}

func (s *Source) DatabaseDialect() string {
	return s.Dialect
}

func (s *Source) SpannerProjectID() string {
	return s.ProjectID
}

func (s *Source) SpannerInstanceID() string {
	return s.InstanceID
}

func (s *Source) SpannerDatabaseID() string {
	return s.DatabaseID
}

func initSpannerClient(ctx context.Context, tracer trace.Tracer, name, project, instance, dbname string) (*spanner.Client, error) {
	//nolint:all // Reassigned ctx
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, name)
	defer span.End()

	// Configure the connection to the database
	db := fmt.Sprintf("projects/%s/instances/%s/databases/%s", project, instance, dbname)

	// Configure session pool to automatically clean inactive transactions
	sessionPoolConfig := spanner.SessionPoolConfig{
		TrackSessionHandles: true,
		InactiveTransactionRemovalOptions: spanner.InactiveTransactionRemovalOptions{
			ActionOnInactiveTransaction: spanner.WarnAndClose,
		},
	}

	// Create spanner client
	userAgent, err := util.UserAgentFromContext(ctx)
	if err != nil {
		return nil, err
	}
	client, err := spanner.NewClientWithConfig(ctx, db, spanner.ClientConfig{SessionPoolConfig: sessionPoolConfig, UserAgent: userAgent})
	if err != nil {
		return nil, fmt.Errorf("unable to create new client: %w", err)
	}

	return client, nil
}
