---
title: "mysql-execute-sql"
type: docs
weight: 1
description: >
  A "mysql-execute-sql" tool executes a SQL statement against a MySQL
  database.
aliases:
- /resources/tools/mysql-execute-sql
---

## About

A `mysql-execute-sql` tool executes a SQL statement against a MySQL
database. It's compatible with any of the following sources:

- [cloud-sql-mysql](../../sources/cloud-sql-mysql.md)
- [mysql](../../sources/mysql.md)

`mysql-execute-sql` takes one input parameter `sql` and run the sql
statement against the `source`.

> **Note:** This tool is intended for developer assistant workflows with
> human-in-the-loop and shouldn't be used for production agents.

## Example

```yaml
tools:
 execute_sql_tool:
    kind: mysql-execute-sql
    source: my-mysql-instance
    description: Use this tool to execute sql statement.
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be "mysql-execute-sql".                                                                     |
| source      |                   string                   |     true     | Name of the source the SQL should execute on.                                                    |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                               |
