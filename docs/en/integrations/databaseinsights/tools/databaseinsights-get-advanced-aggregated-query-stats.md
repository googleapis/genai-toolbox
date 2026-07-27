---
title: "databaseinsights-get-advanced-aggregated-query-stats"
type: docs
weight: 1
description: >
  Fetches aggregated query performance statistics (latency, execution count, wait time) for AlloyDB instances.
---

## About

The `databaseinsights-get-advanced-aggregated-query-stats` tool fetches aggregated performance metrics for queries executed on AlloyDB instances with Advanced Query Insights enabled. It ranks queries by total execution time (`sum(execution_time) desc`) to help identify top resource-consuming queries.

## Compatible Sources

{{< compatible-sources >}}

## Requirements

### IAM Permissions

- `databaseinsights.queryStats.fetch` on the target location/project.

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

Filter stats to a specific database.

### `username` (String, Optional)

Filter stats to a specific database user.

### `query_id` (String, Optional)

Fetch stats for a single specific query hash.

### `page_size` (Integer, Optional)

Maximum number of query stats to return (Default: `20`).

### `page_token` (String, Optional)

Token for fetching subsequent pages of results.

## Example

```yaml
kind: tool
name: get_aggregated_query_stats
type: databaseinsights-get-advanced-aggregated-query-stats
source: database-insights-source
```

## Output Format

Returns a structured object containing `results` (rows of metrics) and `metadata` (schema columns).

## Reference

| **field**   | **type** | **required** | **description**                                    |
| :---------- | :------: | :----------: | :------------------------------------------------- |
| type        |  string  |     true     | Must be "databaseinsights-get-advanced-aggregated-query-stats". |
| source      |  string  |     true     | Name of the source the tool should execute on.     |
| description |  string  |    false     | Optional description override.                     |
