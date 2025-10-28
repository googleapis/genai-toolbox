---
title: "tidb-execute-sql"
type: docs
weight: 1
description: > 
  A "tidb-execute-sql" tool executes a SQL statement against a TiDB
  database.
aliases:
- /resources/tools/tidb-execute-sql
---

## About

A `tidb-execute-sql` tool executes a SQL statement against a TiDB
database. It's compatible with the following source:

- [tidb](../../sources/tidb.md)

`tidb-execute-sql` takes one input parameter `sql` and run the sql
statement against the `source`.

> **Note:** This tool is intended for developer assistant workflows with
> human-in-the-loop and shouldn't be used for production agents.

## Example

```yaml
tools:
 execute_sql_tool:
    kind: tidb-execute-sql
    source: my-tidb-instance
    description: Use this tool to execute sql statement.
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be "tidb-execute-sql".                                                                     |
| source      |                   string                   |     true     | Name of the source the SQL should execute on.                                                    |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                               |
