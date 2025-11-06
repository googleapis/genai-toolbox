---
title: "mongodb-list-collection-names"
type: docs
weight: 2
description: >
  A "mongodb-list-collection-names" tool returns all collection names from a specified database in the MongoDB source.
aliases:
- /resources/tools/mongodb-list-collection-names
---

## About

A `mongodb-list-collection-names` tool returns all collection names from a specified database in the MongoDB source.
It's compatible with the following sources:

- [mongodb](../../sources/mongodb.md)

`mongodb-list-collection-names` accepts the following parameter:
- **`database`** (required if not specified in config): The name of the database to list collections from.

The tool can be configured in two ways:
1. **With database in config:** The database name is fixed in the tool configuration.
2. **Without database in config:** The database name must be provided as a parameter when invoking the tool,
   allowing agents to dynamically query different databases.

## Examples

### Example 1: Database specified in configuration

```yaml
tools:
  mongodb_list_collections:
    kind: mongodb-list-collection-names
    source: my-mongodb-source
    database: my_database
    description: Use this tool to list all collections in the my_database database.
```

### Example 2: Database specified as parameter

```yaml
tools:
  mongodb_list_collections:
    kind: mongodb-list-collection-names
    source: my-mongodb-source
    description: Use this tool to list all collections in a specified database.
    params:
      - name: database
        description: The name of the database to list collections from
        type: string
        required: true
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be "mongodb-list-collection-names".                                                         |
| source      |                   string                   |     true     | Name of the MongoDB source.                                                                      |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                               |
| database    |                   string                   |    false     | Name of the database to list collections from. If not provided, must be passed as a parameter.   |
| params      |            array of parameters             |    false     | Additional parameters for the tool.                                                              |
