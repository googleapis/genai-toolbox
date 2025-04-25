---
title: "bigquery-sql"
type: docs
weight: 1
description: >
  A "bigquery-sql" tool executes a pre-defined SQL statement.
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

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be "bigquery-sql".                                                                          |
| source      |                   string                   |     true     | Name of the source the GoogleSQL should execute on.                                              |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                               |
| statement   |                   string                   |     true     | The GoogleSQL statement to execute.                                                              |
| parameters  | [parameters](_index#specifying-parameters) |    false     | List of [parameters](_index#specifying-parameters) that will be inserted into the SQL statement. |
