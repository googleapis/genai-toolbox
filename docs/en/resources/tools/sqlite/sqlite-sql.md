---
title: "sqlite-sql"
type: docs
weight: 1
description: >
  Execute SQL statements against a SQLite database.
aliases:
- /resources/tools/sqlite-sql
---

## About

A `sqlite-sql` tool executes SQL statements against a SQLite database.
It's compatible with any of the following sources:

- [sqlite](../../sources/sqlite.md)

SQLite uses the `?` placeholder for parameters in SQL statements. Parameters are
bound in the order they are provided.

The statement field supports any valid SQLite SQL statement, including `SELECT`,
`INSERT`, `UPDATE`, `DELETE`, `CREATE/ALTER/DROP` table statements, and other
DDL statements.

### Example

> **Note:** This tool uses parameterized queries to prevent SQL injections.
> Query parameters can be used as substitutes for arbitrary expressions.
> Parameters cannot be used as substitutes for identifiers, column names, table
> names, or other parts of the query.

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

### Example with Template Parameters

> **Note:** This tool allows direct modifications to the SQL statement,
> including identifiers, column names, and table names. **This makes it more
> vulnerable to SQL injections**. Using basic parameters only (see above) is
> recommended for performance and safety reasons. For more details, please check
> [templateParameters](_index.md#template-parameters).

```yaml
tools:
 list_table:
    kind: sqlite-sql
    source: my-sqlite-db
    statement: |
      SELECT * FROM {{.tableName}};
    description: |
      Use this tool to list all information from a specific table.
      Example:
      {{
          "tableName": "flights",
      }}
    templateParameters:
      - name: tableName
        type: string
        description: Table to select from
```

## Reference

| **field**          |                  **type**                        | **required** | **description**                                                                                                                            |
|--------------------|:------------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------------------------------------------------|
| kind               |                   string                         |     true     | Must be "sqlite-sql".                                                                                                                      |
| source             |                   string                         |     true     | Name of the source the SQLite source configuration.                                                                                        |
| description        |                   string                         |     true     | Description of the tool that is passed to the LLM.                                                                                         |
| statement          |                   string                         |     true     | The SQL statement to execute.                                                                                                              |
| parameters         | [parameters](../_index.md#specifying-parameters)       |    false     | List of [parameters](../_index.md#specifying-parameters) that will be inserted into the SQL statement.                                           |
| templateParameters | [templateParameters](_index.md#template-parameters) |    false     | List of [templateParameters](_index.md#template-parameters) that will be inserted into the SQL statement before executing prepared statement. |
