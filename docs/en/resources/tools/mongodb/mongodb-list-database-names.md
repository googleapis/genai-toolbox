---
title: "mongodb-list-database-names"
type: docs
weight: 1
description: >
  A "mongodb-list-database-names" tool returns all database names from the MongoDB source.
aliases:
- /resources/tools/mongodb-list-database-names
---

## About

A `mongodb-list-database-names` tool returns all database names from the MongoDB source.
It's compatible with the following sources:

- [mongodb](../../sources/mongodb.md)

This tool does not accept any parameters and returns a list of all database names
accessible through the configured MongoDB connection.

## Example

```yaml
tools:
  mongodb_list_database_names:
    kind: mongodb-list-database-names
    source: my-mongodb-source
    description: Use this tool to list all available databases in MongoDB.
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be "mongodb-list-database-names".                                                           |
| source      |                   string                   |     true     | Name of the MongoDB source.                                                                      |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                               |
