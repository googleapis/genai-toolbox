---
title: "Oracle"
type: docs
description: "Details of the Oracle prebuilt configuration."
---

## Oracle

*   `--prebuilt` value: `oracledb`
*   **Environment Variables:**
   
    *   `ORACLE_CONNECTION_STRING`: The connection string for the Oracle server (e.g., "hostname:port/servicename").
    *   `ORACLE_USERNAME`: The database username.
    *   `ORACLE_PASSWORD`: The password for the database user.
    *   `ORACLE_WALLET`: The path to the Oracle DB Wallet file for databases that support this authentication type.
    *   `ORACLE_USE_OCI`: A boolean flag (`true` or `false`) indicating whether to use the OCI-based driver. Setting to `true` is required for features like Oracle Wallet and requires the Oracle Instant Client libraries to be installed.
*   **Permissions:**
    *   Database-level permissions (e.g., `SELECT`, `INSERT`) are required to execute queries.
    *   For queries on DBA views like `dba_data_files` and `dba_free_space`, access typically requires elevated database privileges (like `SELECT_CATALOG_ROLE` or direct grants) that a standard user may not have.
*   **Tools:**
    *   `execute_sql`: Executes a SQL query.
    *   `list_tables`: Lists tables in the database.
    *   `list_active_sessions`: Lists active database sessions.
    *   `get_query_plan`: Generate a full execution plan for a single SQL statement.
    *   `list_top_sql_by_resource`: Lists top SQL statements by resource usage.
    *   `list_tablespace_usage`: Lists tablespace usage.
    *   `list_invalid_objects`: Lists invalid objects.
