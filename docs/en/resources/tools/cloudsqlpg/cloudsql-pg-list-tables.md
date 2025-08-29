---
title: "cloudsql-pg-list-tables"
type: docs
weight: 1
description: >
  The "cloudsql-pg-list-tables" tool lists schema information for all or specified tables in an CloudSQL for PostgreSQL database.
aliases:
- /resources/tools/cloudsql-pg-list-tables
---

## About

The `cloudsql-pg-list-tables` tool retrieves schema information for all or specified tables in an CloudSQL for PostgreSQL database.
It's compatible with [cloud-sql-pg](../../sources/cloud-sql-pg.md) source.

`cloudsql-pg-list-tables` lists detailed schema information (object type, columns, constraints, indexes, triggers, owner, comment) as JSON for user-created tables (ordinary or partitioned). Filters by a comma-separated list of names. If names are omitted, it lists all tables in user schemas. The output format can be set to `simple` which will return only the table names or `detailed` which is the default.

## Example

```yaml
tools:
  cloudsql_pg_list_tables:
    kind: cloudsql-pg-list-tables
    source: cloudsql-pg-source
    description: Use this tool to retrieve schema information for all or specified tables. Output format can be simple (only table names) or detailed.
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be "cloudsql-pg-list-tables".                                                               |
| source      |                   string                   |     true     | Name of the source the SQL should execute on.                                                    |
| description |                   string                   |     true     | Description of the tool that is passed to the agent.                                             |
