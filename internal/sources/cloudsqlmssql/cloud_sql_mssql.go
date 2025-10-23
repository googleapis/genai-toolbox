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

package cloudsqlmssql

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"slices"

	"cloud.google.com/go/cloudsqlconn/sqlserver/mssql"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/util"
	"go.opentelemetry.io/otel/trace"
)

const SourceKind string = "cloud-sql-mssql"

// validate interface
var _ sources.SourceConfig = Config{}

func init() {
	if !sources.Register(SourceKind, newConfig) {
		panic(fmt.Sprintf("source kind %q already registered", SourceKind))
	}
}

func newConfig(ctx context.Context, name string, decoder *yaml.Decoder) (sources.SourceConfig, error) {
	actual := Config{Name: name, IPType: "public"} // Default IPType
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return nil, err
	}
	return actual, nil
}

type Config struct {
	// Cloud SQL MSSQL configs
	Name      string         `yaml:"name" validate:"required"`
	Kind      string         `yaml:"kind" validate:"required"`
	Project   string         `yaml:"project" validate:"required"`
	Region    string         `yaml:"region" validate:"required"`
	Instance  string         `yaml:"instance" validate:"required"`
	IPAddress string         `yaml:"ipAddress" validate:"required"`
	IPType    sources.IPType `yaml:"ipType" validate:"required"`
	User      string         `yaml:"user" validate:"required"`
	Password  string         `yaml:"password" validate:"required"`
	Database  string         `yaml:"database" validate:"required"`
}

func (r Config) SourceConfigKind() string {
	// Returns Cloud SQL MSSQL source kind
	return SourceKind
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	// Initializes a Cloud SQL MSSQL source
	db, err := initCloudSQLMssqlConnection(ctx, tracer, r.Name, r.Project, r.Region, r.Instance, r.IPAddress, r.IPType.String(), r.User, r.Password, r.Database)
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
		Kind: SourceKind,
		Db:   db,
	}
	return s, nil
}

var _ sources.Source = &Source{}

type Source struct {
	// Cloud SQL MSSQL struct with connection pool
	Name string `yaml:"name"`
	Kind string `yaml:"kind"`
	Db   *sql.DB
}

func (s *Source) SourceKind() string {
	// Returns Cloud SQL MSSQL source kind
	return SourceKind
}

func (s *Source) MSSQLDB() *sql.DB {
	// Returns a Cloud SQL MSSQL database connection pool
	return s.Db
}

func initCloudSQLMssqlConnection(ctx context.Context, tracer trace.Tracer, name, project, region, instance, ipAddress, ipType, user, pass, dbname string) (*sql.DB, error) {
	//nolint:all // Reassigned ctx
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, name)
	defer span.End()

	userAgent, err := util.UserAgentFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Create dsn
	query := url.Values{}
	query.Add("app name", userAgent)
	query.Add("database", dbname)
	query.Add("cloudsql", fmt.Sprintf("%s:%s:%s", project, region, instance))

	url := &url.URL{
		Scheme:   "sqlserver",
		User:     url.UserPassword(user, pass),
		Host:     ipAddress,
		RawQuery: query.Encode(),
	}

	// Get dial options
	opts, err := sources.GetCloudSQLOpts(ipType, userAgent, false)
	if err != nil {
		return nil, err
	}

	// Register sql server driver
	if !slices.Contains(sql.Drivers(), "cloudsql-sqlserver-driver") {
		_, err := mssql.RegisterDriver("cloudsql-sqlserver-driver", opts...)
		if err != nil {
			return nil, err
		}
	}

	// Open database connection
	db, err := sql.Open(
		"cloudsql-sqlserver-driver",
		url.String(),
	)
	if err != nil {
		return nil, err
	}
	return db, nil
}
