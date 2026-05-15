---
title: cloud-sql-admin-execute-many
type: docs
weight: 1
description: >
  A "cloud-sql-admin-execute-many" tool executes multiple SQL statements against a specific Cloud SQL instance provided at runtime.
---

## About

The `cloud-sql-admin-execute-many` tool executes multiple SQL statements against a specific Cloud SQL instance identified by project, instanceId, and database parameters provided at runtime.

This tool is useful for executing arbitrary SQL queries across multiple database instances without needing to configure a separate tool for each instance.

> **Note:** This tool is intended for developer assistant workflows with human-in-the-loop and shouldn't be used for production agents.

## Compatible Sources

{{< compatible-sources >}}

## Parameters

The following parameters are required at runtime when invoking the tool:

| **Parameter** | **Type** | **Description**               |
| :------------ | :------- | :---------------------------- |
| `project`     | string   | The GCP project ID.           |
| `instanceId`  | string   | The Cloud SQL instance ID.    |
| `database`    | string   | The database name.            |
| `sql`         | string   | The SQL statement to execute. |

## Example

```yaml
kind: tool
name: execute_sql_many_tool
type: cloud-sql-admin-execute-many
source: my-cloud-sql-admin-source
description: Use this tool to execute sql statements on a specific instance.
```

## Reference

| **field**   | **type** | **required** | **description**                                      |
| :---------- | :------- | :----------- | :--------------------------------------------------- |
| type        | string   | true         | Must be "cloud-sql-admin-execute-many".              |
| source      | string   | true         | Name of the `cloud-sql-admin` source.                |
| description | string   | true         | Description of the tool that is passed to the agent. |
