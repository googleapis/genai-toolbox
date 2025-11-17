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

package sources

import (
	"context"

	"fmt"

	"github.com/goccy/go-yaml"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// SourceConfigFactory defines the function signature for creating a SourceConfig.
type SourceConfigFactory func(ctx context.Context, name string, decoder *yaml.Decoder) (SourceConfig, error)

var sourceRegistry = make(map[string]SourceConfigFactory)

// Register registers a new source kind with its factory.
// It returns false if the kind is already registered.
func Register(kind string, factory SourceConfigFactory) bool {
	if _, exists := sourceRegistry[kind]; exists {
		// Source with this kind already exists, do not overwrite.
		return false
	}
	sourceRegistry[kind] = factory
	return true
}

// DecodeConfig decodes a source configuration using the registered factory for the given kind.
func DecodeConfig(ctx context.Context, kind string, name string, decoder *yaml.Decoder) (SourceConfig, error) {
	factory, found := sourceRegistry[kind]
	if !found {
		return nil, fmt.Errorf("unknown source kind: %q", kind)
	}
	sourceConfig, err := factory(ctx, name, decoder)
	if err != nil {
		return nil, fmt.Errorf("unable to parse source %q as %q: %w", name, kind, err)
	}
	return sourceConfig, err
}

// SourceConfig is the interface for configuring a source.
type SourceConfig interface {
	SourceConfigKind() string
	Initialize(ctx context.Context, tracer trace.Tracer) (Source, error)
}

// Source is the interface for the source itself.
type Source interface {
	SourceKind() string
	ToConfig() SourceConfig
}

// InitConnectionSpan adds a span for database pool connection initialization
func InitConnectionSpan(ctx context.Context, tracer trace.Tracer, sourceKind, sourceName string) (context.Context, trace.Span) {
	ctx, span := tracer.Start(
		ctx,
		"toolbox/server/source/connect",
		trace.WithAttributes(attribute.String("source_kind", sourceKind)),
		trace.WithAttributes(attribute.String("source_name", sourceName)),
	)
	return ctx, span
}
