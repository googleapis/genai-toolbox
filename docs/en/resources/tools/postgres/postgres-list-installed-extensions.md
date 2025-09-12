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

The `postgres-list-installed-extensions` tool retrieves all PostgreSQL extensions installed on a Postgres database. It's compatible with any of the following sources:

- [alloydb-postgres](../../sources/alloydb-pg.md)
- [cloud-sql-postgres](../../sources/cloud-sql-pg.md)
- [postgres](../../sources/postgres.md)

`postgres-list-installed-extensions` lists all installed PostgreSQL extensions (extension name, version, schema, owner, description) as JSON. The does not support any input parameter.

## Example

```yaml
tools:
  list_installed_extensions:
    kind: postgres-list-installed-extensions
    source: postgres-source
    description: List all installed PostgreSQL extensions with their name, version, schema, owner, and description.
```

## Reference

|**name**                |**version**|**schema**|**owner**   |**description**                                                                                                        |
|--------------------|-------|------|--------|-------------------------------------------------------------------------------------------------------------------|
|address_standardizer|3.5.2  |public|postgres|Used to parse an address into constituent elements. Generally used to support geocoding address normalization step.|
|amcheck             |1.4    |public|postgres|functions for verifying relation integrity                                                                         |
|anon                |1.0.0  |public|admin   |Data anonymization tools                                                                                           |
|autoinc             |1.0    |public|admin   |functions for autoincrementing fields                                                                              |
