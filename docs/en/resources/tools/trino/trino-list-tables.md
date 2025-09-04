---
title: "trino-list-tables"
type: docs
weight: 9
description: >
  A "trino-list-tables" tool lists tables within a specific schema with
  filtering options and detailed information.
aliases:
- /resources/tools/trino-list-tables
---

## About

A `trino-list-tables` tool provides comprehensive table listing capabilities
for Trino schemas. It can list tables and views with optional filtering,
include column counts for each table, and supports SQL LIKE pattern matching
for table names.

This tool is compatible with:
- [trino](../../sources/trino.md)

## Features

- **Table and View Listing**: Lists tables and optionally views in a schema
- **Pattern Matching**: Filter tables using SQL LIKE patterns
- **Column Counts**: Optionally include column count for each table
- **Current Schema Support**: Uses current catalog/schema if not specified
- **Type Separation**: Distinguishes between tables and views
- **Detailed Metadata**: Provides comprehensive information for each table

## Example

```yaml
tools:
  list_tables:
    kind: trino-list-tables
    source: my-trino-instance
    description: Lists tables with filtering and detailed information
```

## Parameters

| **parameter**     | **type** | **required** | **description**                                                                     |
|-------------------|:--------:|:------------:|--------------------------------------------------------------------------------------|
| catalog           | string   | false        | Catalog name. If not provided, uses current catalog                                 |
| schema            | string   | false        | Schema name. If not provided, uses current schema                                   |
| table_filter      | string   | false        | Filter tables by name pattern (supports SQL LIKE wildcards: % and _)                |
| include_views     | boolean  | false        | If true, includes views in the results. Default: true                               |
| include_details   | boolean  | false        | If true, includes additional details like column count. Default: false              |

## Response Structure

The tool returns a JSON object containing tables with their metadata:

```json
{
  "catalog": "hive",
  "schema": "sales",
  "tables": [
    {
      "catalogName": "hive",
      "schemaName": "sales",
      "tableName": "orders",
      "tableType": "BASE TABLE",
      "columnCount": 15
    },
    {
      "catalogName": "hive",
      "schemaName": "sales",
      "tableName": "customers",
      "tableType": "BASE TABLE",
      "columnCount": 8
    },
    {
      "catalogName": "hive",
      "schemaName": "sales",
      "tableName": "sales_summary",
      "tableType": "VIEW",
      "columnCount": 5
    }
  ],
  "totalCount": 3,
  "tableCount": 2,
  "viewCount": 1
}
```

## Usage Examples

### List All Tables in Current Schema
```yaml
# Lists all tables and views in the current schema
parameters: {}
```

### List Tables in Specific Schema
```yaml
# Lists tables in a specific catalog and schema
parameters:
  catalog: "hive"
  schema: "sales"
```

### Filter Tables by Pattern
```yaml
# Lists tables whose names start with "customer"
parameters:
  catalog: "hive"
  schema: "sales"
  table_filter: "customer%"
```

### Tables Only (No Views)
```yaml
# Lists only tables, excluding views
parameters:
  include_views: false
```

### With Column Counts
```yaml
# Lists tables with column count information
parameters:
  schema: "analytics"
  include_details: true
```

### Complex Filter Example
```yaml
# Find all tables with "order" in the name, including details
parameters:
  catalog: "hive"
  schema: "sales"
  table_filter: "%order%"
  include_details: true
  include_views: true
```

## Response Fields

| **field**      | **type**   | **description**                                              |
|----------------|------------|--------------------------------------------------------------|
| catalog        | string     | The catalog containing the tables                           |
| schema         | string     | The schema containing the tables                            |
| tables         | array      | List of table objects                                       |
| catalogName    | string     | Catalog name (in each table object)                         |
| schemaName     | string     | Schema name (in each table object)                          |
| tableName      | string     | Name of the table                                           |
| tableType      | string     | Type: "BASE TABLE" or "VIEW"                                |
| columnCount    | integer    | Number of columns (only when include_details is true)       |
| totalCount     | integer    | Total number of tables and views returned                   |
| tableCount     | integer    | Number of tables (BASE TABLE) in results                    |
| viewCount      | integer    | Number of views in results                                  |

## SQL LIKE Pattern Syntax

The `table_filter` parameter supports SQL LIKE pattern matching:
- `%` - Matches any sequence of characters
- `_` - Matches any single character
- `\%` - Matches a literal percent sign
- `\_` - Matches a literal underscore

Examples:
- `customer%` - Tables starting with "customer"
- `%order%` - Tables containing "order"
- `user_` - Tables like "users", "user1", etc.
- `temp_%_2024` - Tables like "temp_data_2024", "temp_log_2024"

## Notes

- Results are ordered by table type (tables first, then views) and table name
- Column counts require an additional query per table when `include_details` is true
- Empty schemas will return an empty tables array
- When catalog/schema are not specified, uses `CURRENT_CATALOG` and `CURRENT_SCHEMA`
- Large schemas may take longer when `include_details` is enabled

## Reference

| **field**      | **type** | **required** | **description**                                                |
|----------------|:--------:|:------------:|----------------------------------------------------------------|
| kind           | string   | true         | Must be "trino-list-tables".                                  |
| source         | string   | true         | Name of the Trino source to connect to.                       |
| description    | string   | true         | Description of the tool that is passed to the LLM.            |
| authRequired   | array    | false        | List of authentication services required.                     |
