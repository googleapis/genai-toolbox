---
title: "MySQL"
type: docs
description: "Details of the MySQL prebuilt configuration."
---

## MySQL

*   `--prebuilt` value: `mysql`
*   **Environment Variables:**
    *   `MYSQL_HOST`: The hostname or IP address of the MySQL server.
    *   `MYSQL_PORT`: The port number for the MySQL server.
    *   `MYSQL_DATABASE`: The name of the database to connect to.
    *   `MYSQL_USER`: The database username.
    *   `MYSQL_PASSWORD`: The password for the database user.
*   **Permissions:**
    *   Database-level permissions (e.g., `SELECT`, `INSERT`) are required to
        execute queries.
*   **Tools:**
    *   `execute_sql`: Executes a SQL query.
    *   `list_tables`: Lists tables in the database.
    *   `get_query_plan`: Provides information about how MySQL executes a SQL
        statement.
    *   `list_active_queries`: Lists ongoing queries.
    *   `list_tables_missing_unique_indexes`: Looks for tables that do not have
        primary or unique key contraint.
    *   `list_table_fragmentation`: Displays table fragmentation in MySQL.
    *   `list_table_stats`: Displays table statistics in MySQL.
