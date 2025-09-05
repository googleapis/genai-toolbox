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

package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"go.opentelemetry.io/otel/trace"
	_ "modernc.org/sqlite" // Pure Go SQLite driver
)

const SourceType string = "sqlite"

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
	Name     string `yaml:"name" validate:"required"`
	Type     string `yaml:"kind" validate:"required"`
	Database string `yaml:"database" validate:"required"` // Path to SQLite database file
}

func (r Config) SourceConfigType() string {
	return SourceType
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	db, err := initSQLiteConnection(ctx, tracer, r.Name, r.Database)
	if err != nil {
		return nil, fmt.Errorf("unable to create db connection: %w", err)
	}

	err = db.PingContext(context.Background())
	if err != nil {
		return nil, fmt.Errorf("unable to connect successfully: %w", err)
	}

	s := &Source{
		Name: r.Name,
		Type: SourceType,
		Db:   db,
	}
	return s, nil
}

var _ sources.Source = &Source{}

type Source struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`
	Db   *sql.DB
}

func (s *Source) SourceType() string {
	return SourceType
}

func (s *Source) SQLiteDB() *sql.DB {
	return s.Db
}

func initSQLiteConnection(ctx context.Context, tracer trace.Tracer, name, dbPath string) (*sql.DB, error) {
	//nolint:all // Reassigned ctx
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceType, name)
	defer span.End()

	// Open database connection
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}

	// Set some reasonable defaults for SQLite
	db.SetMaxOpenConns(1) // SQLite only supports one writer at a time
	db.SetMaxIdleConns(1)

	return db, nil
}
