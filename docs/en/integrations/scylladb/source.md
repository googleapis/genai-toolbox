---
title: "ScyllaDB Source"
type: docs
linkTitle: "Source"
weight: 1
description: >
  ScyllaDB is a high-performance NoSQL database compatible with Apache Cassandra CQL, offering lower latency and higher throughput through its shard-per-core architecture.
no_list: true
---

## About

[ScyllaDB][scylladb-docs] is a high-performance NoSQL database that is 
compatible with Apache Cassandra's CQL protocol. It is designed to provide 
predictable performance at scale, optimize cloud infrastructure, 
rapidly scale clusters with global replication and high availability,

ScyllaDB supports both self-hosted deployments and [ScyllaDB Cloud][scylladb-cloud],
a fully-managed DBaaS available on AWS and GCP.

[scylladb-docs]: https://docs.scylladb.com/
[scylladb-cloud]: https://cloud.scylladb.com/

## Available Tools

{{< list-tools >}}

## Example

### Self-hosted ScyllaDB

```yaml
kind: source
name: my-scylladb-source
type: scylladb
hosts:
    - 127.0.0.1
keyspace: my_keyspace
protoVersion: 4
username: ${USER_NAME}
password: ${PASSWORD}
```

### ScyllaDB Cloud

When connecting to ScyllaDB Cloud, set `localDC` to the datacenter name shown
on the **Connect** tab of your cluster in the [ScyllaDB Cloud Console][scylladb-cloud].
This enables DC-aware token-aware load balancing, which is required for
ScyllaDB Cloud connections.

```yaml
kind: source
name: my-scylladb-cloud-source
type: scylladb
hosts:
    - node-0.your-cluster.us-east-1.cloud.scylladb.com
    - node-1.your-cluster.us-east-1.cloud.scylladb.com
    - node-2.your-cluster.us-east-1.cloud.scylladb.com
keyspace: my_keyspace
username: ${USER_NAME}
password: ${PASSWORD}
localDC: AWS_US_EAST_1
```

{{< notice tip >}}
Use environment variable replacement with the format ${ENV_NAME}
instead of hardcoding your secrets into the configuration file.
{{< /notice >}}

## Reference

| **field**              | **type** | **required** | **description**                                                                                                                                         |
|------------------------|:--------:|:------------:|---------------------------------------------------------------------------------------------------------------------------------------------------------|
| type                   |  string  |     true     | Must be "scylladb".                                                                                                                                     |
| hosts                  | string[] |     true     | List of contact point addresses (e.g., ["192.168.1.1:9042", "192.168.1.2:9042"]). The default port is 9042 if not specified.                            |
| keyspace               |  string  |    false     | Name of the ScyllaDB keyspace to connect to (e.g., "my_keyspace").                                                                                      |
| protoVersion           | integer  |    false     | CQL native protocol version (e.g., 4).                                                                                                                  |
| username               |  string  |    false     | Name of the ScyllaDB user to connect as (e.g., "scylla").                                                                                               |
| password               |  string  |    false     | Password of the ScyllaDB user.                                                                                                                          |
| localDC                |  string  |    false     | Datacenter name for DC-aware load balancing (e.g., "AWS_US_EAST_1"). Required for ScyllaDB Cloud connections.                                           |
| caPath                 |  string  |    false     | Path to a CA certificate file. Use when connecting to a self-hosted ScyllaDB cluster with a private/custom CA. Not needed for ScyllaDB Cloud.           |
| certPath               |  string  |    false     | Path to the client certificate file for mutual TLS (mTLS). Required only when the server demands client certificate authentication.                     |
| keyPath                |  string  |    false     | Path to the client private key file for mutual TLS (mTLS). Required together with `certPath`.                                                           |
| enableHostVerification |   bool   |    false     | Whether to verify the server's hostname against its TLS certificate. Defaults to `false`.                                                               |
