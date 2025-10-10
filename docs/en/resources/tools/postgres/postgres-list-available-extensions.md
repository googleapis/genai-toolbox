---
title: "postgres-list-available-extensions"
type: docs
weight: 1
description: >
  The "postgres-list-available-extensions" tool retrieves all PostgreSQL
  extensions available for installation on a Postgres database.
aliases:
- /resources/tools/postgres-list-available-extensions
---

## About

The `postgres-list-available-extensions` tool retrieves all PostgreSQL
extensions available for installation on a Postgres database. It's compatible
with any of the following sources:

- [alloydb-postgres](../../sources/alloydb-pg.md)
- [cloud-sql-postgres](../../sources/cloud-sql-pg.md)
- [postgres](../../sources/postgres.md)

`postgres-list-available-extensions` lists all PostgreSQL extensions available
for installation (extension name, default version description) as JSON. The does
not support any input parameter.

## Example

```yaml
tools:
  list_available_extensions:
    kind: postgres-list-available-extensions
    source: postgres-source
    description: Discover all PostgreSQL extensions available for installation on this server, returning name, default_version, and description.
```

## Reference

| **name**             | **default_version** | **description**                                                                                                     |
|----------------------|---------------------|---------------------------------------------------------------------------------------------------------------------|
| address_standardizer | 3.5.2               | Used to parse an address into constituent elements. Generally used to support geocoding address normalization step. |
| amcheck              | 1.4                 | functions for verifying relation integrity                                                                          |
| anon                 | 1.0.0               | Data anonymization tools                                                                                            |
| autoinc              | 1.0                 | functions for autoincrementing fields                                                                               |
