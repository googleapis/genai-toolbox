---
title: "SAP HANA"
type: docs
weight: 1
description: >
  SAP HANA is a high-performance in-memory database and application platform.

---

## About

[SAP HANA][hana-docs] is a high-performance in-memory database and application 
platform that enables businesses to process massive amounts of data in real time.
It supports both transactional and analytical workloads in a single system.

[hana-docs]: https://help.sap.com/docs/SAP_HANA_PLATFORM

## Available Tools

- [`hana-sql`](../tools/hana/hana-sql.md)  
  Execute parameterized SQL queries as prepared statements in SAP HANA.

- [`hana-execute-sql`](../tools/hana/hana-execute-sql.md)  
  Run ad-hoc SQL statements in SAP HANA.

### Pre-built Configurations

The HANA source includes pre-built tools for common database operations:
- `execute_sql` - Execute arbitrary SQL statements
- `list_tables` - List tables in a given schema

## Requirements

### Database User

This source uses standard SAP HANA authentication. You will need to [create a
HANA database user][hana-users] with appropriate permissions to access the
target database and schemas.

[hana-users]: https://help.sap.com/docs/SAP_HANA_PLATFORM/b3ee5778bc2e4a089d3299b82ec762a7/c511a2c3bb571014a8cbfacb5b5da03a.html

### Network Access

- **HANA Cloud**: Connections automatically use TLS encryption
- **On-premise HANA**: May require network configuration and proxy settings
- Use `HTTPS_PROXY` environment variable if corporate proxy is required

## Example

```yaml
sources:
    my-hana-source:
        kind: hana
        host: ${HANA_HOST}
        port: ${HANA_PORT}
        database: ${HANA_DATABASE}
        user: ${HANA_USER}
        password: ${HANA_PASSWORD}
        queryTimeout: 30s
```

{{< notice tip >}}
Use environment variable replacement with the format ${ENV_NAME}
instead of hardcoding your secrets into the configuration file.
{{< /notice >}}

{{< notice info >}}
The go-hdb driver automatically negotiates TLS when required (e.g. HANA Cloud).
Additional TLS parameters can be passed via the DSN if needed.
{{< /notice >}}

## Reference

|   **field**   |  **type** | **required** | **description**                                                        |
|---------------|:---------:|:------------:|------------------------------------------------------------------------|
| kind          |  string   |     true     | Must be "hana".                                                        |
| host          |  string   |     true     | Hostname or IP address to connect to (e.g. "hana.example.com")        |
| port          |  string   |     true     | Port to connect to (e.g. "39015" for tenant DB, "39013" for system)  |
| database      |  string   |     true     | Name of the HANA database/tenant to connect to (e.g. "HXE")          |
| user          |  string   |     true     | Name of the HANA user to connect as (e.g. "DBADMIN")                 |
| password      |  string   |     true     | Password of the HANA user (e.g. "MyPassword123")                     |
| queryTimeout  |  string   |     false    | Query timeout duration (e.g. "30s", "5m"). Maps to DSN timeout.     |

## Common Port Numbers

| **HANA Type**     | **Default Port** | **Description**                    |
|-------------------|:----------------:|------------------------------------|
| HANA Cloud        |       443        | HANA Cloud (uses HTTPS)           |
| Tenant Database   |      39015       | Multi-tenant container database    |
| System Database   |      39013       | System database (single-tenant)   |
| Express Edition   |      39017       | HANA Express Edition default      |
