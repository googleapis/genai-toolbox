---
title: "postgres-list-installed-extensions"
type: docs
weight: 1
description: >
  The "postgres-list-installed-extensions" tool retrieves all PostgreSQL extensions installed on a Postgres database.
aliases:
- /resources/tools/postgres-list-installed-extensions
---

## About

The `postgres-list-installed-extensions` tool retrieves all PostgreSQL
extensions installed on a Postgres database. It's compatible with any of the
following sources:

- [alloydb-postgres](../../sources/alloydb-pg.md)
- [cloud-sql-postgres](../../sources/cloud-sql-pg.md)
- [postgres](../../sources/postgres.md)

`postgres-list-installed-extensions` lists all installed PostgreSQL extensions
(extension name, version, schema, owner, description) as JSON. The does not
support any input parameter.

## Example

```yaml
tools:
  list_installed_extensions:
    kind: postgres-list-installed-extensions
    source: postgres-source
    description: List all installed PostgreSQL extensions with their name, version, schema, owner, and description.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| kind        |  string  |     true     | Must be "postgres-list-active-queries".            |
| source      |  string  |     true     | Name of the source the SQL should execute on.      |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
