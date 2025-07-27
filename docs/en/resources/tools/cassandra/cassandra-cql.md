---
title: "cassandra-cql"
type: docs
weight: 1
description: > 
  A "cassandra-cql" tool executes a pre-defined CQL statement against a Cassandra
  database.
aliases:
- /resources/tools/cassandra-cql
---

## About

A `cassandra-cql` tool executes a pre-defined CQL statement against a Cassandra
database. It's compatible with any of the following sources:

- [cassandra](../sources/cassandra.md)

The specified CQL statement is executed as a [prepared statement][cassandra-prepare],
and expects parameters in the CQL query to be in the form of placeholders `?`.

[cassandra-prepare]: https://docs.datastax.com/en/developer/go-driver/4.8/cql-prepared-statements/

## Example

> **Note:** This tool uses parameterized queries to prevent CQL injections.
> Query parameters can be used as substitutes for arbitrary expressions.
> Parameters cannot be used as substitutes for keyspaces, table names, column names,
> or other parts of the query.

```yaml
tools:
  search_users_by_email:
    kind: cassandra-cql
    source: my-cassandra-cluster
    statement: |
      SELECT user_id, email, first_name, last_name, created_at 
      FROM users 
      WHERE email = ?
    description: |
      Use this tool to retrieve specific user information by their email address.
      Takes an email address and returns user details including user ID, email, 
      first name, last name, and account creation timestamp.
      Do NOT use this tool with a user ID or other identifiers.
      Example:
      {{
          "email": "user@example.com",
      }}
    parameters:
      - name: email
        type: string
        description: User's email address
```

### Example with Template Parameters

> **Note:** This tool allows direct modifications to the CQL statement,
> including keyspaces, table names, and column names. **This makes it more
> vulnerable to CQL injections**. Using basic parameters only (see above) is
> recommended for performance and safety reasons. For more details, please check
> [templateParameters](_index#template-parameters).

```yaml
tools:
  list_keyspace_table:
    kind: cassandra-cql
    source: my-cassandra-cluster
    statement: |
      SELECT * FROM {{.keyspace}}.{{.tableName}};
    description: |
      Use this tool to list all information from a specific table in a keyspace.
      Example:
      {{
          "keyspace": "my_keyspace",
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

| **field**          |                  **type**                        | **required** | **description**                                                                                                                            |
|--------------------|:------------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------------------------------------------------|
| kind               |                   string                         |     true     | Must be "cassandra-cql".                                                                                                                   |
| source             |                   string                         |     true     | Name of the source the CQL should execute on.                                                                                              |
| description        |                   string                         |     true     | Description of the tool that is passed to the LLM.                                                                                         |
| statement          |                   string                         |     true     | CQL statement to execute.                                                                                                                  |
| authRequired       |                []string                         |    false     | List of authentication requirements for the source.                                                                                        |
| parameters         | [parameters](_index#specifying-parameters)       |    false     | List of [parameters](_index#specifying-parameters) that will be inserted into the CQL statement.                                           |
| templateParameters | [templateParameters](_index#template-parameters) |    false     | List of [templateParameters](_index#template-parameters) that will be inserted into the CQL statement before executing prepared statement. |
