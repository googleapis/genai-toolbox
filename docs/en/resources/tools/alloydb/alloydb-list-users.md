---
title: "alloydb-list-users"
type: docs
weight: 1
description: >
  The "alloydb-list-users" tool lists all database users within an AlloyDB cluster.
aliases:
- /resources/tools/alloydb-list-users
---

## About

The `alloydb-list-users` tool lists all database users within an AlloyDB cluster. It is compatible with [alloydb-admin](../../sources/alloydb-admin.md) source.
The tool takes the following input parameters:
	
| Parameter  | Type   | Description                                                                              | Required |
| :--------- | :----- | :--------------------------------------------------------------------------------------- | :------- |
| `project`  | string | The GCP project ID to list users for.                                                    | Yes      |
| `cluster`  | string | The ID of the cluster to list users from.                                                | Yes      |
| `location` | string | The location of the cluster (e.g., 'us-central1').                                       | Yes      |

## Example

```yaml
tools:
  list_users:
    kind: alloydb-list-users
    source: alloydb-admin-source
    description: Use this tool to list all database users within an AlloyDB cluster
```
## Reference
| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be alloydb-list-users.                                                                  |
| source      |                   string                   |     true     | The name of an `alloydb-admin` source.                                                                       |
| description |                   string                   |     false    | Description of the tool that is passed to the agent.                                             |