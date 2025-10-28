---
title: "Cassandra"
type: docs
weight: 1
description: >
  Apache Cassandra is a NoSQL distributed database known for its horizontal scalability, distributed architecture, and flexible schema definition.
---

## About

[Apache Cassandra][cassandra-docs] is a NoSQL distributed database. By design, NoSQL databases are lightweight, open-source, non-relational, and largely distributed. Counted among their strengths are horizontal scalability, distributed architectures, and a flexible approach to schema definition.

[cassandra-docs]: https://cassandra.apache.org/

## Available Tools

- [`cassandra-cql`](../tools/cassandra/cassandra-cql.md)  
  Run parameterized CQL queries in Cassandra.


## Example

```yaml
sources:
    my-cassandra-source:
        kind: cassandra
        hosts:
            - 127.0.0.1
        keyspace: my_keyspace
        protoVersion: 4
        username: ${USER_NAME}
        password: ${PASSWORD}
        caPath: /path/to/ca.crt # Optional: path to CA certificate
        certPath: /path/to/client.crt # Optional: path to client certificate
        keyPath: /path/to/client.key # Optional: path to client key
        enableHostVerification: true # Optional: enable host verification
```

{{< notice tip >}}
Use environment variable replacement with the format ${ENV_NAME}
instead of hardcoding your secrets into the configuration file.
{{< /notice >}}

## Reference

| **field**              | **type**  | **required** | **description**                                                                                       |
|------------------------|:---------:|:------------:|-------------------------------------------------------------------------------------------------------|
| kind                   |  string   |     true     | Must be "cassandra".                                                                                  |
| hosts                  |  string[] |     true     | List of IP addresses to connect to (e.g., ["192.168.1.1:9042", "192.168.1.2:9042","192.168.1.3:9042"]). The default port is 9042 if not specified.                                    |
| keyspace               |  string   |     true     | Name of the Cassandra keyspace to connect to (e.g., "my_keyspace").                                   |
| protoVersion           |  integer  |    false     | Protocol version for the Cassandra connection (e.g., 4).                                              |
| username               |  string   |    false     | Name of the Cassandra user to connect as (e.g., "my-cassandra-user").                                 |
| password               |  string   |    false     | Password of the Cassandra user (e.g., "my-password").                                                 |
| caPath                 |  string   |    false     | Path to the CA certificate for SSL/TLS (e.g., "/path/to/ca.crt").                                     |
| certPath               |  string   |    false     | Path to the client certificate for SSL/TLS (e.g., "/path/to/client.crt").                             |
| keyPath                |  string   |    false     | Path to the client key for SSL/TLS (e.g., "/path/to/client.key").                                     |
| enableHostVerification |  boolean  |    false     | Enable host verification for SSL/TLS (e.g., true). By default, host verification is disabled.         |
