---
title: "SQLite"
type: docs
description: "Details of the SQLite prebuilt configuration."
---

## SQLite

*   `--prebuilt` value: `sqlite`
*   **Environment Variables:**
    *   `SQLITE_DATABASE`: The path to the SQLite database file (e.g.,
        `./sample.db`).
*   **Permissions:**
    *   File system read/write permissions for the specified database file.
*   **Tools:**
    *   `execute_sql`: Executes a SQL query.
    *   `list_tables`: Lists tables in the database.
