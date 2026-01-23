---
title: cloud-sql-postgres-create-instance
type: docs
weight: 10
description: Create a Cloud SQL for PostgreSQL instance.
---

The `cloud-sql-postgres-create-instance` tool creates a Cloud SQL for PostgreSQL
instance using the Cloud SQL Admin API.

{{< notice info >}}
This tool uses a `source` of kind `cloud-sql-admin`.
{{< /notice >}}

## Example

```yaml
tools:
  create-sql-instance:
    kind: cloud-sql-postgres-create-instance
    source: cloud-sql-admin-source
    description: "Creates a Postgres instance using `Production` and `Development` presets. For the `Development` template, it chooses a 2 vCPU, 16 GiB RAM, 100 GiB SSD configuration with Non-HA/zonal availability. For the `Production` template, it chooses an 8 vCPU, 64 GiB RAM, 250 GiB SSD configuration with HA/regional availability. The Enterprise Plus edition is used in both cases. The default database version is `POSTGRES_17`. The agent should ask the user if they want to use a different version."
```

## Reference

### Tool Configuration

| **field**   | **type** | **required** | **description**                                  |
| ----------- | :------: | :----------: | ------------------------------------------------ |
| kind        |  string  |     true     | Must be "cloud-sql-postgres-create-instance".    |
| source      |  string  |     true     | The name of the `cloud-sql-admin` source to use. |
| description |  string  |     false    | A description of the tool.                       |

### Tool Inputs

| **parameter**   | **type** | **required** | **description**                                                                                                                                          |
|-----------------|:--------:|:------------:|----------------------------------------------------------------------------------------------------------------------------------------------------------|
| project         |  string  |     true     | The project ID.                                                                                                                                          |
| name            |  string  |     true     | The name of the instance.                                                                                                                                |
| databaseVersion |  string  |    false     | The database version for Postgres. If not specified, defaults to the latest available version (e.g., POSTGRES_17).                                       |
| rootPassword    |  string  |     true     | The root password for the instance.                                                                                                                      |
| editionPreset   |  string  |    false     | The edition of the instance. Can be `Production` or `Development`. This determines the default machine type and availability. Defaults to `Development`. |
