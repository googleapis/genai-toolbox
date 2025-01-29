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
	"slices"

	"cloud.google.com/go/cloudsqlconn/sqlserver/mssql"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"go.opentelemetry.io/otel/trace"
)

const SourceKind string = "cloud-sql-mssql"

// validate interface
var _ sources.SourceConfig = Config{}

type Config struct {
	// Cloud SQL MSSQL configs
	Name      string `yaml:"name"`
	Kind      string `yaml:"kind"`
	Project   string `yaml:"project"`
	Region    string `yaml:"region"`
	Instance  string `yaml:"instance"`
	IPAddress string `yaml:"ipAddress"`
	IPType    string `yaml:"ipType"`
	User      string `yaml:"user"`
	Password  string `yaml:"password"`
	Database  string `yaml:"database"`
}

func (r Config) SourceConfigKind() string {
	// Returns Cloud SQL MSSQL source kind
	return SourceKind
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer, userAgent string) (sources.Source, error) {
	// Initializes a Cloud SQL MSSQL source
	db, err := initCloudSQLMssqlConnection(ctx, tracer, r.Name, r.Project, r.Region, r.Instance, r.IPAddress, r.IPType, r.User, r.Password, r.Database, userAgent)
	if err != nil {
		return nil, fmt.Errorf("unable to create db connection: %w", err)
	}

	// Verify db connection
	err = db.Ping()
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

func initCloudSQLMssqlConnection(ctx context.Context, tracer trace.Tracer, name, project, region, instance, ipAddress, ipType, user, pass, dbname, userAgent string) (*sql.DB, error) {
	//nolint:all // Reassigned ctx
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, name)
	defer span.End()

	// Create dsn
	dsn := fmt.Sprintf("sqlserver://%s:%s@%s?database=%s&cloudsql=%s:%s:%s", user, pass, ipAddress, dbname, project, region, instance)

	// Get dial options
	opts, err := sources.GetCloudSQLOpts(ipType, userAgent)
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
		dsn,
	)
	if err != nil {
		return nil, err
	}
	return db, nil
}
