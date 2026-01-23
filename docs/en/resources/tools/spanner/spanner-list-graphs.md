---
title: "spanner-list-graphs"
type: docs
weight: 3
description: >
  A "spanner-list-graphs" tool retrieves schema information about graphs in a
  Google Cloud Spanner database.
---

## About

A `spanner-list-graphs` tool retrieves comprehensive schema information about
graphs in a Cloud Spanner database. It returns detailed metadata including
node tables, edge tables, labels and property declarations. It's compatible with:

- [spanner](../../sources/spanner.md)

This tool is read-only and executes pre-defined SQL queries against the
`INFORMATION_SCHEMA` tables to gather metadata. 
{{< notice warning >}}
The tool only works for the GoogleSQL 
source dialect, as Spanner Graph isn't available in the PostgreSQL dialect.
{{< /notice >}}

## Features

- **Comprehensive Schema Information**: Returns node tables, edge tables, labels
  and property declarations
- **Flexible Filtering**: Can list all graphs or filter by specific graph names
- **Output Format Options**: Choose between simple (graph names only) or detailed
  (full schema information) output

## Example

### Basic Usage - List All Graphs

```yaml
sources:
  my-spanner-db:
    kind: spanner
    project: ${SPANNER_PROJECT}
    instance: ${SPANNER_INSTANCE}
    database: ${SPANNER_DATABASE}
    dialect: googlesql  # wont work for postgresql

tools:
  list_all_graphs:
    kind: spanner-list-graphs
    source: my-spanner-db
    description: Lists all graphs with their complete schema information
```

### List Specific Graphs

```yaml
tools:
  list_specific_graphs:
    kind: spanner-list-graphs
    source: my-spanner-db
    description: |
      Lists schema information for specific graphs.
      Example usage:
      {
        "graph_names": "FinGraph,SocialGraph",
        "output_format": "detailed"
      }
```

## Parameters

The tool accepts two optional parameters:

| **parameter** | **type** | **default** | **description**                                                                                      |
|---------------|:--------:|:-----------:|------------------------------------------------------------------------------------------------------|
| graph_names   |  string  |     ""      | Comma-separated list of graph names to filter. If empty, lists all graphs in user-accessible schemas |
| output_format |  string  | "detailed"  | Output format: "simple" returns only graph names, "detailed" returns full schema information         |

## Output Format

### Simple Format

When `output_format` is set to "simple", the tool returns a minimal JSON structure:

```json
[
  {
    "object_details": {
      "name": "FinGraph"
    },
    "object_name": "FinGraph",
    "schema_name": ""
  },
  {
    "object_details": {
      "name": "SocialGraph"
    },
    "object_name": "SocialGraph",
    "schema_name": ""
  }
]
```

### Detailed Format

When `output_format` is set to "detailed" (default), the tool returns
comprehensive schema information:

```json
[
  {
    "object_details": {
      "catalog": "",
      "edge_tables": [
        {
          "baseCatalogName": "",
          "baseSchemaName": "",
          "baseTableName": "Knows",
          "destinationNodeTable": {
            "edgeTableColumns": [
              "DstId"
            ],
            "nodeTableColumns": [
              "Id"
            ],
            "nodeTableName": "Person"
          },
          "keyColumns": [
            "SrcId",
            "DstId"
          ],
          "kind": "EDGE",
          "labelNames": [
            "Knows"
          ],
          "name": "Knows",
          "propertyDefinitions": [
            {
              "propertyDeclarationName": "DstId",
              "valueExpressionSql": "DstId"
            },
            {
              "propertyDeclarationName": "SrcId",
              "valueExpressionSql": "SrcId"
            }
          ],
          "sourceNodeTable": {
            "edgeTableColumns": [
              "SrcId"
            ],
            "nodeTableColumns": [
              "Id"
            ],
            "nodeTableName": "Person"
          }
        }
      ],
      "labels": [
        {
          "name": "Knows",
          "propertyDeclarationNames": [
            "DstId",
            "SrcId"
          ]
        },
        {
          "name": "Person",
          "propertyDeclarationNames": [
            "Id",
            "Name"
          ]
        }
      ],
      "node_tables": [
        {
          "baseCatalogName": "",
          "baseSchemaName": "",
          "baseTableName": "Person",
          "keyColumns": [
            "Id"
          ],
          "kind": "NODE",
          "labelNames": [
            "Person"
          ],
          "name": "Person",
          "propertyDefinitions": [
            {
              "propertyDeclarationName": "Id",
              "valueExpressionSql": "Id"
            },
            {
              "propertyDeclarationName": "Name",
              "valueExpressionSql": "Name"
            }
          ]
        }
      ],
      "object_name": "SocialGraph",
      "property_declarations": [
        {
          "name": "DstId",
          "type": "INT64"
        },
        {
          "name": "Id",
          "type": "INT64"
        },
        {
          "name": "Name",
          "type": "STRING"
        },
        {
          "name": "SrcId",
          "type": "INT64"
        }
      ],
      "schema_name": ""
    },
    "object_name": "SocialGraph",
    "schema_name": ""
  }
]
```

## Use Cases

1. **Database Documentation**: Generate comprehensive documentation of your
   database schema
2. **Schema Validation**: Verify that expected graphs, node and edge exist
3. **Migration Planning**: Understand the current schema before making changes
4. **Development Tools**: Build tools that need to understand database structure
5. **Audit and Compliance**: Track schema changes and ensure compliance with
   data governance policies

## Example with Agent Integration

```yaml
sources:
  spanner-db:
    kind: spanner
    project: my-project
    instance: my-instance
    database: my-database
    dialect: googlesql

tools:
  schema_inspector:
    kind: spanner-list-graphs
    source: spanner-db
    description: |
      Use this tool to inspect database schema information.
      You can:
      - List all graphs by leaving graph_names empty
      - Get specific graph schemas by providing comma-separated graph names
      - Choose between simple (names only) or detailed (full schema) output
      
      Examples:
      1. List all graphs with details: {"output_format": "detailed"}
      2. Get specific graphs: {"graph_names": "FinGraph,SocialGraph", "output_format": "detailed"}
      3. Just get graph names: {"output_format": "simple"}
```

## Reference

| **field**    | **type** | **required** | **description**                                                 |
|--------------|:--------:|:------------:|-----------------------------------------------------------------|
| kind         |  string  |     true     | Must be "spanner-list-graphs"                                   |
| source       |  string  |     true     | Name of the Spanner source to query (dialect must be GoogleSQL) |
| description  |  string  |    false     | Description of the tool that is passed to the LLM               |
| authRequired | string[] |    false     | List of auth services required to invoke this tool              |

## Notes

- This tool is read-only and does not modify any data
- The tool only works for the GoogleSQL source dialect
- Large databases with many graphs may take longer to query
