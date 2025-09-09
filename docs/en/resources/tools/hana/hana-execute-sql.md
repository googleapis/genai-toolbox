---
title: "hana-execute-sql"
type: docs
weight: 1
description: >
  A "hana-execute-sql" tool executes a SQL statement against a SAP HANA
  database.
aliases:
- /resources/tools/hana-execute-sql
---

## About

A `hana-execute-sql` tool executes a SQL statement against a SAP HANA
database. It's compatible with the following source:

- [hana](../../sources/hana.md)

`hana-execute-sql` takes one input parameter `sql` and runs the SQL
statement against the `source`.

> **Note:** This tool is intended for developer assistant workflows with
> human-in-the-loop and shouldn't be used for production agents.

## Example

```yaml
tools:
 execute_sql_tool:
    kind: hana-execute-sql
    source: my-hana-instance
    description: Use this tool to execute SQL statements against SAP HANA. 
      This tool can run any valid HANA SQL including SELECT, INSERT, UPDATE, 
      DELETE, and DDL statements. Use with caution as it has full database access.
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be "hana-execute-sql".                                                                     |
| source      |                   string                   |     true     | Name of the source the SQL should execute on.                                                    |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                               |
