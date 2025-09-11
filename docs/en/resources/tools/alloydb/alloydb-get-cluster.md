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
| `projectId`  | string | The GCP project ID to get cluster for.                                                 | Yes      |
| `locationId` | string | The location of the cluster (e.g., 'us-central1'). | Yes      |
| `clusterId` | string | The ID of the cluster to retrieve. | Yes      |
> **Note**
> This tool authenticates using the credentials configured in its [alloydb-admin](../../sources/alloydb-admin.md) source which can be either [Application Default Credentials](https://cloud.google.com/docs/authentication/application-default-credentials) or client-side OAuth.

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
| kind        |                   string                   |     true     | Must be alloydb-get-cluster.                                                                  |                                               |
| source      |                   string                   |     true     | The name of an `alloydb-admin` source.                                                                       |
| description |                   string                   |     true     | Description of the tool that is passed to the agent.                                             |