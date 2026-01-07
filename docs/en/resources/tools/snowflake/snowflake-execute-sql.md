---
title: "snowflake-execute-sql"
type: docs
weight: 1
description: >
  A "snowflake-execute-sql" tool executes a SQL statement against a Snowflake
  database.
---

## About

A `snowflake-execute-sql` tool executes a SQL statement against a Snowflake
database. It's compatible with any of the following sources:

- [snowflake](../../sources/snowflake.md)

`snowflake-execute-sql` takes one input parameter `sql` and run the sql
statement against the `source`.

> **Note:** This tool is intended for developer assistant workflows with
> human-in-the-loop and shouldn't be used for production agents.

## Example

```yaml
tools:
 execute_sql_tool:
    kind: snowflake-execute-sql
    source: my-snowflake-instance
    description: Use this tool to execute sql statement.
```

## Reference

| **field**    |   **type**    | **required** | **description**                                           |
|--------------|:-------------:|:------------:|-----------------------------------------------------------|
| kind         |    string     |     true     | Must be "snowflake-execute-sql".                          |
| source       |    string     |     true     | Name of the source the SQL should execute on.             |
| description  |    string     |     true     | Description of the tool that is passed to the LLM.        |
| authRequired | array[string] |    false     | List of auth services that are required to use this tool. |
