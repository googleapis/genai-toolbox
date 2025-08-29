---
title: "cloud-sql-mysql-list-tables"
type: docs
weight: 1
description: >
  The "cloud-sql-mysql-list-tables" tool lists schema information for all or specified tables in a CloudSQL for MySQL database.
aliases:
- /resources/tools/cloud-sql-mysql-list-tables
---

## About

The `cloud-sql-mysql-list-tables` tool retrieves schema information for all or specified tables in a CloudSQL for MySQL database.
It is compatible with [cloud-sql-mysql](../../sources/cloud-sql-mysql.md) source.

`cloud-sql-mysql-list-tables` lists detailed schema information (object type, columns, constraints, indexes, triggers, owner, comment) as JSON for user-created tables (ordinary or partitioned). Filters by a comma-separated list of names. If names are omitted, it lists all tables in user schemas. The output format can be set to `simple` which will return only the table names or `detailed` which is the default.

## Example

```yaml
tools:
  cloud_sql_mysql_list_tables:
    kind: cloud-sql-mysql-list-tables
    source: cloud-sql-mysql-source
    description: Use this tool to retrieve schema information for all or specified tables. Output format can be simple (only table names) or detailed.
```

## Reference

| **field**   |                  **type**                  | **required** | **description**                                                                                  |
|-------------|:------------------------------------------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |                   string                   |     true     | Must be "cloud-sql-mysql-list-tables".                                                               |
| source      |                   string                   |     true     | Name of the source the SQL should execute on.                                                    |
| description |                   string                   |     true     | Description of the tool that is passed to the LLM.                                               |