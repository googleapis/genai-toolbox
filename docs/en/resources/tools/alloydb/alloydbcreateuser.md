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

The `alloydb-create-user` tool creates a new database user (`ALLOYDB_BUILT_IN` or `ALLOYDB_IAM_USER`) within a specified cluster.

**Permissions & APIs Required:**
Before using, ensure the following on your GCP project:
1.  The [AlloyDB API](https://console.cloud.google.com/apis/library/alloydb.googleapis.com) is enabled.
2.  The user or service account executing the tool has the following IAM roles:
    -   `roles/alloydb.admin`: To create and manage AlloyDB users.

The tool takes the following input parameters:

| Parameter | Type | Description | Required |
| :--- | :--- | :--- | :--- |
| `projectId` | string | The GCP project ID where the cluster exists. | Yes |
| `locationId` | string | The GCP location where the cluster exists (e.g., `us-central1`). | Yes |
| `clusterId` | string | The ID of the existing cluster where the user will be created. | Yes |
| `userId` | string | The name for the new user. Must be unique within the cluster. | Yes |
| `userType`| string | The type of user. Valid values: `ALLOYDB_BUILT_IN`, `ALLOYDB_IAM_USER`. Default: `ALLOYDB_BUILT_IN`. | No |
| `password` | string | A secure password for the user. Required only if `userType` is `ALLOYDB_BUILT_IN`. | No |
| `databaseRoles` | array(string) | Optional. A list of database roles to grant to the new user (e.g., `pg_read_all_data`). | No |

> **Note**
> This tool does not have a `source` and authenticates using the environment's
[Application Default Credentials](https://cloud.google.com/docs/authentication/application-default-credentials).

## Example

```yaml
tools:
  alloydb_create_user:
    kind: alloydb-create-user
    description: Use this tool to create a new database user for an AlloyDB cluster.
```
## Reference
| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be alloydb-create-user.                                                                  |                                               |
| description |                   string                   |     true     | Description of the tool that is passed to the agent.                                             |