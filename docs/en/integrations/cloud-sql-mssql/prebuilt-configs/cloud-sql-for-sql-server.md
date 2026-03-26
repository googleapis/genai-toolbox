---
title: "Cloud SQL for SQL Server"
type: docs
description: "Details of the Cloud SQL for SQL Server prebuilt configuration."
---

## Cloud SQL for SQL Server

*   `--prebuilt` value: `cloud-sql-mssql`
*   **Environment Variables:**
    *   `CLOUD_SQL_MSSQL_PROJECT`: The GCP project ID.
    *   `CLOUD_SQL_MSSQL_REGION`: The region of your Cloud SQL instance.
    *   `CLOUD_SQL_MSSQL_INSTANCE`: The ID of your Cloud SQL instance.
    *   `CLOUD_SQL_MSSQL_DATABASE`: The name of the database to connect to.
    *   `CLOUD_SQL_MSSQL_USER`: The database username.
    *   `CLOUD_SQL_MSSQL_PASSWORD`: The password for the database user.
    *   `CLOUD_SQL_MSSQL_IP_TYPE`: (Optional) The IP type i.e. "Public" or
        "Private" (Default: Public).
*   **Permissions:**
    *   **Cloud SQL Client** (`roles/cloudsql.client`) to connect to the
        instance.
    *   Database-level permissions (e.g., `SELECT`, `INSERT`) are required to
        execute queries.
*   **Tools:**
    *   `execute_sql`: Executes a SQL query.
    *   `list_tables`: Lists tables in the database.
