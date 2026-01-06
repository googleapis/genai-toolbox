---
title: "KDB+"
type: docs
weight: 1
description: >
  KDB+ is a high-performance, column-oriented relational time-series database.

---

## About

[KDB+][kdb-docs] is a high-performance, column-oriented relational time-series database with over 20 years of active development that has earned it a strong
reputation for reliability, feature robustness, and performance, especially in the financial services industry.

[kdb-docs]: https://kx.com/

## Available Tools

- [`kdb-sql`](../tools/kdb/kdb-sql.md)  
  Execute SQL queries in KDB+.

## Requirements

### Database User

This source only uses standard authentication. You will need to configure a KDB+ user to login to the database with.

## Example

```yaml
sources:
    my-kdb-source:
        kind: kdb
        host: 127.0.0.1
        port: 5001
        user: ${USER_NAME}
        password: ${PASSWORD}
```

{{< notice tip >}}
Use environment variable replacement with the format ${ENV_NAME}
instead of hardcoding your secrets into the configuration file.
{{< /notice >}}

## Reference

|  **field**  |      **type**      | **required** | **description**                                                        |
|-------------|:------------------:|:------------:|------------------------------------------------------------------------|
| kind        |       string       |     true     | Must be "kdb".                                                         |
| host        |       string       |     true     | IP address to connect to (e.g. "127.0.0.1")                            |
| port        |       string       |     true     | Port to connect to (e.g. "5001")                                       |
| user        |       string       |     false    | Name of the KDB+ user to connect as (e.g. "my-kdb-user").              |
| password    |       string       |     false    | Password of the KDB+ user (e.g. "my-password").                        |
