---
title: "MariaDB"
type: docs
weight: 1
description: >
  MariaDB is an open-source relational database compatible with MySQL.

---
## About

MariaDB is a relational database management system derived from MySQL. It
implements the MySQL protocol and client libraries and supports modern SQL
features with a focus on performance and reliability.

## Available Tools

- [`mysql-sql`](../tools/mysql/mysql-sql.md)
  Execute pre-defined prepared SQL queries in MariaDB.

- [`mysql-execute-sql`](../tools/mysql/mysql-execute-sql.md)
  Run parameterized SQL queries in MariaDB.

- [`mysql-list-active-queries`](../tools/mysql/mysql-list-active-queries.md)
  List active queries in MariaDB.

- [`mysql-list-tables`](../tools/mysql/mysql-list-tables.md)
  List tables in a MariaDB database.

- [`mysql-list-tables-missing-unique-indexes`](../tools/mysql/mysql-list-tables-missing-unique-indexes.md)
  List tables in a MariaDB database that do not have primary or unique indices.

- [`mysql-list-table-fragmentation`](../tools/mysql/mysql-list-table-fragmentation.md)
  List table fragmentation in MariaDB tables.

## Requirements

### Database User

This source only uses standard authentication. You will need to [create a
MariaDB user][mariadb-users] to log in to the database.

[mariadb-users]: https://mariadb.com/kb/en/create-user/

## Example
```yaml
sources:
  my_mariadb_db:
    kind: mariadb
    host: "127.0.0.1"
    port: "3306"
    database: "example"
    user: ${MARIADB_USER}
    password: ${MARIADB_PASS}
    queryParams:
      interpolateParams: "true"
```

{{< notice tip >}}
Use environment variables instead of committing credentials to source files.
{{< /notice >}}

## Reference

| field       |  type  | required | description                                    |
| ----------- | :----: | :------: | ---------------------------------------------- |
| kind        | string |   true   | Must be `mariadb`.                             |
| host        | string |   true   | Hostname or IP of the MariaDB server.          |
| port        | string |   true   | Port to connect to (typically `3306`).         |
| database    | string |   true   | Database name to connect to.                   |
| user        | string |   true   | Username used for authentication.              |
| password    | string |   true   | Password for the specified user.               |
| queryParams |   map  |   false  | Optional URL query params appended to the DSN. |