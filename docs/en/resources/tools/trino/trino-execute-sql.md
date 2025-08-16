---
title: "trino-execute-sql"
type: docs
weight: 1
description: >
  A "trino-execute-sql" tool executes a SQL statement against a Trino
  database.
aliases:
- /resources/tools/trino-execute-sql
---

## About

A `trino-execute-sql` tool executes a SQL statement against a Trino
database. It's compatible with any of the following sources:

- [trino](../../sources/trino.md)

`trino-execute-sql` takes one input parameter `sql` and run the sql
statement against the `source`.

> **Note:** This tool is intended for developer assistant workflows with
> human-in-the-loop and shouldn't be used for production agents.

## Example

```yaml
tools:
 execute_sql_tool:
    kind: trino-execute-sql
    source: my-trino-instance
    description: Use this tool to execute sql statement.
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be "trino-execute-sql".                                                                     |
| source      |                   string                   |     true     | Name of the source the SQL should execute on.                                                    |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                               |
