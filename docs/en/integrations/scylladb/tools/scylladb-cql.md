---
title: "scylladb-cql"
type: docs
weight: 1
description: >
  A "scylladb-cql" tool executes a pre-defined CQL statement against a ScyllaDB
  cluster.
---

## About

A `scylladb-cql` tool executes a pre-defined CQL statement against a ScyllaDB
cluster.

The specified CQL statement is executed as a prepared
statement, and expects parameters in the CQL query to be in
the form of placeholders `?`.

## Compatible Sources

{{< compatible-sources >}}

## Example

> **Note:** This tool uses parameterized queries to prevent CQL injections.
> Query parameters can be used as substitutes for arbitrary expressions.
> Parameters cannot be used as substitutes for keyspaces, table names, column
> names, or other parts of the query.

```yaml
kind: tool
name: search_users_by_email
type: scylladb-cql
source: my-scylladb-cluster
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
> [templateParameters](../../../documentation/configuration/tools/_index.md#template-parameters).

```yaml
kind: tool
name: list_keyspace_table
type: scylladb-cql
source: my-scylladb-cluster
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

| **field**          |                   **type**                    | **required** | **description**                                                                                                                         |
|--------------------|:---------------------------------------------:|:------------:|-----------------------------------------------------------------------------------------------------------------------------------------|
| type               |                    string                     |     true     | Must be "scylladb-cql".                                                                                                                 |
| source             |                    string                     |     true     | Name of the source the CQL should execute on.                                                                                           |
| description        |                    string                     |     true     | Description of the tool that is passed to the LLM.                                                                                      |
| statement          |                    string                     |     true     | CQL statement to execute.                                                                                                               |
| authRequired       |                   []string                    |    false     | List of authentication requirements for the source.                                                                                     |
| parameters         |    [parameters](../../../documentation/configuration/tools/_index.md#specifying-parameters)    |    false     | List of [parameters](../../../documentation/configuration/tools/_index.md#specifying-parameters) that will be inserted into the CQL statement.                                           |
| templateParameters | [templateParameters](../../../documentation/configuration/tools/_index.md#template-parameters) |    false     | List of [templateParameters](../../../documentation/configuration/tools/_index.md#template-parameters) that will be inserted into the CQL statement before executing prepared statement. |
