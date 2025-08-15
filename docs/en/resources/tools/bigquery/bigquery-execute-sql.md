---
title: "bigquery-execute-sql"
type: docs
weight: 1
description: >
  A "bigquery-execute-sql" tool executes a SQL statement against BigQuery.
aliases:
- /resources/tools/bigquery-execute-sql
---

## About

A `bigquery-execute-sql` tool executes a SQL statement against BigQuery.
It's compatible with the following sources:

- [bigquery](../../sources/bigquery.md)

`bigquery-execute-sql` takes a required `sql` input parameter and runs the SQL
statement against the configured `source`. It also supports an optional `dry_run`
parameter to validate a query without executing it.

## Write Mode Behavior

The behavior of this tool is influenced by the `write_mode` setting on its `bigquery` source:

- **`allowed` (default):** All SQL statements are permitted.
- **`blocked`:** Only `SELECT` statements are allowed. Any other type of statement (e.g., `INSERT`, `UPDATE`, `CREATE`) will be rejected.
- **`protected`:** This mode enables session-based execution. `SELECT` statements and writes to the session's temporary dataset are allowed (e.g., `CREATE TEMP TABLE ...`). This prevents modifications to permanent datasets while allowing stateful, multi-step operations within a secure session.

> **Note:** This tool is intended for developer assistant workflows with human-in-the-loop and shouldn't be used for production agents.

## Example

```yaml
tools:
 execute_sql_tool:
    kind: bigquery-execute-sql
    source: my-bigquery-source
    description: Use this tool to execute sql statement.
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be "bigquery-execute-sql".                                                                  |
| source      |                   string                   |     true     | Name of the source the SQL should execute on.                                                    |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                               |
