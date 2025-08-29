---
title: "postgres-list-tables"
type: docs
weight: 1
description: >
  The "postgres-list-tables" tool lists schema information for all or specified tables in a Postgres database.
aliases:
- /resources/tools/postgres-list-tables
---

## About

The `postgres-list-tables` tool retrieves schema information for all or specified tables in a Postgres database.
It's compatible with [postgres](../../sources/postgres.md) source.

`postgres-list-tables` lists detailed schema information (object type, columns, constraints, indexes, triggers, owner, comment) as JSON for user-created tables (ordinary or partitioned). Filters by a comma-separated list of names. If names are omitted, it lists all tables in user schemas. The output format can be set to `simple` which will return only the table names or `detailed` which is the default.

## Example

```yaml
tools:
  postgres_list_tables:
    kind: postgres-list-tables
    source: postgres-source
    description: Use this tool to retrieve schema information for all or specified tables. Output format can be simple (only table names) or detailed.
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be "postgres-list-tables".                                                                  |
| source      |                   string                   |     true     | Name of the source the SQL should execute on.                                                    |
| description |                   string                   |     true     | Description of the tool that is passed to the agent.                                             |
