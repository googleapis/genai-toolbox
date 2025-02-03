---
title: "Dgraph"
type: docs
weight: 1
description: >
 Dgraph is a horizontally scalable and distributed graph database.

---

## About

[Dgraph][dgraph-docs] is a horizontally scalable and distributed graph database.
It provides ACID transactions, consistent replication, and linearizable reads.

This source can connect to either a self-managed Dgraph cluster or one hosted on
Dgraph Cloud. If you're new to Dgraph, the fastest way to get started is to
[sign up for Dgraph Cloud][dgraph-login].

[dgraph-docs]: https://dgraph.io/docs
[dgraph-login]: https://cloud.dgraph.io/login

## Requirements 

### Database User

When **connecting to a hosted Dgraph database**, this source uses the API key
for access. If you are using a dedicated environment, you will additionally need
the namespace and user credentials for that namespace.

For **connecting to a local or self-hosted Dgraph database**, use the namespace
and user credentials for that namespace.

## Example

```yaml
sources:
    my-dgraph-source:
        kind: "dgraph"
        dgraphUrl: "https://xxxx.cloud.dgraph.io"
        user: "groot"
        password: "password"
        apiKey: abc123
        namepace : 0
```

## Reference

| **Field**   | **Type** | **Required** | **Description**                                                                                  |
|-------------|:--------:|:------------:|--------------------------------------------------------------------------------------------------|
| kind        |  string  |     true     | Must be "dgraph".                                                                                |
| dgraphUrl   |  string  |     true     | Connection URI (e.g. "https://xxx.cloud.dgraph.io", "https://localhost:8080").                   |
| user        |  string  |     false    | Name of the Dgraph user to connect as (e.g., "groot").                                           |
| password    |  string  |     false    | Password of the Dgraph user (e.g., "password").                                                  |
| apiKey      |  string  |     false    | API key to connect to a Dgraph Cloud instance.                                                   |
| namespace   |  uint64  |     false    | Dgraph namespace (not required for Dgraph Cloud Shared Clusters).                                |
