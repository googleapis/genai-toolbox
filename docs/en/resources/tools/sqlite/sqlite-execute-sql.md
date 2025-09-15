---
title: "sqlite-execute-sql"
type: docs
weight: 1
description: >
  A "sqlite-execute-sql" tool executes a single SQL statement against a SQLite database.
aliases:
- /resources/tools/sqlite-execute-sql
---

## About

A `sqlite-execute-sql` tool executes a single SQL statement against a SQLite database.
It's compatible with any of the following sources:

- [sqlite](../../sources/sqlite.md)

This tool is designed for direct execution of SQL statements. It takes a single `sql` input parameter and runs the SQL statement against the configured SQLite `source`.

> **Note:** This tool is intended for developer assistant workflows with
> human-in-the-loop and shouldn't be used for production agents.

## Example

```yaml
tools:
 execute_sql_tool:
    kind: sqlite-execute-sql
    source: my-sqlite-db
    description: Use this tool to execute a SQL statement.
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                    |
|-------------|:------------------------------------------:|:------------:|----------------------------------------------------|
| kind        |                   string                   |     true     | Must be "sqlite-execute-sql".                      |
| source      |                   string                   |     true     | Name of the source the SQL should execute on.      |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM. |
