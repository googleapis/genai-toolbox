---
title: "trino-table-statistics"
type: docs
weight: 5
description: >
  A "trino-table-statistics" tool retrieves detailed statistics and metadata
  for specific tables in Trino including row counts, column statistics, and
  partition information.
aliases:
- /resources/tools/trino-table-statistics
---

## About

A `trino-table-statistics` tool provides comprehensive statistics and metadata
for individual tables in Trino. It can retrieve row counts, column-level
statistics, partition information, storage details, and optionally update
statistics by running ANALYZE before retrieval.

This tool is compatible with:
- [trino](../../sources/trino.md)

## Features

- **Table Metadata**: Get table type, properties, and basic information
- **Row Statistics**: Retrieve row counts and data size metrics
- **Column Statistics**: Get detailed statistics for each column including
  null counts, distinct values, min/max values
- **Partition Information**: Identify partitioned tables and get partition
  details
- **Storage Information**: Get storage format, location, and file information
  (connector-dependent)
- **Auto-analyze**: Optionally run ANALYZE to update statistics before
  retrieval
- **Flexible Table References**: Support for fully qualified names or current
  catalog/schema

## Example

```yaml
tools:
  table_stats:
    kind: trino-table-statistics
    source: my-trino-instance
    description: Get detailed statistics and metadata for specific tables
```

## Parameters

| **parameter**       | **type** | **required** | **description**                                                                                |
|---------------------|:--------:|:------------:|------------------------------------------------------------------------------------------------|
| table_name          | string   | true         | Table name (can be fully qualified: catalog.schema.table, or just table name)                 |
| catalog             | string   | false        | Catalog name. If not provided, uses current catalog.                                          |
| schema              | string   | false        | Schema name. If not provided, uses current schema.                                            |
| include_columns     | boolean  | false        | If true, includes detailed column statistics. Default: true.                                  |
| include_partitions  | boolean  | false        | If true, includes partition information if table is partitioned. Default: false.              |
| analyze_table       | boolean  | false        | If true, runs ANALYZE on the table before retrieving statistics. Default: false.              |

## Response Structure

The tool returns a JSON object with comprehensive table statistics:

```json
{
  "tableName": "orders",
  "catalogName": "hive",
  "schemaName": "sales",
  "tableType": "BASE TABLE",
  "rowCount": 1000000,
  "dataSizeBytes": 104857600,
  "dataSizeMB": 100.0,
  "lastAnalyzedTime": "2024-01-15T10:30:00Z",
  "tableProperties": {
    "transactional": "true",
    "format": "ORC"
  },
  "columnStatistics": [
    {
      "columnName": "order_id",
      "dataType": "bigint",
      "nullCount": 0,
      "distinctCount": 1000000,
      "minValue": "1",
      "maxValue": "1000000",
      "dataSizeBytes": 8000000
    },
    {
      "columnName": "customer_name",
      "dataType": "varchar",
      "nullCount": 500,
      "distinctCount": 50000,
      "minValue": "Aaron",
      "maxValue": "Zoe",
      "avgLength": 15.5,
      "maxLength": 50,
      "dataSizeBytes": 15500000
    }
  ],
  "partitionInfo": {
    "isPartitioned": true,
    "partitionColumns": ["order_date"],
    "partitionCount": 365,
    "partitions": [
      {
        "partitionValues": {"order_date": "2024-01-01"},
        "rowCount": 2750,
        "dataSizeBytes": 287500
      }
    ]
  },
  "storageInfo": {
    "location": "s3://bucket/path/to/table",
    "inputFormat": "org.apache.hadoop.hive.ql.io.orc.OrcInputFormat",
    "outputFormat": "org.apache.hadoop.hive.ql.io.orc.OrcOutputFormat",
    "compressed": true,
    "numFiles": 100,
    "totalSizeBytes": 104857600
  },
  "accessInfo": {
    "owner": "admin",
    "createdTime": "2023-01-01T00:00:00Z",
    "lastModifiedTime": "2024-01-15T10:30:00Z"
  },
  "errors": []
}
```

## Usage Examples

### Basic Table Statistics
```yaml
# Get statistics for a table using current catalog and schema
parameters:
  table_name: "orders"
  include_columns: true
```

### Fully Qualified Table
```yaml
# Get statistics for a fully qualified table name
parameters:
  table_name: "hive.sales.orders"
  include_columns: true
  include_partitions: true
```

### Update and Retrieve Statistics
```yaml
# Run ANALYZE before getting statistics
parameters:
  table_name: "customers"
  catalog: "hive"
  schema: "sales"
  analyze_table: true
  include_columns: true
```

### Partition Information
```yaml
# Get detailed partition information for a partitioned table
parameters:
  table_name: "events"
  include_partitions: true
  include_columns: false  # Skip column stats for faster response
```

## Notes

- The availability of certain statistics depends on the underlying connector
  and whether statistics have been collected
- Running ANALYZE (analyze_table: true) may take time for large tables
- Storage information availability depends on the connector type
- For better performance, disable include_columns if column statistics are not
  needed
- Partition information retrieval may be slow for tables with many partitions

## Reference

| **field**      | **type** | **required** | **description**                                                |
|----------------|:--------:|:------------:|----------------------------------------------------------------|
| kind           | string   | true         | Must be "trino-table-statistics".                             |
| source         | string   | true         | Name of the Trino source to connect to.                       |
| description    | string   | true         | Description of the tool that is passed to the LLM.            |
