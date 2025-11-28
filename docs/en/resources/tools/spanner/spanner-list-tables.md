---
title: "spanner-list-tables"
type: docs
weight: 3
description: >
  A "spanner-list-tables" tool retrieves schema information about tables in a
  Google Cloud Spanner database.
---

## About

A `spanner-list-tables` tool retrieves comprehensive schema information about
tables in a Cloud Spanner database. It automatically adapts to the database
dialect (GoogleSQL or PostgreSQL) and returns detailed metadata including
columns, constraints, and indexes. It's compatible with:

- [spanner](../../sources/spanner.md)

This tool is read-only and executes pre-defined SQL queries against the
`INFORMATION_SCHEMA` tables to gather metadata. The tool automatically detects
the database dialect from the source configuration and uses the appropriate SQL
syntax.

## Features

- **Automatic Dialect Detection**: Adapts queries based on whether the database
  uses GoogleSQL or PostgreSQL dialect
- **Comprehensive Schema Information**: Returns columns, data types, constraints,
  indexes, and table relationships
- **Flexible Filtering**: Can list all tables or filter by specific table names
- **Output Format Options**: Choose between simple (table names only) or detailed
  (full schema information) output

## Example

### Basic Usage - List All Tables

```yaml
sources:
  my-spanner-db:
    kind: spanner
    project: ${SPANNER_PROJECT}
    instance: ${SPANNER_INSTANCE}
    database: ${SPANNER_DATABASE}
    dialect: googlesql  # or postgresql

tools:
  list_all_tables:
    kind: spanner-list-tables
    source: my-spanner-db
    description: Lists all tables with their complete schema information
```

### List Specific Tables

```yaml
tools:
  list_specific_tables:
    kind: spanner-list-tables
    source: my-spanner-db
    description: |
      Lists schema information for specific tables.
      Example usage:
      {
        "table_names": "users,orders,products",
        "output_format": "detailed"
      }
```

## Parameters

The tool accepts two optional parameters:

| **parameter** | **type** | **default** | **description**                                                                                      |
|---------------|:--------:|:-----------:|------------------------------------------------------------------------------------------------------|
| table_names   |  string  |     ""      | Comma-separated list of table names to filter. If empty, lists all tables in user-accessible schemas |
| output_format |  string  | "detailed"  | Output format: "simple" returns only table names, "detailed" returns full schema information         |

## Output Format

### Simple Format

When `output_format` is set to "simple", the tool returns a minimal JSON structure:

```json
[
  {
    "schema_name": "public",
    "object_name": "users",
    "object_details": {
      "name": "users"
    }
  },
  {
    "schema_name": "public",
    "object_name": "orders",
    "object_details": {
      "name": "orders"
    }
  }
]
```

### Detailed Format

When `output_format` is set to "detailed" (default), the tool returns
comprehensive schema information:

```json
[
  {
    "schema_name": "public",
    "object_name": "users",
    "object_details": {
      "schema_name": "public",
      "object_name": "users",
      "object_type": "BASE TABLE",
      "columns": [
        {
          "column_name": "id",
          "data_type": "INT64",
          "ordinal_position": 1,
          "is_not_nullable": true,
          "column_default": null
        },
        {
          "column_name": "email",
          "data_type": "STRING(255)",
          "ordinal_position": 2,
          "is_not_nullable": true,
          "column_default": null
        }
      ],
      "constraints": [
        {
          "constraint_name": "PK_users",
          "constraint_type": "PRIMARY KEY",
          "constraint_definition": "PRIMARY KEY (id)",
          "constraint_columns": [
            "id"
          ],
          "foreign_key_referenced_table": null,
          "foreign_key_referenced_columns": []
        }
      ],
      "indexes": [
        {
          "index_name": "idx_users_email",
          "index_type": "INDEX",
          "is_unique": true,
          "is_null_filtered": false,
          "interleaved_in_table": null,
          "index_key_columns": [
            {
              "column_name": "email",
              "ordering": "ASC"
            }
          ],
          "storing_columns": []
        }
      ]
    }
  }
]
```

## Use Cases

1. **Database Documentation**: Generate comprehensive documentation of your
   database schema
2. **Schema Validation**: Verify that expected tables and columns exist
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
    kind: spanner-list-tables
    source: spanner-db
    description: |
      Use this tool to inspect database schema information.
      You can:
      - List all tables by leaving table_names empty
      - Get specific table schemas by providing comma-separated table names
      - Choose between simple (names only) or detailed (full schema) output
      
      Examples:
      1. List all tables with details: {"output_format": "detailed"}
      2. Get specific tables: {"table_names": "users,orders", "output_format": "detailed"}
      3. Just get table names: {"output_format": "simple"}
```

## Reference

| **field**    | **type** | **required** | **description**                                    |
|--------------|:--------:|:------------:|----------------------------------------------------|
| kind         |  string  |     true     | Must be "spanner-list-tables"                      |
| source       |  string  |     true     | Name of the Spanner source to query                |
| description  |  string  |    false     | Description of the tool that is passed to the LLM  |
| authRequired | string[] |    false     | List of auth services required to invoke this tool |

## Notes

- This tool is read-only and does not modify any data
- The tool automatically handles both GoogleSQL and PostgreSQL dialects
- Large databases with many tables may take longer to query
