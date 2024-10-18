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

    "cloud.google.com/go/alloydbconn"
    "github.com/jackc/pgx/v5/pgxpool"
)

const AlloyDBPgKind string = "alloydb-postgres"

// validate interface
var _ Config = AlloyDBPgConfig{}

type AlloyDBPgConfig struct {
	Name     string `yaml:"name"`
	Kind     string `yaml:"kind"`
	Project  string `yaml:"project"`
	Region   string `yaml:"region"`
    Cluster  string `yaml:"cluster"`
	Instance string `yaml:"instance"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

func (r AlloyDBPgConfig) sourceKind() string {
	return AlloyDBPgKind
}

func (r AlloyDBPgConfig) Initialize() (Source, error) {
	pool, err := initAlloyDBPgConnectionPool(r.Project, r.Region, r.Cluster, r.Instance, r.User, r.Password, r.Database)
	if err != nil {
		return nil, fmt.Errorf("Unable to create pool: %w", err)
	}

	err = pool.Ping(context.Background())
	if err != nil {
		return nil, fmt.Errorf("Unable to connect successfully: %w", err)
	}

	s := AlloyDBPgSource{
		Name: r.Name,
		Kind: AlloyDBPgKind,
		Pool: pool,
	}
	return s, nil
}

var _ Source = AlloyDBPgSource{}

type AlloyDBPgSource struct {
	Name string `yaml:"name"`
	Kind string `yaml:"kind"`
	Pool *pgxpool.Pool
}

func initAlloyDBPgConnectionPool(project, region, cluster, instance, user, pass, dbname string) (*pgxpool.Pool, error) {
	// Configure the driver to connect to the database
	dsn := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", user, pass, dbname)
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("Unable to parse connection uri: %w", err)
	}

	// Create a new dialer with any options
	d, err := alloydbconn.NewDialer(context.Background())
	if err != nil {
		return nil, fmt.Errorf("Unable to parse connection uri: %w", err)
	}

	// Tell the driver to use the AlloyDB Go Connector to create connections
	i := fmt.Sprintf("projects/%s/locations/%s/clusters/%s/instances/%s", project, region, cluster, instance)
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
