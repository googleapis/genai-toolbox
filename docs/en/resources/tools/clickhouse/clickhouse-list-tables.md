---
title: "clickhouse-list-tables"
type: docs
weight: 4
description: >
  A "clickhouse-list-tables" tool lists all tables in a specific ClickHouse database.
aliases:
- /resources/tools/clickhouse-list-tables
---

## About

A `clickhouse-list-tables` tool lists all available tables in a specified 
ClickHouse database. It's compatible with the [clickhouse](../../sources/clickhouse.md) source.

This tool executes the `SHOW TABLES FROM <database>` command and returns a list 
of all tables in the specified database that are accessible to the configured 
user, making it useful for schema exploration and table discovery tasks.

## Example

```yaml
tools:
  list_clickhouse_tables:
    kind: clickhouse-list-tables
    source: my-clickhouse-instance
    description: List all tables in a specific ClickHouse database
```

## Parameters

| **parameter** | **type** | **required** | **description**                            |
|---------------|:--------:|:------------:|---------------------------------------------|
| database      | string   | true         | The database to list tables from.          |

## Return Value

The tool returns an array of objects, where each object contains:
- `name`: The name of the table
- `database`: The database the table belongs to

Example response:
```json
[
  {"name": "users", "database": "analytics"},
  {"name": "events", "database": "analytics"},
  {"name": "products", "database": "analytics"},
  {"name": "orders", "database": "analytics"}
]
```

## Reference

| **field**          | **type**           | **required** | **description**                                           |
|--------------------|:------------------:|:------------:|-----------------------------------------------------------|
| kind               | string             | true         | Must be "clickhouse-list-tables".                        |
| source             | string             | true         | Name of the ClickHouse source to list tables from.       |
| description        | string             | true         | Description of the tool that is passed to the LLM.       |
| authRequired       | array of string    | false        | Authentication services required to use this tool.       |
| parameters         | array of Parameter | false        | Parameters for the tool (see Parameters section above).  |