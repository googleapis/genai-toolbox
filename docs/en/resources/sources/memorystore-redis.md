---
title: "Memorystore for Redis"
linkTitle: "Memorystore (Redis)"
type: docs
weight: 1
description: >
    Memorystore for Redis is a fully-managed Redis service.
    
---

## About

Memorystore for Redis is a fully-managed, in-memory data store service built on open-source Redis. It allows you to deploy highly scalable and available Redis instances on Google Cloud Platform.

If you are new to Memorystore for Redis, you can try creating and connecting to a Redis instance by following this [guide][ms-redis-quickstart].

[ms-redis-quickstart]:
    https://cloud.google.com/memorystore/docs/redis/create-instance-console

## Requirements

### Memorystore Redis

[AUTH string][auth] is a Universally Unique Identifier (UUID) that serves as the
password for connection to Memorystore Redis. If you enable the AUTH feature on
your Memorystore instance, incoming client connections must authenticate in
order to connect.

Specify your AUTH string in the password field:

```yaml
sources:
    my-redis-instance:
     kind: memorystore-redis
     address: 127.0.0.1
     password: ${MY_AUTH_STRING} # Omit this field if you don't have an AUTH string.
     database: 1
     # clusterEnabled: false
     # useIAM: false  # Non-cluster Redis instance does not support IAM authentication
```

{{< notice tip >}}
Use environment variable replacement with the format ${ENV_NAME}
instead of hardcoding your secrets into the configuration file.
{{< /notice >}}

### Memorystore Redis Cluster

Memorystore Redis Cluster supports IAM authentication. Grant your account the
required [IAM role][iam] and make sure to set `clusterEnabled` to `true`.

Here is an example tools.yaml config for Memorystore Redis Cluster instances
using IAM authentication:

```yaml
sources:
    my-redis-cluster-instance:
     kind: memorystore-redis
     address: 127.0.0.1
     useIAM: true
     clusterEnabled: true
```

[iam]: https://cloud.google.com/memorystore/docs/cluster/about-iam-auth

## Reference

| **field**      | **type** | **required** | **description**                                                                                              |
|----------------|:--------:|:------------:|--------------------------------------------------------------------------------------------------------------|
| kind           |  string  |     true     | Must be "memorystore-redis".                                                                                 |
| address        |  string  |     true     | Primary endpoint for the Memorystore Redis instance to connect to.                                           |
| password       |  string  |    false     | If you have [Redis AUTH][auth] enabled, specify the AUTH string here                                         |
| database       |   int    |    false     | The Redis database to connect to. Not applicable for cluster enabled instances. The default database is `0`. |
| clusterEnabled |   bool   |    false     | Set it to `true` if using a Redis Cluster instance. Defaults to `false`.                                     |
| useIAM         |  string  |    false     | Set it to `true` if you are using IAM authentication. Defaults to `false`.                                   |

[auth]: https://cloud.google.com/memorystore/docs/redis/about-redis-auth
