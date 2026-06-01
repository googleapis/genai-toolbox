---
title: "postgres-list-views"
type: docs
weight: 1
description: >
  The "postgres-list-views" tool lists views in a Postgres database, with a default limit of 50 rows.
---

## About

The `postgres-list-views` tool retrieves a list of top N (default 50) views from
a Postgres database, excluding those in system schemas (`pg_catalog`,
`information_schema`).

`postgres-list-views` lists detailed view information (schemaname, viewname,
ownername, definition) as JSON for views in a database. The tool takes the following input
parameters:

- `view_name` (optional): A string pattern to filter view names. Default: `""`
- `schema_name` (optional): A string pattern to filter schema names. Default: `""`
- `limit` (optional): The maximum number of rows to return. Default: `50`.

## Compatible Sources

{{< compatible-sources others="integrations/alloydb, integrations/cloud-sql-pg">}}

## Example

```yaml
kind: tool
name: list_views
type: postgres-list-views
source: cloudsql-pg-source
```

## Reference

| **field**   | **type** | **required** | **description**                                      |
|-------------|:--------:|:------------:|------------------------------------------------------|
| type        |  string  |     true     | Must be "postgres-list-views".                       |
| source      |  string  |     true     | Name of the source the SQL should execute on.        |
| description |  string  |    false     | Description of the tool that is passed to the agent. |
