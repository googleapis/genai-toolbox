---
title: "duckdb-sql"
type: docs
weight: 1
description: >
  Execute SQL statements against a DuckDB database using the DuckDB SQL tools configuration.
aliases:
- /resources/tools/duckdb-sql
---

## About

A `duckdb-sql` tool executes a pre-defined SQL statement against a [DuckDB](https://duckdb.org/) database. It is compatible with any DuckDB source configuration as defined in the [DuckDB source documentation](../../sources/duckdb.md).

The specified SQL statement is executed as a prepared statement, and parameters are inserted according to their position: e.g., `$1` is the first parameter, `$2` is the second, and so on. If template parameters are included, they are resolved before execution of the prepared statement. 

DuckDB's SQL dialect closely follows the conventions of the PostgreSQL dialect, with a few exceptions listed in the [DuckDB PostgreSQL Compatibility documentation](https://duckdb.org/docs/stable/sql/dialect/postgresql_compatibility.html). For an introduction to DuckDB's SQL dialect, refer to the [DuckDB SQL Introduction](https://duckdb.org/docs/stable/sql/introduction).

### Concepts

DuckDB is a relational database management system (RDBMS). Data is stored in relations (tables), where each table is a named collection of rows. Each row in a table has the same set of named columns, each with a specific data type. Tables are stored within schemas, and a collection of schemas constitutes the entire database.

For more details, see the [DuckDB SQL Introduction](https://duckdb.org/docs/stable/sql/introduction).

## Example

> **Note:** This tool uses parameterized queries to prevent SQL injections. Query parameters can be used as substitutes for arbitrary expressions but cannot be used for identifiers, column names, table names, or other parts of the query.

```yaml
tools:
  search-users:
    kind: duckdb-sql
    source: my-duckdb
    description: Search users by name and age
    statement: SELECT * FROM users WHERE name LIKE $1 AND age >= $2
    parameters:
      - name: name
        type: string
        description: The name to search for
      - name: min_age
        type: integer
        description: Minimum age
```

## Example with Template Parameters

> **Note:** Template parameters allow direct modifications to the SQL statement, including identifiers, column names, and table names, which makes them more vulnerable to SQL injections. Using basic parameters (see above) is recommended for performance and safety. For more details, see the [templateParameters](#template-parameters) section.

```yaml
tools:
  list_table:
    kind: duckdb-sql
    source: my-duckdb
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

### Configuration Fields

| **field**          | **type**                        | **required** | **description**                                                                                                                            |
|--------------------|:-------------------------------:|:------------:|--------------------------------------------------------------------------------------------------------------------------------------------|
| kind               | string                         |     true     | Must be "duckdb-sql".                                                                                                                      |
| source             | string                         |     true     | Name of the DuckDB source configuration (see [DuckDB source documentation](../../sources/duckdb.md)).                                         |
| description        | string                         |     true     | Description of the tool that is passed to the LLM.                                                                                         |
| statement          | string                         |     true     | The SQL statement to execute.                                                                                                              |
| authRequired       | []string                       |    false     | List of authentication requirements for the tool (if any).                                                                                 |
| parameters         | [parameters](../#specifying-parameters)       |    false     | List of parameters that will be inserted into the SQL statement                      |
| templateParameters | [templateParameters](#template-parameters) |    false     | List of template parameters that will be inserted into the SQL statement before executing the prepared statement.                           |
