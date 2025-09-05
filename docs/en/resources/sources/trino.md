---
title: "Trino"
type: docs
weight: 1
description: >
  Trino is a distributed SQL query engine for big data analytics.
---

## About

[Trino][trino-docs] is a distributed SQL query engine designed for fast analytic queries against data of any size. It allows you to query data where it lives, including Hive, Cassandra, relational databases or even proprietary data stores.

[trino-docs]: https://trino.io/docs/

## Available Tools

- [`trino-sql`](../tools/trino/trino-sql.md)  
  Execute parameterized SQL queries against Trino.

- [`trino-execute-sql`](../tools/trino/trino-execute-sql.md)  
  Execute arbitrary SQL queries against Trino.

## Requirements

### Trino Cluster

You need access to a running Trino cluster with appropriate user permissions for the catalogs and schemas you want to query.

## Example

```yaml
sources:
    my-trino-source:
    kind: trino
    host: trino.example.com
    port: "8080"
    user: ${TRINO_USER}  # Optional for anonymous access
    password: ${TRINO_PASSWORD}  # Optional
    catalog: hive
    schema: default
```

{{< notice tip >}}
Use environment variable replacement with the format ${ENV_NAME}
instead of hardcoding your secrets into the configuration file.
{{< /notice >}}

## Reference

|  **field**  |      **type**      | **required** | **description**                                                        |
|-------------|:------------------:|:------------:|------------------------------------------------------------------------|
| type        |       string       |     true     | Must be "trino".                                                       |
| host        |       string       |     true     | Trino coordinator hostname (e.g. "trino.example.com")                 |
| port        |       string       |     true     | Trino coordinator port (e.g. "8080", "8443")                          |
| user        |       string       |     false    | Username for authentication (e.g. "analyst"). Optional for anonymous access. |
| password    |       string       |     false    | Password for basic authentication                                      |
| catalog     |       string       |     true     | Default catalog to use for queries (e.g. "hive")                      |
| schema      |       string       |     true     | Default schema to use for queries (e.g. "default")                    |
| queryTimeout|       string       |     false    | Query timeout duration (e.g. "30m", "1h")                             |
| accessToken |       string       |     false    | JWT access token for authentication                                    |
| kerberosEnabled | boolean        |     false    | Enable Kerberos authentication (default: false)                       |
| sslEnabled  |       boolean      |     false    | Enable SSL/TLS (default: false)                                       |
