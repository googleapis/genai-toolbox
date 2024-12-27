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
)

const (
	InstrumentationName = "github.com/googleapis/genai-toolbox/internal/opentel"

	toolsetGetCountName = "toolbox.server.toolset.get.count"
	toolGetCountName    = "toolbox.server.tool.get.count"
	toolInvokeCountName = "toolbox.server.tool.invoke.count"
	operationActiveName = "toolbox.server.operation.active"
)

type Metric struct {
	meter                        metric.Meter
	toolsetGetCounter            metric.Int64Counter
	toolGetCounter               metric.Int64Counter
	toolInvokeCounter            metric.Int64Counter
	operationActiveUpDownCounter metric.Int64UpDownCounter
}

// createCustomMetric creates all the custom metrics for toolbox
func createCustomMetric(versionString string) (*Metric, error) {
	meter := otel.Meter(InstrumentationName, metric.WithInstrumentationVersion(versionString))
	toolsetGetCounter, err := meter.Int64Counter(
		toolsetGetCountName,
		metric.WithDescription("Number of toolset GET API calls."),
		metric.WithUnit("{call}"),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create %s metric: %w", toolsetGetCountName, err)
	}

	toolGetCounter, err := meter.Int64Counter(
		toolGetCountName,
		metric.WithDescription("Number of tool GET API calls."),
		metric.WithUnit("{call}"),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create %s metric: %w", toolGetCountName, err)
	}

	toolInvokeCounter, err := meter.Int64Counter(
		toolInvokeCountName,
		metric.WithDescription("Number of tool Invoke API calls."),
		metric.WithUnit("{call}"),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create %s metric: %w", toolInvokeCountName, err)
	}

	operationActiveUpDownCounter, err := meter.Int64UpDownCounter(
		operationActiveName,
		metric.WithDescription("Number of active request."),
		metric.WithUnit("{call}"),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create %s metric: %w", operationActiveName, err)
	}

	customMetric := &Metric{
		meter:                        meter,
		toolsetGetCounter:            toolsetGetCounter,
		toolGetCounter:               toolGetCounter,
		toolInvokeCounter:            toolInvokeCounter,
		operationActiveUpDownCounter: operationActiveUpDownCounter,
	}
	return customMetric, nil
}

// ToolsetGetCounter retrieves the toolsetGetCounter metric
func (m *Metric) ToolsetGetCounter() metric.Int64Counter {
	return m.toolsetGetCounter
}

// ToolGetCounter retrieves the toolGetCounter metric
func (m *Metric) ToolGetCounter() metric.Int64Counter {
	return m.toolGetCounter
}

// ToolInvokeCounter retrieves the toolInvokeCounter metric
func (m *Metric) ToolInvokeCounter() metric.Int64Counter {
	return m.toolInvokeCounter
}

// OperationActiveUpDownCounter retrieves the operationActiveUpDownCounter metric
func (m *Metric) OperationActiveUpDownCounter() metric.Int64UpDownCounter {
	return m.operationActiveUpDownCounter
}
