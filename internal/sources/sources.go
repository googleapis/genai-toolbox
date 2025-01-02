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

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// SourceConfig is the interface for configuring a source.
type SourceConfig interface {
	SourceConfigKind() string
	Initialize(ctx context.Context, tracer trace.Tracer) (Source, error)
}

// Source is the interface for the source itself.
type Source interface {
	SourceKind() string
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
