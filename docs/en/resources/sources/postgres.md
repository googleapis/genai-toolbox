---
title: "PostgreSQL"
type: docs
weight: 1
description: >
  PostgreSQL is a powerful, open source object-relational database.

---

## About

[PostgreSQL][pg-docs] is a powerful, open source object-relational database
system with over 35 years of active development that has earned it a strong
reputation for reliability, feature robustness, and performance.

[pg-docs]: https://www.postgresql.org/

## Requirements

### Database User

This source only uses standard authentication. You will need to [create a
PostgreSQL user][pg-users] to login to the database with.

[pg-users]: https://www.postgresql.org/docs/current/sql-createuser.html

## Example

```yaml
sources:
    my-pg-source:
        kind: postgres
        host: 127.0.0.1
        port: 5432
        database: my_db
        user: ${USER_NAME}
        password: ${PASSWORD}
```

{{< notice tip >}}
Use environment variable replacement with the format ${ENV_NAME}
instead of hardcoding your secrets into the configuration file.
{{< /notice >}}

## Reference

| **field** | **type** | **required** | **description**                                                        |
|-----------|:--------:|:------------:|------------------------------------------------------------------------|
| kind      |  string  |     true     | Must be "postgres".                                                    |
| host      |  string  |     true     | IP address to connect to (e.g. "127.0.0.1")                            |
| port      |  string  |     true     | Port to connect to (e.g. "5432")                                       |
| database  |  string  |     true     | Name of the Postgres database to connect to (e.g. "my_db").            |
| user      |  string  |     true     | Name of the Postgres user to connect as (e.g. "my-pg-user").           |
| password  |  string  |     true     | Password of the Postgres user (e.g. "my-password").                    |
