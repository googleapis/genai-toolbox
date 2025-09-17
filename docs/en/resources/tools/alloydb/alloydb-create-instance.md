---
title: "alloydb-create-instance"
type: docs
weight: 1
description: >
  The "alloydb-create-instance" tool creates a new AlloyDB instance within a specified cluster.
aliases:
- /resources/tools/alloydb-create-instance
---

## About

The `alloydb-create-instance` tool creates a new AlloyDB instance (PRIMARY or READ_POOL) within a specified cluster. It is compatible with [alloydb-admin](../../sources/alloydb-admin.md) source.
This tool provisions a new instance with a **public IP address** and sets the `password.enforce_complexity` database flag to `on`, requiring new database passwords to meet complexity rules.

  **Permissions & APIs Required:**
  Before using, ensure the following on your GCP project:
  1. The [AlloyDB API](https://console.cloud.google.com/apis/library/alloydb.googleapis.com) is enabled.
  2. The user or service account executing the tool has one of the following IAM roles:
    - `roles/alloydb.admin` (the AlloyDB Admin predefined IAM role)
    - `roles/owner` (the Owner basic IAM role)
    - `roles/editor` (the Editor basic IAM role)

The tool takes the following input parameters:

| Parameter | Type | Description | Required |
| :--- | :--- | :--- | :--- |
| `project`     | string | The GCP project ID where the cluster exists.                                                      | Yes |
| `location`    | string | The GCP location where the cluster exists (e.g., `us-central1`).                                  | Yes |
| `cluster`     | string | The ID of the existing cluster to add this instance to.                                           | Yes |
| `instance`    | string | A unique identifier for the new AlloyDB instance.                                                 | Yes |
| `instanceType`| string | The type of instance. Valid values are: `PRIMARY` and `READ_POOL`. Default: `PRIMARY`             | No  |
| `displayName` | string | An optional, user-friendly name for the instance.                                                            | No |
| `nodeCount`   | int    | The number of nodes for a read pool. Required only if `instanceType` is `READ_POOL`. Default: `1` | No  |
## Example

```yaml
tools:
  create_instance:
    kind: alloydb-create-instance
    source: alloydb-admin-source
    description: Use this tool to create a new AlloyDB instance within a specified cluster.
```
## Reference
| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be alloydb-create-instance.                                                                 |
| source      |                   string                   |     true     | The name of an `alloydb-admin` source.                                                           |
| description |                   string                   |     false    | Description of the tool that is passed to the agent.                                             |
