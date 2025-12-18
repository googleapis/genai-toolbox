---
title: "ClickHouse"
type: docs
weight: 1
description: >
  ClickHouse is an open-source, OLTP database.

---

## About

[ClickHouse][clickhouse-docs] is a fast, open-source, column-oriented database

[clickhouse-docs]: https://clickhouse.com/docs

## Available Tools

- [`clickhouse-execute-sql`](../tools/clickhouse/clickhouse-execute-sql.md)  
  Execute parameterized SQL queries in ClickHouse with query logging.

- [`clickhouse-sql`](../tools/clickhouse/clickhouse-sql.md)  
  Execute SQL queries as prepared statements in ClickHouse.

## Requirements

### Database User

This source uses standard ClickHouse authentication. You will need to [create a
ClickHouse user][clickhouse-users] (or with [ClickHouse
Cloud][clickhouse-cloud]) to connect to the database with. The user should have
appropriate permissions for the operations you plan to perform.

[clickhouse-cloud]:
    https://clickhouse.com/docs/getting-started/quick-start/cloud#connect-with-your-app
[clickhouse-users]: https://clickhouse.com/docs/en/sql-reference/statements/create/user

### Network Access

ClickHouse supports multiple protocols:

- **HTTPS protocol** (default port 8443) - Secure HTTP access (default)
- **HTTP protocol** (default port 8123) - Good for web-based access

## Example

### Secure Connection Example

```yaml
sources:
    secure-clickhouse-source:
        kind: clickhouse
        host: clickhouse.example.com
        port: "8443"
        database: analytics
        user: ${CLICKHOUSE_USER}
        password: ${CLICKHOUSE_PASSWORD}
        protocol: https
        secure: true
```

### HTTP Protocol Example

```yaml
sources:
    http-clickhouse-source:
        kind: clickhouse
        host: localhost
        port: "8123"
        database: logs
        user: ${CLICKHOUSE_USER}
        password: ${CLICKHOUSE_PASSWORD}
        protocol: http
        secure: false
```

{{< notice tip >}}
Use environment variable replacement with the format ${ENV_NAME}
instead of hardcoding your secrets into the configuration file.
{{< /notice >}}

## Reference

| **field** | **type** | **required** | **description**                                                                     |
|-----------|:--------:|:------------:|-------------------------------------------------------------------------------------|
| kind      |  string  |     true     | Must be "clickhouse".                                                               |
| host      |  string  |     true     | IP address or hostname to connect to (e.g. "127.0.0.1" or "clickhouse.example.com") |
| port      |  string  |     true     | Port to connect to (e.g. "8443" for HTTPS, "8123" for HTTP)                         |
| database  |  string  |     true     | Name of the ClickHouse database to connect to (e.g. "my_database").                 |
| user      |  string  |     true     | Name of the ClickHouse user to connect as (e.g. "analytics_user").                  |
| password  |  string  |    false     | Password of the ClickHouse user (e.g. "my-password").                               |
| protocol  |  string  |    false     | Connection protocol: "https" (default) or "http".                                   |
| secure    | boolean  |    false     | Whether to use a secure connection (TLS). Default: false.                           |
