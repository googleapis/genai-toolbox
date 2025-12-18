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

package singlestore

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"go.opentelemetry.io/otel/trace"
)

// SourceKind for SingleStore source
const SourceKind string = "singlestore"

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

// Config holds the configuration parameters for connecting to a SingleStore database.
type Config struct {
	Name         string `yaml:"name" validate:"required"`
	Kind         string `yaml:"kind" validate:"required"`
	Host         string `yaml:"host" validate:"required"`
	Port         string `yaml:"port" validate:"required"`
	User         string `yaml:"user" validate:"required"`
	Password     string `yaml:"password" validate:"required"`
	Database     string `yaml:"database" validate:"required"`
	QueryTimeout string `yaml:"queryTimeout"`
}

// SourceConfigKind returns the kind of the source configuration.
func (r Config) SourceConfigKind() string {
	return SourceKind
}

// Initialize sets up the SingleStore connection pool and returns a Source.
func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	pool, err := initSingleStoreConnectionPool(ctx, tracer, r.Name, r.Host, r.Port, r.User, r.Password, r.Database, r.QueryTimeout)
	if err != nil {
		return nil, fmt.Errorf("unable to create pool: %w", err)
	}

	err = pool.PingContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to connect successfully: %w", err)
	}

	s := &Source{
		Config: r,
		Pool:   pool,
	}
	return s, nil
}

var _ sources.Source = &Source{}

// Source represents a SingleStore database source and holds its connection pool.
type Source struct {
	Config
	Pool *sql.DB
}

// SourceKind returns the kind of the source configuration.
func (s *Source) SourceKind() string {
	return SourceKind
}

func (s *Source) ToConfig() sources.SourceConfig {
	return s.Config
}

// SingleStorePool returns the underlying *sql.DB connection pool for SingleStore.
func (s *Source) SingleStorePool() *sql.DB {
	return s.Pool
}

func initSingleStoreConnectionPool(ctx context.Context, tracer trace.Tracer, name, host, port, user, pass, dbname, queryTimeout string) (*sql.DB, error) {
	//nolint:all // Reassigned ctx
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, name)
	defer span.End()

	// Configure the driver to connect to the database
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&vector_type_project_format=JSON", user, pass, host, port, dbname)

	// Add connection attributes to DSN
	customAttrs := []string{"_connector_name"}
	customAttrValues := []string{"MCP toolbox for Databases"}

	customAttrStrs := make([]string, len(customAttrs))
	for i := range customAttrs {
		customAttrStrs[i] = fmt.Sprintf("%s:%s", customAttrs[i], customAttrValues[i])
	}
	dsn += "&connectionAttributes=" + url.QueryEscape(strings.Join(customAttrStrs, ","))

	// Add query timeout to DSN if specified
	if queryTimeout != "" {
		timeout, err := time.ParseDuration(queryTimeout)
		if err != nil {
			return nil, fmt.Errorf("invalid queryTimeout %q: %w", queryTimeout, err)
		}
		dsn += "&readTimeout=" + timeout.String()
	}

	// Interact with the driver directly as you normally would
	pool, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}
	return pool, nil
}
