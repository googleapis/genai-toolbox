---
title: "alloydb-pg-create-cluster"
type: docs
weight: 1
description: >
  The "alloydb-pg-create-cluster" tool creates a new AlloyDB for PostgreSQL cluster in a specified project and location.
aliases:
- /resources/tools/alloydb-pg-create-cluster
---

## About

The `alloydb-pg-create-cluster` tool creates a new AlloyDB for PostgreSQL cluster in a specified project and location.
The tool takes the following input parameters:
	* `project` : The GCP project ID where the cluster will be created.
    * `clusterId`: A unique identifier for the new AlloyDB cluster.
    * `location` (optional): The GCP location where the cluster will be created. Default: `us-central1`.
    * `network` (optional): The name of the VPC network to connect the cluster to. Default: `default`.
	* `user` (optional): The name for the initial superuser. Default: `postgres`. The initial database will always be named 'postgres'.
    * `password` : A secure password for the initial 'postgres' user or the custom user provided.

## Example

```yaml
tools:
  alloydb_pg_create_cluster:
    kind: alloydb-pg-create-cluster
    description: Use this tool to create a new AlloyDB cluster in a given project and location.
```
## Reference
| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be alloydb-pg-create-cluster.                                                                  |                                               |
| description |                   string                   |     true     | Description of the tool that is passed to the agent.                                             |