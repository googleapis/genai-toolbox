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

package postgres

import (
	"context"
	"fmt"
	"net/url"
	"sort"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/exp/maps"
)

const SourceKind string = "postgres"

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
	Name     string `yaml:"name" validate:"required"`
	Kind     string `yaml:"kind" validate:"required"`
	Host     string `yaml:"host" validate:"required"`
	Port     string `yaml:"port" validate:"required"`
	User     string `yaml:"user" validate:"required"`
	Password string `yaml:"password" validate:"required"`
	Database string `yaml:"database" validate:"required"`
	// SSLMode is a shortcut for the sslmode query parameter (disable, require, verify-full …).
	// If provided it is added to QueryParams unless the user already set sslmode explicitly.
	SSLMode     string            `yaml:"sslmode"`
	QueryParams map[string]string `yaml:"queryParams"`
}

func (r Config) SourceConfigKind() string {
	return SourceKind
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	qp := maps.Clone(r.QueryParams)
	if r.SSLMode != "" {
		// Do not overwrite if user already specified sslmode in QueryParams
		if _, ok := qp["sslmode"]; !ok {
			qp["sslmode"] = r.SSLMode
		}
	}

	pool, err := initPostgresConnectionPool(ctx, tracer, r.Name, r.Host, r.Port, r.User, r.Password, r.Database, qp)
	if err != nil {
		return nil, fmt.Errorf("unable to create pool: %w", err)
	}

	err = pool.Ping(ctx)
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
	Pool *pgxpool.Pool
}

func (s *Source) SourceKind() string {
	return SourceKind
}

func (s *Source) PostgresPool() *pgxpool.Pool {
	return s.Pool
}

func initPostgresConnectionPool(ctx context.Context, tracer trace.Tracer, name, host, port, user, pass, dbname string, queryParams map[string]string) (*pgxpool.Pool, error) {
	//nolint:all // Reassigned ctx
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, name)
	defer span.End()

	// urlExample := "postgres:dd//username:password@localhost:5432/database_name"
	url := &url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(user, pass),
		Host:     fmt.Sprintf("%s:%s", host, port),
		Path:     dbname,
		RawQuery: ConvertParamMapToRawQuery(queryParams),
	}
	pool, err := pgxpool.New(ctx, url.String())
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	return pool, nil
}

func ConvertParamMapToRawQuery(queryParams map[string]string) string {
	if len(queryParams) == 0 {
		return ""
	}
	keys := make([]string, 0, len(queryParams))
	for k := range queryParams {
		if queryParams[k] != "" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	values := url.Values{}
	for _, k := range keys {
		values.Set(k, queryParams[k])
	}
	return values.Encode()
}
