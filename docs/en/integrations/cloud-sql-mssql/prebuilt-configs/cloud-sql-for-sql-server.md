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

*   **Toolsets:**
    *   `admin`: Tools for administering instances, databases, clusters, and users.
        *   **Tools:** `create_instance`, `get_instance`, `list_instances`, `create_database`, `list_databases`, `create_user`, `wait_for_operation`
    *   `data`: Tools for executing queries, listing tables, views, schemas, and interacting with core data.
        *   **Tools:** `execute_sql`, `list_tables`
    *   `monitor`: Tools for checking system metrics, tracking active queries, identifying locks, and monitoring performance.
        *   **Tools:** `get_system_metrics`
    *   `lifecycle`: Tools for managing the lifecycle of instances, including backups, restores, and upgrades.
        *   **Tools:** `create_backup`, `restore_backup`, `clone_instance`, `list_instances`, `get_instance`, `wait_for_operation`
