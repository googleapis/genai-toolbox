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

`bigquery-execute-sql` takes one input parameter `sql` and runs the SQL
statement against the `source`. If the associated `bigquery` source has a
`datasets` list configured, this tool will parse the SQL statement to identify
accessed datasets and return an error if any of them are not in the allowed
list. Note that the toolbox cannot determine which datasets are accessed within
an `EXECUTE IMMEDIATE` statement. Therefore, when `datasets` restrictions are
active, any SQL from this tool containing `EXECUTE IMMEDIATE` will be rejected.
It also supports an optional `dry_run` parameter to validate a query without 
executing it.

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
