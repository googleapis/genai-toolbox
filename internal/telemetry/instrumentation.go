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

package telemetry

import (
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const (
	TracerName = "github.com/googleapis/mcp-toolbox/internal/opentel"
	MetricName = "github.com/googleapis/mcp-toolbox/internal/opentel"

	// OTel semconv metrics
	mcpOperationDurationName  = "mcp.server.operation.duration"
	mcpSessionDurationName    = "mcp.server.session.duration"
	mcpActiveSessionsName     = "toolbox.server.mcp.active_sessions"
	toolExecutionDurationName = "toolbox.tool.execution.duration"
)

// Instrumentation defines the telemetry instrumentation for toolbox
type Instrumentation struct {
	Tracer                trace.Tracer
	meter                 metric.Meter
	McpOperationDuration  metric.Float64Histogram
	McpSessionDuration    metric.Float64Histogram
	McpActiveSessions     metric.Int64UpDownCounter
	ToolExecutionDuration metric.Float64Histogram
}

func CreateTelemetryInstrumentation(versionString string) (*Instrumentation, error) {
	tracer := otel.Tracer(
		TracerName,
		trace.WithInstrumentationVersion(versionString),
	)

	meter := otel.Meter(MetricName, metric.WithInstrumentationVersion(versionString))

	mcpOperationDuration, err := meter.Float64Histogram(
		mcpOperationDurationName,
		metric.WithDescription("Duration of a single MCP JSON-RPC operation."),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create %s metric: %w", mcpOperationDurationName, err)
	}

	mcpSessionDuration, err := meter.Float64Histogram(
		mcpSessionDurationName,
		metric.WithDescription("Duration of an MCP session."),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create %s metric: %w", mcpSessionDurationName, err)
	}

	mcpActiveSessions, err := meter.Int64UpDownCounter(
		mcpActiveSessionsName,
		metric.WithDescription("Current count of active MCP sessions."),
		metric.WithUnit("{session}"),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create %s metric: %w", mcpActiveSessionsName, err)
	}

	toolExecutionDuration, err := meter.Float64Histogram(
		toolExecutionDurationName,
		metric.WithDescription("Duration of backend tool execution."),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create %s metric: %w", toolExecutionDurationName, err)
	}

	instrumentation := &Instrumentation{
		Tracer:                tracer,
		meter:                 meter,
		McpOperationDuration:  mcpOperationDuration,
		McpSessionDuration:    mcpSessionDuration,
		McpActiveSessions:     mcpActiveSessions,
		ToolExecutionDuration: toolExecutionDuration,
	}
	return instrumentation, nil
}
