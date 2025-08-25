---
title: "alloydb-pg-list-tables"
type: docs
weight: 1
description: >
  The "alloydb-pg-list-tables" tool lists schema information for all or specified tables in an AlloyDB for PostgreSQL database.
aliases:
- /resources/tools/alloydb-pg-list-tables
---

## About

The `alloydb-pg-list-tables` tool lists schema information for all or specified tables in an AlloyDB for PostgreSQL database.
It's compatible with the following sources:

- [alloydb-pg](../../sources/alloydb-pg.md)

`alloydb-pg-list-tables` lists detailed schema information (object type, columns, constraints, indexes, triggers, owner, comment) as JSON for user-created tables (ordinary or partitioned). Filters by a comma-separated list of names. If names are omitted, it lists all tables in user schemas. The output format can be set to `simple` which will return only the table names or `detailed` which is the default.

## Example

```yaml
tools:
  alloydb_pg_list_tables:
    kind: alloydb-pg-list-tables
    source: my-alloydb-pg-source
    description: Use this tool to get schema information for a given table.
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be "alloydb-pg-list-tables".                                                               |
| source      |                   string                   |     true     | Name of the source the SQL should execute on.                                                    |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                               |