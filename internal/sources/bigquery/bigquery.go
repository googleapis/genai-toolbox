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

package bigquery

import (
	"context"
	"fmt"
	"sync"
	"time"

	bigqueryapi "cloud.google.com/go/bigquery"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	bigqueryrestapi "google.golang.org/api/bigquery/v2"
	"google.golang.org/api/option"
)

const SourceKind string = "bigquery"

// validate interface
var _ sources.SourceConfig = Config{}

type BigqueryClientCreator func(tokenString tools.OAuthAccessToken) (*bigqueryapi.Client, *bigqueryrestapi.Service, error)

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
	// BigQuery configs
	Name     string `yaml:"name" validate:"required"`
	Kind     string `yaml:"kind" validate:"required"`
	Project  string `yaml:"project" validate:"required"`
	Location string `yaml:"location"`
}

func (r Config) SourceConfigKind() string {
	// Returns BigQuery source kind
	return SourceKind
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	// Initializes a BigQuery Google SQL source
	client, restService, clientCreator, err := initBigQueryConnection(ctx, tracer, r.Name, r.Project, r.Location)
	if err != nil {
		return nil, err
	}

	s := &Source{
		Name:          r.Name,
		Kind:          SourceKind,
		Client:        client,
		RestService:   restService,
		Location:      r.Location,
		ClientCreator: clientCreator,
	}
	return s, nil

}

var _ sources.Source = &Source{}

type Source struct {
	// BigQuery Google SQL struct with client
	Name          string `yaml:"name"`
	Kind          string `yaml:"kind"`
	Location      string `yaml:"location"`
	Client        *bigqueryapi.Client
	RestService   *bigqueryrestapi.Service
	ClientCreator BigqueryClientCreator
}

func (s *Source) SourceKind() string {
	// Returns BigQuery Google SQL source kind
	return SourceKind
}

func (s *Source) BigQueryClient() *bigqueryapi.Client {
	return s.Client
}

func (s *Source) BigQueryRestService() *bigqueryrestapi.Service {
	return s.RestService
}

func (s *Source) BigQueryClientCreator() BigqueryClientCreator {
	return s.ClientCreator
}

func initBigQueryConnection(
	ctx context.Context,
	tracer trace.Tracer,
	name string,
	project string,
	location string,
) (*bigqueryapi.Client, *bigqueryrestapi.Service, BigqueryClientCreator, error) {
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, name)
	defer span.End()

	cred, err := google.FindDefaultCredentials(ctx, bigqueryapi.Scope)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to find default Google Cloud credentials with scope %q: %w", bigqueryapi.Scope, err)
	}

	userAgent, err := util.UserAgentFromContext(ctx)
	if err != nil {
		return nil, nil, nil, err
	}

	// Initialize the high-level BigQuery client
	client, err := bigqueryapi.NewClient(ctx, project, option.WithUserAgent(userAgent), option.WithCredentials(cred))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create BigQuery client for project %q: %w", project, err)
	}
	client.Location = location

	// Initialize the low-level BigQuery REST service using the same credentials
	restService, err := bigqueryrestapi.NewService(ctx, option.WithUserAgent(userAgent), option.WithCredentials(cred))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create BigQuery v2 service: %w", err)
	}

	clientCreator := newBigQueryClientCreator(ctx, project, location, userAgent)
	return client, restService, clientCreator, nil
}

var (
	clientCache *ClientCache
	once        sync.Once
)

func initBigQueryConnectionWithOAuthToken(
	ctx context.Context,
	project string,
	location string,
	userAgent string,
	tokenString tools.OAuthAccessToken,
) (*bigqueryapi.Client, *bigqueryrestapi.Service, error) {
	// Construct token source
	token := &oauth2.Token{
		AccessToken: string(tokenString),
	}
	ts := oauth2.StaticTokenSource(token)

	// Initialize the BigQuery client with tokenSource
	client, err := bigqueryapi.NewClient(ctx, project, option.WithUserAgent(userAgent), option.WithTokenSource(ts))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create BigQuery client for project %q: %w", project, err)
	}
	client.Location = location

	// Initialize the low-level BigQuery REST service using the same credentials
	restService, err := bigqueryrestapi.NewService(ctx, option.WithUserAgent(userAgent), option.WithTokenSource(ts))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create BigQuery v2 service: %w", err)
	}

	return client, restService, nil
}

// newBigQueryClientCreator returns a cache-aware client creater
func newBigQueryClientCreator(
	ctx context.Context,
	project string,
	location string,
	userAgent string,
) BigqueryClientCreator {

	cache := GetClientCache()
	return func(tokenString tools.OAuthAccessToken) (*bigqueryapi.Client, *bigqueryrestapi.Service, error) {
		return cache.GetOrCreateClient(ctx, project, location, userAgent, tokenString)
	}
}

// cachedClient holds the BigQuery clients and their expiration time.
type cachedClient struct {
	client      *bigqueryapi.Client
	restService *bigqueryrestapi.Service
	expiry      time.Time
}

// ClientCache manages a thread-safe cache of BigQuery clients.
type ClientCache struct {
	mu      sync.RWMutex
	clients map[tools.OAuthAccessToken]*cachedClient
}

// GetClientCache initializes the clientCache and start a cleanup go routine
func GetClientCache() *ClientCache {
	once.Do(func() {
		clientCache = &ClientCache{
			clients: make(map[tools.OAuthAccessToken]*cachedClient),
		}
		// Clean up expired clients periodically.
		go clientCache.cleanupLoop(5 * time.Minute)
	})
	return clientCache
}

// cleanupLoop periodically removes expired clients from the cache.
func (c *ClientCache) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		for token, cached := range c.clients {
			if time.Now().After(cached.expiry) {
				_ = cached.client.Close()
				delete(c.clients, token)
			}
		}
		c.mu.Unlock()
	}
}

// GetOrCreateClient retrieves a cached client or creates a new one if not found.
func (c *ClientCache) GetOrCreateClient(
	ctx context.Context,
	project string,
	location string,
	userAgent string,
	tokenString tools.OAuthAccessToken,
) (*bigqueryapi.Client, *bigqueryrestapi.Service, error) {

	// Check for an existing client using a read lock.
	c.mu.RLock()
	cached, found := c.clients[tokenString]
	if found && time.Now().Before(cached.expiry) {
		c.mu.RUnlock()
		return cached.client, cached.restService, nil
	}
	c.mu.RUnlock()

	// Use a write lock to create a new client.
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check in case another goroutine created it while waiting for the lock.
	cached, found = c.clients[tokenString]
	if found && time.Now().Before(cached.expiry) {
		return cached.client, cached.restService, nil
	}

	// Create the new client
	client, restService, err := initBigQueryConnectionWithOAuthToken(ctx, project, location, userAgent, tokenString)
	if err != nil {
		return nil, nil, err
	}

	// Store the new client in the cache with a 55-minute expiry.
	newCachedClient := &cachedClient{
		client:      client,
		restService: restService,
		expiry:      time.Now().Add(55 * time.Minute),
	}
	c.clients[tokenString] = newCachedClient

	return client, restService, nil
}
