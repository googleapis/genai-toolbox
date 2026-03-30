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
