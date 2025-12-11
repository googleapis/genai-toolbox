---
title: "looker-get-connection-table-columns"
type: docs
weight: 1
description: >
  A "looker-get-connection-table-columns" tool returns all the columns for each table specified.
aliases:
- /resources/tools/looker-get-connection-table-columns
---

## About

A `looker-get-connection-table-columns` tool returns all the columnes for each table specified.

It's compatible with the following sources:

- [looker](../../sources/looker.md)

`looker-get-connection-table-columns` accepts a `conn` parameter, a `schema` parameter, a `tables` parameter with a comma separated list of tables, and an optional `db` parameter.

## Example

```yaml
tools:
    get_connection_table_columns:
        kind: looker-get-connection-table-columns
        source: looker-source
        description: |
          This tool retrieves a list of columns for one or more specified tables within a
          given database schema and connection.

          Parameters:
          - connection_name (required): The name of the database connection, obtained from `get_connections`.
          - schema (required): The name of the schema where the tables reside, obtained from `get_connection_schemas`.
          - tables (required): A comma-separated string of table names for which to retrieve columns
            (e.g., "users,orders,products"), obtained from `get_connection_tables`.
          - database (optional): The name of the database to filter by. Only applicable for connections
            that support multiple databases (check with `get_connections`).

          Output:
          A JSON array of objects, where each object represents a column and contains details
          such as `table_name`, `column_name`, `data_type`, and `is_nullable`.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "looker-get-connection-table-columns".     |
| source      |  string  |     true     | Name of the source Looker instance.                |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
