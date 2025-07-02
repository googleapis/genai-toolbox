---
title: "postgres-execute-sql"
type: docs
weight: 1
description: > 
  A "postgres-execute-sql" tool executes a SQL statement against a Postgres
  database.
aliases:
- /resources/tools/postgres-execute-sql
---

## About

A `postgres-execute-sql` tool executes a SQL statement against a Postgres
database. It's compatible with any of the following sources:

- [alloydb-postgres](../sources/alloydb-pg.md)
- [cloud-sql-postgres](../sources/cloud-sql-pg.md)
- [postgres](../sources/postgres.md)

`postgres-execute-sql` takes one input parameter `sql` and run the sql
statement against the `source`.

> **Note:** This tool is intended for developer assistant workflows with
> human-in-the-loop and shouldn't be used for production agents.

## Example

```yaml
tools:
 execute_sql_tool:
    kind: postgres-execute-sql
    source: my-pg-instance
    description: Use this tool to execute sql statement.
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be "postgres-execute-sql".                                                                  |
| source      |                   string                   |     true     | Name of the source the SQL should execute on.                                                    |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                               |
