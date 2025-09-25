---
title: "kdb-execute-sql"
type: docs
weight: 1
description: >
  A "kdb-execute-sql" tool executes a SQL statement against a KDB
  database.
aliases:
- /resources/tools/kdb-execute-sql
---

## About

A `kdb-execute-sql` tool executes a SQL statement against a KDB
database. It's compatible with the following source:

- [kdb](../../sources/kdb.md)

`kdb-execute-sql` takes one input parameter `sql` and run the sql
statement against the `source`.

> **Note:** This tool is intended for developer assistant workflows with
> human-in-the-loop and shouldn't be used for production agents.

## Example

```yaml
tools:
 execute_sql_tool:
    kind: kdb-execute-sql
    source: my-kdb-instance
    description: Use this tool to execute sql statement.
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be "kdb-execute-sql".                                                                  |
| source      |                   string                   |     true     | Name of the source the SQL should execute on.                                                    |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                               |
