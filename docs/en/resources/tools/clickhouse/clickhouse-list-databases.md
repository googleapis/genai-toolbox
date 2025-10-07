---
title: "clickhouse-list-databases"
type: docs
weight: 3
description: >
  A "clickhouse-list-databases" tool lists all databases in a ClickHouse instance.
aliases:
- /resources/tools/clickhouse-list-databases
---

## About

A `clickhouse-list-databases` tool lists all available databases in a ClickHouse
instance. It's compatible with the [clickhouse](../../sources/clickhouse.md)
source.

This tool executes the `SHOW DATABASES` command and returns a list of all 
databases accessible to the configured user, making it useful for database 
discovery and exploration tasks.

## Example

```yaml
tools:
  list_clickhouse_databases:
    kind: clickhouse-list-databases
    source: my-clickhouse-instance
    description: List all available databases in the ClickHouse instance
```

## Return Value

The tool returns an array of objects, where each object contains:
- `name`: The name of the database

Example response:
```json
[
  {"name": "default"},
  {"name": "system"},
  {"name": "analytics"},
  {"name": "user_data"}
]
```

## Reference

| **field**    |      **type**      | **required** | **description**                                       |
|--------------|:------------------:|:------------:|-------------------------------------------------------|
| kind         |       string       |     true     | Must be "clickhouse-list-databases".                  |
| source       |       string       |     true     | Name of the ClickHouse source to list databases from. |
| description  |       string       |     true     | Description of the tool that is passed to the LLM.    |
| authRequired |  array of string   |    false     | Authentication services required to use this tool.    |
| parameters   | array of Parameter |    false     | Parameters for the tool (typically not used).         |
