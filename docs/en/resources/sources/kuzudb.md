---
title: "KùzuDB"
type: docs
weight: 1
description: > 
    KùzuDB is an open-source, embedded graph database built for query speed and scalability, optimized for complex join-heavy analytical workloads using the Cypher query language.
---

## About

[KuzuDB](https://kuzudb.com/) is an embedded graph database designed for high query speed and scalability, optimized for complex, join-heavy analytical workloads on large graph datasets. It provides a lightweight, in-process integration with applications, making it ideal for scenarios requiring fast and efficient graph data processing.

KuzuDB has the following core features:

- **Property Graph Data Model and Cypher Query Language**: Supports the property graph model and uses Cypher, a powerful and expressive query language for graph databases.
- **Embedded Integration**: Runs in-process with applications, eliminating the need for a separate server.
- **Columnar Disk-Based Storage**: Utilizes columnar storage for efficient data access and management.
- **Columnar and Compressed Sparse Row-Based (CSR) Adjacency List and Join Indices**: Optimizes storage and query performance for large graphs.
- **Vectorized and Factorized Query Processing**: Enhances query execution speed through advanced processing techniques.
- **Novel and Efficient Join Algorithms**: Improves performance for complex join operations.
- **Multi-Core Query Parallelism**: Leverages multiple cores for faster query execution.
- **Serializable ACID Transactions**: Ensures data consistency and reliability with full ACID compliance.


## Available Tools

- [`kuzudb-cypher`](../tools/kuzudb/kuzudb-cypher.md)  
  Execute pre-defined Cypher queries with placeholder parameters.

## Requirements

### Database File

You need a directory to store the KuzuDB database files. This can be:

- An existing database file
- A path where a new database file should be created
- `:memory:` for an in-memory database

## Example

```yaml
sources:
  my-kuzu-db:
    kind: "kuzudb"
    database: "/path/to/database.db"
    bufferPoolSize: 1073741824  # 1GB
    maxNumThreads: 4
    enableCompression: true
    readOnly: false
    maxDbSize: 5368709120  # 5GB
```

For an in-memory database:

```yaml
sources:
  my-kuzu-memory-db:
    kind: "kuzudb"
    database: ":memory:"
    bufferPoolSize: 1073741824  # 1GB
    maxNumThreads: 4
    enableCompression: true
    readOnly: false
    maxDbSize: 5368709120  # 5GB
```

## Reference

### Configuration Fields

| **Field**          | **Type** | **Required** | **Description**                                                                 |
|--------------------|:--------:|:------------:|---------------------------------------------------------------------------------|
| kind               | string   | true         | Must be "kuzudb".                                                                 |
| database           | string   | false        | Path to the database directory. Default is ":memory:" which creates an in-memory database. |
| bufferPoolSize     | uint64   | false        | Size of the buffer pool in bytes (e.g., 1073741824 for 1GB).                    |
| maxNumThreads      | uint64   | false        | Maximum number of threads for query execution.                                   |
| enableCompression  | bool     | false        | Enables or disables data compression. Default is true.                           |
| readOnly           | bool     | false        | Sets the database to read-only mode if true. Default is false.                   |
| maxDbSize          | uint64   | false        | Maximum database size in bytes (e.g., 5368709120 for 5GB).                      |
