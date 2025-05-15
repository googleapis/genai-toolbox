---
title: "bigtable-sql"
type: docs
weight: 1
description: > 
  A "bigtable-sql" tool executes a pre-defined SQL statement against a Google 
  Cloud Bigtable instance.
---

## About

A `bigtable-sql` tool executes a pre-defined SQL statement against a Bigtable
instance. It's compatible with any of the following sources:

- [bigtable](../sources/bigtable.md)

### GoogleSQL

Bigtable supports SQL queries. The integration with Toolbox supports `googlesql`
dialect, the specified SQL statement is executed as a [data manipulation
language (DML)][bigtable-googlesql] statements, and specified parameters will
inserted according to their name: e.g. `@name`.

[bigtable-googlesql]: https://cloud.google.com/bigtable/docs/googlesql-overview

## Example

```yaml
tools:
 search_user_by_id_or_name:
    kind: bigtable-sql
    source: my-bigtable-instance
    statement: |
      SELECT 
        TO_INT64(cf[ 'id' ]) as id, 
        CAST(cf[ 'name' ] AS string) as name, 
      FROM 
        mytable 
      WHERE 
        TO_INT64(cf[ 'id' ]) = @id 
        OR CAST(cf[ 'name' ] AS string) = @name;
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
      - name: name
        type: string
        description: Name of the user
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be "bigtable-sql".                                                                          |
| source      |                   string                   |     true     | Name of the source the SQL should execute on.                                                    |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                               |
| statement   |                   string                   |     true     | SQL statement to execute on.                                                                     |
| parameters  | [parameters](_index#specifying-parameters) |    false     | List of [parameters](_index#specifying-parameters) that will be inserted into the SQL statement. |

## Tips

- [Bigtable Studio][bigtable-studio] is a useful to explore and manage your
  Bigtable data. If you're unfamiliar with the query syntax, [Query
  Builder][bigtable-querybuilder] lets you build a query, run it against a
  table, and then view the results in the console.
- Some Python libraries limit the use of underscore columns such as `_key`. A
  workaround would be to leverage Bigtable [Logical
  Views][bigtable-logical-view] to rename the columns.

[bigtable-studio]: https://cloud.google.com/bigtable/docs/manage-data-using-console
[bigtable-logical-view]: https://cloud.google.com/bigtable/docs/create-manage-logical-views
[bigtable-querybuilder]: https://cloud.google.com/bigtable/docs/query-builder
