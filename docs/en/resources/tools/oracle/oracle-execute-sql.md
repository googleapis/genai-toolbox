---
title: "oracle-execute-sql"
type: docs
weight: 1
description: > 
  An "oracle-execute-sql" tool executes a SQL statement against an Oracle database.
aliases:
- /resources/tools/oracle-execute-sql
---

## About

An `oracle-execute-sql` tool executes a SQL statement against an Oracle
database. It's compatible with the following source:

- [oracle](../../sources/oracle.md)

`oracle-execute-sql` takes one input parameter `sql` and runs the sql
statement against the `source`.

> **Note:** This tool is intended for developer assistant workflows with
> human-in-the-loop and shouldn't be used for production agents.

## Example

```yaml
tools:
  execute_sql_tool:
    kind: oracle-execute-sql
    source: my-oracle-instance
    description: Use this tool to execute sql statement.
