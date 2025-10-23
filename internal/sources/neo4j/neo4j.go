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

package neo4j

import (
	"context"
	"fmt"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/util"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	neo4jconf "github.com/neo4j/neo4j-go-driver/v5/neo4j/config"
	"go.opentelemetry.io/otel/trace"
)

const SourceKind string = "neo4j"

// validate interface
var _ sources.SourceConfig = Config{}

func init() {
	if !sources.Register(SourceKind, newConfig) {
		panic(fmt.Sprintf("source kind %q already registered", SourceKind))
	}
}

func newConfig(ctx context.Context, name string, decoder *yaml.Decoder) (sources.SourceConfig, error) {
	actual := Config{Name: name, Database: "neo4j"} // Default database
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return nil, err
	}
	return actual, nil
}

type Config struct {
	Name     string `yaml:"name" validate:"required"`
	Kind     string `yaml:"kind" validate:"required"`
	Uri      string `yaml:"uri" validate:"required"`
	User     string `yaml:"user" validate:"required"`
	Password string `yaml:"password" validate:"required"`
	Database string `yaml:"database" validate:"required"`
}

func (r Config) SourceConfigKind() string {
	return SourceKind
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	driver, err := initNeo4jDriver(ctx, tracer, r.Uri, r.User, r.Password, r.Name)
	if err != nil {
		return nil, fmt.Errorf("unable to create driver: %w", err)
	}

	err = driver.VerifyConnectivity(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to connect successfully: %w", err)
	}

	if r.Database == "" {
		r.Database = "neo4j"
	}
	s := &Source{
		Name:     r.Name,
		Kind:     SourceKind,
		Database: r.Database,
		Driver:   driver,
	}
	return s, nil
}

var _ sources.Source = &Source{}

type Source struct {
	Name     string `yaml:"name"`
	Kind     string `yaml:"kind"`
	Database string `yaml:"database"`
	Driver   neo4j.DriverWithContext
}

func (s *Source) SourceKind() string {
	return SourceKind
}

func (s *Source) Neo4jDriver() neo4j.DriverWithContext {
	return s.Driver
}

func (s *Source) Neo4jDatabase() string {
	return s.Database
}

func initNeo4jDriver(ctx context.Context, tracer trace.Tracer, uri, user, password, name string) (neo4j.DriverWithContext, error) {
	//nolint:all // Reassigned ctx
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, name)
	defer span.End()

	auth := neo4j.BasicAuth(user, password, "")
	userAgent, err := util.UserAgentFromContext(ctx)
	if err != nil {
		return nil, err
	}
	driver, err := neo4j.NewDriverWithContext(uri, auth, func(config *neo4jconf.Config) {
		config.UserAgent = userAgent
	})
	if err != nil {
		return nil, fmt.Errorf("unable to create connection driver: %w", err)
	}
	return driver, nil
}
