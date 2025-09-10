---
title: "cloud-sql-create-instances"
type: docs
weight: 10
description: >
  Create a Cloud SQL instance.
---

The `cloud-sql-create-instances` tool creates a Cloud SQL instance using the Cloud SQL Admin API.

{{< notice info >}}
This tool uses a `source` of kind `http`, and the `baseUrl` for that source must be `https://sqladmin.googleapis.com/`.
{{< /notice >}}

{{< notice info >}}
The toolbox automatically generates a bearer token on behalf of the user with the `https://www.googleapis.com/auth/sqlservice.admin` scope to authenticate requests.
{{< /notice >}}

## Example

```yaml
tools:
  create-sql-instance:
    kind: cloud-sql-create-instances
    description: "Create a Cloud SQL instance."
    source: http-source
```

## Reference

### Tool Configuration

| **field**   | **type** | **required** | **description**                                                                                                  |
| ----------- | :------: | :----------: | ---------------------------------------------------------------------------------------------------------------- |
| kind        |  string  |     true     | Must be "cloud-sql-create-instances".                                                                            |
| description |  string  |     true     | A description of the tool.                                                                                       |
| source      |  string  |     true     | The name of the `http` source to use. The source's `baseUrl` must be `https://sqladmin.googleapis.com/`.         |

### Tool Inputs

| **parameter**     | **type** | **required** | **description**                                                                                                                                                    |
| ----------------- | :------: | :----------: | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| project           |  string  |     true     | The project ID.                                                                                                                                                    |
| name              |  string  |     true     | The name of the instance.                                                                                                                                          |
| databaseVersion   |  string  |     true     | The database version. If not specified, defaults to the latest available version for the engine (e.g., POSTGRES_17, MYSQL_8_4, SQLSERVER_2022_STANDARD).             |
| rootPassword      |  string  |     true     | The root password for the instance.                                                                                                                                |
| editionPreset     |  string  |     true     | The edition of the instance. Can be `Production` or `Development`. This determines the default machine type and availability. Defaults to `Development`.             |
