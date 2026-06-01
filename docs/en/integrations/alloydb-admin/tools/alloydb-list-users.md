---
title: alloydb-list-users
type: docs
weight: 1
description: "The \"alloydb-list-users\" tool lists all database users within an AlloyDB cluster.\n"
---

## About

The `alloydb-list-users` tool lists all database users within an AlloyDB
cluster.

The tool takes the following input parameters:

| Parameter  | Type   | Description                                        | Required |
| :--------- | :----- | :------------------------------------------------- | :------- |
| `project`  | string | The GCP project ID to list users for.              | Yes      |
| `cluster`  | string | The ID of the cluster to list users from.          | Yes      |
| `location` | string | The location of the cluster (e.g., 'us-central1'). | Yes      |


## Compatible Sources

{{< compatible-sources >}}

## Example

```yaml
kind: tool
name: list_users
type: alloydb-list-users
source: alloydb-admin-source
description: Use this tool to list all database users within an AlloyDB cluster
```

## Reference

| **field**   | **type** | **required** | **description**                                      |
| ----------- | :------: | :----------: | ---------------------------------------------------- |
| type        |  string  |     true     | Must be alloydb-list-users.                          |
| source      |  string  |     true     | The name of an `alloydb-admin` source.               |
| description |  string  |     false    | Description of the tool that is passed to the agent. |
