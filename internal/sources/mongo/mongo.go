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
package mongo

import (
	"context"
	"fmt"
	"github.com/googleapis/genai-toolbox/internal/sources"
	mongodb "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/otel/trace"
)

const SourceKind string = "mongo"

// validate interface
var _ sources.SourceConfig = Config{}

type Config struct {
	Name string `yaml:"name" validate:"required"`
	Kind string `yaml:"kind" validate:"required"`
	URL  string `yaml:"url" validate:"required"`
}

func (r Config) SourceConfigKind() string {
	return SourceKind
}

// Initialize initializes a MongoBD Source instance.
func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	client, err := mongodb.Connect(ctx, options.Client().ApplyURI(r.URL))
	if err != nil {
		return nil, err
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("could not connect to database %s, %s", r.URL, err)
	}

	s := &Source{
		Name:   r.Name,
		Kind:   SourceKind,
		URL:    r.URL,
		Client: client,
	}
	return s, nil
}

var _ sources.Source = &Source{}

type Source struct {
	Name   string `yaml:"name"`
	Kind   string `yaml:"kind"`
	URL    string `yaml:"url"`
	Client *mongodb.Client
}

func (s *Source) SourceKind() string {
	return SourceKind
}
