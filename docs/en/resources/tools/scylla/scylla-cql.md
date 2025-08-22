---
title: "scylla-cql"
type: docs
weight: 1
description: >
  A "scylla-cql" tool executes a pre-defined CQL statement against a Scylla
  database.
aliases:
- /resources/tools/scylla-cql
---

## About

A `scylla-cql` tool executes a pre-defined CQL statement against a Scylla
database. It's compatible with any of the following sources:

- [scylla](../../sources/scylla.md)

The specified CQL statement is executed as a [prepared statement][cql-prepare],
and specified parameters will be inserted according to their position: e.g. the
first `?` will be replaced with the first parameter specified, the second `?` 
will be replaced with the second parameter, and so on. If template parameters 
are included, they will be resolved before execution of the prepared statement.

[cql-prepare]: https://docs.scylladb.com/stable/cql/prepared-statements.html

## Example

> **Note:** This tool uses parameterized queries to prevent CQL injections.
> Query parameters can be used as substitutes for values in WHERE clauses,
> INSERT statements, and UPDATE statements. Parameters cannot be used as 
> substitutes for identifiers, column names, table names, or other parts of 
> the query structure.

```yaml
tools:
 search_users_by_status:
    kind: scylla-cql
    source: my-scylla-instance
    statement: |
      SELECT user_id, name, email, status
      FROM users
      WHERE status = ?
      AND created_at >= ?
      LIMIT 10
    description: |
      Use this tool to get information for users with a specific status.
      Takes a status and creation date and returns user information.
      Do NOT use this tool with a user id. Do NOT guess a status or date.
      Status must be one of: active, inactive, suspended.
      Example:
      {{
          "status": "active",
          "created_date": "2024-01-01",
      }}
      Example:
      {{
          "status": "inactive",
          "created_date": "2024-01-15",
      }}
    parameters:
      - name: status
        type: string
        description: User status filter
      - name: created_date
        type: string
        description: Minimum creation date in YYYY-MM-DD format
```

### Example with Template Parameters

> **Note:** This tool allows direct modifications to the CQL statement,
> including identifiers, column names, and table names. **This makes it more
> vulnerable to CQL injections**. Using basic parameters only (see above) is
> recommended for performance and safety reasons. For more details, please check
> [templateParameters](..#template-parameters).

```yaml
tools:
 list_table:
    kind: scylla-cql
    source: my-scylla-instance
    statement: |
      SELECT * FROM {{.keyspace}}.{{.tableName}}
    description: |
      Use this tool to list all information from a specific table.
      Example:
      {{
          "keyspace": "mykeyspace",
          "tableName": "users",
      }}
    templateParameters:
      - name: keyspace
        type: string
        description: Keyspace containing the table
      - name: tableName
        type: string
        description: Table to select from
```

## Reference

| **field**           |                  **type**                                 | **required** | **description**                                                                                                                            |
|---------------------|:---------------------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------------------------------------------------|
| kind                |                   string                                  |     true     | Must be "scylla-cql".                                                                                                                      |
| source              |                   string                                  |     true     | Name of the source the CQL should execute on.                                                                                              |
| description         |                   string                                  |     true     | Description of the tool that is passed to the LLM.                                                                                         |
| statement           |                   string                                  |     true     | CQL statement to execute.                                                                                                                  |
| parameters          | [parameters](../#specifying-parameters)                |    false     | List of [parameters](../#specifying-parameters) that will be inserted into the CQL statement.                                           |
| templateParameters  |  [templateParameters](..#template-parameters)         |    false     | List of [templateParameters](..#template-parameters) that will be inserted into the CQL statement before executing prepared statement. |
