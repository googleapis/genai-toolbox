---
title: "couchbase"
type: docs
weight: 1
description: > 
  A "couchbase" source connects to a Couchbase database.
---

## About

A `couchbase` source establishes a connection to a Couchbase database cluster, allowing tools to execute SQL queries against it.

## Example

```yaml
sources:
  my-couchbase-instance:
    kind: couchbase
    connection_string: couchbase://localhost:8091
    bucket: travel-sample
    scope: inventory
    username: Administrator
    password: password
```

## Reference

| **field**           | **type** | **required** | **description**                                                                                                             |
|---------------------|:--------:|:------------:|-----------------------------------------------------------------------------------------------------------------------------|
| kind                | string   |    true      | Must be "couchbase".                                                                                                         |
| connection_string   | string   |    true      | Connection string for the Couchbase cluster.                                                                                |
| bucket              | string   |    true      | Name of the bucket to connect to.                                                                                           |
| scope               | string   |    true      | Name of the scope within the bucket.                                                                                        |
| username            | string   |    false     | Username for authentication.                                                                                                |
| password            | string   |    false     | Password for authentication.                                                                                                |
| client_cert         | string   |    false     | Path to client certificate file for TLS authentication.                                                                     |
| client_cert_password| string   |    false     | Password for the client certificate.                                                                                        |
| client_key          | string   |    false     | Path to client key file for TLS authentication.                                                                             |
| client_key_password | string   |    false     | Password for the client key.                                                                                                |
| ca_cert             | string   |    false     | Path to CA certificate file.                                                                                                |
| no_ssl_verify       | boolean  |    false     | If true, skip server certificate verification.                                                                              |
| profile             | string   |    false     | Name of the connection profile to apply.                                                                                    |

## Tools

### Couchbase SQL

This source can be used with the "couchbase-sql" tool to execute SQL++ queries against your Couchbase database.

Example:

```yaml
tools:
    list-airports:
        kind: "couchbase-sql"
        source: "my-couchbase-source"
        description: "List airports in a specific country"
        statement: "SELECT * FROM airport WHERE country = $country LIMIT $limit"
        parameters:
            country:
                type: "string"
                description: "Country code (e.g. 'FR' for France)"
                required: true
            limit:
                type: "integer"
                description: "Maximum number of results"
                default: 10
```
