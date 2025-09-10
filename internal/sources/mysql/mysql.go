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

package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"go.opentelemetry.io/otel/trace"
)

const SourceKind string = "mysql"

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
	Name         string            `yaml:"name" validate:"required"`
	Kind         string            `yaml:"kind" validate:"required"`
	Host         string            `yaml:"host" validate:"required"`
	Port         string            `yaml:"port" validate:"required"`
	User         string            `yaml:"user" validate:"required"`
	Password     string            `yaml:"password" validate:"required"`
	Database     string            `yaml:"database" validate:"required"`
	QueryTimeout string            `yaml:"queryTimeout"`
	QueryParams  map[string]string `yaml:"queryParams"`
}

func (r Config) SourceConfigKind() string {
	return SourceKind
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	pool, err := initMySQLConnectionPool(ctx, tracer, r.Name, r.Host, r.Port, r.User, r.Password, r.Database, r.QueryTimeout, r.QueryParams)
	if err != nil {
		return nil, fmt.Errorf("unable to create pool: %w", err)
	}

	err = pool.PingContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to connect successfully: %w", err)
	}

	s := &Source{
		Name: r.Name,
		Kind: SourceKind,
		Pool: pool,
	}
	return s, nil
}

var _ sources.Source = &Source{}

type Source struct {
	Name string `yaml:"name"`
	Kind string `yaml:"kind"`
	Pool *sql.DB
}

func (s *Source) SourceKind() string {
	return SourceKind
}

func (s *Source) MySQLPool() *sql.DB {
	return s.Pool
}

func initMySQLConnectionPool(ctx context.Context, tracer trace.Tracer, name, host, port, user, pass, dbname, queryTimeout string, queryParams map[string]string) (*sql.DB, error) {
	//nolint:all // Reassigned ctx
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, name)
	defer span.End()

	// Build query parameters via url.Values for deterministic order and proper escaping.
	values := url.Values{}

	// Derive readTimeout from queryTimeout when provided.
	if queryTimeout != "" {
		timeout, err := time.ParseDuration(queryTimeout)
		if err != nil {
			return nil, fmt.Errorf("invalid queryTimeout %q: %w", queryTimeout, err)
		}
		values.Set("readTimeout", timeout.String())
	}

	// Custom user parameters
	for k, v := range queryParams {
		if v == "" {
			continue // skip empty values
		}
		values.Set(k, v)
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", user, pass, host, port, dbname)
	if enc := values.Encode(); enc != "" {
		dsn += "&" + enc
	}

	// Interact with the driver directly as you normally would
	pool, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}
	return pool, nil
}
