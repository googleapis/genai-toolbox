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

package sources

import (
	"context"
	"fmt"
	"net"

	"cloud.google.com/go/cloudsqlconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const CloudSQLPgKind string = "cloud-sql-postgres"

// validate interface
var _ Config = CloudSQLPgConfig{}

type CloudSQLPgConfig struct {
	Name     string `yaml:"name"`
	Kind     string `yaml:"kind"`
	Project  string `yaml:"project"`
	Region   string `yaml:"region"`
	Instance string `yaml:"instance"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

func (r CloudSQLPgConfig) sourceKind() string {
	return CloudSQLPgKind
}

func (r CloudSQLPgConfig) Initialize() (Source, error) {
	pool, err := initCloudSQLPgConnectionPool(r.Project, r.Region, r.Instance, r.User, r.Password, r.Database)
	if err != nil {
		return nil, fmt.Errorf("unable to create pool: %w", err)
	}

	err = pool.Ping(context.Background())
	if err != nil {
		return nil, fmt.Errorf("unable to connect successfully: %w", err)
	}

	s := CloudSQLPgSource{
		Name: r.Name,
		Kind: CloudSQLPgKind,
		Pool: pool,
	}
	return s, nil
}

var _ Source = CloudSQLPgSource{}

type CloudSQLPgSource struct {
	Name string `yaml:"name"`
	Kind string `yaml:"kind"`
	Pool *pgxpool.Pool
}

func initCloudSQLPgConnectionPool(project, region, instance, user, pass, dbname string) (*pgxpool.Pool, error) {
	// Configure the driver to connect to the database
	dsn := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", user, pass, dbname)
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("unable to parse connection uri: %w", err)
	}

	// Create a new dialer with any options
	d, err := cloudsqlconn.NewDialer(context.Background())
	if err != nil {
		return nil, fmt.Errorf("unable to parse connection uri: %w", err)
	}

	// Tell the driver to use the Cloud SQL Go Connector to create connections
	i := fmt.Sprintf("%s:%s:%s", project, region, instance)
	config.ConnConfig.DialFunc = func(ctx context.Context, _ string, instance string) (net.Conn, error) {
		return d.Dial(ctx, i)
	}

	// Interact with the driver directly as you normally would
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, err
	}
	return pool, nil
}
