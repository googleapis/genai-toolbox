---
title: "mssql-execute-sql"
type: docs
weight: 1
description: > 
  A "mssql-execute-sql" tool executes a SQL statement against a SQL Server
  database.
---

## About

A `mssql-execute-sql` tool executes a SQL statement against a SQL Server
database. It's compatible with any of the following sources:

- [cloud-sql-mssql](../sources/cloud-sql-mssql.md)
- [mssql](../sources/mssql.md)

`mssql-execute-sql` takes one input parameter `sql` and run the sql
statement against the `source`.

## Example

```yaml
tools:
 execute_sql_tool:
    kind: mssql-execute-sql
    source: my-mssql-instance
    description: Use this tool to execute sql statement.
```
## Reference
| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be "mssql-execute-sql".                                                                          |
| source      |                   string                   |     true     | Name of the source the SQL should execute on.                                                    |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.