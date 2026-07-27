---
title: "databaseinsights-get-index-recommendations"
type: docs
weight: 5
description: >
  Fetches index recommendations for specified query IDs across databases on AlloyDB instances.
---

## About

The `databaseinsights-get-index-recommendations` tool retrieves index advisor recommendations for specified query IDs across databases. It identifies missing indexes, provides exact SQL creation commands, and calculates estimated performance gains.

## Compatible Sources

{{< compatible-sources >}}

## Requirements

### IAM Permissions

- `databaseinsights.indexRecommendations.batchQuery` on the target location/project.

## Parameters

### `parent` (String, Required)

Project and location in the format `projects/{project_id}/locations/{location}`.

### `full_resource_name` (String, Required)

The full resource identifier for the database instance (e.g., `//alloydb.googleapis.com/projects/{project_id}/locations/{location}/clusters/{cluster_id}/instances/{instance_id}`).

### `database_query_ids` (Array of Objects, Required)

A list of objects specifying database names and target query IDs (e.g., `[{"database": "dbname", "query_ids": ["12345"]}]`).

## Example

```yaml
kind: tool
name: get_index_recommendations
type: databaseinsights-get-index-recommendations
source: database-insights-source
```

## Output Format

Returns index recommendations (`sql_command`, `schema`, `relation`, `columns`, `estimated_storage_size_bytes`) and predicted query performance improvements.

## Reference

| **field**   | **type** | **required** | **description**                                    |
| :---------- | :------: | :----------: | :------------------------------------------------- |
| type        |  string  |     true     | Must be "databaseinsights-get-index-recommendations". |
| source      |  string  |     true     | Name of the source the tool should execute on.     |
| description |  string  |    false     | Optional description override.                     |
