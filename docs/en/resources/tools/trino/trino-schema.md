---
title: "trino-schema"
type: docs
weight: 3
description: >
  A "trino-schema" tool retrieves comprehensive schema information from a Trino
  database including catalogs, schemas, tables, columns, and cluster details.
aliases:
- /resources/tools/trino-schema
---

## About

A `trino-schema` tool provides comprehensive schema introspection for Trino
databases. It retrieves detailed information about the database structure
including all catalogs, schemas, tables, columns, and cluster configuration.
The tool includes built-in caching to improve performance for repeated schema
queries.

This tool is compatible with:
- [trino](../../sources/trino.md)

## Features

- **Complete Schema Discovery**: Retrieves all catalogs, schemas, tables, and
  column definitions
- **Cluster Information**: Provides details about Trino cluster nodes,
  coordinators, and workers
- **Statistics**: Calculates and returns database statistics including table
  counts, column counts, and table type distributions
- **Caching**: Built-in cache with configurable expiration to optimize
  performance for repeated queries
- **Concurrent Extraction**: Uses parallel processing to efficiently gather
  schema information

## Example

```yaml
tools:
  get_schema:
    kind: trino-schema
    source: my-trino-instance
    description: Get comprehensive schema information with caching
    cacheExpireMinutes: 10  # Optional: cache expiration in minutes (default: 10)
```

## Response Structure

The tool returns a JSON object with the following structure:

```json
{
  "catalogs": [
    {
      "name": "catalog_name",
      "schemas": [
        {
          "name": "schema_name",
          "tables": [
            {
              "name": "table_name",
              "type": "BASE TABLE",
              "columns": [
                {
                  "name": "column_name",
                  "dataType": "varchar",
                  "position": 1,
                  "isNullable": "YES",
                  "defaultValue": null,
                  "comment": "Column description"
                }
              ]
            }
          ]
        }
      ]
    }
  ],
  "clusterInfo": {
    "totalNodes": 5,
    "coordinators": [...],
    "workers": [...],
    "version": "410"
  },
  "statistics": {
    "totalCatalogs": 3,
    "totalSchemas": 15,
    "totalTables": 125,
    "totalColumns": 1250,
    "tablesByType": {
      "BASE TABLE": 100,
      "VIEW": 25
    }
  }
}
```

## Reference

| **field**           | **type** | **required** | **description**                                                               |
|---------------------|:--------:|:------------:|-------------------------------------------------------------------------------|
| kind                | string   | true         | Must be "trino-schema".                                                      |
| source              | string   | true         | Name of the Trino source to connect to.                                      |
| description         | string   | true         | Description of the tool that is passed to the LLM.                           |
| cacheExpireMinutes  | integer  | false        | Cache expiration time in minutes. Default is 10.                             |
