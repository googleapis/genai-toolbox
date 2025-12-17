---
title: "postgres-list-publication-tables"
type: docs
weight: 1
description: >
 The "postgres-list-publication-tables" tool lists publication tables in a Postgres database.
aliases:
- /resources/tools/postgres-list-publication-tables
---

## About

The `postgres-list-publication-tables` tool lists all publication tables in the database. It's compatible with any of the following sources:

- [alloydb-postgres](../../sources/alloydb-pg.md)
- [cloud-sql-postgres](../../sources/cloud-sql-pg.md)
- [postgres](../../sources/postgres.md)

`postgres-list-publication-tables` lists detailed information as JSON for publication tables. A publication table in PostgreSQL is a 
table that is explicitly included as a source for replication within a publication (a set of changes generated from a table or group 
of tables) as part of the logical replication feature. The tool takes the following input parameters:

- `table_names` (optional): Filters by a comma-separated list of table names. Default: `""`
- `publication_names` (optional): Filters by a comma-separated list of publication names. Default: `""`
- `schema_names` (optional): Filters by a comma-separated list of schema names. Default: `""`
- `limit` (optional): The maximum number of rows to return. Default: `50`

## Example

```yaml
tools:
  list_indexes:
    kind: postgres-list-publication-tables
    source: postgres-source
    description: |
      Lists all tables that are explicitly part of a publication in the database.
      Tables that are part of a publication via 'FOR ALL TABLES' are not included,
      unless they are also explicitly added to the publication.
      Returns the publication name, schema name, and table name, along with
      definition details indicating if it publishes all tables, whether it
      replicates inserts, updates, deletes, or truncates, and the publication
      owner.
```

The response is a JSON array with the following elements:
```json
{
  "publication_name": "Name of the publication",
  "schema_name": "Name of the schema the table belongs to",
  "table_name": "Name of the table",
  "publishes_all_tables": "boolean indicating if the publication was created with FOR ALL TABLES",
  "publishes_inserts": "boolean indicating if INSERT operations are replicated",
  "publishes_updates": "boolean indicating if UPDATE operations are replicated",
  "publishes_deletes": "boolean indicating if DELETE operations are replicated",
  "publishes_truncates": "boolean indicating if TRUNCATE operations are replicated",
  "publication_owner": "Username of the database role that owns the publication"
}
```

## Reference

| **field**   | **type** | **required** | **description**                                      |
|-------------|:--------:|:------------:|------------------------------------------------------|
| kind        |  string  |     true     | Must be "postgres-list-publication-tables".          |
| source      |  string  |     true     | Name of the source the SQL should execute on.        |
| description |  string  |    false     | Description of the tool that is passed to the agent. |
