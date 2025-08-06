---
title: "clickhouse-list-databases"
type: docs
weight: 5
description: >
  A "clickhouse-list-databases" tool lists all databases in a ClickHouse instance 
  with their metadata.
aliases:
- /resources/tools/clickhouse-list-databases
---

## About

A `clickhouse-list-databases` tool lists all databases in a ClickHouse instance 
with their metadata including engine and comment information. It's compatible 
with the [clickhouse](../../sources/clickhouse.md) source.

This tool provides database-level information including:
- Database names
- Database engines
- Comments and descriptions

## Example

```yaml
tools:
  list_databases_tool:
    kind: clickhouse-list-databases
    source: my-clickhouse-instance
    description: List all databases in the ClickHouse instance with their metadata.
```

## Parameters

This tool takes no parameters and lists all databases in the ClickHouse instance.

## Usage Example

```json
{}
```

## Output Format

The tool returns detailed JSON information for each database including:
- `database_name`: Name of the database
- `database_details`: JSON object containing:
  - `database_name`: Database name
  - `engine`: Database engine type
  - `comment`: Database comment/description

## Reference

| **field**   | **type** | **required** | **description**                                         |
|-------------|:--------:|:------------:|---------------------------------------------------------|
| kind        |  string  |     true     | Must be "clickhouse-list-databases".                   |
| source      |  string  |     true     | Name of the ClickHouse source to query.                |
| description |  string  |     true     | Description of the tool that is passed to the LLM.     |