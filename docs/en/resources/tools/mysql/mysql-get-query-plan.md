---
title: "mysql-get-query-plan"
type: docs
weight: 1
description: >
  A "mysql-get-query-plan" tool gets the execution plan for a SQL statement against a MySQL
  database.
aliases:
- /resources/tools/mysql-get-query-plan
---

## About

A `mysql-get-query-plan` tool gets the execution plan for a SQL statement against a MySQL
database. It's compatible with any of the following sources:

- [cloud-sql-mysql](../../sources/cloud-sql-mysql.md)
- [mysql](../../sources/mysql.md)

`mysql-get-query-plan` takes one input parameter `sql_statement` and gets the execution plan for the sql
statement against the `source`.

## Example

```yaml
tools:
 get_query_plan_tool:
    kind: mysql-get-query-plan
    source: my-mysql-instance
    description: Use this tool to get the execution plan for a sql statement.
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be "mysql-get-query-plan".                                                                     |
| source      |                   string                   |     true     | Name of the source the SQL should execute on.                                                    |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                               |
