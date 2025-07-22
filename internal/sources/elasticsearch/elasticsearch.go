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
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
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
	APIKey    string   `yaml:"apiKey"`
	CloudID   string   `yaml:"cloudId"`
	CACert    string   `yaml:"caCert"`
	SkipTLS   bool     `yaml:"skipTLS"`
}

func (r Config) SourceConfigKind() string {
	return SourceKind
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	client, err := initElasticsearchConnection(ctx, tracer, r.Name, r)
	if err != nil {
		return nil, err
	}

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
	Client *elasticsearch.Client
}

func (s *Source) SourceKind() string {
	return SourceKind
}

func (s *Source) ElasticsearchClient() *elasticsearch.Client {
	return s.Client
}

func initElasticsearchConnection(
	ctx context.Context,
	tracer trace.Tracer,
	name string,
	config Config,
) (*elasticsearch.Client, error) {
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceKind, name)
	defer span.End()

	// Build Elasticsearch configuration
	cfg := elasticsearch.Config{
		Addresses: config.Addresses,
		Username:  config.Username,
		Password:  config.Password,
		APIKey:    config.APIKey,
		CloudID:   config.CloudID,
	}

	// Configure TLS if needed
	if config.SkipTLS {
		cfg.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}

	// Create the Elasticsearch client
	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Elasticsearch client: %w", err)
	}

	// Test the connection
	res, err := client.Info()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Elasticsearch: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("Elasticsearch connection error: %s", res.Status())
	}

	return client, nil
}