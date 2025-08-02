---
title: "Connect from your IDE"
type: docs
weight: 1
description: >
  List of guides detailing how to connect your AI tools (IDEs) to Toolbox using MCP.
aliases:
- /how-to/connect_tools_using_mcp
---

## `--prebuilt` Flag

The `--prebuilt` flag allows you to use predefined tool configurations for common database types without creating a custom `tools.yaml` file.

### Usage

```bash
./toolbox --prebuilt <source-type> [other-flags]
```

### Supported Source Types

The following prebuilt configurations are available:

- `alloydb-postgres` - AlloyDB PostgreSQL with execute_sql and list_tables tools
- `bigquery` - BigQuery with execute_sql, get_dataset_info, get_table_info, list_dataset_ids, and list_table_ids tools
- `cloud-sql-mysql` - Cloud SQL MySQL with execute_sql and list_tables tools
- `cloud-sql-postgres` - Cloud SQL PostgreSQL with execute_sql and list_tables tools
- `cloud-sql-mssql` - Cloud SQL SQL Server with execute_sql and list_tables tools
- `postgres` - PostgreSQL with execute_sql and list_tables tools
- `spanner` - Spanner (GoogleSQL) with execute_sql, execute_sql_dql, and list_tables tools
- `spanner-postgres` - Spanner (PostgreSQL) with execute_sql, execute_sql_dql, and list_tables tools

### Examples

#### PostgreSQL with STDIO transport
```bash
./toolbox --prebuilt postgres --stdio
```

This is commonly used in MCP client configurations:

#### BigQuery remote HTTP transport
```bash
./toolbox --prebuilt bigquery [--port 8080]
```

### Environment Variables

When using `--prebuilt`, you still need to provide database connection details through environment variables. The specific variables depend on the source type, see the documentation per database below for the complete list:

For PostgreSQL-based sources:
- `POSTGRES_HOST`
- `POSTGRES_PORT`
- `POSTGRES_DATABASE`
- `POSTGRES_USER`
- `POSTGRES_PASSWORD`


## Notes

The `--prebuilt` flag was added in version 0.6.0.