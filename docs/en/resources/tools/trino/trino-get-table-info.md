---
title: "trino-get-table-info"
type: docs
weight: 6
description: >
  A "trino-get-table-info" tool retrieves comprehensive metadata about a specific
  table including columns, statistics, and sample data.
aliases:
- /resources/tools/trino-get-table-info
---

## About

A `trino-get-table-info` tool provides comprehensive metadata retrieval for
individual tables in Trino. It retrieves detailed information about table
structure, columns, data types, statistics, and can optionally include sample
data and CREATE TABLE statements.

This tool is compatible with:
- [trino](../../sources/trino.md)

## Features

- **Comprehensive Metadata**: Retrieves table type, columns, data types, and properties
- **Column Details**: Includes nullability, defaults, and comments for each column
- **Statistics Integration**: Optionally runs SHOW STATS to get column-level statistics
- **Sample Data**: Can retrieve sample rows to understand table content
- **CREATE TABLE Statement**: Retrieves the DDL for table recreation
- **Flexible References**: Supports fully qualified or partial table names

## Example

```yaml
tools:
  get_table_info:
    kind: trino-get-table-info
    source: my-trino-instance
    description: Gets comprehensive table metadata including columns and statistics
```

## Parameters

| **parameter**   | **type** | **required** | **description**                                                                              |
|-----------------|:--------:|:------------:|----------------------------------------------------------------------------------------------|
| table_name      | string   | true         | Table name (can be fully qualified: catalog.schema.table)                                   |
| catalog         | string   | false        | Catalog name. If not provided and table_name is not fully qualified, uses current catalog   |
| schema          | string   | false        | Schema name. If not provided and table_name is not fully qualified, uses current schema     |
| include_stats   | boolean  | false        | If true, includes table statistics using SHOW STATS. Default: false                         |
| include_sample  | boolean  | false        | If true, includes sample data from the table. Default: false                                |
| sample_size     | integer  | false        | Number of sample rows to include if include_sample is true. Default: 5                      |

## Response Structure

The tool returns a JSON object with comprehensive table metadata:

```json
{
  "catalogName": "hive",
  "schemaName": "sales",
  "tableName": "orders",
  "tableType": "BASE TABLE",
  "columns": [
    {
      "columnName": "order_id",
      "dataType": "bigint",
      "ordinalPosition": 1,
      "isNullable": "NO",
      "columnDefault": null,
      "columnComment": "Primary key",
      "dataSize": 8000000,
      "distinctValuesCount": 1000000,
      "nullsFraction": 0.0,
      "minValue": "1",
      "maxValue": "1000000"
    },
    {
      "columnName": "customer_name",
      "dataType": "varchar(100)",
      "ordinalPosition": 2,
      "isNullable": "YES",
      "columnDefault": null,
      "columnComment": null,
      "dataSize": 15500000,
      "distinctValuesCount": 50000,
      "nullsFraction": 0.01,
      "minValue": "Aaron",
      "maxValue": "Zoe"
    }
  ],
  "columnCount": 2,
  "rowCount": 1000000,
  "dataSizeBytes": 104857600,
  "createStatement": "CREATE TABLE hive.sales.orders (\n  order_id bigint NOT NULL,\n  customer_name varchar(100)\n)",
  "sampleData": [
    {
      "order_id": 1,
      "customer_name": "John Doe"
    },
    {
      "order_id": 2,
      "customer_name": "Jane Smith"
    }
  ],
  "statistics": [
    {
      "columnName": "order_id",
      "dataSize": 8000000.0,
      "distinctValuesCount": 1000000.0,
      "nullsFraction": 0.0,
      "rowCount": 1000000.0,
      "lowValue": "1",
      "highValue": "1000000"
    }
  ]
}
```

## Usage Examples

### Basic Table Information
```yaml
# Get basic table structure
parameters:
  table_name: "orders"
```

### Fully Qualified Table with Statistics
```yaml
# Get comprehensive information including statistics
parameters:
  table_name: "hive.sales.orders"
  include_stats: true
```

### Table with Sample Data
```yaml
# Get table info with sample data
parameters:
  table_name: "customers"
  catalog: "hive"
  schema: "sales"
  include_sample: true
  sample_size: 10
```

### Complete Table Analysis
```yaml
# Get all available information
parameters:
  table_name: "products"
  include_stats: true
  include_sample: true
```

## Notes

- Statistics availability depends on whether ANALYZE has been run on the table
- Sample data retrieval may be slow for large tables
- CREATE TABLE statement format may vary by connector
- Column statistics are only included when `include_stats` is true
- The tool automatically handles current catalog/schema when not specified

## Reference

| **field**      | **type** | **required** | **description**                                                |
|----------------|:--------:|:------------:|----------------------------------------------------------------|
| kind           | string   | true         | Must be "trino-get-table-info".                               |
| source         | string   | true         | Name of the Trino source to connect to.                       |
| description    | string   | true         | Description of the tool that is passed to the LLM.            |
