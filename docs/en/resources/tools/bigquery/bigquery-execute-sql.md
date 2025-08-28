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

`bigquery-execute-sql` executes a GoogleSQL statement. It takes an `sql`
parameter for the query string and an optional `dry_run` parameter to validate
the query without running it. Its behavior changes based on the source
configuration:

- **Without `datasets` restriction:** The tool can execute any valid GoogleSQL
  query.
- **With `datasets` restriction:** The tool performs a dry run to analyze the query.
  It will reject the query if it attempts to access any table outside the
  allowed `datasets` list. To enforce this restriction, the following operations
  are also disallowed:
  - **Dataset-level operations** (e.g., `CREATE SCHEMA`, `ALTER SCHEMA`).
  - **Unanalyzable operations** where the accessed tables cannot be determined
    statically (e.g., `EXECUTE IMMEDIATE`, `CREATE PROCEDURE`, `CALL`).

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
