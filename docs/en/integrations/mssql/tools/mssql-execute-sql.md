---
title: "mssql-execute-sql"
type: docs
weight: 1
description: >
  A "mssql-execute-sql" tool executes a SQL statement against a SQL Server
  database.
---

## About

A `mssql-execute-sql` tool executes a SQL statement against a SQL Server
database. It's compatible with any of the following sources:

`mssql-execute-sql` takes one input parameter `sql` and run the sql
statement against the `source`.

> **Note:** This tool is intended for developer assistant workflows with
> human-in-the-loop and shouldn't be used for production agents.

## Compatible Sources

{{< compatible-sources others="integrations/cloud-sql-mssql">}}

## Example

```yaml
kind: tool
name: execute_sql_tool
type: mssql-execute-sql
source: my-mssql-instance
description: Use this tool to execute sql statement.
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                    |
|-------------|:------------------------------------------:|:------------:|----------------------------------------------------|
| type        |                   string                   |     true     | Must be "mssql-execute-sql".                       |
| source      |                   string                   |     true     | Name of the source the SQL should execute on.      |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM. |
