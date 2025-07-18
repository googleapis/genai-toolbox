---
title: "Redis"
linkTitle: "Redis"
type: docs
weight: 1
description: >
    Redis is an open-source, in-memory data structure store.
    
---

## About

Redis is an open-source, in-memory data structure store, used as a database,
cache, and message broker. It supports data structures such as strings, hashes,
lists, sets, sorted sets with range queries, bitmaps, hyperloglogs, and
geospatial indexes with radius queries.

If you are new to Redis, you can find installation and getting started guides on
the [official Redis website](https://redis.io/docs/getting-started/).

## Available Tools

- [`redis`](../tools/redis/redis.md)  
  Run Redis commands and interact with key-value pairs.

## Requirements

### Redis

[AUTH string][auth] is a password for connection to Redis. If you have the
`requirepass` directive set in your Redis configuration, incoming client
connections must authenticate in order to connect.

Specify your AUTH string in the password field:

```yaml
sources:
    my-redis-instance:
     kind: redis
     address:
       - 127.0.0.1
     username: ${MY_USER_NAME}
     password: ${MY_AUTH_STRING} # Omit this field if you don't have a password.
     # database: 0
     # clusterEnabled: false
     # useGCPIAM: false
```

{{< notice tip >}}
Use environment variable replacement with the format ${ENV_NAME}
instead of hardcoding your secrets into the configuration file.
{{< /notice >}}

### Memorystore For Redis

Memorystore standalone instances support authentication using an [AUTH][auth]
string.

Here is an example tools.yaml config with [AUTH][auth] enabled:

```yaml
sources:
    my-redis-cluster-instance:
     kind: memorystore-redis
     address:
       - 127.0.0.1
     password: ${MY_AUTH_STRING}
     # useGCPIAM: false
     # clusterEnabled: false
```

Memorystore Redis Cluster supports IAM authentication instead. Grant your
account the required [IAM role][iam] and make sure to set `useGCPIAM` to `true`.

Here is an example tools.yaml config for Memorystore Redis Cluster instances
using IAM authentication:

```yaml
sources:
    my-redis-cluster-instance:
     kind: memorystore-redis
     address: 127.0.0.1
     useGCPIAM: true
     clusterEnabled: true
```

[iam]: https://cloud.google.com/memorystore/docs/cluster/about-iam-auth

## Reference

| **field**      | **type** | **required** | **description**                                                                                                                 |
|----------------|:--------:|:------------:|---------------------------------------------------------------------------------------------------------------------------------|
| kind           |  string  |     true     | Must be "memorystore-redis".                                                                                                    |
| address        |  string  |     true     | Primary endpoint for the Memorystore Redis instance to connect to.                                                              |
| username       |  string  |    false     | If you are using a non-default user, specify the user name here. If you are using Memorystore for Redis, leave this field blank |
| password       |  string  |    false     | If you have [Redis AUTH][auth] enabled, specify the AUTH string here                                                            |
| database       |   int    |    false     | The Redis database to connect to. Not applicable for cluster enabled instances. The default database is `0`.                    |
| clusterEnabled |   bool   |    false     | Set it to `true` if using a Redis Cluster instance. Defaults to `false`.                                                        |
| useGCPIAM      |  string  |    false     | Set it to `true` if you are using GCP's IAM authentication. Defaults to `false`.                                                |

[auth]: https://cloud.google.com/memorystore/docs/redis/about-redis-auth
