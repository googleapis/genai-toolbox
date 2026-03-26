---
title: "MySQL Source"
linkTitle: "Source"
type: docs
weight: 1
description: >
  MySQL is a relational database management system that stores and manages data.
no_list: true
---

## About

[MySQL][mysql-docs] is a relational database management system (RDBMS) that
stores and manages data. It's a popular choice for developers because of its
reliability, performance, and ease of use.

[mysql-docs]: https://www.mysql.com/

## Available Tools

{{< list-tools >}}

## Requirements

### Database User

This source only uses standard authentication. You will need to [create a
MySQL user][mysql-users] to login to the database with.

[mysql-users]: https://dev.mysql.com/doc/refman/8.4/en/user-names.html

## Example

```yaml
kind: source
name: my-mysql-source
type: mysql
host: 127.0.0.1
port: 3306
database: my_db
user: ${USER_NAME}
password: ${PASSWORD}
# Optional TLS and other driver parameters. For example, enable preferred TLS:
# queryParams:
#     tls: preferred
queryTimeout: 30s # Optional: query timeout duration
```

{{< notice tip >}}
Use environment variable replacement with the format ${ENV_NAME}
instead of hardcoding your secrets into the configuration file.
{{< /notice >}}

## Reference

| **field**    |      **type**      | **required** | **description**                                                                                                                                 |
|--------------|:------------------:|:------------:|-------------------------------------------------------------------------------------------------------------------------------------------------|
| type         |       string       |     true     | Must be "mysql".                                                                                                                                |
| host         |       string       |     true     | IP address to connect to (e.g. "127.0.0.1").                                                                                                    |
| port         |       string       |     true     | Port to connect to (e.g. "3306").                                                                                                               |
| user         |       string       |    false     | Name of the MySQL user to connect as (e.g. "my-mysql-user").                                                                                    |
| password     |       string       |    false     | Password of the MySQL user (e.g. "my-password").                                                                                                |
| database     |       string       |    false     | Name of the MySQL database to connect to (e.g. "my_db").                                                                                        |
| queryTimeout |       string       |    false     | Maximum time to wait for query execution (e.g. "30s", "2m"). By default, no timeout is applied.                                                 |
| queryParams  | map<string,string> |    false     | Arbitrary DSN parameters passed to the driver (e.g. `tls: preferred`, `charset: utf8mb4`). Useful for enabling TLS or other connection options. |
