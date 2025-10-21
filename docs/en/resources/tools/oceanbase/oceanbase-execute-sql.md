---
title: "oceanbase-execute-sql"
type: docs
weight: 1
description: > 
  An "oceanbase-execute-sql" tool executes a SQL statement against an OceanBase database.
aliases:
- /resources/tools/oceanbase-execute-sql
---

## About

An `oceanbase-execute-sql` tool executes a SQL statement against an OceanBase
database. It's compatible with the following source:

- [oceanbase](../../sources/oceanbase.md)

`oceanbase-execute-sql` takes one input parameter `sql` and runs the sql
statement against the `source`.

> **Note:** This tool is intended for developer assistant workflows with
> human-in-the-loop and shouldn't be used for production agents.

## Example

```yaml
tools:
  execute_sql_tool:
    kind: oceanbase-execute-sql
    source: my-oceanbase-instance
    description: Use this tool to execute sql statement.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "oceanbase-execute-sql".                   |
| source      |  string  |     true     | Name of the source the SQL should execute on.      |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |