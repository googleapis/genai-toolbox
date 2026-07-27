---
title: "databaseinsights-get-advanced-time-series-query-stats"
type: docs
weight: 3
description: >
  Fetches time-series performance trends for queries executed on AlloyDB instances.
---

## About

The `databaseinsights-get-advanced-time-series-query-stats` tool retrieves historical time-series performance metrics for AlloyDB queries to help analyze latency trends over time and identify performance spikes.

## Compatible Sources

{{< compatible-sources >}}

## Requirements

### IAM Permissions

- `databaseinsights.queryTimeSeries.fetch` on the target location/project.

## Parameters

### `parent` (String, Required)

Project and location in the format `projects/{project_id}/locations/{location}`.

### `full_resource_name` (String, Required)

The full resource identifier for the database instance (e.g., `//alloydb.googleapis.com/projects/{project_id}/locations/{location}/clusters/{cluster_id}/instances/{instance_id}`).

### `start_time` (String, Optional)

Beginning of the time interval in RFC3339 format.

### `end_time` (String, Optional)

End of the time interval in RFC3339 format.

### `database` (String, Optional)

Filter results to a specific database.

### `username` (String, Optional)

Filter results to a specific database user.

### `query_id` (String, Optional)

Fetch history for a single specific query hash.

## Example

```yaml
kind: tool
name: get_query_time_series
type: databaseinsights-get-advanced-time-series-query-stats
source: database-insights-source
```

## Output Format

Returns a list of time-series data points with metrics like rate of execution time and execution count.

## Reference

| **field**   | **type** | **required** | **description**                                    |
| :---------- | :------: | :----------: | :------------------------------------------------- |
| type        |  string  |     true     | Must be "databaseinsights-get-advanced-time-series-query-stats". |
| source      |  string  |     true     | Name of the source the tool should execute on.     |
| description |  string  |    false     | Optional description override.                     |
