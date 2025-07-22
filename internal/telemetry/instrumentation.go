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
	TracerName = "github.com/googleapis/genai-toolbox/internal/opentel"
	MetricName = "github.com/googleapis/genai-toolbox/internal/opentel"

	toolsetGetCountName = "toolbox.server.toolset.get.count"
	toolGetCountName    = "toolbox.server.tool.get.count"
	toolInvokeCountName = "toolbox.server.tool.invoke.count"
	mcpSseCountName     = "toolbox.server.mcp.sse.count"
	mcpPostCountName    = "toolbox.server.mcp.post.count"
)

// Instrumentation defines the telemetry instrumentation for toolbox
type Instrumentation struct {
	Tracer     trace.Tracer
	meter      metric.Meter
	ToolsetGet metric.Int64Counter
	ToolGet    metric.Int64Counter
	ToolInvoke metric.Int64Counter
	McpSse     metric.Int64Counter
	McpPost    metric.Int64Counter
}

func CreateTelemetryInstrumentation(versionString string) (*Instrumentation, error) {
	tracer := otel.Tracer(
		TracerName,
		trace.WithInstrumentationVersion(versionString),
	)

	meter := otel.Meter(MetricName, metric.WithInstrumentationVersion(versionString))
	toolsetGet, err := meter.Int64Counter(
		toolsetGetCountName,
		metric.WithDescription("Number of toolset GET API calls."),
		metric.WithUnit("{call}"),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create %s metric: %w", toolsetGetCountName, err)
	}

	toolGet, err := meter.Int64Counter(
		toolGetCountName,
		metric.WithDescription("Number of tool GET API calls."),
		metric.WithUnit("{call}"),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create %s metric: %w", toolGetCountName, err)
	}

	toolInvoke, err := meter.Int64Counter(
		toolInvokeCountName,
		metric.WithDescription("Number of tool Invoke API calls."),
		metric.WithUnit("{call}"),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create %s metric: %w", toolInvokeCountName, err)
	}

	mcpSse, err := meter.Int64Counter(
		mcpSseCountName,
		metric.WithDescription("Number of MCP SSE connection requests."),
		metric.WithUnit("{connection}"),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create %s metric: %w", mcpSseCountName, err)
	}

	mcpPost, err := meter.Int64Counter(
		mcpPostCountName,
		metric.WithDescription("Number of MCP Post calls."),
		metric.WithUnit("{call}"),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create %s metric: %w", mcpPostCountName, err)
	}

	instrumentation := &Instrumentation{
		Tracer:     tracer,
		meter:      meter,
		ToolsetGet: toolsetGet,
		ToolGet:    toolGet,
		ToolInvoke: toolInvoke,
		McpSse:     mcpSse,
		McpPost:    mcpPost,
	}
	return instrumentation, nil
}
