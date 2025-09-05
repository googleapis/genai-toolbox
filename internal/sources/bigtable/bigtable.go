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

package bigtable

import (
	"context"
	"fmt"

	"cloud.google.com/go/bigtable"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/util"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/api/option"
)

const SourceType string = "bigtable"

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
	Name     string `yaml:"name" validate:"required"`
	Type     string `yaml:"kind" validate:"required"`
	Project  string `yaml:"project" validate:"required"`
	Instance string `yaml:"instance" validate:"required"`
}

func (r Config) SourceConfigType() string {
	return SourceType
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	client, err := initBigtableClient(ctx, tracer, r.Name, r.Project, r.Instance)
	if err != nil {
		return nil, fmt.Errorf("unable to create client: %w", err)
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
	Client *bigtable.Client
}

func (s *Source) SourceType() string {
	return SourceType
}

func (s *Source) BigtableClient() *bigtable.Client {
	return s.Client
}

func initBigtableClient(ctx context.Context, tracer trace.Tracer, name, project, instance string) (*bigtable.Client, error) {
	//nolint:all // Reassigned ctx
	ctx, span := sources.InitConnectionSpan(ctx, tracer, SourceType, name)
	defer span.End()

	// Set up Bigtable data operations client.
	poolSize := 10
	userAgent, err := util.UserAgentFromContext(ctx)
	if err != nil {
		return nil, err
	}

	client, err := bigtable.NewClient(ctx, project, instance, option.WithUserAgent(userAgent), option.WithGRPCConnectionPool(poolSize))

	if err != nil {
		return nil, fmt.Errorf("unable to create bigtable.NewClient: %w", err)
	}

	return client, nil
}
