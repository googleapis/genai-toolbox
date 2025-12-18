---
title: alloydb-list-clusters
type: docs
weight: 1
description: "The \"alloydb-list-clusters\" tool lists the AlloyDB clusters in a given project and location.\n"
aliases: [/resources/tools/alloydb-list-clusters]
---

## About

The `alloydb-list-clusters` tool retrieves AlloyDB cluster information for all
or specified locations in a given project. It is compatible with
[alloydb-admin](../../sources/alloydb-admin.md) source.

`alloydb-list-clusters` tool lists the detailed information of AlloyDB
cluster(cluster name, state, configuration, etc) for a given project and
location. The tool takes the following input parameters:

| Parameter  | Type   | Description                                                                                      | Required |
| :--------- | :----- | :----------------------------------------------------------------------------------------------- | :------- |
| `project`  | string | The GCP project ID to list clusters for.                                                         | Yes      |
| `location` | string | The location to list clusters in (e.g., 'us-central1'). Use `-` for all locations. Default: `-`. | No       |

## Example

```yaml
tools:
  list_clusters:
    kind: alloydb-list-clusters
    source: alloydb-admin-source
    description: Use this tool to list all AlloyDB clusters in a given project and location.
```

## Reference

| **field**   | **type** | **required** | **description**                                      |
| ----------- | :------: | :----------: | ---------------------------------------------------- |
| kind        |  string  |     true     | Must be alloydb-list-clusters.                       |
| source      |  string  |     true     | The name of an `alloydb-admin` source.               |
| description |  string  |     false    | Description of the tool that is passed to the agent. |
