---
title: "trino-list-schemas"
type: docs
weight: 8
description: >
  A "trino-list-schemas" tool lists schemas within a specific catalog with
  optional system schema filtering.
aliases:
- /resources/tools/trino-list-schemas
---

## About

A `trino-list-schemas` tool provides schema discovery within Trino catalogs.
It lists all schemas in a specified catalog (or the current catalog) with
counts of tables and views in each schema. The tool can optionally filter out
system schemas for cleaner results.

This tool is compatible with:
- [trino](../../sources/trino.md)

## Features

- **Schema Discovery**: Lists all schemas in a catalog
- **Object Counts**: Shows table and view counts per schema
- **System Schema Filtering**: Option to exclude system schemas
- **Current Catalog Support**: Uses current catalog if not specified
- **Structured Response**: Returns organized JSON with schema details

## Example

```yaml
tools:
  list_schemas:
    kind: trino-list-schemas
    source: my-trino-instance
    description: Lists schemas with table and view counts
```

## Parameters

| **parameter**    | **type** | **required** | **description**                                                          |
|------------------|:--------:|:------------:|---------------------------------------------------------------------------|
| catalog          | string   | false        | Catalog name to list schemas from. If not provided, uses current catalog |
| include_system   | boolean  | false        | If true, includes system schemas. Default: false                         |

## Response Structure

The tool returns a JSON object containing schemas with their metadata:

```json
{
  "catalog": "hive",
  "schemas": [
    {
      "catalogName": "hive",
      "schemaName": "sales",
      "tableCount": 25,
      "viewCount": 10
    },
    {
      "catalogName": "hive",
      "schemaName": "marketing",
      "tableCount": 15,
      "viewCount": 5
    },
    {
      "catalogName": "hive",
      "schemaName": "analytics",
      "tableCount": 30,
      "viewCount": 20
    }
  ],
  "totalCount": 3
}
```

## Usage Examples

### List Schemas in Current Catalog
```yaml
# Lists schemas in the current catalog, excluding system schemas
parameters: {}
```

### List Schemas in Specific Catalog
```yaml
# Lists all schemas in the hive catalog
parameters:
  catalog: "hive"
```

### Include System Schemas
```yaml
# Lists all schemas including system schemas
parameters:
  catalog: "postgresql"
  include_system: true
```

### Current Catalog with System Schemas
```yaml
# Lists all schemas in current catalog including system schemas
parameters:
  include_system: true
```

## Response Fields

| **field**      | **type**   | **description**                                           |
|----------------|------------|-----------------------------------------------------------|
| catalog        | string     | The catalog containing the schemas                       |
| schemas        | array      | List of schema objects                                   |
| catalogName    | string     | Name of the catalog (in each schema object)             |
| schemaName     | string     | Name of the schema                                       |
| tableCount     | integer    | Number of tables in the schema                           |
| viewCount      | integer    | Number of views in the schema                            |
| totalCount     | integer    | Total number of schemas returned                         |

## Notes

- System schemas filtered by default include: `information_schema`, `pg_catalog`, `sys`
- Table and view counts only include accessible objects
- Empty schemas (with no tables or views) are still included
- Results are ordered alphabetically by schema name
- When catalog is not specified, uses `CURRENT_CATALOG` from the session

## Reference

| **field**      | **type** | **required** | **description**                                                |
|----------------|:--------:|:------------:|----------------------------------------------------------------|
| kind           | string   | true         | Must be "trino-list-schemas".                                 |
| source         | string   | true         | Name of the Trino source to connect to.                       |
| description    | string   | true         | Description of the tool that is passed to the LLM.            |
| authRequired   | array    | false        | List of authentication services required.                     |
