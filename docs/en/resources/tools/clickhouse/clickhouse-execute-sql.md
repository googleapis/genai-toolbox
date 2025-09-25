---
title: "clickhouse-execute-sql"
type: docs
weight: 1
description: >
  A "clickhouse-execute-sql" tool executes a SQL statement against a ClickHouse
  database.
aliases:
- /resources/tools/clickhouse-execute-sql
---

## About

A `clickhouse-execute-sql` tool executes a SQL statement against a ClickHouse
database. It's compatible with the [clickhouse](../../sources/clickhouse.md)
source.

`clickhouse-execute-sql` takes one input parameter `sql` and runs the SQL
statement against the specified `source`. This tool includes query logging
capabilities for monitoring and debugging purposes.

> **Note:** This tool is intended for developer assistant workflows with
> human-in-the-loop and shouldn't be used for production agents.

## Example

```yaml
tools:
  execute_sql_tool:
    kind: clickhouse-execute-sql
    source: my-clickhouse-instance
    description: Use this tool to execute SQL statements against ClickHouse.
```

## Parameters

| **parameter** | **type** | **required** | **description**                                   |
|---------------|:--------:|:------------:|---------------------------------------------------|
| sql           |  string  |     true     | The SQL statement to execute against the database |

## Reference

| **field**   | **type** | **required** | **description**                                       |
|-------------|:--------:|:------------:|-------------------------------------------------------|
| kind        |  string  |     true     | Must be "clickhouse-execute-sql".                     |
| source      |  string  |     true     | Name of the ClickHouse source to execute SQL against. |
| description |  string  |     true     | Description of the tool that is passed to the LLM.    |