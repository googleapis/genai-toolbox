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

package clickhouse

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"time"

	_ "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"go.opentelemetry.io/otel/trace"
)

const SourceKind string = "clickhouse"

const (
	// DefaultMaxOpenConns is the default maximum number of open connections to the database.
	DefaultMaxOpenConns = 25
	// DefaultMaxIdleConns is the default maximum number of idle connections in the pool.
	DefaultMaxIdleConns = 5
	// DefaultConnMaxLifetime is the default maximum lifetime of a connection.
	DefaultConnMaxLifetime = 5 * time.Minute
)

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
	Name             string `yaml:"name" validate:"required"`
	Kind             string `yaml:"kind" validate:"required"`
	Host             string `yaml:"host" validate:"required"`
	Port             string `yaml:"port" validate:"required"`
	Database         string `yaml:"database" validate:"required"`
	User             string `yaml:"user" validate:"required"`
	Password         string `yaml:"password"`
	Protocol         string `yaml:"protocol"`
	Secure           bool   `yaml:"secure"`
	MaxOpenConns     *int   `yaml:"maxOpenConns" validate:"omitempty,gt=0"`
	MaxIdleConns     *int   `yaml:"maxIdleConns" validate:"omitempty,gt=0"`
	ConnMaxLifetime  string `yaml:"connMaxLifetime"`
}

func (r Config) SourceConfigKind() string {
	return SourceKind
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	pool, err := initClickHouseConnectionPool(ctx, tracer, r.Name, r.Host, r.Port, r.User, r.Password, r.Database, r.Protocol, r.Secure, r.MaxOpenConns, r.MaxIdleConns, r.ConnMaxLifetime)
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

type Source struct {
	Config
	Pool *sql.DB
}

func (s *Source) SourceKind() string {
	return SourceKind
}

func (s *Source) ToConfig() sources.SourceConfig {
	return s.Config
}

func (s *Source) ClickHousePool() *sql.DB {
	return s.Pool
}

func validateConfig(protocol string) error {
	validProtocols := map[string]bool{"http": true, "https": true}

	if protocol != "" && !validProtocols[protocol] {
		return fmt.Errorf("invalid protocol: %s, must be one of: http, https", protocol)
	}
	return nil
}

func initClickHouseConnectionPool(ctx context.Context, tracer trace.Tracer, name, host, port, user, pass, dbname, protocol string, secure bool, maxOpenConns, maxIdleConns *int, connMaxLifetime string) (*sql.DB, error) {
	//nolint:all // Reassigned ctx
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, name)
	defer span.End()

	if protocol == "" {
		protocol = "https"
	}

	if err := validateConfig(protocol); err != nil {
		return nil, err
	}

	encodedUser := url.QueryEscape(user)
	encodedPass := url.QueryEscape(pass)

	var dsn string
	scheme := protocol
	if protocol == "http" && secure {
		scheme = "https"
	}
	dsn = fmt.Sprintf("%s://%s:%s@%s:%s/%s", scheme, encodedUser, encodedPass, host, port, dbname)
	if scheme == "https" {
		dsn += "?secure=true&skip_verify=false"
	}

	pool, err := sql.Open("clickhouse", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}

	// Set MaxOpenConns with default value if not specified
	maxOpen := DefaultMaxOpenConns
	if maxOpenConns != nil {
		maxOpen = *maxOpenConns
	}
	pool.SetMaxOpenConns(maxOpen)

	// Set MaxIdleConns with default value if not specified
	maxIdle := DefaultMaxIdleConns
	if maxIdleConns != nil {
		maxIdle = *maxIdleConns
	}
	pool.SetMaxIdleConns(maxIdle)

	// Set ConnMaxLifetime with default value if not specified
	connLifetime := DefaultConnMaxLifetime
	if connMaxLifetime != "" {
		parsedLifetime, err := time.ParseDuration(connMaxLifetime)
		if err != nil {
			return nil, fmt.Errorf("invalid connMaxLifetime %q: %w", connMaxLifetime, err)
		}
		connLifetime = parsedLifetime
	}
	pool.SetConnMaxLifetime(connLifetime)

	return pool, nil
}
