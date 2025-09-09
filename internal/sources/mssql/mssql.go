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

package mssql

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	_ "github.com/microsoft/go-mssqldb"
	"go.opentelemetry.io/otel/trace"
)

const SourceType string = "mssql"

// validate interface
var _ sources.SourceConfig = Config{}

func init() {
	if !sources.Register(SourceType, newConfig) {
		panic(fmt.Sprintf("source type %q already registered", SourceType))
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
	// Cloud SQL MSSQL configs
	Name     string `yaml:"name" validate:"required"`
	Type     string `yaml:"type" validate:"required"`
	Host     string `yaml:"host" validate:"required"`
	Port     string `yaml:"port" validate:"required"`
	User     string `yaml:"user" validate:"required"`
	Password string `yaml:"password" validate:"required"`
	Database string `yaml:"database" validate:"required"`
	Encrypt  string `yaml:"encrypt"`
}

func (r Config) SourceConfigType() string {
	// Returns Cloud SQL MSSQL source type
	return SourceType
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	// Initializes a MSSQL source
	db, err := initMssqlConnection(ctx, tracer, r.Name, r.Host, r.Port, r.User, r.Password, r.Database, r.Encrypt)
	if err != nil {
		return nil, fmt.Errorf("unable to create db connection: %w", err)
	}

	// Verify db connection
	err = db.PingContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to connect successfully: %w", err)
	}

	s := &Source{
		Name: r.Name,
		Type: SourceType,
		Db:   db,
	}
	return s, nil
}

var _ sources.Source = &Source{}

type Source struct {
	// Cloud SQL MSSQL struct with connection pool
	Name string `yaml:"name"`
	Type string `yaml:"type"`
	Db   *sql.DB
}

func (s *Source) SourceType() string {
	// Returns Cloud SQL MSSQL source type
	return SourceType
}

func (s *Source) MSSQLDB() *sql.DB {
	// Returns a Cloud SQL MSSQL database connection pool
	return s.Db
}

func initMssqlConnection(
	ctx context.Context,
	tracer trace.Tracer,
	name, host, port, user, pass, dbname, encrypt string,
) (
	*sql.DB,
	error,
) {
	//nolint:all // Reassigned ctx
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceType, name)
	defer span.End()

	// Create dsn
	query := url.Values{}
	query.Add("database", dbname)
	if encrypt != "" {
		query.Add("encrypt", encrypt)
	}

	url := &url.URL{
		Scheme:   "sqlserver",
		User:     url.UserPassword(user, pass),
		Host:     fmt.Sprintf("%s:%s", host, port),
		RawQuery: query.Encode(),
	}

	// Open database connection
	db, err := sql.Open("sqlserver", url.String())
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}
	return db, nil
}
