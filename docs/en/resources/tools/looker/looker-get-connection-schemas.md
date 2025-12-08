---
title: "looker-get-connection-schemas"
type: docs
weight: 1
description: >
  A "looker-get-connection-schemas" tool returns all the schemas in a connection.
aliases:
- /resources/tools/looker-get-connection-schemas
---

## About

A `looker-get-connection-schemas` tool returns all the schemas in a connection.

It's compatible with the following sources:

- [looker](../../sources/looker.md)

`looker-get-connection-schemas` accepts a `conn` parameter and an optional `db` parameter.

## Example

```yaml
tools:
    get_connection_schemas:
        kind: looker-get-connection-schemas
        source: looker-source
        description: |
          get_connection_schemas Tool

          This tool retrieves a list of database schemas available through a specified
          Looker connection.

          Parameters:
          - connection_name (required): The name of the database connection, obtained from `get_connections`.
          - database (optional): An optional database name to filter the schemas.
            Only applicable for connections that support multiple databases.

          Output:
          A JSON array of strings, where each string is the name of an available schema.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "looker-get-connection-schemas".           |
| source      |  string  |     true     | Name of the source Looker instance.                |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
