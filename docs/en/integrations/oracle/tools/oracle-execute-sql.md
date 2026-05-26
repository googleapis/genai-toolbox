---
title: "oracle-execute-sql"
type: docs
weight: 1
description: > 
  An "oracle-execute-sql" tool executes a SQL statement against an Oracle database.
---

## About

An `oracle-execute-sql` tool executes a SQL statement against an Oracle
database.

`oracle-execute-sql` takes one input parameter `sql` and runs the sql
statement against the `source`.

> **Note:** This tool is intended for developer assistant workflows with
> human-in-the-loop and shouldn't be used for production agents.

## Compatible Sources

{{< compatible-sources >}}

## Example

```yaml
kind: tool
name: execute_sql_tool
type: oracle-execute-sql
source: my-oracle-instance
description: Use this tool to execute sql statement.
```
