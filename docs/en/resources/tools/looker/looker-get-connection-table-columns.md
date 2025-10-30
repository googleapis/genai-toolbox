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
          get_connection_table_columns Tool

          This tool will list the columns available from a connection, for all the tables
          given in a comma separated list of table names, filtered by the 
          schema name and optional database name.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "looker-get-connection-table-columns".     |
| source      |  string  |     true     | Name of the source Looker instance.                |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |