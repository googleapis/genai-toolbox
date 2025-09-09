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
package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/trace"
)

const SourceType string = "redis"

// validate interface
var _ sources.SourceConfig = Config{}

func init() {
	if !sources.Register(SourceType, newConfig) {
		panic(fmt.Sprintf("source type %q already registered", SourceType))
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
	Name           string   `yaml:"name" validate:"required"`
	Type           string   `yaml:"type" validate:"required"`
	Address        []string `yaml:"address" validate:"required"`
	Username       string   `yaml:"username"`
	Password       string   `yaml:"password"`
	Database       int      `yaml:"database"`
	UseGCPIAM      bool     `yaml:"useGCPIAM"`
	ClusterEnabled bool     `yaml:"clusterEnabled"`
}

func (r Config) SourceConfigType() string {
	return SourceType
}

// RedisClient is an interface for `redis.Client` and `redis.ClusterClient
type RedisClient interface {
	Do(context.Context, ...any) *redis.Cmd
}

var _ RedisClient = (*redis.Client)(nil)
var _ RedisClient = (*redis.ClusterClient)(nil)

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	client, err := initRedisClient(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("error initializing Redis client: %s", err)
	}
	s := &Source{
		Name:   r.Name,
		Type:   SourceType,
		Client: client,
	}
	return s, nil
}

func initRedisClient(ctx context.Context, r Config) (RedisClient, error) {
	var authFn func(ctx context.Context) (username string, password string, err error)
	if r.UseGCPIAM {
		// Pass in an access token getter fn for IAM auth
		authFn = func(ctx context.Context) (username string, password string, err error) {
			token, err := sources.GetIAMAccessToken(ctx)
			if err != nil {
				return "", "", err
			}
			return "default", token, nil
		}
	}

	var client RedisClient
	var err error
	if r.ClusterEnabled {
		// Create a new Redis Cluster client
		clusterClient := redis.NewClusterClient(&redis.ClusterOptions{
			Addrs: r.Address,
			// PoolSize applies per cluster node and not for the whole cluster.
			PoolSize:                   10,
			ConnMaxIdleTime:            60 * time.Second,
			MinIdleConns:               1,
			CredentialsProviderContext: authFn,
			Username:                   r.Username,
			Password:                   r.Password,
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
		Addr:                       r.Address[0],
		PoolSize:                   10,
		ConnMaxIdleTime:            60 * time.Second,
		MinIdleConns:               1,
		DB:                         r.Database,
		CredentialsProviderContext: authFn,
		Username:                   r.Username,
		Password:                   r.Password,
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
	Type   string `yaml:"type"`
	Client RedisClient
}

func (s *Source) SourceType() string {
	return SourceType
}

func (s *Source) RedisClient() RedisClient {
	return s.Client
}
