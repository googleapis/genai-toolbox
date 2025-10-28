---
title: "firebird-execute-sql"
type: docs
weight: 1
description: > 
  A "firebird-execute-sql" tool executes a SQL statement against a Firebird
  database.
aliases:
- /resources/tools/firebird-execute-sql
---

## About

A `firebird-execute-sql` tool executes a SQL statement against a Firebird
database. It's compatible with the following source:

- [firebird](../../sources/firebird.md)

`firebird-execute-sql` takes one input parameter `sql` and runs the sql
statement against the `source`.

> **Note:** This tool is intended for developer assistant workflows with
> human-in-the-loop and shouldn't be used for production agents.

## Example

```yaml
tools:
  execute_sql_tool:
    kind: firebird-execute-sql
    source: my_firebird_db
    description: Use this tool to execute a SQL statement against the Firebird database.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "firebird-execute-sql".                    |
| source      |  string  |     true     | Name of the source the SQL should execute on.      |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
