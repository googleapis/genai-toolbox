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

	"github.com/jackc/pgx/v5/pgxpool"
)

const PostgresKind string = "postgres"

// validate interface
var _ Config = PostgresConfig{}

type PostgresConfig struct {
	Name     string `yaml:"name"`
	Kind     string `yaml:"kind"`
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

func (r PostgresConfig) sourceKind() string {
	return PostgresKind
}

func (r PostgresConfig) Initialize() (Source, error) {
	pool, err := initPostgresConnectionPool(r.Host, r.Port, r.User, r.Password, r.Database)
	if err != nil {
		return nil, fmt.Errorf("Unable to create pool: %w", err)
	}

	err = pool.Ping(context.Background())
	if err != nil {
		return nil, fmt.Errorf("Unable to connect successfully: %w", err)
	}

	s := PostgresSource{
		Name: r.Name,
		Kind: PostgresKind,
		Pool: pool,
	}
	return s, nil
}

var _ Source = PostgresSource{}

type PostgresSource struct {
	Name string `yaml:"name"`
	Kind string `yaml:"kind"`
	Pool *pgxpool.Pool
}

func initPostgresConnectionPool(host, port, user, pass, dbname string) (*pgxpool.Pool, error) {
    // urlExample := "postgres:dd//username:password@localhost:5432/database_name"
	i := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", user, pass, host, port, dbname)
    pool, err := pgxpool.New(context.Background(), i)
	if err != nil {
		return nil, fmt.Errorf("Unable to create connection pool: %w", err)
	}

	return pool, nil
}
