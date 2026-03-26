---
title: "ClickHouse"
type: docs
description: "Details of the ClickHouse prebuilt configuration."
---

## ClickHouse

*   `--prebuilt` value: `clickhouse`
*   **Environment Variables:**
    *   `CLICKHOUSE_HOST`: The hostname or IP address of the ClickHouse server.
    *   `CLICKHOUSE_PORT`: The port number of the ClickHouse server.
    *   `CLICKHOUSE_USER`: The database username.
    *   `CLICKHOUSE_PASSWORD`: The password for the database user.
    *   `CLICKHOUSE_DATABASE`: The name of the database to connect to.
    *   `CLICKHOUSE_PROTOCOL`: The protocol to use (e.g., http).
*   **Tools:**
    *   `execute_sql`: Use this tool to execute SQL.
    *   `list_databases`: Use this tool to list all databases in ClickHouse.
    *   `list_tables`: Use this tool to list all tables in a specific ClickHouse database.
