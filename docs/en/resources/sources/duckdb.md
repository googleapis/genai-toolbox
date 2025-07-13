---
title: DuckDB
linkTitle: DuckDB
type: docs
weight: 1
description: >
  DuckDB is an in-process SQL OLAP database management system designed for analytical query processing.
---

## About

[DuckDB](https://duckdb.org/) is an embedded analytical database management system that runs in-process with the client application. It is optimized for analytical workloads, providing high performance for complex queries with minimal setup.

DuckDB has the following notable characteristics:

- In-process, serverless database engine
- Supports complex SQL queries for analytical processing
- Can operate on in-memory or persistent storage
- Zero-configuration - no external dependencies or server setup required
- Highly optimized for columnar data storage and query execution

For more details, refer to the [DuckDB Documentation](https://duckdb.org/).

## Requirements

### Database File

To use DuckDB, you can either:

- Specify a file path for a persistent database stored on the filesystem
- Omit the file path to use an in-memory database

## Example

For a persistent DuckDB database:

```yaml
sources:
    my-duckdb:
        name: "my-duckdb"
        kind: "duckdb"
        dbFilePath: "/path/to/database.db"
        configurations:
            memory_limit: "2GB"
            threads: "4"
```

For an in-memory DuckDB database:

```yaml
sources:
    my-duckdb-memory:
        name: "my-duckdb-memory"
        kind: "duckdb"
```

## Reference

### Configuration Fields

| **field**         | **type**          | **required** | **description**                                                                 |
|-------------------|:-----------------:|:------------:|---------------------------------------------------------------------------------|
| name              | string            |     true     | A unique identifier for the DuckDB instance.                                     |
| kind              | string            |     true     | Must be "duckdb".                                                               |
| dbFilePath        | string            |    false     | Path to the DuckDB database file. Omit for an in-memory database.                |
| configurations    | map[string]string |    false     | Additional DuckDB configuration options (e.g., `memory_limit`, `threads`).       |

For a complete list of available configuration options, refer to the [DuckDB Configuration Documentation](https://duckdb.org/docs/stable/configuration/overview.html#local-configuration-options).


For more details on the Go implementation, see the [go-duckdb package documentation](https://pkg.go.dev/github.com/scottlepp/go-duckdb#section-readme).