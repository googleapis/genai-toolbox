---
title: "mysql-list-tables-missing-unique-indexes"
type: docs
weight: 1
description: >
  A "mysql-list-tables-missing-unique-indexes" tool lists tables that do not have primary or unique indices in a MySQL instance.
aliases:
- /resources/tools/mysql-list-tables-missing-unique-indexes
---

## About

A `mysql-list-tables-missing-unique-indexes` tool searches tables that do not
have primary or unique indices in a MySQL database. It's compatible with:

- [cloud-sql-mysql](../../sources/cloud-sql-mysql.md)
- [mysql](../../sources/mysql.md)

`mysql-list-tables-missing-unique-indexes` outputs table names, including
`table_schema` and `table_name` in JSON format. It takes 2 optional input
parameters:

- `table_schema` (optional): Only check tables in this specific schema/database.
  Search all visible tables in all visible databases if not specified.
- `limit` (optional):  max number of queries to return, default `50`.

## Example

```yaml
tools:
  list_tables_missing_unique_indexes:
    kind: mysql-list-tables-missing-unique-indexes
    source: my-mysql-instance
    description: Find tables that do not have primary or unique key constraint. A primary key or unique key is the only mechanism that guaranttes a row is unique. Without them, the database-level protection against data integrity issues will be missing.
```

The response is a json array with the following fields:

```json
{
  "table_schema": "the schema/database this table belongs to",
  "table_name": "name of the table",
}
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "mysql-list-active-queries".               |
| source      |  string  |     true     | Name of the source the SQL should execute on.      |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
