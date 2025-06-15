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
	Name        string `yaml:"name" validate:"required"`
	Kind        string `yaml:"kind" validate:"required"`
	Host        string `yaml:"host" validate:"required"`
	Port        string `yaml:"port" validate:"required"`
	User        string `yaml:"user" validate:"required"`
	Password    string `yaml:"password"`
	Database    string `yaml:"database" validate:"required"`
	Protocol    string `yaml:"protocol"` // native, http, https
	Secure      bool   `yaml:"secure"`
	Compression string `yaml:"compression"` // lz4, zstd, gzip, none
}

func (r Config) SourceConfigKind() string {
	return SourceKind
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	pool, err := initClickHouseConnectionPool(ctx, tracer, r.Name, r.Host, r.Port, r.User, r.Password, r.Database, r.Protocol, r.Secure, r.Compression)
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

func (s *Source) ClickHousePool() *sql.DB {
	return s.Pool
}

func validateConfig(protocol, compression string) error {
	validProtocols := map[string]bool{"native": true, "http": true, "https": true}
	validCompression := map[string]bool{"lz4": true, "zstd": true, "gzip": true, "none": true}

	if protocol != "" && !validProtocols[protocol] {
		return fmt.Errorf("invalid protocol: %s, must be one of: native, http, https", protocol)
	}
	if compression != "" && !validCompression[compression] {
		return fmt.Errorf("invalid compression: %s, must be one of: lz4, zstd, gzip, none", compression)
	}
	return nil
}

func initClickHouseConnectionPool(ctx context.Context, tracer trace.Tracer, name, host, port, user, pass, dbname, protocol string, secure bool, compression string) (*sql.DB, error) {
	//nolint:all // Reassigned ctx
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, name)
	defer span.End()

	// Set default protocol if not specified
	if protocol == "" {
		protocol = "native"
	}

	// Set default compression if not specified
	if compression == "" {
		compression = "lz4"
	}

	// Validate configuration parameters
	if err := validateConfig(protocol, compression); err != nil {
		return nil, err
	}

	// URL encode credentials to prevent injection and handle special characters
	encodedUser := url.QueryEscape(user)
	encodedPass := url.QueryEscape(pass)

	// Build DSN based on protocol
	var dsn string
	switch protocol {
	case "http", "https":
		scheme := protocol
		if protocol == "http" && secure {
			scheme = "https"
		}
		dsn = fmt.Sprintf("%s://%s:%s@%s:%s/%s?compress=%s", scheme, encodedUser, encodedPass, host, port, dbname, compression)
	default: // native
		dsn = fmt.Sprintf("clickhouse://%s:%s@%s:%s/%s?compress=%s", encodedUser, encodedPass, host, port, dbname, compression)
		if secure {
			dsn += "&secure=true"
		}
	}

	pool, err := sql.Open("clickhouse", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}

	// Configure connection pool limits for better resource management
	pool.SetMaxOpenConns(25)
	pool.SetMaxIdleConns(5)
	pool.SetConnMaxLifetime(5 * time.Minute)

	return pool, nil
}
