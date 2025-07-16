---
title: "MySQL"
type: docs
weight: 1
description: >
  MySQL is a relational database management system that stores and manages data.
---

## About

[MySQL][mysql-docs] is a relational database management system (RDBMS) that
stores and manages data. It's a popular choice for developers because of its
reliability, performance, and ease of use.

[mysql-docs]: https://www.mysql.com/

## Requirements

### Database User

This source only uses standard authentication. You will need to [create a
MySQL user][mysql-users] to login to the database with.

[mysql-users]: https://dev.mysql.com/doc/refman/8.4/en/user-names.html

## Example

```yaml
sources:
    my-mysql-source:
        kind: mysql
        host: 127.0.0.1
        port: 3306
        database: my_db
        user: ${USER_NAME}
        password: ${PASSWORD}
        queryTimeout: 30s # Optional: query timeout duration
```

{{< notice tip >}}
Use environment variable replacement with the format ${ENV_NAME}
instead of hardcoding your secrets into the configuration file.
{{< /notice >}}

## Reference

| **field**    | **type** | **required** | **description**                                                                                 |
| ------------ | :------: | :----------: | ----------------------------------------------------------------------------------------------- |
| kind         |  string  |     true     | Must be "mysql".                                                                                |
| host         |  string  |     true     | IP address to connect to (e.g. "127.0.0.1").                                                    |
| port         |  string  |     true     | Port to connect to (e.g. "3306").                                                               |
| database     |  string  |     true     | Name of the MySQL database to connect to (e.g. "my_db").                                        |
| user         |  string  |     true     | Name of the MySQL user to connect as (e.g. "my-mysql-user").                                    |
| password     |  string  |     true     | Password of the MySQL user (e.g. "my-password").                                                |
| queryTimeout |  string  |    false     | Maximum time to wait for query execution (e.g. "30s", "2m"). By default, no timeout is applied. |

## Example Tools

- [`mysql-sql`](../tools/mysql/mysql-sql.md)  
  Execute pre-defined prepared SQL queries in MySQL.

- [`mysql-execute-sql`](../tools/mysql/mysql-execute-sql.md)  
  Run parameterized SQL queries in MySQL.
