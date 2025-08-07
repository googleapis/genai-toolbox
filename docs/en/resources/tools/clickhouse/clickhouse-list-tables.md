---
title: "clickhouse-list-tables"
type: docs
weight: 4
description: >
  A "clickhouse-list-tables" tool lists detailed schema information for tables 
  in ClickHouse databases.
aliases:
- /resources/tools/clickhouse-list-tables
---

## About

A `clickhouse-list-tables` tool lists detailed schema information for tables 
in a ClickHouse database. It's compatible with the 
[clickhouse](../../sources/clickhouse.md) source.

This tool provides comprehensive table information including:
- Table names and database schemas
- Engine types and configurations
- Key definitions (primary key, sorting key, partition key)
- Storage statistics (row counts, data sizes)
- Table comments and metadata

The tool can list all tables in the current database or filter by specific 
table names.

## Example

```yaml
tools:
  list_tables_tool:
    kind: clickhouse-list-tables
    source: my-clickhouse-instance
    description: List detailed schema information for ClickHouse tables including engine, keys, and storage statistics.
```

## Parameters

| **parameter** | **type** | **required** | **description**                                                                                    |
|---------------|:--------:|:------------:|----------------------------------------------------------------------------------------------------|
| table_names   |  string  |    false     | Optional comma-separated list of table names to describe. If empty, lists all tables in current database |

## Usage Examples

### List all tables in the current database
```json
{
  "table_names": ""
}
```

### List specific tables
```json
{
  "table_names": "user_events, page_views, transactions"
}
```

### List a single table
```json
{
  "table_names": "analytics_summary"
}
```

## Output Format

The tool returns detailed JSON information for each table including:
- `schema_name`: Database name
- `object_name`: Table name  
- `object_type`: Always "TABLE"
- `engine`: ClickHouse table engine
- `primary_key`: Primary key definition
- `sorting_key`: Sorting key definition
- `partition_key`: Partition key definition
- `total_rows`: Number of rows in the table
- `total_bytes`: Storage size in bytes
- `comment`: Table comment

## Reference

| **field**   | **type** | **required** | **description**                                         |
|-------------|:--------:|:------------:|---------------------------------------------------------|
| kind        |  string  |     true     | Must be "clickhouse-list-tables".                      |
| source      |  string  |     true     | Name of the ClickHouse source to query.                |
| description |  string  |     true     | Description of the tool that is passed to the LLM.     |