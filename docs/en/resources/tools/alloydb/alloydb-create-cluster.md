---
title: alloydb-create-cluster
type: docs
weight: 1
description: "The \"alloydb-create-cluster\" tool creates a new AlloyDB for PostgreSQL cluster in a specified project and location.\n"
aliases: [/resources/tools/alloydb-create-cluster]
---

## About

The `alloydb-create-cluster` tool creates a new AlloyDB for PostgreSQL cluster in a specified project and location. It is compatible with [alloydb-admin](../../sources/alloydb-admin.md) source.
This tool provisions a cluster with a **private IP address** within the specified VPC network.

  **Permissions & APIs Required:**
  Before using, ensure the following on your GCP project:

1.  The [AlloyDB API](https://console.cloud.google.com/apis/library/alloydb.googleapis.com) is enabled.
2.  The user or service account executing the tool has one of the following IAM roles:


    - `roles/alloydb.admin` (the AlloyDB Admin predefined IAM role)
    - `roles/owner` (the Owner basic IAM role)
    - `roles/editor` (the Editor basic IAM role)

The tool takes the following input parameters:

| Parameter  | Type   | Description                                                                                                               | Required |
| :--------- | :----- | :------------------------------------------------------------------------------------------------------------------------ | :------- |
| `project`  | string | The GCP project ID where the cluster will be created.                                                                     | Yes      |
| `cluster`  | string | A unique identifier for the new AlloyDB cluster.                                                                          | Yes      |
| `password` | string | A secure password for the initial user.                                                                                   | Yes      |
| `location` | string | The GCP location where the cluster will be created. Default: `us-central1`. If quota is exhausted then use other regions. | No       |
| `network`  | string | The name of the VPC network to connect the cluster to. Default: `default`.                                                | No       |
| `user`     | string | The name for the initial superuser. Default: `postgres`.                                                                  | No       |

## Example

```yaml
tools:
  create_cluster:
    kind: alloydb-create-cluster
    source: alloydb-admin-source
    description: Use this tool to create a new AlloyDB cluster in a given project and location.
```

## Reference

| **field**   | **type** | **required** | **description**                                      |   |
| ----------- | :------: | :----------: | ---------------------------------------------------- | - |
| kind        |  string  |     true     | Must be alloydb-create-cluster.                      |   |
| source      |  string  |     true     | The name of an `alloydb-admin` source.               |   |
| description |  string  |     false    | Description of the tool that is passed to the agent. |   |
