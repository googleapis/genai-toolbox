---
title: "alloydb-create-cluster"
type: docs
weight: 1
description: >
  The "alloydb-create-cluster" tool creates a new AlloyDB for PostgreSQL cluster in a specified project and location.
aliases:
- /resources/tools/alloydb-create-cluster
---

## About

The `alloydb-create-cluster` tool creates a new AlloyDB for PostgreSQL cluster in a specified project and location. It is compatible with [http](../../sources/http.md) source.
This tool provisions a cluster with a **private IP address** within the specified VPC network.

  **Permissions & APIs Required:**
  Before using, ensure the following on your GCP project:
  1. The [AlloyDB API](https://console.cloud.google.com/apis/library/alloydb.googleapis.com) is enabled.
  2. The user or service account executing the tool has the following IAM roles:
     - `roles/alloydb.admin`: To create and manage the AlloyDB cluster.

The tool takes the following input parameters:

| Parameter | Type | Description | Required |
| :--- | :--- | :--- | :--- |
| `projectId` | string | The GCP project ID where the cluster will be created. | Yes |
| `locationId` | string | The GCP location where the cluster will be created. | Yes |
| `clusterId` | string | A unique identifier for the new AlloyDB cluster. | Yes |
| `password` | string | A secure password for the initial user. | Yes |
| `network` | string | The name of the VPC network to connect the cluster to. Default: `default`. | No |
| `user` | string | The name for the initial superuser. Default: `postgres`. | No |
> **Note**
> This tool authenticates using the environment's
[Application Default Credentials](https://cloud.google.com/docs/authentication/application-default-credentials).
## Example

```yaml
tools:
  alloydb_create_cluster:
    kind: alloydb-create-cluster
    source: http-source
    description: Use this tool to create a new AlloyDB cluster in a given project and location.
```
## Reference
| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be alloydb-create-cluster.                                                                  |                                               |
| source      |                   string                   |     true     | The name of a http source.                                                                       |
| description |                   string                   |     true     | Description of the tool that is passed to the agent.                                             |