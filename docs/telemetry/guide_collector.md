# Use collector to export telemetry (trace and metric) data
Collector receives telemetry data, processes the telemetry, and exports it to a wide variety of observability backends using its components. For a conceptual overview of the Collector, see [Collector][collector].

[collector]: https://opentelemetry.io/docs/collector/

## Using a Collector

1.  **Obtain a Collector binary.** Pull a binary or Docker image for the
    OpenTelemetry contrib collector which includes the GCP exporter plugin
    through one of the following:

    *   Download a [binary or package of the OpenTelemetry
        Collector Contrib](https://github.com/open-telemetry/opentelemetry-collector-releases/releases)
        that is appropriate for your platform, and includes the Google Cloud
        exporter.
    *   Pull a Docker image with `docker pull otel/opentelemetry-collector-contrib`
    *   Create your own main package in Go, that pulls in just the plugins you need.
    *   Use the [OpenTelemetry Collector
        Builder](https://github.com/open-telemetry/opentelemetry-collector-builder)
        to generate the Go main package and `go.mod`.

1. **Set up credentials.** Enable APIs (if using GCP exporters) or set up telemetry backend.

    if using GCP exporters:
    * cloud metrics and cloud trace APIs
    * Ensure that your GCP user has (at minimum) `roles/monitoring.metricWriter` and `roles/cloudtrace.agent`.

1. **Set up the Collector config.** An example of collector config is [provided](#collector-config).

1. **Run the Collector.**

    ```bash
    ./otelcol-contrib --config=collector-config.yaml
    ```

1. **Run toolbox.** Configure it to send them to `http://127.0.0.1:4553` (for HTTP) or the Collector's URL using the `--telemetry-otlp` flag.

    ```bash
    ./toolbox --telemetry-otlp=http://127.0.0.1:4553
    ```

1. **View telemetry.** If you are using GCP exporters, telemetry will be visible in GCP dashboard at [Metrics Explorer][metrics-explorer] and [Trace Explorer][trace-explorer].

[metrics-explorer]: https://console.cloud.google.com/monitoring/metrics-explorer
[trace-explorer]: https://console.cloud.google.com/traces

## Collector config
The below example uses the [Google Cloud Exporter][google-cloud-exporter] (for traces) and the [Google Managed Service for Prometheus Exporter][google-prometheus-exporter] (for metrics).

[google-cloud-exporter]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/exporter/googlecloudexporter
[google-prometheus-exporter]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/exporter/googlemanagedprometheusexporter

```
receivers:
  # Receive OTLP from our application
  otlp:
    protocols:
      http:
        endpoint: "0.0.0.0:4553"

exporters:
  # Export logs and traces using the standard googelcloud exporter
  googlecloud:
    project: ${GOOGLE_CLOUD_PROJECT}
  # Export metrics to Google Managed service for Prometheus
  googlemanagedprometheus:
    project: ${GOOGLE_CLOUD_PROJECT}

processors:
  memory_limiter:
    check_interval: 1s
    limit_percentage: 65
    spike_limit_percentage: 20
  resourcedetection:
    detectors: [gcp]
    timeout: 10s
  # Batch telemetry together to more efficiently send to GCP
  batch:
    send_batch_max_size: 200
    send_batch_size: 200
    timeout: 5s
  # Make sure Google Managed service for Prometheus required labels are set
  resource:
    attributes:
      - { key: "location", value: "us-central1", action: "upsert" }
      - { key: "cluster", value: "no-cluster", action: "upsert" }
      - { key: "namespace", value: "no-namespace", action: "upsert" }
      - { key: "job", value: "us-job", action: "upsert" }
      - { key: "instance", value: "us-instance", action: "upsert" }
  # If running on GCP (e.g. on GKE), detect resource attributes from the environment.
  resourcedetection:
    detectors: ["env", "gcp"]

service:
  pipelines:
    traces:
      receivers: ["otlp"]
      processors: ["memory_limiter","batch", "resourcedetection"]
      exporters: ["googlecloud"]
    metrics:
      receivers: ["otlp"]
      processors: ["memory_limiter","batch", "resourcedetection", "resource"]
      exporters: ["googlemanagedprometheus"]
```
