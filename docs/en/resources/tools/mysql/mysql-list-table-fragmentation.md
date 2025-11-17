---
title: "mysql-list-table-fragmentation"
type: docs
weight: 1
description: >
  A "mysql-list-table-fragmentation" tool lists top N fragemented tables in MySQL.
aliases:
- /resources/tools/mysql-list-table-fragmentation
---

## About

A `mysql-list-table-fragmentation` tool checks table fragmentation of MySQL
tables by calculating the size of the data and index files in bytes and
comparing with free space allocated to each table. This tool calculates
`fragmentation_percentage` which represents the proportion of free space
relative to the total data and index size. It's compatible with

- [cloud-sql-mysql](../../sources/cloud-sql-mysql.md)
- [mysql](../../sources/mysql.md)

`mysql-list-table-fragmentation` outputs detailed information as JSON , ordered
by the fragmentation percentage in descending order.
This tool takes 4 optional input parameters:

- `table_schema` (optional): The database where fragmentation check is to be
  executed. Check all tables visible to the current user if not specified.
- `table_name` (optional): Name of the table to be checked. Check all tables
  visible to the current user if not specified.
- `data_free_threshold_bytes` (optional): Only show tables with at least this
  much free space in bytes. Default 1.
- `limit` (optional): Max rows to return, default 10.

## Example

```yaml
tools:
  list_table_fragmentation:
    kind: mysql-list-table-fragmentation
    source: my-mysql-instance
    description: List table fragmentation in MySQL, by calculating the size of the data and index files and free space allocated to each table. The query calculates fragmentation percentage which represents the proportion of free space relative to the total data and index size. Storage can be reclaimed for tables with high fragmentation using OPTIMIZE TABLE.
```

The response is a json array with the following fields:

```json
{
  "table_schema": "The schema/database this table belongs to",
  "table_name": "Name of this table",
  "data_size": "Size of the table data in bytes",
  "index_size": "Size of the table's indexes in bytes",
  "data_free": "Free space (bytes) available in the table's data file",
  "fragmentation_percentage": "How much fragementation this table has",
}
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "mysql-list-table-fragmentation".          |
| source      |  string  |     true     | Name of the source the SQL should execute on.      |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
