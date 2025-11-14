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

package elasticsearch

import (
	"context"
	"fmt"
	"net/http"

	"github.com/elastic/elastic-transport-go/v8/elastictransport"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/util"
	"go.opentelemetry.io/otel/trace"
)

const SourceKind string = "elasticsearch"

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
	Name      string   `yaml:"name" validate:"required"`
	Kind      string   `yaml:"kind" validate:"required"`
	Addresses []string `yaml:"addresses" validate:"required"`
	Username  string   `yaml:"username"`
	Password  string   `yaml:"password"`
	APIKey    string   `yaml:"apikey"`
}

func (c Config) SourceConfigKind() string {
	return SourceKind
}

type EsClient interface {
	esapi.Transport
	elastictransport.Instrumented
}

type Source struct {
	Config
	Client EsClient
}

var _ sources.Source = &Source{}

// tracerProviderAdapter adapts a Tracer to implement the TracerProvider interface
type tracerProviderAdapter struct {
	trace.TracerProvider
	tracer trace.Tracer
}

// Tracer implements the TracerProvider interface
func (t *tracerProviderAdapter) Tracer(name string, options ...trace.TracerOption) trace.Tracer {
	return t.tracer
}

// Initialize creates a new Elasticsearch Source instance.
func (c Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	tracerProvider := &tracerProviderAdapter{tracer: tracer}

	ua, err := util.UserAgentFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting user agent from context: %w", err)
	}

	// Create a new Elasticsearch client with the provided configuration
	cfg := elasticsearch.Config{
		Addresses:       c.Addresses,
		Instrumentation: elasticsearch.NewOpenTelemetryInstrumentation(tracerProvider, false),
		Header:          http.Header{"User-Agent": []string{ua + " go-elasticsearch/" + elasticsearch.Version}},
	}

	// Client need either username and password or an API key
	if c.Username != "" && c.Password != "" {
		cfg.Username = c.Username
		cfg.Password = c.Password
	} else if c.APIKey != "" {
		// API key will be set below
		cfg.APIKey = c.APIKey
	} else {
		// If neither username/password nor API key is provided, we throw an error
		return nil, fmt.Errorf("elasticsearch source %q requires either username/password or an API key", c.Name)
	}

	client, err := elasticsearch.NewBaseClient(cfg)
	if err != nil {
		return nil, err
	}

	// Test connection
	res, err := esapi.InfoRequest{
		Instrument: client.InstrumentationEnabled(),
	}.Do(ctx, client)

	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("elasticsearch connection failed: status %d", res.StatusCode)
	}

	s := &Source{
		Config: c,
		Client: client,
	}
	return s, nil
}

// SourceKind returns the kind string for this source.
func (s *Source) SourceKind() string {
	return SourceKind
}

func (s *Source) ToConfig() sources.SourceConfig {
	return s.Config
}

func (s *Source) ElasticsearchClient() EsClient {
	return s.Client
}
