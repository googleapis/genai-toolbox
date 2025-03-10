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
	"database/sql"
	"fmt"

	_ "github.com/googleapis/go-sql-spanner"

	"github.com/googleapis/genai-toolbox/internal/sources"
	"go.opentelemetry.io/otel/trace"
)

const SourceKind string = "spanner"

// validate interface
var _ sources.SourceConfig = Config{}

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
	// Initializes a Spanner source
	db, err := initSpannerDb(ctx, tracer, r.Name, r.Project, r.Instance, r.Database)
	if err != nil {
		return nil, fmt.Errorf("unable to create db connection: %w", err)
	}

	// Verify db connection
	err = db.PingContext(context.Background())
	if err != nil {
		return nil, fmt.Errorf("unable to connect successfully: %w", err)
	}

	s := &Source{
		Name:    r.Name,
		Kind:    SourceKind,
		Db:      db,
		Dialect: r.Dialect.String(),
	}
	return s, nil
}

var _ sources.Source = &Source{}

type Source struct {
	Name    string `yaml:"name"`
	Kind    string `yaml:"kind"`
	Db      *sql.DB
	Dialect string
}

func (s *Source) SourceKind() string {
	return SourceKind
}

func (s *Source) SpannerDb() *sql.DB {
	return s.Db
}

func (s *Source) DatabaseDialect() string {
	return s.Dialect
}

func initSpannerDb(ctx context.Context, tracer trace.Tracer, name, project, instance, dbname string) (*sql.DB, error) {
	//nolint:all // Reassigned ctx
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, name)
	defer span.End()

	// Create DSN
	dsn := fmt.Sprintf("projects/%s/instances/%s/databases/%s", project, instance, dbname)

	// Open DB connection
	db, err := sql.Open("spanner", dsn)
	if err != nil {
		return nil, err
	}

	return db, nil
}
