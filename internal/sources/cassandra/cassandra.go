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

package cassandra

import (
	"context"
	"fmt"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"go.opentelemetry.io/otel/trace"
)

const SourceKind string = "cassandra"

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
	Name string `yaml:"name" validate:"required"`
	Kind string `yaml:"kind" validate:"required"`
	Uri  string `yaml:"uri" validate:"required"` // Cassandra Connection IP
}

func (r Config) SourceConfigKind() string {
	return SourceKind
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	session, err := initCassandraSession(ctx, tracer, r.Name, r.Uri)
	if err != nil {
		return nil, fmt.Errorf("unable to create session: %w", err)
	}

	hosts := session.GetHosts()
	if len(hosts) == 0 {
		return nil, fmt.Errorf("unable to start session successfully")
	}

	s := &Source{
		Name:    r.Name,
		Kind:    SourceKind,
		Session: session,
	}

	return s, nil
}

var _ sources.Source = &Source{}

type Source struct {
	Name    string `yaml:"name"`
	Kind    string `yaml:"kind"`
	Session *gocql.Session
}

func (s *Source) SourceKind() string {
	return SourceKind
}

func (s *Source) CassandraSession() *gocql.Session {
	return s.Session
}

func initCassandraSession(ctx context.Context, tracer trace.Tracer, name, uri string) (*gocql.Session, error) {
	// Start a tracing span
	_, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, name)
	defer span.End()

	// Create a new Cassandra Session
	cluster := gocql.NewCluster(uri)
	session, err := cluster.CreateSession()
	if err != nil {
		return nil, fmt.Errorf("unable to create Cassandra session: %w", err)
	}

	return session, nil
}
