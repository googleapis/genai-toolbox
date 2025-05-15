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
	"fmt"
	"time"

	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/trace"
)

const SourceKind string = "memorystore-redis"

// validate interface
var _ sources.SourceConfig = Config{}

type Config struct {
	Name           string `yaml:"name" validate:"required"`
	Kind           string `yaml:"kind" validate:"required"`
	Address        string `yaml:"address" validate:"required"`
	ClusterEnabled bool   `yaml:"clusterEnabled"`
	Password       string `yaml:"password"`
	Database       int    `yaml:"database"`
}

func (r Config) SourceConfigKind() string {
	return SourceKind
}

// RedisClient is an interface for `redis.Client` and `redis.ClusterClient
type RedisClient interface {
	Do(context.Context, ...any) *redis.Cmd
}

var _ RedisClient = (*redis.Client)(nil)
var _ RedisClient = (*redis.ClusterClient)(nil)

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	client, err := initMemorystoreRedisClient(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("error initializing Redis client: %s", err)
	}
	s := &Source{
		Name:   r.Name,
		Kind:   SourceKind,
		Client: client,
	}
	return s, nil
}

func initMemorystoreRedisClient(ctx context.Context, r Config) (RedisClient, error) {
	var client RedisClient
	var err error
	if r.ClusterEnabled {
		// Create a new Redis Cluster client
		clusterClient := redis.NewClusterClient(&redis.ClusterOptions{
			Addrs: []string{r.Address},
			// PoolSize applies per cluster node and not for the whole cluster.
			PoolSize:        10,
			ConnMaxIdleTime: 60 * time.Second,
			MinIdleConns:    1,
		})
		err = clusterClient.ForEachShard(ctx, func(ctx context.Context, shard *redis.Client) error {
			return shard.Ping(ctx).Err()
		})
		if err != nil {
			return nil, fmt.Errorf("unable to connect to redis cluster: %s", err)
		}
		client = clusterClient
		return client, nil
	}

	// Create a new Redis client
	standaloneClient := redis.NewClient(&redis.Options{
		Addr:            r.Address,
		PoolSize:        10,
		ConnMaxIdleTime: 60 * time.Second,
		MinIdleConns:    1,
	})
	_, err = standaloneClient.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("unable to connect to redis: %s", err)
	}
	client = standaloneClient
	return client, nil
}

var _ sources.Source = &Source{}

type Source struct {
	Name   string `yaml:"name"`
	Kind   string `yaml:"kind"`
	Client RedisClient
}

func (s *Source) SourceKind() string {
	return SourceKind
}

func (s *Source) RedisClient() RedisClient {
	return s.Client
}
