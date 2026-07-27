---
title: "databaseinsights-get-advanced-aggregated-wait-event-stats"
type: docs
weight: 2
description: >
  Fetches aggregated wait event statistics for AlloyDB instances to identify performance bottlenecks.
---

## About

The `databaseinsights-get-advanced-aggregated-wait-event-stats` tool retrieves aggregated wait event metrics (IO, Lock, CPU contention) to help diagnose database performance bottlenecks.

## Compatible Sources

{{< compatible-sources >}}

## Requirements

### IAM Permissions

- `databaseinsights.waitEventStats.fetch` on the target location/project.

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

Filter wait event stats to a specific database.

### `username` (String, Optional)

Filter wait event stats to a specific database user.

### `query_id` (String, Optional)

Breakdown wait events for a specific query hash.

### `page_size` (Integer, Optional)

Maximum number of results to return (Default: `20`).

### `page_token` (String, Optional)

Token for fetching subsequent pages of results.

### `view` (String, Optional)

Aggregation level (`WAIT_CLASS` or `WAIT_EVENT`). Defaults to `WAIT_CLASS`.

## Example

```yaml
kind: tool
name: get_aggregated_wait_event_stats
type: databaseinsights-get-advanced-aggregated-wait-event-stats
source: database-insights-source
```

## Output Format

Returns a structured object containing `results` (wait event categories and time spent) and `metadata`.

## Reference

| **field**   | **type** | **required** | **description**                                    |
| :---------- | :------: | :----------: | :------------------------------------------------- |
| type        |  string  |     true     | Must be "databaseinsights-get-advanced-aggregated-wait-event-stats". |
| source      |  string  |     true     | Name of the source the tool should execute on.     |
| description |  string  |    false     | Optional description override.                     |
