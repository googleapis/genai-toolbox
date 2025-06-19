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

package yugabytedb

import (
	"context"
	"fmt"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/yugabyte/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/trace"
)

const SourceKind string = "yugabytedb"

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
	Name                            string `yaml:"name" validate:"required"`
	Kind                            string `yaml:"kind" validate:"required"`
	Host                            string `yaml:"host" validate:"required"`
	Port                            string `yaml:"port" validate:"required"`
	User                            string `yaml:"user" validate:"required"`
	Password                        string `yaml:"password" validate:"required"`
	Database                        string `yaml:"database" validate:"required"`
	LoadBalance                     string `yaml:"load_balance"`                     // optional
	TopologyKeys                    string `yaml:"topology_keys"`                    // optional
	YBServersRefreshInterval        string `yaml:"yb_servers_refresh_interval"`      // optional
	FallBackToTopologyKeysOnly      string `yaml:"fallback_to_topology_keys_only"`   // optional
	FailedHostReconnectDelaySeconds string `yaml:"failed_host_reconnect_delay_secs"` // optional
}

func (r Config) SourceConfigKind() string {
	return SourceKind
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	pool, err := initYugabyteDBConnectionPool(ctx, tracer, r.Name, r.Host, r.Port, r.User, r.Password, r.Database, r.LoadBalance, r.TopologyKeys, r.YBServersRefreshInterval, r.FallBackToTopologyKeysOnly, r.FailedHostReconnectDelaySeconds)
	if err != nil {
		return nil, fmt.Errorf("Unable to create pool: %w", err)
	}

	err = pool.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("Unable to connect successfully: %w", err)
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
	Pool *pgxpool.Pool
}

func (s *Source) SourceKind() string {
	return SourceKind
}

func (s *Source) YugabyteDBPool() *pgxpool.Pool {
	return s.Pool
}

func initYugabyteDBConnectionPool(ctx context.Context, tracer trace.Tracer, name, host, port, user, pass, dbname, loadBalance, topologyKeys, refreshInterval, explicitFallback, failedHostTTL string) (*pgxpool.Pool, error) {
	//nolint:all // Reassigned ctx
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, name)
	defer span.End()
	// urlExample := "postgres://username:password@localhost:5433/database_name"
	i := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", user, pass, host, port, dbname)
	if loadBalance == "true" {
		i = fmt.Sprintf("%s?load_balance=%s", i, loadBalance)
		if topologyKeys != "" {
			i = fmt.Sprintf("%s&topology_keys=%s", i, topologyKeys)
			if explicitFallback == "true" {
				i = fmt.Sprintf("%s&fallback_to_topology_keys_only=%s", i, explicitFallback)
			}
		}
		if refreshInterval != "" {
			i = fmt.Sprintf("%s&yb_servers_refresh_interval=%s", i, refreshInterval)
		}
		if failedHostTTL != "" {
			i = fmt.Sprintf("%s&failed_host_reconnect_delay_secs=%s", i, failedHostTTL)
		}
	}
	pool, err := pgxpool.New(ctx, i)
	if err != nil {
		return nil, fmt.Errorf("Unable to create connection pool: %w", err)
	}

	return pool, nil
}
