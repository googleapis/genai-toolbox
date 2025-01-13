# Use collector to export telemetry (trace and metric) data
Collector receives telemetry data, processes the telemetry, and exports it to a wide variety of observability backends using its components. 

## Collector
The OpenTelemetry Collector removes the need to run, operate, and maintain multiple
agents/collector. This works well with scalability and supports open source
observability data formats senidng to one or more open source or commercial
backends. In addition, collector also provide other benefits such as allowing
your service to offload data quickly while it take care of additional handling
like retries, batching, encryption, or even sensitive data filtering.

To run a collector, you will have to provide a configuration file. The
configuration file consists of four classes of pipeline component that access
telemetry data.
- `Receivers`
- `Processors`
- `Exporters`
- `Connectors`

Example of setting up the classes of pipeline components (in this example, we
don't use connectors):

```yaml
receivers:
  otlp:
    protocols:
      http:
        endpoint: "127.0.0.1:4553"

exporters:
  googlecloud:
    project: <YOUR_GOOGLE_CLOUD_PROJECT>

processors:
  batch:
    send_batch_size: 200
```

After each pipeline component is configured, you will enable it within the
`service` section of the configuration file.

```yaml
service:
  pipelines:
    traces:
      receivers: ["otlp"]
      processors: ["batch"]
      exporters: ["googlecloud"]
```

For a conceptual overview of the Collector, see [Collector][collector].

[collector]: https://opentelemetry.io/docs/collector/

## Using a Collector
There are a couple of steps to run and use a Collector.

1.  Obtain a Collector binary. Pull a binary or Docker image for the
    OpenTelemetry contrib collector.

1. Set up credentials for telemetry backend.

1. Set up the Collector config.
    Below are some examples for setting up the Collector config:
    - [Google Cloud Exporter][google-cloud-exporter]
    - [Google Managed Service for Prometheus Exporter][google-prometheus-exporter]

1. Run the Collector with the configuration file.

    ```bash
    ./otelcol-contrib --config=collector-config.yaml
    ```

1. Run toolbox with the `--telemetry-otlp` flag. Configure it to send them to
   `http://127.0.0.1:4553` (for HTTP) or the Collector's URL.

    ```bash
    ./toolbox --telemetry-otlp=http://127.0.0.1:4553
    ```

1. Once telemetry datas are collected, you can view them in your telemetry
   backend. If you are using GCP exporters, telemetry will be visible in GCP
   dashboard at [Metrics Explorer][metrics-explorer] and [Trace
   Explorer][trace-explorer].

> [!NOTE]
> If you are exporting to Google Cloud monitoring, we recommend that you use
> the Google Cloud Exporter for traces and the Google Managed Service for
> Prometheus Exporter for metrics.

[google-cloud-exporter]:
    https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/exporter/googlecloudexporter
[google-prometheus-exporter]:
    https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/exporter/googlemanagedprometheusexporter#example-configuration
[metrics-explorer]: https://console.cloud.google.com/monitoring/metrics-explorer
[trace-explorer]: https://console.cloud.google.com/traces
