---
title: "OceanBase"
type: docs
weight: 1
description: >
  OceanBase is a distributed relational database that provides high availability, scalability, and compatibility with MySQL.
---

## About

[OceanBase][oceanbase-docs] is a distributed relational database management system (RDBMS) that provides high availability, scalability, and strong consistency. It's designed to handle large-scale data processing and is compatible with MySQL, making it easy for developers to migrate from MySQL to OceanBase.

[oceanbase-docs]: https://www.oceanbase.com/

## Requirements

### Database User

This source only uses standard authentication. You will need to create an OceanBase user to login to the database with. OceanBase supports MySQL-compatible user management syntax.

### Network Connectivity

Ensure that your application can connect to the OceanBase cluster. OceanBase typically runs on ports 2881 (for MySQL protocol) or 3881 (for MySQL protocol with SSL).

## Example

```yaml
sources:
    my-oceanbase-source:
        kind: oceanbase
        host: 127.0.0.1
        port: 2881
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
| ------------ | :------: | :----------: |-------------------------------------------------------------------------------------------------|
| kind         |  string  |     true     | Must be "oceanbase".                                                                            |
| host         |  string  |     true     | IP address to connect to (e.g. "127.0.0.1").                                                    |
| port         |  string  |     true     | Port to connect to (e.g. "2881").                                                               |
| database     |  string  |     true     | Name of the OceanBase database to connect to (e.g. "my_db").                                    |
| user         |  string  |     true     | Name of the OceanBase user to connect as (e.g. "my-oceanbase-user").                            |
| password     |  string  |     true     | Password of the OceanBase user (e.g. "my-password").                                            |
| queryTimeout |  string  |    false     | Maximum time to wait for query execution (e.g. "30s", "2m"). By default, no timeout is applied. |

## Features

### MySQL Compatibility

OceanBase is highly compatible with MySQL, supporting most MySQL SQL syntax, data types, and functions. This makes it easy to migrate existing MySQL applications to OceanBase.

### High Availability

OceanBase provides automatic failover and data replication across multiple nodes, ensuring high availability and data durability.

### Scalability

OceanBase can scale horizontally by adding more nodes to the cluster, making it suitable for large-scale applications.

### Strong Consistency

OceanBase provides strong consistency guarantees, ensuring that all transactions are ACID compliant. 