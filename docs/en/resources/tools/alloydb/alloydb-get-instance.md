---
title: "alloydb-get-instance"
type: docs
weight: 1
description: >
  The "alloydb-get-instance" tool retrieves details for a specific AlloyDB instance.
aliases:
- /resources/tools/alloydb-get-instance
---

## About

The `alloydb-get-instance` tool retrieves detailed information for a single, specified AlloyDB instance. It is compatible with [alloydb-admin](../../sources/alloydb-admin.md) source.
	
| Parameter  | Type   | Description                                                                              | Required |
| :--------- | :----- | :--------------------------------------------------------------------------------------- | :------- |
| `project`  | string | The GCP project ID to get instance for.                                                  | Yes      |
| `location` | string | The location of the instance (e.g., 'us-central1').                                      | Yes      |
| `cluster`  | string | The ID of the cluster.                                                                   | Yes      |
| `instance` | string | The ID of the instance to retrieve.                                                      | Yes      |

## Example

```yaml
tools:
  get_specific_instance:
    kind: alloydb-get-instance
    source: my-alloydb-admin-source
    description: Use this tool to retrieve details for a specific AlloyDB instance.
```
## Reference
| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be alloydb-get-instance.                                                                  |
| source      |                   string                   |     true     | The name of an `alloydb-admin` source.                                                                       |
| description |                   string                   |     false    | Description of the tool that is passed to the agent.                                             |
