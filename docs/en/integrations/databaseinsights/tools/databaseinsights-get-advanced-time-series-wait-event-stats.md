---
title: "databaseinsights-get-advanced-time-series-wait-event-stats"
type: docs
weight: 4
description: >
  Fetches time-series wait event statistics for AlloyDB instances over time.
---

## About

The `databaseinsights-get-advanced-time-series-wait-event-stats` tool retrieves time-series wait event metrics (IO, Lock, CPU) for an AlloyDB instance over a specified period to visualize how contention evolves.

## Compatible Sources

{{< compatible-sources >}}

## Requirements

### IAM Permissions

- `databaseinsights.waitEventTimeSeries.fetch` on the target location/project.

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

Filter wait event history to a specific database.

### `username` (String, Optional)

Filter wait event history to a specific database user.

### `query_id` (String, Optional)

Breakdown wait events over time for a specific query hash.

### `view` (String, Optional)

Aggregation level (`WAIT_CLASS` or `WAIT_EVENT`). Defaults to `WAIT_CLASS`.

## Example

```yaml
kind: tool
name: get_wait_event_time_series
type: databaseinsights-get-advanced-time-series-wait-event-stats
source: database-insights-source
```

## Output Format

Returns a dataset of time-series points showing the rate of time spent in wait states.

## Reference

| **field**   | **type** | **required** | **description**                                    |
| :---------- | :------: | :----------: | :------------------------------------------------- |
| type        |  string  |     true     | Must be "databaseinsights-get-advanced-time-series-wait-event-stats". |
| source      |  string  |     true     | Name of the source the tool should execute on.     |
| description |  string  |    false     | Optional description override.                     |
