---
title: "PostgreSQL"
type: docs
description: "Details of the PostgreSQL prebuilt configuration."
---

## PostgreSQL

*   `--prebuilt` value: `postgres`
*   **Environment Variables:**
    *   `POSTGRES_HOST`: (Optional) The hostname or IP address of the PostgreSQL server.
    *   `POSTGRES_PORT`: (Optional) The port number for the PostgreSQL server.
    *   `POSTGRES_DATABASE`: The name of the database to connect to.
    *   `POSTGRES_USER`: The database username.
    *   `POSTGRES_PASSWORD`: The password for the database user.
    *   `POSTGRES_QUERY_PARAMS`: (Optional) Raw query to be added to the db
        connection string.
*   **Permissions:**
    *   Database-level permissions (e.g., `SELECT`, `INSERT`) are required to
        execute queries.
*   **Tools:**
    *   `execute_sql`: Executes a SQL query.
    *   `list_tables`: Lists tables in the database.
    *   `list_active_queries`: Lists ongoing queries.
    *   `list_available_extensions`: Discover all PostgreSQL extensions available for installation.
    *   `list_installed_extensions`: List all installed PostgreSQL extensions.
    *   `long_running_transactions`: Identifies and lists database transactions that exceed a specified time limit.
    *   `list_locks`: Identifies all locks held by active processes.
    *   `replication_stats`: Lists each replica's process ID and sync state.
    *   `list_autovacuum_configurations`: Lists autovacuum configurations in the
        database.
    *   `list_memory_configurations`: Lists memory-related configurations in the
        database.
    *   `list_top_bloated_tables`: List top bloated tables in the database.
    *   `list_replication_slots`: Lists replication slots in the database.
    *   `list_invalid_indexes`: Lists invalid indexes in the database.
    *   `get_query_plan`: Generate the execution plan of a statement.
    *   `list_views`: Lists views in the database from pg_views with a default
        limit of 50 rows. Returns schemaname, viewname and the ownername.
    *   `list_schemas`: Lists schemas in the database.
    *   `database_overview`: Fetches the current state of the PostgreSQL server.
    *   `list_triggers`: Lists triggers in the database.
    *   `list_indexes`: List available user indexes in a PostgreSQL database.
    *   `list_sequences`: List sequences in a PostgreSQL database.
    *   `list_query_stats`: Lists query statistics.
    *   `get_column_cardinality`: Gets column cardinality.
    *   `list_table_stats`: Lists table statistics.
    *   `list_publication_tables`: List publication tables in a PostgreSQL database.
    *   `list_tablespaces`: Lists tablespaces in the database.
    *   `list_pg_settings`: List configuration parameters for the PostgreSQL server.
    *   `list_database_stats`: Lists the key performance and activity statistics for
        each database in the PostgreSQL server.
    *   `list_roles`: Lists all the user-created roles in PostgreSQL database.
    *   `list_stored_procedure`: Lists stored procedures.

*   **Toolsets:**
    *   `data`: Tools for executing queries, listing tables, views, schemas, and interacting with core data.
        *   **Tools:** `execute_sql`, `list_tables`, `list_views`, `list_schemas`, `list_triggers`, `list_indexes`, `list_sequences`, `list_stored_procedure`
    *   `monitor`: Tools for checking system metrics, tracking active queries, identifying locks, and monitoring performance.
        *   **Tools:** `list_query_stats`, `get_query_plan`, `list_database_stats`, `list_active_queries`, `long_running_transactions`, `list_locks`
    *   `health`: Tools for auditing database health, identifying bloat, vacuum configurations, and analyzing tables/indexes.
        *   **Tools:** `list_top_bloated_tables`, `list_invalid_indexes`, `list_table_stats`, `get_column_cardinality`, `list_autovacuum_configurations`, `list_tablespaces`, `database_overview`, `list_pg_settings`
    *   `view-config`: Tools for viewing and managing system-level configurations and parameters.
        *   **Tools:** `list_available_extensions`, `list_installed_extensions`, `list_memory_configurations`, `list_pg_settings`, `database_overview`
    *   `replication`: Tools for monitoring replication health, replication slots, publication tables, and cluster synchronization.
        *   **Tools:** `replication_stats`, `list_replication_slots`, `list_publication_tables`, `list_roles`, `list_pg_settings`, `database_overview`
