---
title: "alloydb-get-user"
type: docs
weight: 1
description: >
  The "alloydb-get-user" tool retrieves details for a specific AlloyDB user.
aliases:
- /resources/tools/alloydb-get-user
---

## About

The `alloydb-get-user` tool retrieves detailed information for a single, specified AlloyDB user. It is compatible with [alloydb-admin](../../sources/alloydb-admin.md) source.
	
| Parameter  | Type   | Description                                                                              | Required |
| :--------- | :----- | :--------------------------------------------------------------------------------------- | :------- |
| `project`  | string | The GCP project ID to get user for.                                                 | Yes      |
| `location` | string | The location of the cluster (e.g., 'us-central1'). | Yes      |
| `cluster` | string | The ID of the cluster to retrieve the user from. | Yes      |
| `user` | string | The ID of the user to retrieve. | Yes      |
> **Note**
> This tool authenticates using the credentials configured in its [alloydb-admin](../../sources/alloydb-admin.md) source which can be either [Application Default Credentials](https://cloud.google.com/docs/authentication/application-default-credentials) or client-side OAuth.

## Example

```yaml
tools:
  get_specific_user:
    kind: alloydb-get-user
    source: my-alloydb-admin-source
    description: Use this tool to retrieve details for a specific AlloyDB user.
```
## Reference
| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be alloydb-get-user.                                                                  |                                               |
| source      |                   string                   |     true     | The name of an `alloydb-admin` source.                                                                       |
| description |                   string                   |     true     | Description of the tool that is passed to the agent.                                             |