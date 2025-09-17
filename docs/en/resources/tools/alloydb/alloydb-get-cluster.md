---
title: "alloydb-get-cluster"
type: docs
weight: 1
description: >
  The "alloydb-get-cluster" tool retrieves details for a specific AlloyDB cluster.
aliases:
- /resources/tools/alloydb-get-cluster
---

## About

The `alloydb-get-cluster` tool retrieves detailed information for a single, specified AlloyDB cluster. It is compatible with [alloydb-admin](../../sources/alloydb-admin.md) source.
	
| Parameter  | Type   | Description                                                                              | Required |
| :--------- | :----- | :--------------------------------------------------------------------------------------- | :------- |
| `project`  | string | The GCP project ID to get cluster for.                                                   | Yes      |
| `location` | string | The location of the cluster (e.g., 'us-central1').                                       | Yes      |
| `cluster`  | string | The ID of the cluster to retrieve.                                                       | Yes      |

## Example

```yaml
tools:
  get_specific_cluster:
    kind: alloydb-get-cluster
    source: my-alloydb-admin-source
    description: Use this tool to retrieve details for a specific AlloyDB cluster.
```
## Reference
| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be alloydb-get-cluster.                                                                  |
| source      |                   string                   |     true     | The name of an `alloydb-admin` source.                                                                               |
| description |                   string                   |     false    | Description of the tool that is passed to the agent.                                                                                |
