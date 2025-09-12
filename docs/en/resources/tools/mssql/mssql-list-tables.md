---
title: "mssql-list-tables"
type: docs
weight: 1
description: >
  The "mssql-list-tables" tool lists schema information for all or specified tables in a SQL server database.
aliases:
- /resources/tools/mssql-list-tables
---

## About

The `mssql-list-tables` tool retrieves schema information for all or specified tables in a SQL server database. It is compatible with any of the following sources:

- [cloud-sql-mssql](../../sources/cloud-sql-mssql.md)
- [mssql](../../sources/mssql.md)

`mssql-list-tables` lists detailed schema information (object type, columns, constraints, indexes, triggers, owner, comment) as JSON for user-created tables (ordinary or partitioned). Filters by a comma-separated list of names. If names are omitted, it lists all tables in user schemas. The output format can be set to `simple` which will return only the table names or `detailed` which is the default.

The tool takes the following input parameters:

| Parameter  | Type   | Description                                                                              | Required |
| :--------- | :----- | :--------------------------------------------------------------------------------------- | :------- |
| `table_names`  | string | Filters by a comma-separated list of names. By default, it lists all tables in user schemas. Default: `""`                                                 | No      |
| `output_format` | string | Indicate the output format of table schema. `simple` will return only the table names, `detailed` will return the full table information. Default: `detailed`. | No       |

## Example

```yaml
tools:
  mssql_list_tables:
    kind: mssql-list-tables
    source: mssql-source
    description: Use this tool to retrieve schema information for all or specified tables. Output format can be simple (only table names) or detailed.
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be "mssql-list-tables".                                                          |
| source      |                   string                   |     true     | Name of the source the SQL should execute on.                                                    |
| description |                   string                   |     true     | Description of the tool that is passed to the agent.                                             |
