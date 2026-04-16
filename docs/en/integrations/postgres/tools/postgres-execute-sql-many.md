---
title: "postgres-execute-sql-many"
type: docs
weight: 1
description: >
  A "postgres-execute-sql-many" tool executes a SQL statement against a specific Cloud SQL Postgres instance provided at runtime.
---

## About

A `postgres-execute-sql-many` tool executes a SQL statement against a specific Cloud SQL Postgres instance identified by project, region, instance, and database parameters provided at runtime.

This tool is useful for executing arbitrary SQL queries across multiple database instances without needing to configure a separate tool for each instance.

> **Note:** This tool is intended for developer assistant workflows with human-in-the-loop and shouldn't be used for production agents.

## Compatible Sources

{{< compatible-sources others="integrations/cloud-sql-admin" >}}

## Parameters

The following parameters are required at runtime when invoking the tool:

| **Parameter** | **Type** | **Description**               |
| :------------ | :------- | :---------------------------- |
| `project`     | string   | The GCP project ID.           |
| `region`      | string   | The GCP region.               |
| `instance`    | string   | The Cloud SQL instance ID.    |
| `database`    | string   | The database name.            |
| `sql`         | string   | The SQL statement to execute. |

## Example

```yaml
kind: tool
name: execute_sql_many_tool
type: postgres-execute-sql-many
source: my-cloud-sql-admin-source
description: Use this tool to execute sql statement on a specific instance.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
| :---------- | :------- | :----------- | :------------------------------------------------- |
| type        | string   | true         | Must be "postgres-execute-sql-many".               |
| source      | string   | true         | Name of the `cloud-sql-admin` source.              |
| description | string   | true         | Description of the tool that is passed to the LLM. |
