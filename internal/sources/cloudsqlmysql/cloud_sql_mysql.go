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

package cloudsqlmysql

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"strings"

	"cloud.google.com/go/cloudsqlconn"
	"github.com/go-sql-driver/mysql"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"go.opentelemetry.io/otel/trace"
)

const SourceKind string = "cloud-sql-mysql"

// validate interface
var _ sources.SourceConfig = Config{}

type Config struct {
	Name     string         `yaml:"name"`
	Kind     string         `yaml:"kind"`
	Project  string         `yaml:"project"`
	Region   string         `yaml:"region"`
	Instance string         `yaml:"instance"`
	IPType   sources.IPType `yaml:"ipType"`
	User     string         `yaml:"user"`
	Password string         `yaml:"password"`
	Database string         `yaml:"database"`
}

func (r Config) SourceConfigKind() string {
	return SourceKind
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	pool, err := initCloudSQLMySQLConnectionPool(ctx, tracer, r.Name, r.Project, r.Region, r.Instance, r.IPType.String(), r.User, r.Password, r.Database)
	if err != nil {
		return nil, fmt.Errorf("unable to create pool: %w", err)
	}

	err = pool.PingContext(context.Background())
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

func (s *Source) MySQLPool() *sql.DB {
	return s.Pool
}

func getDialOpts(ipType string) ([]cloudsqlconn.DialOption, error) {
	switch strings.ToLower(ipType) {
	case "private":
		return []cloudsqlconn.DialOption{cloudsqlconn.WithPrivateIP()}, nil
	case "public":
		return []cloudsqlconn.DialOption{cloudsqlconn.WithPublicIP()}, nil
	default:
		return nil, fmt.Errorf("invalid ipType %s", ipType)
	}
}

func initCloudSQLMySQLConnectionPool(ctx context.Context, tracer trace.Tracer, name, project, region, instance, ipType, user, pass, dbname string) (*sql.DB, error) {
	//nolint:all // Reassigned ctx
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, name)
	defer span.End()

	// Create a new dialer with options
	dialOpts, err := getDialOpts(ipType)
	if err != nil {
		return nil, err
	}
	d, err := cloudsqlconn.NewDialer(context.Background())
	if err != nil {
		return nil, fmt.Errorf("unable to parse connection uri: %w", err)
	}

	// Tell the driver to use the Cloud SQL Go Connector to create connections
	i := fmt.Sprintf("%s:%s:%s", project, region, instance)
	mysql.RegisterDialContext("cloudsqlconn", func(ctx context.Context, addr string) (net.Conn, error) {
		return d.Dial(ctx, i, dialOpts...)
	})

	// Configure the driver to connect to the database
	dbURI := fmt.Sprintf("%s:%s@cloudsqlconn(localhost:3306)/%s?parseTime=true", user, pass, dbname)

	// Interact with the driver directly as you normally would
	pool, err := sql.Open("mysql", dbURI)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}
	return pool, nil
}
