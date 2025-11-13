---
title: "SingleStore"
type: docs
weight: 1
description: >
  SingleStore is the cloud-native database built with speed and scale to power data-intensive applications.
---

## About

[SingleStore][singlestore-docs] is a distributed SQL database built to power intelligent applications. It is both relational and multi-model, enabling developers to easily build and scale applications and workloads.

SingleStore is built around Universal Storage which combines in-memory rowstore and on-disk columnstore data formats to deliver a single table type that is optimized to handle both transactional and analytical workloads.

[singlestore-docs]: https://docs.singlestore.com/

## Available Tools

- [`singlestore-sql`](../tools/singlestore/singlestore-sql.md)  
  Execute pre-defined prepared SQL queries in SingleStore.

- [`singlestore-execute-sql`](../tools/singlestore/singlestore-execute-sql.md)  
  Run parameterized SQL queries in SingleStore.

## Requirements

### Database User

This source only uses standard authentication. You will need to [create a
database user][singlestore-user] to login to the database with.

[singlestore-user]: https://docs.singlestore.com/cloud/reference/sql-reference/security-management-commands/create-user/

## Example

```yaml
sources:
    my-singlestore-source:
        kind: singlestore
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
| kind         |  string  |     true     | Must be "singlestore".                                                                          |
| host         |  string  |     true     | IP address to connect to (e.g. "127.0.0.1").                                                    |
| port         |  string  |     true     | Port to connect to (e.g. "3306").                                                               |
| database     |  string  |     true     | Name of the SingleStore database to connect to (e.g. "my_db").                                  |
| user         |  string  |     true     | Name of the SingleStore database user to connect as (e.g. "admin").                             |
| password     |  string  |     true     | Password of the SingleStore database user.                                                      |
| queryTimeout |  string  |    false     | Maximum time to wait for query execution (e.g. "30s", "2m"). By default, no timeout is applied. |
