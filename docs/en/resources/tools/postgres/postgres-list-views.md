---
title: "postgres-list-views"
type: docs
weight: 1
description: >
  The "postgres-list-views" tool lists views in a Postgres database, with a default limit of 50 rows.
aliases:
- /resources/tools/postgres-list-views
---

## About

The `postgres-list-views` tool retrieves a list of top N (default 50) views from
a Postgres database, excluding those in system schemas (`pg_catalog`,
`information_schema`). It's compatible with any of the following sources:

- [alloydb-postgres](../../sources/alloydb-pg.md)
- [cloud-sql-postgres](../../sources/cloud-sql-pg.md)
- [postgres](../../sources/postgres.md)

`postgres-list-views` lists detailed view information (schemaname, viewname,
ownername, definition) as JSON for views in a database. The tool takes the following input
parameters:

- `view_name` (optional): A string pattern to filter view names. Default: `""`
- `schema_name` (optional): A string pattern to filter schema names. Default: `""`
- `limit` (optional): The maximum number of rows to return. Default: `50`.

## Example

```yaml
tools:
  list_views:
    kind: postgres-list-views
    source: cloudsql-pg-source
```

## Reference

| **field**   | **type** | **required** | **description**                                      |
|-------------|:--------:|:------------:|------------------------------------------------------|
| kind        |  string  |     true     | Must be "postgres-list-views".                       |
| source      |  string  |     true     | Name of the source the SQL should execute on.        |
| description |  string  |    false     | Description of the tool that is passed to the agent. |
