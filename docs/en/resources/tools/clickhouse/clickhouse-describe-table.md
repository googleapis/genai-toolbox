---
title: "clickhouse-describe-table"
type: docs
weight: 3
description: >
  A "clickhouse-describe-table" tool gets metadata about a specific 
  ClickHouse table including schema, and engine details.
aliases:
- /resources/tools/clickhouse-describe-table
---

## About

A `clickhouse-describe-table` tool retrieves metadata about a 
specific table in a ClickHouse database. It's compatible with the 
[clickhouse](../../sources/clickhouse.md) source.

This tool provides detailed information including:
- Table schema (columns, data types, constraints)
- Engine configuration and settings
- Storage statistics (row count, data size)
- Key definitions (primary key, sorting key, partition key)
- Column metadata and comments

## Example

```yaml
tools:
  describe_table_tool:
    kind: clickhouse-describe-table
    source: my-clickhouse-instance
    description: Get metadata about ClickHouse tables including schema and storage details.
```

## Parameters

| **parameter** | **type** | **required** | **description**                                                          |
|---------------|:--------:|:------------:|--------------------------------------------------------------------------|
| table_name    |  string  |     true     | The name of the table to describe                                       |
| database_name |  string  |    false     | The database name (optional, defaults to current database if not specified) |

## Usage Examples

### Describe a table in the current database

```json
{
  "table_name": "user_events"
}
```

### Describe a table in a specific database

```json
{
  "table_name": "analytics_events",
  "database_name": "production_analytics"
}
```

## Reference

| **field**   | **type** | **required** | **description**                                         |
|-------------|:--------:|:------------:|---------------------------------------------------------|
| kind        |  string  |     true     | Must be "clickhouse-describe-table".                   |
| source      |  string  |     true     | Name of the ClickHouse source to query.                |
| description |  string  |     true     | Description of the tool that is passed to the LLM.     |