---
title: "AlloyDB Admin"
linkTitle: "AlloyDB Admin"
type: docs
weight: 2
description: >
  The AlloyDB Admin source provides tools for managing AlloyDB for PostgreSQL database.
---

## About

[AlloyDB for PostgreSQL][alloydb-docs] is a fully-managed, PostgreSQL-compatible
database for demanding transactional workloads. It provides enterprise-grade
performance and availability while maintaining 100% compatibility with
open-source PostgreSQL.

The `alloydb-admin` source provides tools to perform tasks like creating and listing clusters, instances and users in your AlloyDB database.

> Note: For data plane operations such as connecting to the database and executing queries, please use the `alloydb-pg` source.

If you are new to AlloyDB for PostgreSQL, you can [create a free trial
cluster][alloydb-free-trial].

[alloydb-docs]: https://cloud.google.com/alloydb/docs
[alloydb-free-trial]: https://cloud.google.com/alloydb/docs/create-free-trial-cluster

## Available Tools

- [`alloydb-list-clusters`](../tools/alloydb/alloydb-list-clusters.md)  
  Lists all AlloyDB clusters in a given project and location.

- [`alloydb-list-instances`](../tools/alloydb/alloydb-list-instances.md)  
  Lists all AlloyDB instances within a specific cluster.

- [`alloydb-list-users`](../tools/alloydb/alloydb-list-users.md)  
  Lists all database users within a specific AlloyDB cluster.

### Pre-built Configurations

- [AlloyDB Admin API using MCP](https://googleapis.github.io/genai-toolbox/how-to/connect-ide/alloydb_pg_admin_mcp/)  
Create your AlloyDB database with MCP Toolbox.

## Requirements

### IAM Permissions

Ensure that the IAM identity has been given the following IAM roles (or corresponding
permissions):

- `roles/alloydb.admin`

## Example

```yaml
sources:
    my-alloydb-admin-source:
        kind: alloydb-admin
```

## Reference

| **field** | **type** | **required** | **description**                                                                                                          |
|-----------|:--------:|:------------:|--------------------------------------------------------------------------------------------------------------------------|
| kind      |  string  |     true     | Must be "alloydb-admin".                                                                                              |
