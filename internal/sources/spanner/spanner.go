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
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/util"
	"go.opentelemetry.io/otel/trace"
)

const SourceKind string = "spanner"

// validate interface
var _ sources.SourceConfig = Config{}

type Config struct {
	Name     string          `yaml:"name"`
	Kind     string          `yaml:"kind"`
	Project  string          `yaml:"project"`
	Instance string          `yaml:"instance"`
	Dialect  sources.Dialect `yaml:"dialect"`
	Database string          `yaml:"database"`
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
		Name:    r.Name,
		Kind:    SourceKind,
		Client:  client,
		Dialect: r.Dialect.String(),
	}
	return s, nil
}

var _ sources.Source = &Source{}

type Source struct {
	Name    string `yaml:"name"`
	Kind    string `yaml:"kind"`
	Client  *spanner.Client
	Dialect string
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
	client, err := spanner.NewClientWithConfig(context.Background(), db, spanner.ClientConfig{SessionPoolConfig: sessionPoolConfig, UserAgent: ctx.Value(util.UserAgentKey).(string)})
	if err != nil {
		return nil, fmt.Errorf("unable to create new client: %w", err)
	}

	return client, nil
}
