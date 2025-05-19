---
title: "Memorystore for Valkey"
linkTitle: "Memorystore (Valkey)"
type: docs
weight: 1
description: >
Memorystore for Valkey is a fully-managed Valkey service.
    
---

## About

Memorystore for Valkey is a fully-managed, in-memory data store service built on open-source Valkey. It allows you to deploy highly scalable and available Valkey instances on Google Cloud Platform.

If you are new to Memorystore for Valkey, you can try creating and connecting to a Valkey instance by following this [guide][quickstart].

[quickstart]: https://cloud.google.com/memorystore/docs/valkey/create-instances

## Requirements

### IAM Authentication

Memorystore Valkey supports IAM authentication. Grant your account the
required [IAM role][iam] and set `useIAM` to `true`.

## Example

```yaml
sources:
    my-valkey-instance:
     kind: memorystore-valkey
     address: 127.0.0.1
     database: 1
     useIAM: true
     disableCache: true
```

[iam]: https://cloud.google.com/memorystore/docs/valkey/about-iam-auth

## Reference

| **field**      | **type** | **required** | **description**                                                                                              |
|----------------|:--------:|:------------:|--------------------------------------------------------------------------------------------------------------|
| kind           |  string  |     true     | Must be "memorystore-valkey".                                                                                 |
| address        |  string  |     true     | Primary endpoint for the Memorystore Valkey instance to connect to.                                           |
| database       |   int    |    false     | The Valkey database to connect to. Not applicable for cluster enabled instances. The default database is `0`. |
| clusterEnabled |   bool   |    false     | Set it to `true` if using a Valkey Cluster instance. Defaults to `false`.                                     |
| useIAM         |  bool  |    false     | Set it to `true` if you are using IAM authentication. Defaults to `false`.                                   |
| disableCache   |  bool  |    false     | Set it to `true` if you want to enable client-side caching. Defaults to `false`.                                   |
