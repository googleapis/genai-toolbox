---
title: "Spanner (GoogleSQL dialect)"
type: docs
description: "Details of the Spanner (GoogleSQL dialect) prebuilt configuration."
---

## Spanner (GoogleSQL dialect)

*   `--prebuilt` value: `spanner`
*   **Environment Variables:**
    *   `SPANNER_PROJECT`: The GCP project ID.
    *   `SPANNER_INSTANCE`: The Spanner instance ID.
    *   `SPANNER_DATABASE`: The Spanner database ID.
*   **Permissions:**
    *   **Cloud Spanner Database Reader** (`roles/spanner.databaseReader`) to
        execute DQL queries and list tables.
    *   **Cloud Spanner Database User** (`roles/spanner.databaseUser`) to
        execute DML queries.
*   **Tools:**
    *   `execute_sql`: Executes a DML SQL query.
    *   `execute_sql_dql`: Executes a DQL SQL query.
    *   `list_tables`: Lists tables in the database.
    *   `list_graphs`: Lists graphs in the database.
