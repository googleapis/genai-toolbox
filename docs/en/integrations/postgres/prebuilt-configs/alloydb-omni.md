---
title: "AlloyDB Omni"
type: docs
description: "Details of the AlloyDB Omni prebuilt configuration."
---

## AlloyDB Omni

*   `--prebuilt` value: `alloydb-omni`
*   **Environment Variables:**
    *   `ALLOYDB_OMNI_HOST`: (Optional) The hostname or IP address (Default: localhost).
    *   `ALLOYDB_OMNI_PORT`: (Optional) The port number (Default: 5432).
    *   `ALLOYDB_OMNI_DATABASE`: The name of the database to connect to.
    *   `ALLOYDB_OMNI_USER`: The database username.
    *   `ALLOYDB_OMNI_PASSWORD`: (Optional) The password for the database user.
    *   `ALLOYDB_OMNI_QUERY_PARAMS`: (Optional) Connection query parameters.
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
    *   `list_columnar_configurations`: List AlloyDB Omni columnar-related configurations.
    *   `list_columnar_recommended_columns`: Lists columns that AlloyDB Omni recommends adding to the columnar engine.
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
        each database in the AlloyDB instance.
    *   `list_roles`: Lists all the user-created roles in PostgreSQL database.
    *   `list_stored_procedure`: Lists stored procedures.
