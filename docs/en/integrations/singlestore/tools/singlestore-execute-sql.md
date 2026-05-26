---
title: "singlestore-execute-sql"
type: docs
weight: 1
description: >
  A "singlestore-execute-sql" tool executes a SQL statement against a SingleStore
  database.
---

## About

A `singlestore-execute-sql` tool executes a SQL statement against a SingleStore
database.

`singlestore-execute-sql` takes one input parameter `sql` and runs the sql
statement against the `source`.

> **Note:** This tool is intended for developer assistant workflows with
> human-in-the-loop and shouldn't be used for production agents.

## Compatible Sources

{{< compatible-sources >}}

## Example

```yaml
kind: tool
name: execute_sql_tool
type: singlestore-execute-sql
source: my-s2-instance
description: Use this tool to execute sql statement
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| type        |  string  |     true     | Must be "singlestore-execute-sql".                 |
| source      |  string  |     true     | Name of the source the SQL should execute on.      |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
