---
title: "clickhouse-execute-sql"
type: docs
weight: 1
description: >
  A "clickhouse-execute-sql" tool executes a SQL statement against a ClickHouse
  database.
---

## About

A `clickhouse-execute-sql` tool executes a SQL statement against a ClickHouse
database.

`clickhouse-execute-sql` takes one input parameter `sql` and runs the SQL
statement against the specified `source`. This tool includes query logging
capabilities for monitoring and debugging purposes.

> **Note:** This tool is intended for developer assistant workflows with
> human-in-the-loop and shouldn't be used for production agents.


## Compatible Sources

{{< compatible-sources >}}

## Parameters

| **parameter** | **type** | **required** | **description**                                   |
|---------------|:--------:|:------------:|---------------------------------------------------|
| sql           |  string  |     true     | The SQL statement to execute against the database |


## Example

```yaml
kind: tool
name: execute_sql_tool
type: clickhouse-execute-sql
source: my-clickhouse-instance
description: Use this tool to execute SQL statements against ClickHouse.
```

## Reference

| **field**   | **type** | **required** | **description**                                       |
|-------------|:--------:|:------------:|-------------------------------------------------------|
| type        |  string  |     true     | Must be "clickhouse-execute-sql".                     |
| source      |  string  |     true     | Name of the ClickHouse source to execute SQL against. |
| description |  string  |     true     | Description of the tool that is passed to the LLM.    |
