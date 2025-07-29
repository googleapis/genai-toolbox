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

- [bigquery](../sources/bigquery.md)

`bigquery-execute-sql` takes one input parameter `sql` and runs the sql
statement against the `source`.

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
