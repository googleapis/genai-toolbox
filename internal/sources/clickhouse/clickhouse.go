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
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
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
	pool, err := initClickHouseConnectionPool(ctx, tracer, r)
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

func (s *Source) RunSQL(ctx context.Context, statement string, params parameters.ParamValues) (any, error) {
	var sliceParams []any
	if params != nil {
		sliceParams = params.AsSlice()
	}
	results, err := s.ClickHousePool().QueryContext(ctx, statement, sliceParams...)
	if err != nil {
		return nil, fmt.Errorf("unable to execute query: %w", err)
	}
	defer results.Close()

	cols, err := results.Columns()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve rows column name: %w", err)
	}

	// create an array of values for each column, which can be re-used to scan each row
	rawValues := make([]any, len(cols))
	values := make([]any, len(cols))
	for i := range rawValues {
		values[i] = &rawValues[i]
	}

	colTypes, err := results.ColumnTypes()
	if err != nil {
		return nil, fmt.Errorf("unable to get column types: %w", err)
	}

	var out []any
	for results.Next() {
		err := results.Scan(values...)
		if err != nil {
			return nil, fmt.Errorf("unable to parse row: %w", err)
		}
		vMap := make(map[string]any)
		for i, name := range cols {
			// ClickHouse driver may return specific types that need handling
			switch colTypes[i].DatabaseTypeName() {
			case "String", "FixedString":
				if rawValues[i] != nil {
					// Handle potential []byte to string conversion if needed
					if b, ok := rawValues[i].([]byte); ok {
						vMap[name] = string(b)
					} else {
						vMap[name] = rawValues[i]
					}
				} else {
					vMap[name] = nil
				}
			default:
				vMap[name] = rawValues[i]
			}
		}
		out = append(out, vMap)
	}

	if err := results.Err(); err != nil {
		return nil, fmt.Errorf("errors encountered by results.Scan: %w", err)
	}

	return out, nil
}

func validateConfig(protocol string) error {
	validProtocols := map[string]bool{"http": true, "https": true}

	if protocol != "" && !validProtocols[protocol] {
		return fmt.Errorf("invalid protocol: %s, must be one of: http, https", protocol)
	}
	return nil
}

func initClickHouseConnectionPool(ctx context.Context, tracer trace.Tracer, config Config) (*sql.DB, error) {
	//nolint:all // Reassigned ctx
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, config.Name)
	defer span.End()

	protocol := config.Protocol
	if protocol == "" {
		protocol = "https"
	}

	if err := validateConfig(protocol); err != nil {
		return nil, err
	}

	encodedUser := url.QueryEscape(config.User)
	encodedPass := url.QueryEscape(config.Password)

	var dsn string
	scheme := protocol
	if protocol == "http" && config.Secure {
		scheme = "https"
	}
	dsn = fmt.Sprintf("%s://%s:%s@%s:%s/%s", scheme, encodedUser, encodedPass, config.Host, config.Port, config.Database)
	if scheme == "https" {
		dsn += "?secure=true&skip_verify=false"
	}

	pool, err := sql.Open("clickhouse", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}

	// Set MaxOpenConns with default value if not specified
	maxOpen := DefaultMaxOpenConns
	if config.MaxOpenConns != nil {
		maxOpen = *config.MaxOpenConns
	}
	pool.SetMaxOpenConns(maxOpen)

	// Set MaxIdleConns with default value if not specified
	maxIdle := DefaultMaxIdleConns
	if config.MaxIdleConns != nil {
		maxIdle = *config.MaxIdleConns
	}
	pool.SetMaxIdleConns(maxIdle)

	// Set ConnMaxLifetime with default value if not specified
	connLifetime := DefaultConnMaxLifetime
	if config.ConnMaxLifetime != "" {
		parsedLifetime, err := time.ParseDuration(config.ConnMaxLifetime)
		if err != nil {
			return nil, fmt.Errorf("invalid connMaxLifetime %q: %w", config.ConnMaxLifetime, err)
		}
		connLifetime = parsedLifetime
	}
	pool.SetConnMaxLifetime(connLifetime)

	return pool, nil
}
