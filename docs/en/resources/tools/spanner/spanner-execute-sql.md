---
title: "spanner-execute-sql"
type: docs
weight: 1
description: >
  A "spanner-execute-sql" tool executes a SQL statement against a Spanner
  database.
aliases:
- /resources/tools/spanner-execute-sql
---

## About

A `spanner-execute-sql` tool executes a SQL statement against a Spanner
database. It's compatible with any of the following sources:

- [spanner](../../sources/spanner.md)

`spanner-execute-sql` takes one input parameter `sql` and run the sql
statement against the `source`.

> **Note:** This tool is intended for developer assistant workflows with
> human-in-the-loop and shouldn't be used for production agents.

## Example

```yaml
tools:
 execute_sql_tool:
    kind: spanner-execute-sql
    source: my-spanner-instance
    description: Use this tool to execute sql statement.
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| type        |                   string                   |     true     | Must be "spanner-execute-sql".                                                                   |
| source      |                   string                   |     true     | Name of the source the SQL should execute on.                                                    |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                               |
| readOnly    |                   bool                     |     false    | When set to `true`, the `statement` is run as a read-only transaction. Default: `false`.         |
