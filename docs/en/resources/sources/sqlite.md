---
title: "SQLite"
linkTitle: "SQLite"
type: docs
weight: 1
description: >
  SQLite is a C-language library that implements a small, fast, self-contained, 
  high-reliability, full-featured, SQL database engine.
---

## About

[SQLite](https://sqlite.org/) is a software library that provides a relational
database management system. The lite in SQLite means lightweight in terms of
setup, database administration, and required resources.

SQLite has the following notable characteristics:

- Self-contained with no external dependencies
- Serverless - the SQLite library accesses its storage files directly
- Single database file that can be easily copied or moved
- Zero-configuration - no setup or administration needed
- Transactional with ACID properties

## Available Tools

- [`sqlite-sql`](../tools/sqlite/sqlite-sql.md)  
  Run SQL queries against a local SQLite database.

## Requirements

### Database File

You need a SQLite database file. This can be:

- An existing database file
- A path where a new database file should be created
- `:memory:` for an in-memory database

## Example

```yaml
sources:
    my-sqlite-db:
        kind: "sqlite"
        database: "/path/to/database.db"
```

For an in-memory database:

```yaml
sources:
    my-sqlite-memory-db:
        kind: "sqlite"
        database: ":memory:"
```

## Reference

### Configuration Fields

| **field** | **type** | **required** | **description**                                                                                                     |
|-----------|:--------:|:------------:|---------------------------------------------------------------------------------------------------------------------|
| kind      |  string  |     true     | Must be "spanner".                                                                                                  |
| database  |  string  |     true     | Path to SQLite database file, or ":memory:" for an in-memory database.                                              |

### Connection Properties

SQLite connections are configured with these defaults for optimal performance:

- `MaxOpenConns`: 1 (SQLite only supports one writer at a time)
- `MaxIdleConns`: 1
