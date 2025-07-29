---
title: "bigquery-sql"
type: docs
weight: 1
description: >
  A "bigquery-sql" tool executes a pre-defined SQL statement.
aliases:
- /resources/tools/bigquery-sql
---

## About

A `bigquery-sql` tool executes a pre-defined SQL statement. It's compatible with
the following sources:

- [bigquery](../sources/bigquery.md)

### GoogleSQL

BigQuery uses [GoogleSQL][bigquery-googlesql] for querying data. The integration
with Toolbox supports this dialect. The specified SQL statement is executed, and
parameters can be inserted into the query. BigQuery supports both named parameters
(e.g., `@name`) and positional parameters (`?`), but they cannot be mixed in the
same query.

[bigquery-googlesql]: https://cloud.google.com/bigquery/docs/reference/standard-sql/

## Example

> **Note:** This tool uses [parameterized
> queries](https://cloud.google.com/bigquery/docs/parameterized-queries) to
> prevent SQL injections. Query parameters can be used as substitutes for
> arbitrary expressions. Parameters cannot be used as substitutes for
> identifiers, column names, table names, or other parts of the query.

```yaml
tools:
  # Example: Querying a user table in BigQuery
  search_users_bq:
    kind: bigquery-sql
    source: my-bigquery-source
    statement: |
      SELECT
        id,
        name,
        email
      FROM
        `my-project.my-dataset.users`
      WHERE
        id = @id OR email = @email;
    description: |
      Use this tool to get information for a specific user.
      Takes an id number or a name and returns info on the user.

      Example:
      {{
          "id": 123,
          "name": "Alice",
      }}
    parameters:
      - name: id
        type: integer
        description: User ID
      - name: email
        type: string
        description: Email address of the user
```

### Example with Template Parameters

> **Note:** This tool allows direct modifications to the SQL statement,
> including identifiers, column names, and table names. **This makes it more
> vulnerable to SQL injections**. Using basic parameters only (see above) is
> recommended for performance and safety reasons. For more details, please check
> [templateParameters](_index#template-parameters).

```yaml
tools:
 list_table:
    kind: bigquery-sql
    source: my-bigquery-source
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
| kind               |                   string                         |     true     | Must be "bigquery-sql".                                                                                                                    |
| source             |                   string                         |     true     | Name of the source the GoogleSQL should execute on.                                                                                        |
| description        |                   string                         |     true     | Description of the tool that is passed to the LLM.                                                                                         |
| statement          |                   string                         |     true     | The GoogleSQL statement to execute.                                                                                                        |
| parameters         | [parameters](_index#specifying-parameters)       |    false     | List of [parameters](_index#specifying-parameters) that will be inserted into the SQL statement.                                           |
| templateParameters | [templateParameters](_index#template-parameters) |    false     | List of [templateParameters](_index#template-parameters) that will be inserted into the SQL statement before executing prepared statement. |
