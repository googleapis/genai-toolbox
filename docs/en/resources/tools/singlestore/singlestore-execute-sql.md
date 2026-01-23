---
title: "singlestore-execute-sql"
type: docs
weight: 1
description: >
  A "singlestore-execute-sql" tool executes a SQL statement against a SingleStore
  database.
aliases:
- /resources/tools/singlestore-execute-sql
---

## About

A `singlestore-execute-sql` tool executes a SQL statement against a SingleStore
database. It's compatible with the following sources:

- [singlestore](../../sources/singlestore.md)

`singlestore-execute-sql` takes one input parameter `sql` and runs the sql
statement against the `source`.

> **Note:** This tool is intended for developer assistant workflows with
> human-in-the-loop and shouldn't be used for production agents.

## Example

```yaml
tools:
 execute_sql_tool:
    kind: singlestore-execute-sql
    source: my-s2-instance
    description: Use this tool to execute sql statement
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "singlestore-execute-sql".                 |
| source      |  string  |     true     | Name of the source the SQL should execute on.      |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
