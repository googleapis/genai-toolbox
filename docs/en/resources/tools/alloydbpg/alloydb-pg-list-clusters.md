---
title: "alloydb-pg-list-clusters"
type: docs
weight: 1
description: >
  The "alloydb-pg-list-clusters" tool lists the AlloyDB clusters in a given project and location.
aliases:
- /resources/tools/alloydb-pg-list-clusters
---

## About

The `alloydb-pg-list-clusters` tool retrieves AlloyDB cluster information for all or specified locations in a given project.

`alloydb-pg-list-clusters` tool lists the detailed information of AlloyDB cluster(cluster name, state, configuration, etc) for a given project and location. The tool takes the following input parameters:
	* `project` : The GCP project ID to list clusters for.
	* `location` (optional): The location to list clusters in (e.g., 'us-central1'). Use '-' to list clusters across all locations. Default: `"-"`.

## Example

```yaml
tools:
  alloydb_pg_list_clusters:
    kind: alloydb-pg-list-clusters
    description: Use this tool to list all AlloyDB clusters in a given project and location.
```
## Reference
| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be alloydb-pg-list-clusters.                                                                  |                                               |
| description |                   string                   |     true     | Description of the tool that is passed to the agent.                                             |