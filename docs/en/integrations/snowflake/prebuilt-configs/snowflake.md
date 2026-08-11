---
title: "Snowflake"
type: docs
description: "Details of the Snowflake prebuilt configuration."
---

## Snowflake

*   `--prebuilt` value: `snowflake`
*   **Environment Variables:**
    *   `SNOWFLAKE_ACCOUNT`: The Snowflake account.
    *   `SNOWFLAKE_USER`: The database username.
    *   `SNOWFLAKE_PASSWORD`: The password for the database user.
    *   `SNOWFLAKE_DATABASE`: The name of the database to connect to.
    *   `SNOWFLAKE_SCHEMA`: The schema name.
    *   `SNOWFLAKE_WAREHOUSE`: The warehouse name.
    *   `SNOWFLAKE_ROLE`: The role name.
*   **Tools:**
    *   `execute_sql`: Use this tool to execute SQL.
    *   `list_tables`: Lists detailed schema information for user-created tables.

*   **Toolsets:**
    *   `snowflake_tools`: Tools for executing queries and listing tables in Snowflake.
        *   **Tools:** `execute_sql`, `list_tables`
