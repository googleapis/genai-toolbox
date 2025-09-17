---
title: "alloydb-create-user"
type: docs
weight: 2 
description: >
  The "alloydb-create-user" tool creates a new database user within a specified AlloyDB cluster.
aliases:
- /resources/tools/alloydb-create-user
---

## About

The `alloydb-create-user` tool creates a new database user (`ALLOYDB_BUILT_IN` or `ALLOYDB_IAM_USER`) within a specified cluster. It is compatible with [alloydb-admin](../../sources/alloydb-admin.md) source.

**Permissions & APIs Required:**
Before using, ensure the following on your GCP project:
1.  The [AlloyDB API](https://console.cloud.google.com/apis/library/alloydb.googleapis.com) is enabled.
2. The user or service account executing the tool has one of the following IAM roles:
    - `roles/alloydb.admin` (the AlloyDB Admin predefined IAM role)
    - `roles/owner` (the Owner basic IAM role)
    - `roles/editor` (the Editor basic IAM role)

The tool takes the following input parameters:

| Parameter | Type | Description | Required |
| :--- | :--- | :--- | :--- |
| `project`       | string        | The GCP project ID where the cluster exists.                                                         | Yes |
| `cluster`       | string        | The ID of the existing cluster where the user will be created.                                                        | Yes |
| `location`      | string        | The GCP location where the cluster exists (e.g., `us-central1`).                                                 | Yes |
| `user`          | string        | The name for the new user. Must be unique within the cluster.                                                        | Yes |
| `userType`      | string        | The type of user. Valid values: `ALLOYDB_BUILT_IN` and `ALLOYDB_IAM_USER`. `ALLOYDB_IAM_USER` is recommended.          | Yes |
| `password`      | string        | A secure password for the user. Required only if `userType` is `ALLOYDB_BUILT_IN`.                                          | No  |
| `databaseRoles` | array(string) | Optional. A list of database roles to grant to the new user (e.g., `pg_read_all_data`).                                     | No  |

## Example

```yaml
tools:
  create_user:
    kind: alloydb-create-user
    source: alloydb-admin-source
    description: Use this tool to create a new database user for an AlloyDB cluster.
```
## Reference
| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be alloydb-create-user.                                                                     |
| source      |                   string                   |     true     | The name of an `alloydb-admin` source.                                                           |
| description |                   string                   |     false    | Description of the tool that is passed to the agent.                                             |
