---
title: "looker-get-connection-tables"
type: docs
weight: 1
description: >
  A "looker-get-connection-tables" tool returns all the tables in a connection.
aliases:
- /resources/tools/looker-get-connection-tables
---

## About

A `looker-get-connection-tables` tool returns all the tables in a connection.

It's compatible with the following sources:

- [looker](../../sources/looker.md)

`looker-get-connection-tables` accepts a `conn` parameter, a `schema` parameter, and an optional `db` parameter.

## Example

```yaml
tools:
    get_connection_tables:
        kind: looker-get-connection-tables
        source: looker-source
        description: |
          get_connection_tables Tool

          This tool will list the tables available from a connection, filtered by the 
          schema name and optional database name.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "looker-get-connection-tables".            |
| source      |  string  |     true     | Name of the source Looker instance.                |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |