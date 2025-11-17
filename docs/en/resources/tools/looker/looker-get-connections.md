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

          This tool will list all the connections available in the Looker system, as
          well as the dialect name, the default schema, the database if applicable,
          and whether the connection supports multiple databases.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "looker-get-connections".                  |
| source      |  string  |     true     | Name of the source Looker instance.                |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
