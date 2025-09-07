---
title: "trino-list-catalogs"
type: docs
weight: 7
description: >
  A "trino-list-catalogs" tool lists all available catalogs in the Trino cluster
  with schema counts.
aliases:
- /resources/tools/trino-list-catalogs
---

## About

A `trino-list-catalogs` tool provides a simple way to discover all available
catalogs in a Trino cluster. It returns a structured list of catalogs along
with the count of schemas in each catalog, helping users understand the
available data sources.

This tool is compatible with:
- [trino](../../sources/trino.md)

## Features

- **Catalog Discovery**: Lists all accessible catalogs in the cluster
- **Schema Counts**: Provides the number of schemas in each catalog
- **No Parameters Required**: Simple, parameterless operation
- **Structured Response**: Returns JSON with catalog information

## Example

```yaml
tools:
  list_catalogs:
    kind: trino-list-catalogs
    source: my-trino-instance
    description: Lists all available catalogs with schema counts
```

## Parameters

This tool requires no parameters.

## Response Structure

The tool returns a JSON object containing all catalogs:

```json
{
  "catalogs": [
    {
      "catalogName": "hive",
      "schemaCount": 15
    },
    {
      "catalogName": "mysql",
      "schemaCount": 3
    },
    {
      "catalogName": "postgresql",
      "schemaCount": 8
    },
    {
      "catalogName": "system",
      "schemaCount": 4
    }
  ],
  "totalCount": 4
}
```

## Usage Examples

### List All Catalogs
```yaml
# No parameters needed - lists all available catalogs
tools:
  catalogs:
    kind: trino-list-catalogs
    source: trino-source
    description: Get all available catalogs
```

### With Authentication
```yaml
# Use with authentication if required
tools:
  secure_catalogs:
    kind: trino-list-catalogs
    source: secure-trino
    description: List catalogs with authentication
    authRequired:
      - my-auth-service
```

## Response Fields

| **field**      | **type**   | **description**                                           |
|----------------|------------|-----------------------------------------------------------|
| catalogs       | array      | List of catalog objects                                  |
| catalogName    | string     | Name of the catalog                                      |
| schemaCount    | integer    | Number of schemas in the catalog                         |
| totalCount     | integer    | Total number of catalogs available                       |

## Notes

- The tool returns all catalogs visible to the authenticated user
- Schema counts include all schemas, including system schemas
- Catalogs with no schemas will show a count of 0
- The list is ordered alphabetically by catalog name

## Reference

| **field**      | **type** | **required** | **description**                                                |
|----------------|:--------:|:------------:|----------------------------------------------------------------|
| kind           | string   | true         | Must be "trino-list-catalogs".                                |
| source         | string   | true         | Name of the Trino source to connect to.                       |
| description    | string   | true         | Description of the tool that is passed to the LLM.            |
| authRequired   | array    | false        | List of authentication services required.                     |
