---
title: "looker-get-connection-databases"
type: docs
weight: 1
description: >
  A "looker-get-connection-databases" tool returns all the databases in a connection.
aliases:
- /resources/tools/looker-get-connection-databases
---

## About

A `looker-get-connection-databases` tool returns all the databases in a connection.

It's compatible with the following sources:

- [looker](../../sources/looker.md)

`looker-get-connection-databases` accepts a `conn` parameter.

## Example

```yaml
tools:
    get_connection_databases:
        kind: looker-get-connection-databases
        source: looker-source
        description: |
          get_connection_databases Tool

          This tool will list the databases available from a connection if the connection
          supports multiple databases.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "looker-get-connection-databases".         |
| source      |  string  |     true     | Name of the source Looker instance.                |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
