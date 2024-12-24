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
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/contrib/propagators/autoprop"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// setupOTelSDK bootstraps the OpenTelemetry pipeline.
// If it does not return an error, make sure to call shutdown for proper cleanup.
func SetupOTel(ctx context.Context, versionString string) (shutdown func(context.Context) error, err error) {
	var shutdownFuncs []func(context.Context) error

	// shutdown calls cleanup functions registered via shutdownFuncs.
	// The errors from the calls are joined.
	// Each registered cleanup will be invoked once.
	shutdown = func(ctx context.Context) error {
		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		shutdownFuncs = nil
		return err
	}

	// handleErr calls shutdown for cleanup and makes sure that all errors are returned.
	handleErr := func(inErr error) {
		err = errors.Join(inErr, shutdown(ctx))
	}

	// Configure Context Propagation to use the default W3C traceparent format.
	otel.SetTextMapPropagator(autoprop.NewTextMapPropagator())

	res, err := newResource(versionString)
	if err != nil {
		errMsg := fmt.Errorf("unable to set up resource: %w", err)
		handleErr(errMsg)
		return
	}

	tracerProvider, err := newTracerProvider(res)
	if err != nil {
		errMsg := fmt.Errorf("unable to set up trace provider: %w", err)
		handleErr(errMsg)
		return
	}
	shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
	otel.SetTracerProvider(tracerProvider)

	meterProvider, err := newMeterProvider(res)
	if err != nil {
		errMsg := fmt.Errorf("unable to set up meter provider: %w", err)
		handleErr(errMsg)
		return
	}
	shutdownFuncs = append(shutdownFuncs, meterProvider.Shutdown)
	otel.SetMeterProvider(meterProvider)

	return shutdown, nil
}

// newResource create default resources for telemetry data.
// Resource represents the entity producing telemetry.
func newResource(versionString string) (*resource.Resource, error) {
	// Ensure default SDK resources and the required service name are set.
	r, err := resource.New(
		context.Background(),
		resource.WithFromEnv(),                    // Discover and provide attributes from OTEL_RESOURCE_ATTRIBUTES and OTEL_SERVICE_NAME environment variables.
		resource.WithTelemetrySDK(),               // Discover and provide information about the OTel SDK used.
		resource.WithOS(),                         // Discover and provide OS information.
		resource.WithContainer(),                  // Discover and provide container information.
		resource.WithHost(),                       //Discover and provide host information.
		resource.WithSchemaURL(semconv.SchemaURL), // Set the schema url.
		resource.WithAttributes( // Add other custom resource attributes.
			semconv.ServiceName("Toolbox"),
			semconv.ServiceVersion(versionString),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("trace provider fail to set up resource: %w", err)
	}
	return r, nil
}

// newTracerProvider creates TracerProvider.
// TracerProvider is a factory for Tracers and is responsible for creating spans.
func newTracerProvider(r *resource.Resource) (*trace.TracerProvider, error) {
	// TODO: stdout used for dev, will be updated to OTLP exporter
	traceExporter, err := stdouttrace.New(
		stdouttrace.WithPrettyPrint())
	if err != nil {
		return nil, err
	}

	traceProvider := trace.NewTracerProvider(
		trace.WithBatcher(traceExporter,
			// TODO: Default is 5s. Set to 1s for dev purposes.
			trace.WithBatchTimeout(time.Second)),
		trace.WithResource(r),
	)
	return traceProvider, nil
}

// newMeterProvider creates MeterProvider.
// MeterProvider is a factory for Meters, and is responsible for creating metrics.
func newMeterProvider(r *resource.Resource) (*metric.MeterProvider, error) {
	// TODO: stdout used for dev, will be updated to OTLP exporter
	metricExporter, err := stdoutmetric.New(
		stdoutmetric.WithPrettyPrint())
	if err != nil {
		return nil, err
	}

	meterProvider := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(metricExporter,
			// TODO: Default is 1m. Set to 10s for dev purposes.
			metric.WithInterval(3*time.Second))),
		metric.WithResource(r),
	)
	return meterProvider, nil
}
