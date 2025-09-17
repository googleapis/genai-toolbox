---
title: alloydb-list-instances
type: docs
weight: 1
description: "The \"alloydb-list-instances\" tool lists the AlloyDB instances for a given project, cluster and location.\n"
aliases: [/resources/tools/alloydb-list-instances]
---

## About

The `alloydb-list-instances` tool retrieves AlloyDB instance information for all or specified clusters and locations in a given project. It is compatible with [alloydb-admin](../../sources/alloydb-admin.md) source.

`alloydb-list-instances` tool lists the detailed information of AlloyDB instances (instance name, type, IP address, state, configuration, etc) for a given project, cluster and location. The tool takes the following input parameters:

| Parameter  | Type   | Description                                                                                                | Required |
| :--------- | :----- | :--------------------------------------------------------------------------------------------------------- | :------- |
| `project`  | string | The GCP project ID to list instances for.                                                                  | Yes      |
| `cluster`  | string | The ID of the cluster to list instances from. Use '-' to get results for all clusters. Default: `-`.       | No       |
| `location` | string | The location of the cluster (e.g., 'us-central1'). Use '-' to get results for all locations. Default: `-`. | No       |

## Example

```yaml
tools:
  list_instances:
    kind: alloydb-list-instances
    source: alloydb-admin-source
    description: Use this tool to list all AlloyDB instances for a given project, cluster and location.
```

## Reference

| **field**   | **type** | **required** | **description**                                      |
| ----------- | :------: | :----------: | ---------------------------------------------------- |
| kind        |  string  |     true     | Must be alloydb-list-instances.                      |
| source      |  string  |     true     | The name of an `alloydb-admin` source.               |
| description |  string  |     false    | Description of the tool that is passed to the agent. |
