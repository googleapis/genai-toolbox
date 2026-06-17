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

** Security **

The tool runs the supplied statement as `EXPLAIN FORMAT=JSON <sql_statement>`.
A plain `EXPLAIN` (without `ANALYZE`) only computes the query plan; it never
executes the wrapped statement, so `SELECT`, `INSERT`, `UPDATE`, and `DELETE`
inputs all return a plan without side effects.

Two execution vectors are blocked structurally rather than by parsing the
input:

- **`EXPLAIN ANALYZE` (which does execute the statement) is unreachable.** The
  tool fixes the `FORMAT=JSON` prefix, and MySQL's grammar requires `ANALYZE`
  to appear *before* `FORMAT=`. A statement beginning with `ANALYZE` therefore
  lands after `FORMAT=JSON` and is rejected by the server as a syntax error.
- **Multiple statements are not run.** The MySQL driver does not enable
  multi-statement execution by default, so input such as
  `SELECT 1; DROP TABLE t` is rejected by the server rather than executed.

As defense in depth, configure the `source` with a **least-privilege database
user** scoped to only the objects the agent needs to plan against. This bounds
what any statement — including those that `EXPLAIN` does plan — can reach, and
is the recommended control for this tool. Avoid enabling the driver's
multi-statement option on the source.

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
