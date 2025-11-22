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

package cockroachdb

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"time"

	crdbpgx "github.com/cockroachdb/cockroach-go/v2/crdb/crdbpgxv5"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/util"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/trace"
)

const SourceKind string = "cockroachdb"

var _ sources.SourceConfig = Config{}

func init() {
	if !sources.Register(SourceKind, newConfig) {
		panic(fmt.Sprintf("source kind %q already registered", SourceKind))
	}
}

func newConfig(ctx context.Context, name string, decoder *yaml.Decoder) (sources.SourceConfig, error) {
	actual := Config{Name: name, MaxRetries: 5, RetryBaseDelay: "500ms"}
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return nil, err
	}
	return actual, nil
}

type Config struct {
	Name           string            `yaml:"name" validate:"required"`
	Kind           string            `yaml:"kind" validate:"required"`
	Host           string            `yaml:"host" validate:"required"`
	Port           string            `yaml:"port" validate:"required"`
	User           string            `yaml:"user" validate:"required"`
	Password       string            `yaml:"password"`
	Database       string            `yaml:"database" validate:"required"`
	QueryParams    map[string]string `yaml:"queryParams"`
	MaxRetries     int               `yaml:"maxRetries"`
	RetryBaseDelay string            `yaml:"retryBaseDelay"`
}

func (r Config) SourceConfigKind() string {
	return SourceKind
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	retryBaseDelay, err := time.ParseDuration(r.RetryBaseDelay)
	if err != nil {
		return nil, fmt.Errorf("invalid retryBaseDelay: %w", err)
	}

	pool, err := initCockroachDBConnectionPoolWithRetry(ctx, tracer, r.Name, r.Host, r.Port, r.User, r.Password, r.Database, r.QueryParams, r.MaxRetries, retryBaseDelay)
	if err != nil {
		return nil, fmt.Errorf("unable to create pool: %w", err)
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
	Pool *pgxpool.Pool
}

func (s *Source) SourceKind() string {
	return SourceKind
}

func (s *Source) ToConfig() sources.SourceConfig {
	return s.Config
}

func (s *Source) CockroachDBPool() *pgxpool.Pool {
	return s.Pool
}

func (s *Source) PostgresPool() *pgxpool.Pool {
	return s.Pool
}

// ExecuteTxWithRetry executes a function within a transaction with automatic retry logic
// using the official CockroachDB retry mechanism from cockroach-go/v2
func (s *Source) ExecuteTxWithRetry(ctx context.Context, fn func(pgx.Tx) error) error {
	return crdbpgx.ExecuteTx(ctx, s.Pool, pgx.TxOptions{}, fn)
}

// Query executes a query using the connection pool.
// For read-only queries, connection-level retry is sufficient.
// For write operations requiring transaction retry, use ExecuteTxWithRetry directly.
func (s *Source) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return s.Pool.Query(ctx, sql, args...)
}

func initCockroachDBConnectionPoolWithRetry(ctx context.Context, tracer trace.Tracer, name, host, port, user, pass, dbname string, queryParams map[string]string, maxRetries int, baseDelay time.Duration) (*pgxpool.Pool, error) {
	//nolint:all
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, name)
	defer span.End()

	userAgent, err := util.UserAgentFromContext(ctx)
	if err != nil {
		userAgent = "genai-toolbox"
	}
	if queryParams == nil {
		queryParams = make(map[string]string)
	}
	if _, ok := queryParams["application_name"]; !ok {
		queryParams["application_name"] = userAgent
	}

	connURL := &url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(user, pass),
		Host:     fmt.Sprintf("%s:%s", host, port),
		Path:     dbname,
		RawQuery: ConvertParamMapToRawQuery(queryParams),
	}

	var pool *pgxpool.Pool
	for attempt := 0; attempt <= maxRetries; attempt++ {
		pool, err = pgxpool.New(ctx, connURL.String())
		if err == nil {
			err = pool.Ping(ctx)
		}

		if err == nil {
			return pool, nil
		}

		if attempt < maxRetries {
			backoff := baseDelay * time.Duration(math.Pow(2, float64(attempt)))
			time.Sleep(backoff)
		}
	}

	return nil, fmt.Errorf("failed to connect to CockroachDB after %d retries: %w", maxRetries, err)
}

func ConvertParamMapToRawQuery(queryParams map[string]string) string {
	values := url.Values{}
	for k, v := range queryParams {
		values.Add(k, v)
	}
	return values.Encode()
}
