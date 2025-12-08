---
title: "looker-get-connections"
type: docs
weight: 1
description: >
  A "looker-get-connections" tool returns all the connections in the source.
aliases:
- /resources/tools/looker-get-connections
---

## About

A `looker-get-connections` tool returns all the connections in the source.

It's compatible with the following sources:

- [looker](../../sources/looker.md)

`looker-get-connections` accepts no parameters.

## Example

```yaml
tools:
    get_connections:
        kind: looker-get-connections
        source: looker-source
        description: |
          get_connections Tool

          This tool retrieves a list of all database connections configured in the Looker system.

          Parameters:
          This tool takes no parameters.

          Output:
          A JSON array of objects, each representing a database connection and including details such as:
          - `name`: The connection's unique identifier.
          - `dialect`: The database dialect (e.g., "mysql", "postgresql", "bigquery").
          - `default_schema`: The default schema for the connection.
          - `database`: The associated database name (if applicable).
          - `supports_multiple_databases`: A boolean indicating if the connection can access multiple databases.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "looker-get-connections".                  |
| source      |  string  |     true     | Name of the source Looker instance.                |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
