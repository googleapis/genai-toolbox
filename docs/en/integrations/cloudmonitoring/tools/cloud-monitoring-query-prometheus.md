---
title: cloud-monitoring-query-prometheus
type: docs
weight: 1
description: The "cloud-monitoring-query-prometheus" tool fetches time series metrics for a project using a given prometheus query.
---

The `cloud-monitoring-query-prometheus` tool fetches timeseries metrics data
from Google Cloud Monitoring for a project using a given prometheus query.

## About

The `cloud-monitoring-query-prometheus` tool allows you to query all metrics
available in Google Cloud Monitoring using the Prometheus Query Language
(PromQL).

### Use Cases

- **Ad-hoc analysis:** Quickly investigate performance issues by executing
  direct promql queries for a database instance.
- **Prebuilt Configs:** Use the already added prebuilt tools mentioned in
  prebuilt-tools.md to query the databases system/query level metrics.

Here are some common use cases for the `cloud-monitoring-query-prometheus` tool:

- **Monitoring resource utilization:** Track CPU, memory, and disk usage for
  your database instance (Can use the [prebuilt
  tools](../../../documentation/configuration/prebuilt-configs/_index.md)).
- **Monitoring query performance:** Monitor latency, execution_time, wait_time
  for database instance or even for the queries running (Can use the [prebuilt
  tools](../../../documentation/configuration/prebuilt-configs/_index.md)).
- **System Health:** Get the overall system health for the database instance
  (Can use the [prebuilt tools](../../../documentation/configuration/prebuilt-configs/_index.md)).

## Compatible Sources

{{< compatible-sources >}}

## Requirements

To use this tool, you need to have the following IAM role on your Google Cloud
project:

- `roles/monitoring.viewer`

## Parameters

| Name        | Type   | Description                      |
|-------------|--------|----------------------------------|
| `projectId` | string | The Google Cloud project ID.     |
| `query`     | string | The Prometheus query to execute. |


## Example

Here are some examples of how to use the `cloud-monitoring-query-prometheus`
tool.

```yaml
kind: tool
name: get_wait_time_metrics
type: cloud-monitoring-query-prometheus
source: cloud-monitoring-source
description: |
  This tool fetches system wait time information for AlloyDB cluster, instance. Get the `projectID`, `clusterID` and `instanceID` from the user intent. To use this tool, you must provide the Google Cloud `projectId` and a PromQL `query`.
  Generate `query` using these metric details:
  metric: `alloydb.googleapis.com/instance/postgresql/wait_time`,  monitored_resource: `alloydb.googleapis.com/Instance`. labels: `cluster_id`, `instance_id`, `wait_event_type`, `wait_event_name`.
  Basic time series example promql query: `avg_over_time({"__name__"="alloydb.googleapis.com/instance/postgresql/wait_time","monitored_resource"="alloydb.googleapis.com/Instance","instance_id"="alloydb-instance"}[5m])`
```

## Reference

| **field**   | **type** | **required** | **description**                                      |
|-------------|:--------:|:------------:|------------------------------------------------------|
| type        |  string  |     true     | Must be cloud-monitoring-query-prometheus.           |
| source      |  string  |     true     | The name of an `cloud-monitoring` source.            |
| description |  string  |     true     | Description of the tool that is passed to the agent. |
