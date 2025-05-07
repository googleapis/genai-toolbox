// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package memorystoreredis

import (
	"context"
	"crypto/tls"
	"log"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/valkey-io/valkey-go"
	"go.opentelemetry.io/otel/trace"
)

const SourceKind string = "memorystore-redis"

// validate interface
var _ sources.SourceConfig = Config{}

type Config struct {
	Name     string `yaml:"name" validate:"required"`
	Kind     string `yaml:"kind" validate:"required"`
	Address  string `yaml:"address" validate:"required"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Database int    `yaml:"database"`
}

func (r Config) SourceConfigKind() string {
	return SourceKind
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	// Create a new Redis client
	client := redis.NewClient(&redis.ClusterOptions{
		Addrs: []string{clusterDicEpAddr},
		// PoolSize applies per cluster node and not for the whole cluster.
		PoolSize:            10,
		ConnMaxIdleTime:     60 * time.Second,
		MinIdleConns:        1,
		CredentialsProvider: retrieveTokenFunc,
		TLSConfig: &tls.Config{
			RootCAs: caCertPool,
		},
	})

	// Ping the server to check connectivity (using Do)
	pingCmd := client.B().Ping().Build()
	pingResult, err := client.Do(ctx, pingCmd).ToString()
	if err != nil {
		log.Fatalf("Failed to execute PING command: %v", err)
	}
	log.Printf("PING response: %s\n", pingResult)

	s := &Source{
		Name:   r.Name,
		Kind:   SourceKind,
		Client: client,
	}
	return s, nil
}

var _ sources.Source = &Source{}

type Source struct {
	Name   string `yaml:"name"`
	Kind   string `yaml:"kind"`
	Client valkey.Client
}

func (s *Source) SourceKind() string {
	return SourceKind
}

func (s *Source) RedisClient() valkey.Client {
	return s.Client
}
