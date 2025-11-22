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

package mariadb

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"go.opentelemetry.io/otel/trace"
)

const SourceKind string = "mariadb"

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
	Name        string            `yaml:"name" validate:"required"`
	Kind        string            `yaml:"kind" validate:"required"`
	Host        string            `yaml:"host" validate:"required"`
	Port        string            `yaml:"port" validate:"required"`
	User        string            `yaml:"user" validate:"required"`
	Password    string            `yaml:"password" validate:"required"`
	Database    string            `yaml:"database" validate:"required"`
	QueryParams map[string]string `yaml:"queryParams"`
}

func (r Config) SourceConfigKind() string {
	return SourceKind
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	client, err := initMariaDB(ctx, tracer, r)
	if err != nil {
		return nil, fmt.Errorf("unable to create pool: %w", err)
	}

	// Verify the connection
	err = client.PingContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to connect successfully: %w", err)
	}

	s := &Source{
		Config: r,
		Db:     client,
	}
	return s, nil
}

var _ sources.Source = &Source{}

type Source struct {
	Config
	Db *sql.DB
}

func (s *Source) SourceKind() string {
	return SourceKind
}

func (s *Source) MySQLPool() *sql.DB {
	return s.Db
}

func (s *Source) ToConfig() sources.SourceConfig {
	return s.Config
}

func initMariaDB(ctx context.Context, tracer trace.Tracer, cfg Config) (*sql.DB, error) {
	_, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, cfg.Name)
	defer span.End()

	q := url.Values{}

	// defaults
	q.Set("parseTime", "true")
	q.Set("clientFoundRows", "true")

	// overrides
	for k, v := range cfg.QueryParams {
		q.Set(k, v)
	}

	// driver accepts additional params in DSN, including "interpolateParams=true&tls=skip-verify"
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?%s",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Database,
		q.Encode(),
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}

	return db, nil
}
