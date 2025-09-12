---
title: cloud-sql-mssql-create-instance
type: docs
weight: 10
description: "Create a Cloud SQL for MSSQL instance.\n"
---

The `cloud-sql-mssql-create-instance` tool creates a Cloud SQL for MSSQL instance using the Cloud SQL Admin API.

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
    kind: cloud-sql-mssql-create-instance
       source: cloud-sql-admin-source
    description: "Create a Cloud SQL for MSSQL instance."
```

## Reference

### Tool Configuration

| **field**   | **type** | **required** | **description**                                  |
| ----------- | :------: | :----------: | ------------------------------------------------ |
| kind        |  string  |     true     | Must be "cloud-sql-mssql-create-instance".       |
| description |  string  |     false    | A description of the tool.                       |
| source      |  string  |     true     | The name of the `cloud-sql-admin` source to use. |

### Tool Inputs

| **parameter**   | **type** | **required** | **description**                                                                                                                                          |
| --------------- | :------: | :----------: | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| project         |  string  |     true     | The project ID.                                                                                                                                          |
| name            |  string  |     true     | The name of the instance.                                                                                                                                |
| region          |  string  |     true     | The region of the instance.                                                                                                                              |
| databaseVersion |  string  |     false    | The database version for MSSQL. If not specified, defaults to the latest available version (e.g., SQLSERVER_2022_STANDARD).                              |
| rootPassword    |  string  |     true     | The root password for the instance.                                                                                                                      |
| editionPreset   |  string  |     true     | The edition of the instance. Can be `Production` or `Development`. This determines the default machine type and availability. Defaults to `Development`. |
