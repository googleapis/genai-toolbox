---
title: "MariaDB Source"
linkTitle: "Source"
type: docs
weight: 1
description: >
  MariaDB is an open-source relational database compatible with MySQL.
no_list: true
---
## About

MariaDB is a relational database management system derived from MySQL. It
implements the MySQL protocol and client libraries and supports modern SQL
features with a focus on performance and reliability.

**Note**: MariaDB is supported using the MySQL source.

## Available Tools

{{< list-tools dirs="/integrations/mysql/tools" >}}

## Requirements

### Database User

This source only uses standard authentication. You will need to [create a
MariaDB user][mariadb-users] to log in to the database.

[mariadb-users]: https://mariadb.com/kb/en/create-user/

## Example

```yaml
kind: source
name: my_mariadb_db
type: mysql
host: 127.0.0.1
port: 3306
database: my_db
user: ${MARIADB_USER}
password: ${MARIADB_PASS}
# Optional TLS and other driver parameters. For example, enable preferred TLS:
# queryParams:
#     tls: preferred
queryTimeout: 30s # Optional: query timeout duration
```

{{< notice tip >}}
Use environment variables instead of committing credentials to source files.
{{< /notice >}}


## Reference

| **field**    | **type** | **required** | **description**                                                                                 |
| ------------ | :------: | :----------: | ----------------------------------------------------------------------------------------------- |
| type         |  string  |     true     | Must be `mysql`.                                                                                |
| host         |  string  |     true     | IP address to connect to (e.g. "127.0.0.1").                                                    |
| port         |  string  |     true     | Port to connect to (e.g. "3307").                                                               |
| database     |  string  |     true     | Name of the MariaDB database to connect to (e.g. "my_db").                                        |
| user         |  string  |     true     | Name of the MariaDB user to connect as (e.g. "my-mysql-user").                                    |
| password     |  string  |     true     | Password of the MariaDB user (e.g. "my-password").                                                |
| queryTimeout |  string  |    false     | Maximum time to wait for query execution (e.g. "30s", "2m"). By default, no timeout is applied. |
| queryParams | map<string,string> | false | Arbitrary DSN parameters passed to the driver (e.g. `tls: preferred`, `charset: utf8mb4`). Useful for enabling TLS or other connection options. |
