---
title: "TiDB"
type: docs
weight: 2
description: >
  TiDB is a distributed SQL database that combines the best of traditional RDBMS and NoSQL databases.

---

## About

[TiDB][tidb-docs] is an open-source distributed SQL database that supports Hybrid Transactional and Analytical Processing (HTAP) workloads. It is MySQL-compatible and features horizontal scalability, strong consistency, and high availability.

[tidb-docs]: https://docs.pingcap.com/tidb/stable

## Requirements

### Database User

This source uses standard MySQL protocol authentication. You will need to [create a TiDB user][tidb-users] to login to the database with.

For TiDB Cloud users, you can create database users through the TiDB Cloud console.

[tidb-users]: https://docs.pingcap.com/tidb/stable/user-account-management

## SSL Configuration

- TiDB Cloud

    For TiDB Cloud instances, SSL is automatically enabled when the hostname matches the TiDB Cloud pattern (`gateway*.*.*.tidbcloud.com`). You don't need to explicitly set `use_ssl: true` for TiDB Cloud connections.

- Self-Hosted TiDB

    For self-hosted TiDB instances, you can optionally enable SSL by setting `use_ssl: true` in your configuration.

## Example

- TiDB Cloud

    ```yaml
    sources:
        my-tidb-cloud-source:
            kind: tidb
            host: gateway01.us-west-2.prod.aws.tidbcloud.com
            port: 4000
            database: my_db
            user: ${TIDB_USERNAME}
            password: ${TIDB_PASSWORD}
            # SSL is automatically enabled for TiDB Cloud
    ```

- Self-Hosted TiDB

    ```yaml
    sources:
        my-tidb-source:
            kind: tidb
            host: 127.0.0.1
            port: 4000
            database: my_db
            user: ${TIDB_USERNAME}
            password: ${TIDB_PASSWORD}
            use_ssl: false  # Optional: enable SSL for secure connections
    ```

{{< notice tip >}}
Use environment variable replacement with the format ${ENV_NAME}
instead of hardcoding your secrets into the configuration file.
{{< /notice >}}

## Reference

| **field** | **type** | **required** | **description**                                                                             |
|-----------|:--------:|:------------:|---------------------------------------------------------------------------------------------|
| kind      |  string  |     true     | Must be "tidb".                                                                             |
| host      |  string  |     true     | IP address or hostname to connect to (e.g. "127.0.0.1" or "gateway01.*.tidbcloud.com").    |
| port      |  string  |     true     | Port to connect to (typically "4000" for TiDB).                                            |
| database  |  string  |     true     | Name of the TiDB database to connect to (e.g. "my_db").                                    |
| user      |  string  |     true     | Name of the TiDB user to connect as (e.g. "my-tidb-user").                                 |
| password  |  string  |     true     | Password of the TiDB user (e.g. "my-password").                                            |
| use_ssl   |  boolean |    false     | Whether to use SSL/TLS encryption. Automatically enabled for TiDB Cloud instances.         | 