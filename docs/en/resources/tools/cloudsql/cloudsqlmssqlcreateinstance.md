---
title: cloud-sql-mssql-create-instance
type: docs
weight: 10
description: "Create a Cloud SQL for SQL Server instance."
---

The `cloud-sql-mssql-create-instance` tool creates a Cloud SQL for SQL Server
instance using the Cloud SQL Admin API.

{{< notice info dd>}}
This tool uses a `source` of kind `cloud-sql-admin`.
{{< /notice >}}

## Example

```yaml
tools:
  create-sql-instance:
    kind: cloud-sql-mssql-create-instance
    source: cloud-sql-admin-source
    description: "Creates a SQL Server instance using `Production` and `Development` presets. For the `Development` template, it chooses a 2 vCPU, 8 GiB RAM (`db-custom-2-8192`) configuration with Non-HA/zonal availability. For the `Production` template, it chooses a 4 vCPU, 26 GiB RAM (`db-custom-4-26624`) configuration with HA/regional availability. The Enterprise edition is used in both cases. The default database version is `SQLSERVER_2022_STANDARD`. The agent should ask the user if they want to use a different version."
```

## Reference

### Tool Configuration

| **field**   | **type** | **required** | **description**                                  |
| ----------- | :------: | :----------: | ------------------------------------------------ |
| kind        |  string  |     true     | Must be "cloud-sql-mssql-create-instance".       |
| source      |  string  |     true     | The name of the `cloud-sql-admin` source to use. |
| description |  string  |    false     | A description of the tool.                       |

### Tool Inputs

| **parameter**   | **type** | **required** | **description**                                                                                                                                          |
|-----------------|:--------:|:------------:|----------------------------------------------------------------------------------------------------------------------------------------------------------|
| project         |  string  |     true     | The project ID.                                                                                                                                          |
| name            |  string  |     true     | The name of the instance.                                                                                                                                |
| databaseVersion |  string  |    false     | The database version for SQL Server. If not specified, defaults to the latest available version (e.g., SQLSERVER_2022_STANDARD).                         |
| rootPassword    |  string  |     true     | The root password for the instance.                                                                                                                      |
| editionPreset   |  string  |    false     | The edition of the instance. Can be `Production` or `Development`. This determines the default machine type and availability. Defaults to `Development`. |
