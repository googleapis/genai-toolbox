---
title: "Valkey"
linkTitle: "Valkey"
type: docs
weight: 1
description: >
    Valkey is an open-source, in-memory data structure store, forked from Redis.
    
---

## About

Valkey is an open-source, in-memory data structure store that originated as a
fork of Redis. It's designed to be used as a database, cache, and message
broker, supporting a wide range of data structures like strings, hashes, lists,
sets, sorted sets with range queries, bitmaps, hyperloglogs, and geospatial
indexes with radius queries.

If you're new to Valkey, you can find installation and getting started guides on
the [official Valkey website](https://valkey.io/topics/quickstart/).

## Example

```yaml
sources:
    my-valkey-instance:
     kind: valkey
     address:
       - 127.0.0.1:6379
     username: ${YOUR_USERNAME}
     password: ${YOUR_PASSWORD}
     # database: 0
     # useGCPIAM: false
     # disableCache: false
```

{{< notice tip >}}
Use environment variable replacement with the format ${ENV_NAME}
instead of hardcoding your secrets into the configuration file.
{{< /notice >}}

### IAM Authentication

If you are using GCP's Memorystore for Valkey, you can connect using IAM
authentication. Grant your account the required [IAM role][iam] and set
`useGCPIAM` to `true`:

```yaml
sources:
    my-valkey-instance:
     kind: valkey
     address:
       - 127.0.0.1:6379
     useGCPIAM: true
```

[iam]: https://cloud.google.com/memorystore/docs/valkey/about-iam-auth

## Reference

| **field**    | **type** | **required** | **description**                                                                                                                  |
|--------------|:--------:|:------------:|----------------------------------------------------------------------------------------------------------------------------------|
| kind         |  string  |     true     | Must be "valkey".                                                                                                                |
| address      | []string |     true     | Endpoints for the Valkey instance to connect to.                                                                                 |
| username     |  string  |    false     | If you are using a non-default user, specify the user name here. If you are using Memorystore for Valkey, leave this field blank |
| password     |  string  |    false     | Password for the Valkey instance                                                                                                 |
| database     |   int    |    false     | The Valkey database to connect to. Not applicable for cluster enabled instances. The default database is `0`.                    |
| useGCPIAM    |   bool   |    false     | Set it to `true` if you are using GCP's IAM authentication. Defaults to `false`.                                                 |
| disableCache |   bool   |    false     | Set it to `true` if you want to enable client-side caching. Defaults to `false`.                                                 |
