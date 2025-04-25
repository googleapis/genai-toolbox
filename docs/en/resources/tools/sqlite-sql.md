---
title: "sqlite-sql"
type: docs
weight: 1
description: >
  Execute SQL statements against a SQLite database.
---

## About

A `sqlite-sql` tool executes SQL statements against a SQLite database.
It's compatible with any of the following sources:

- [sqlite](../sources/sqlite.md)

SQLite uses the `?` placeholder for parameters in SQL statements. Parameters are
bound in the order they are provided.

The statement field supports any valid SQLite SQL statement, including `SELECT`, `INSERT`, `UPDATE`, `DELETE`, `CREATE/ALTER/DROP` table statements, and other DDL statements.

### Example

```yaml
tools:
  search-users:
    kind: sqlite-sql
    source: my-sqlite-db
    description: Search users by name and age
    parameters:
      - name: name
        type: string
        description: The name to search for
      - name: min_age
        type: integer
        description: Minimum age
    statement: SELECT * FROM users WHERE name LIKE ? AND age >= ?
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind | string | Yes | Must be "sqlite-sql" |
| source | string | Yes | Name of a SQLite source configuration |
| description | string | Yes | Description of what the tool does |
| parameters | array | No | List of parameters for the SQL statement |
| statement | string | Yes | The SQL statement to execute |
