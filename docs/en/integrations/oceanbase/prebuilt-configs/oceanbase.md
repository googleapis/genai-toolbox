---
title: "OceanBase"
type: docs
description: "Details of the OceanBase prebuilt configuration."
---

## OceanBase

*   `--prebuilt` value: `oceanbase`
*   **Environment Variables:**
    *   `OCEANBASE_HOST`: The hostname or IP address of the OceanBase server.
    *   `OCEANBASE_PORT`: The port number for the OceanBase server.
    *   `OCEANBASE_DATABASE`: The name of the database to connect to.
    *   `OCEANBASE_USER`: The database username.
    *   `OCEANBASE_PASSWORD`: The password for the database user.
*   **Permissions:**
    *   Database-level permissions (e.g., `SELECT`, `INSERT`) are required to
        execute queries.
*   **Tools:**
    *   `execute_sql`: Executes a SQL query.
    *   `list_tables`: Lists tables in the database.
