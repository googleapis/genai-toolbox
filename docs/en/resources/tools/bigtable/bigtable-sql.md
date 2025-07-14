---
title: "bigtable-sql"
type: docs
weight: 1
description: > 
  A "bigtable-sql" tool executes a pre-defined SQL statement against a Google 
  Cloud Bigtable instance.
aliases:
- /resources/tools/bigtable-sql
---

## About

A `bigtable-sql` tool executes a pre-defined SQL statement against a Bigtable
instance. It's compatible with any of the following sources:

- [bigtable](../sources/bigtable.md)

### GoogleSQL

Bigtable supports SQL queries. The integration with Toolbox supports `googlesql`
dialect, the specified SQL statement is executed as a [data manipulation
language (DML)][bigtable-googlesql] statements, and specified parameters will inserted according to their name: e.g. `@name`.

{{<notice note>}}
  Bigtable's GoogleSQL support for DML statements might be limited to certain query types. For detailed information on supported DML statements and use cases, refer to the [Google Cloud Bigtable GoogleSQL overview](https://cloud.google.com/bigtable/docs/googlesql-overview#use-cases).
{{</notice>}}

[bigtable-googlesql]: https://cloud.google.com/bigtable/docs/googlesql-overview
[bigtable-googlesql#usecase]: https://cloud.google.com/bigtable/docs/googlesql-overview#use-cases

## Example

> **Note:** This tool uses parameterized queries to prevent SQL injections.
> Query parameters can be used as substitutes for arbitrary expressions.
> Parameters cannot be used as substitutes for identifiers, column names, table
> names, or other parts of the query.

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

### Example with Template Parameters

> **Note:** This tool allows direct modifications to the SQL statement,
> including identifiers, column names, and table names. **This makes it more
> vulnerable to SQL injections**. Using basic parameters only (see above) is
> recommended for performance and safety reasons. For more details, please check
> [templateParameters](_index#template-parameters).

```yaml
tools:
 list_table:
    kind: bigtable-sql
    source: my-bigtable-instance
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
| kind               |                   string                         |     true     | Must be "bigtable-sql".                                                                                                                    |
| source             |                   string                         |     true     | Name of the source the SQL should execute on.                                                                                              |
| description        |                   string                         |     true     | Description of the tool that is passed to the LLM.                                                                                         |
| statement          |                   string                         |     true     | SQL statement to execute on.                                                                                                               |
| parameters         | [parameters](_index#specifying-parameters)       |    false     | List of [parameters](_index#specifying-parameters) that will be inserted into the SQL statement.                                           |
| templateParameters | [templateParameters](_index#template-parameters) |    false     | List of [templateParameters](_index#template-parameters) that will be inserted into the SQL statement before executing prepared statement. |

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
