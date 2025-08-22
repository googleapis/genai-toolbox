---
title: "Scylla"
type: docs
weight: 1
description: >
  Scylla is a high-performance NoSQL database compatible with Apache Cassandra.
---

## About

[Scylla][scylla-docs] is a NoSQL database that delivers the performance and low latency of C++ combined with the simplicity of Apache Cassandra compatibility. It provides predictable low latency at any scale while maintaining full compatibility with Cassandra Query Language (CQL) and drivers.

[scylla-docs]: https://docs.scylladb.com/

## Available Tools

- [`scylla-cql`](../tools/scylla/scylla-cql.md)  
  Execute parameterized CQL queries against Scylla.

- [`scylla-execute-cql`](../tools/scylla/scylla-execute-cql.md)  
  Execute arbitrary CQL queries against Scylla.

## Requirements

### Scylla Cluster

You need access to a running Scylla cluster with appropriate user permissions for the keyspace you want to access.

## Example

```yaml
sources:
    my-scylla-source:
    kind: scylla
    hosts:
      - scylla1.example.com
      - scylla2.example.com
      - scylla3.example.com
    port: "9042"
    keyspace: mykeyspace
    username: ${SCYLLA_USER}  # Optional
    password: ${SCYLLA_PASSWORD}  # Optional
    consistency: QUORUM  # Optional
```

{{< notice tip >}}
Use environment variable replacement with the format ${ENV_NAME}
instead of hardcoding your secrets into the configuration file.
{{< /notice >}}

## Reference

|  **field**  |      **type**      | **required** | **description**                                                        |
|-------------|:------------------:|:------------:|------------------------------------------------------------------------|
| kind        |       string       |     true     | Must be "scylla".                                                      |
| hosts       |    []string        |     true     | List of Scylla node hostnames (e.g. ["host1", "host2"])               |
| port        |       string       |     true     | Scylla native protocol port (default: "9042")                          |
| keyspace    |       string       |     true     | Default keyspace to use for queries                                    |
| username    |       string       |     false    | Username for authentication                                            |
| password    |       string       |     false    | Password for authentication                                            |
| consistency |       string       |     false    | Consistency level (e.g. "ONE", "QUORUM", "ALL", "LOCAL_QUORUM")       |
| connectTimeout|      string       |     false    | Connection timeout duration (e.g. "10s", "30s")                       |
| timeout     |       string       |     false    | Query timeout duration (e.g. "30s", "1m")                             |
| disableInitialHostLookup | boolean |     false    | Disable initial host lookup (default: false)                          |
| numConnections |      int        |     false    | Number of connections per host (default: 2)                           |
| protoVersion |        int        |     false    | CQL protocol version (default: auto-detect)                           |
| sslEnabled  |       boolean      |     false    | Enable SSL/TLS (default: false)                                       |
