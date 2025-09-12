---
title: "cloud-sql-postgres-create-instance"
type: docs
weight: 10
description: >
  Create a Cloud SQL for PostgreSQL instance.
---

The `cloud-sql-postgres-create-instance` tool creates a Cloud SQL for PostgreSQL instance using the Cloud SQL Admin API.

{{< notice info >}}
This tool uses a `source` of kind `cloud-sql-admin`.
{{< /notice >}}

{{< notice info >}}
The toolbox automatically generates a bearer token on behalf of the user with the `https://www.googleapis.com/auth/sqlservice.admin` scope to authenticate requests.
{{< /notice >}}

## Example

```yaml
tools:
  create-sql-instance:
    kind: cloud-sql-postgres-create-instance
    description: "Create a Cloud SQL for PostgreSQL instance."
    source: http-source
```

## Reference

### Tool Configuration

| **field**   | **type** | **required** | **description**                                                                                                  |
| ----------- | :------: | :----------: | ---------------------------------------------------------------------------------------------------------------- |
| kind        |  string  |     true     | Must be "cloud-sql-postgres-create-instance".                                                                   |
| description |  string  |     true     | A description of the tool.                                                                                       |
| source      |  string  |     true     | The name of the `cloud-sql-admin` source to use.                                                                 |

### Tool Inputs

| **parameter**     | **type** | **required** | **description**                                                                                                                                                    |
| ----------------- | :------: | :----------: | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| project           |  string  |     true     | The project ID.                                                                                                                                                    |
| name              |  string  |     true     | The name of the instance.                                                                                                                                          |
| databaseVersion   |  string  |    false     | The database version for Postgres. If not specified, defaults to the latest available version (e.g., POSTGRES_17).                                                   |
| rootPassword      |  string  |     true     | The root password for the instance.                                                                                                                                |
| editionPreset     |  string  |     true     | The edition of the instance. Can be `Production` or `Development`. This determines the default machine type and availability. Defaults to `Development`.             |
