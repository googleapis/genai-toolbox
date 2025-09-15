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
| `projectId`  | string | The GCP project ID to list users for.                                                 | Yes      |
| `clusterId` | string | The ID of the cluster to list users from.                                                | Yes      |
| `locationId` | string | The location of the cluster (e.g., 'us-central1'). | Yes       |
> **Note**
> This tool authenticates using the credentials configured in its [alloydb-admin](../../sources/alloydb-admin.md) source which can be either [Application Default Credentials](https://cloud.google.com/docs/authentication/application-default-credentials) or client-side OAuth.

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
| kind        |                   string                   |     true     | Must be alloydb-list-users.                                                                  |                                               |
| source      |                   string                   |     true     | The name of an `alloydb-admin` source.                                                                       |
| description |                   string                   |     true     | Description of the tool that is passed to the agent.                                             |