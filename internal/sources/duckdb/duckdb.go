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

package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	_ "github.com/marcboeker/go-duckdb/v2"
	"go.opentelemetry.io/otel/trace"
)

const SourceKind string = "duckdb"

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

type Source struct {
	Name string `yaml:"name"`
	Kind string `yaml:"kind"`
	Db   *sql.DB
}

// SourceKind implements sources.Source.
func (s *Source) SourceKind() string {
	return SourceKind
}

func (s *Source) DuckDb() *sql.DB {
	return s.Db
}

// validate Source
var _ sources.Source = &Source{}

type Config struct {
	Name           string            `yaml:"name" validate:"required"`
	Kind           string            `yaml:"kind" validate:"required"`
	DatabaseFile   string            `yaml:"dbFilePath,omitempty"`
	Configurations map[string]string `yaml:"configurations,omitempty"`
}

func (r Config) SourceConfigKind() string {
	return SourceKind
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	db, err := initDuckDbConnection(ctx, tracer, r.Name, r.DatabaseFile, r.Configurations)
	if err != nil {
		return nil, fmt.Errorf("unable to create db connection: %w", err)
	}

	err = db.PingContext(context.Background())
	if err != nil {
		return nil, fmt.Errorf("unable to connect sucessfully: %w", err)
	}

	s := &Source{
		Name: r.Name,
		Kind: r.Kind,
		Db:   db,
	}
	return s, nil
}

// validate interface
var _ sources.SourceConfig = Config{}

func initDuckDbConnection(ctx context.Context, tracer trace.Tracer, name string, dbFilePath string, duckdbConfiguration map[string]string) (*sql.DB, error) {
	//nolint:all // Reassigned ctx
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, name)
	defer span.End()

	var configStr string = getDuckDbConfiguration(dbFilePath, duckdbConfiguration)

	//Open database connection
	db, err := sql.Open("duckdb", configStr)
	if err != nil {
		return nil, fmt.Errorf("unable to open duckdb connection: %w", err)
	}
	return db, nil
}

func getDuckDbConfiguration(dbFilePath string, duckdbConfiguration map[string]string) string {
	if len(duckdbConfiguration) == 0 {
		return dbFilePath
	}

	params := url.Values{}
	for key, value := range duckdbConfiguration {
		params.Set(key, value)
	}

	var configStr strings.Builder
	configStr.WriteString(dbFilePath)
	configStr.WriteString("?")
	configStr.WriteString(params.Encode())

	return configStr.String()
}
