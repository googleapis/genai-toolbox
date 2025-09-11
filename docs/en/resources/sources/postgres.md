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

## Available Tools

- [`postgres-sql`](../tools/postgres/postgres-sql.md)  
  Execute SQL queries as prepared statements in PostgreSQL.

- [`postgres-execute-sql`](../tools/postgres/postgres-execute-sql.md)  
  Run parameterized SQL statements in PostgreSQL.

- [`postgres-list-tables`](../tools/postgres/postgres-list-tables.md)  
  List tables in a PostgreSQL database.

### Pre-built Configurations

- [PostgreSQL using MCP](https://googleapis.github.io/genai-toolbox/how-to/connect-ide/postgres_mcp/)  
Connect your IDE to PostgreSQL using Toolbox.

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

|  **field**  |      **type**      | **required** | **description**                                                        |
|-------------|:------------------:|:------------:|------------------------------------------------------------------------|
| kind        |       string       |     true     | Must be "postgres".                                                    |
| host        |       string       |     true     | IP address to connect to (e.g. "127.0.0.1")                            |
| port        |       string       |     true     | Port to connect to (e.g. "5432")                                       |
| database    |       string       |     true     | Name of the Postgres database to connect to (e.g. "my_db").            |
| user        |       string       |     true     | Name of the Postgres user to connect as (e.g. "my-pg-user").           |
| password    |       string       |     true     | Password of the Postgres user (e.g. "my-password").                    |
| queryParams |  map[string]string |     false    | Raw query to be added to the db connection string.                     |
