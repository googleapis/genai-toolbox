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

package mongodb

import (
	"context"
	"fmt"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/util"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/otel/trace"
)

const SourceType string = "mongodb"

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
	Name string `yaml:"name" validate:"required"`
	Type string `yaml:"kind" validate:"required"`
	Uri  string `yaml:"uri" validate:"required"` // MongoDB Atlas connection URI
}

func (r Config) SourceConfigType() string {
	return SourceType
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	client, err := initMongoDBClient(ctx, tracer, r.Name, r.Uri)
	if err != nil {
		return nil, fmt.Errorf("unable to create MongoDB client: %w", err)
	}

	// Verify the connection
	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to connect successfully: %w", err)
	}

	s := &Source{
		Name:   r.Name,
		Type:   SourceType,
		Client: client,
	}
	return s, nil
}

var _ sources.Source = &Source{}

type Source struct {
	Name   string `yaml:"name"`
	Type   string `yaml:"type"`
	Client *mongo.Client
}

func (s *Source) SourceType() string {
	return SourceType
}

func (s *Source) MongoClient() *mongo.Client {
	return s.Client
}

func initMongoDBClient(ctx context.Context, tracer trace.Tracer, name, uri string) (*mongo.Client, error) {
	// Start a tracing span
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceType, name)
	defer span.End()

	userAgent, err := util.UserAgentFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Create a new MongoDB client
	clientOpts := options.Client().ApplyURI(uri).SetAppName(userAgent)
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("unable to create MongoDB client: %w", err)
	}

	return client, nil
}
