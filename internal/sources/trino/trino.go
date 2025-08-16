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

package trino

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	_ "github.com/trinodb/trino-go-client/trino"
	"go.opentelemetry.io/otel/trace"
)

const SourceKind string = "trino"

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
	Name            string `yaml:"name" validate:"required"`
	Kind            string `yaml:"kind" validate:"required"`
	Host            string `yaml:"host" validate:"required"`
	Port            string `yaml:"port" validate:"required"`
	User            string `yaml:"user" validate:"required"`
	Password        string `yaml:"password"`
	Catalog         string `yaml:"catalog" validate:"required"`
	Schema          string `yaml:"schema" validate:"required"`
	QueryTimeout    string `yaml:"queryTimeout"`
	AccessToken     string `yaml:"accessToken"`
	KerberosEnabled bool   `yaml:"kerberosEnabled"`
	SSLEnabled      bool   `yaml:"sslEnabled"`
}

func (r Config) SourceConfigKind() string {
	return SourceKind
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	pool, err := initTrinoConnectionPool(ctx, tracer, r.Name, r.Host, r.Port, r.User, r.Password, r.Catalog, r.Schema, r.QueryTimeout, r.AccessToken, r.KerberosEnabled, r.SSLEnabled)
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

func (s *Source) TrinoDB() *sql.DB {
	return s.Pool
}

func initTrinoConnectionPool(ctx context.Context, tracer trace.Tracer, name, host, port, user, password, catalog, schema, queryTimeout, accessToken string, kerberosEnabled, sslEnabled bool) (*sql.DB, error) {
	//nolint:all // Reassigned ctx
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, name)
	defer span.End()

	// Build Trino DSN
	dsn, err := buildTrinoDSN(host, port, user, password, catalog, schema, queryTimeout, accessToken, kerberosEnabled, sslEnabled)
	if err != nil {
		return nil, fmt.Errorf("failed to build DSN: %w", err)
	}

	db, err := sql.Open("trino", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	return db, nil
}

func buildTrinoDSN(host, port, user, password, catalog, schema, queryTimeout, accessToken string, kerberosEnabled, sslEnabled bool) (string, error) {
	// Build query parameters
	query := url.Values{}
	query.Set("catalog", catalog)
	query.Set("schema", schema)
	if queryTimeout != "" {
		query.Set("queryTimeout", queryTimeout)
	}
	if accessToken != "" {
		query.Set("accessToken", accessToken)
	}
	if kerberosEnabled {
		query.Set("KerberosEnabled", "true")
	}

	// Build URL
	scheme := "http"
	if sslEnabled {
		scheme = "https"
	}

	u := &url.URL{
		Scheme:   scheme,
		User:     url.User(user),
		Host:     fmt.Sprintf("%s:%s", host, port),
		RawQuery: query.Encode(),
	}

	if password != "" {
		// For basic auth, set password in userinfo
		u.User = url.UserPassword(user, password)
	}
	return u.String(), nil
}
