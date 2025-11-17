---
title: "postgres-list-schemas"
type: docs
weight: 1
description: >
  The "postgres-list-schemas" tool lists user-defined schemas in a database.
aliases:
- /resources/tools/postgres-list-schemas
---

## About

The `postgres-list-schemas` tool retrieves information about schemas in a
database excluding system and temporary schemas.  It's compatible with any of
the following sources:

- [alloydb-postgres](../../sources/alloydb-pg.md)
- [cloud-sql-postgres](../../sources/cloud-sql-pg.md)
- [postgres](../../sources/postgres.md)

`postgres-list-schemas` lists detailed information as JSON for each schema. The
tool takes the following input parameters:

- `schema_name` (optional): A pattern to filter schema names using SQL LIKE
  operator.
   If omitted, all user-defined schemas are returned.

## Example

```yaml
tools:
  list_schemas:
    kind: postgres-list-schemas
    source: postgres-source
    description: "Lists all schemas in the database ordered by schema name and excluding system and temporary schemas. It returns the schema name, schema owner, grants, number of functions, number of tables and number of views within each schema."
```

The response is a json array with the following elements:

```json
{
  "schema_name": "name of the schema.",
  "owner": "role that owns the schema",
  "grants": "A JSON object detailing the privileges (e.g., USAGE, CREATE) granted to different roles or PUBLIC on the schema.",
  "tables": "The total count of tables within the schema",
  "views": "The total count of views within the schema",
  "functions": "The total count of functions",
}
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "postgres-list-schemas".                   |
| source      |  string  |     true     | Name of the source the SQL should execute on.      |
| description |  string  |    false     | Description of the tool that is passed to the LLM. |
