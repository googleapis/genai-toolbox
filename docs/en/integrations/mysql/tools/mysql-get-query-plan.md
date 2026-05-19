---
title: "mysql-get-query-plan"
type: docs
weight: 1
description: >
  A "mysql-get-query-plan" tool gets the execution plan for a SQL statement against a MySQL
  database.
---

## About

A `mysql-get-query-plan` tool gets the execution plan for a SQL statement against a MySQL
database.

`mysql-get-query-plan` takes one input parameter `sql_statement` and gets the execution plan for the SQL
statement against the `source`.

## Input Validation

To prevent query execution and statement injection through `EXPLAIN`, the
`sql_statement` is validated before it is run:

- It must be a **single** statement. Multiple statements separated by a
  top-level `;` are rejected. A single trailing `;`, and `;` characters inside
  string literals or comments, are allowed.
- It must begin with one of the statement types that `EXPLAIN FORMAT=JSON`
  supports: `SELECT`, `INSERT`, `UPDATE`, `DELETE`, `REPLACE`, `TABLE`, `WITH`,
  or `VALUES`. Leading parentheses (e.g. parenthesized `UNION`) and leading
  comments are permitted.
- The `ANALYZE` keyword is rejected because `EXPLAIN ANALYZE` executes the
  statement instead of only planning it. Other forms such as `FOR CONNECTION`,
  DDL, and `CALL` are rejected as they are not supported with `FORMAT=JSON`.

## Compatible Sources

{{< compatible-sources others="integrations/cloud-sql-mysql">}}

## Example

```yaml
kind: tool
name: get_query_plan_tool
type: mysql-get-query-plan
source: my-mysql-instance
description: Use this tool to get the execution plan for a sql statement.
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| type        |                   string                   |     true     | Must be "mysql-get-query-plan".                                                                     |
| source      |                   string                   |     true     | Name of the source the SQL should execute on.                                                    |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                               |
