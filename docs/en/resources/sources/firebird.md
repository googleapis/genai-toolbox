---
title: "Firebird"
type: docs
weight: 1
description: >
  Firebird is a powerful, cross-platform, and open-source relational database.

---

## About

[Firebird][fb-docs] is a relational database management system offering many ANSI SQL standard features that runs on Linux, Windows, and a variety of Unix platforms. It is known for its small footprint, powerful features, and easy maintenance.

[fb-docs]: https://firebirdsql.org/

## Available Tools

- [`firebird-sql`](../tools/firebird/firebird-sql.md)  
  Execute SQL queries as prepared statements in Firebird.

- [`firebird-execute-sql`](../tools/firebird/firebird-execute-sql.md)  
  Run parameterized SQL statements in Firebird.

## Requirements

### Database User

This source uses standard authentication. You will need to [create a Firebird user][fb-users] to login to the database with.

[fb-users]: https://firebirdsql.org/refdocs/langrefupd25-sql-create-user.html

## Example

```yaml
sources:
    my_firebird_db:
        kind: firebird
        host: "localhost"
        port: 3050
        database: "/path/to/your/database.fdb"
        user: ${FIREBIRD_USER}
        password: ${FIREBIRD_PASS}
```

{{< notice tip >}}
Use environment variable replacement with the format ${ENV_NAME}
instead of hardcoding your secrets into the configuration file.
{{< /notice >}}

## Reference

| **field** | **type** | **required** | **description**                                                        |
|-----------|:--------:|:------------:|------------------------------------------------------------------------|
| kind      |  string  |     true     | Must be "firebird".                                                    |
| host      |  string  |     true     | IP address to connect to (e.g. "127.0.0.1")                            |
| port      |  string  |     true     | Port to connect to (e.g. "3050")                                       |
| database  |  string  |     true     | Path to the Firebird database file (e.g. "/var/lib/firebird/data/test.fdb"). |
| user      |  string  |     true     | Name of the Firebird user to connect as (e.g. "SYSDBA").               |
| password  |  string  |     true     | Password of the Firebird user (e.g. "masterkey").                      |