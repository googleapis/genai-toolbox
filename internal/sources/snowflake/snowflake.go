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

package snowflake

import (
	"context"
	"fmt"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/jmoiron/sqlx"
	_ "github.com/snowflakedb/gosnowflake"
	"go.opentelemetry.io/otel/trace"
)

const SourceKind string = "snowflake"

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
	Name      string `yaml:"name" validate:"required"`
	Kind      string `yaml:"kind" validate:"required"`
	Account   string `yaml:"account" validate:"required"`
	User      string `yaml:"user" validate:"required"`
	Password  string `yaml:"password" validate:"required"`
	Database  string `yaml:"database" validate:"required"`
	Schema    string `yaml:"schema" validate:"required"`
	Warehouse string `yaml:"warehouse"`
	Role      string `yaml:"role"`
}

func (r Config) SourceConfigKind() string {
	return SourceKind
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	db, err := initSnowflakeConnection(ctx, tracer, r.Name, r.Account, r.User, r.Password, r.Database, r.Schema, r.Warehouse, r.Role)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection: %w", err)
	}

	err = db.PingContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to connect successfully: %w", err)
	}

	s := &Source{
		Name: r.Name,
		Kind: SourceKind,
		DB:   db,
	}
	return s, nil
}

var _ sources.Source = &Source{}

type Source struct {
	Name string `yaml:"name"`
	Kind string `yaml:"kind"`
	DB   *sqlx.DB
}

func (s *Source) SourceKind() string {
	return SourceKind
}

func (s *Source) SnowflakeDB() *sqlx.DB {
	return s.DB
}

func initSnowflakeConnection(ctx context.Context, tracer trace.Tracer, name, account, user, password, database, schema, warehouse, role string) (*sqlx.DB, error) {
	//nolint:all // Reassigned ctx
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, name)
	defer span.End()

	// Set defaults for optional parameters
	if warehouse == "" {
		warehouse = "COMPUTE_WH"
	}
	if role == "" {
		role = "ACCOUNTADMIN"
	}

	// Snowflake DSN format: user:password@account/database/schema?warehouse=warehouse&role=role
	dsn := fmt.Sprintf("%s:%s@%s/%s/%s?warehouse=%s&role=%s&protocol=https&timeout=60", user, password, account, database, schema, warehouse, role)
	db, err := sqlx.Connect("snowflake", dsn)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection: %w", err)
	}

	return db, nil
}

