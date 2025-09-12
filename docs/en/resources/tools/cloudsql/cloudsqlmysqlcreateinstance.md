---
title: Cloud SQL Create MySQL Instance
type: docs
weight: 2
description: Create a Cloud SQL for MySQL instance.
---

The `cloud-sql-mysql-create-instance` tool creates a new Cloud SQL for MySQL instance in a specified Google Cloud project.

{{< notice info >}}
This tool uses the `cloud-sql-admin` source, which automatically handles authentication on behalf of the user.
{{< /notice >}}

## Configuration

Here is an example of how to configure the `cloud-sql-mysql-create-instance` tool in your `tools.yaml` file:

```yaml
sources:
  my-cloud-sql-admin-source:
    kind: cloud-sql-admin

tools:
  create_my_mysql_instance:
    kind: cloud-sql-mysql-create-instance
    source: my-cloud-sql-admin-source
    description: Use this tool to create a new Cloud SQL for MySQL instance.
```

## Parameters

The `cloud-sql-mysql-create-instance` tool has the following parameters:

| **field**       | **type** | **required** | **description**                                                                                                 |
| --------------- | :------: | :----------: | --------------------------------------------------------------------------------------------------------------- |
| project         |  string  |     true     | The Google Cloud project ID.                                                                                    |
| name            |  string  |     true     | The name of the instance to create.                                                                             |
| databaseVersion |  string  |     false    | The database version for MySQL. If not specified, defaults to the latest available version (e.g., `MYSQL_8_4`). |
| rootPassword    |  string  |     true     | The root password for the instance.                                                                             |
| editionPreset   |  string  |     true     | The edition of the instance. Can be `Production` or `Development`. Defaults to `Development`.                   |

## Reference

| **field**   | **type** | **required** | **description**                                                |
| ----------- | :------: | :----------: | -------------------------------------------------------------- |
| kind        |  string  |     true     | Must be `cloud-sql-mysql-create-instance`.                     |
| description |  string  |     false    | A description of the tool that is passed to the agent.         |
| source      |  string  |     true     | The name of the `cloud-sql-admin` source to use for this tool. |
