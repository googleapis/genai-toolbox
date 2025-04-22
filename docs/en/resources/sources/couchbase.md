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
        connectionString: couchbase://localhost:8091
        bucket: travel-sample
        scope: inventory
        username: Administrator
        password: password
```

## Reference

| **field**           | **type** | **required** | **description**                                                                                                             |
|---------------------|:--------:|:------------:|-----------------------------------------------------------------------------------------------------------------------------|
| kind                | string   |    true      | Must be "couchbase".                                                                                                         |
| connectionString   | string   |    true      | Connection string for the Couchbase cluster.                                                                                |
| bucket              | string   |    true      | Name of the bucket to connect to.                                                                                           |
| scope               | string   |    true      | Name of the scope within the bucket.                                                                                        |
| username            | string   |    false     | Username for authentication.                                                                                                |
| password            | string   |    false     | Password for authentication.                                                                                                |
| clientCert         | string   |    false     | Path to client certificate file for TLS authentication.                                                                     |
| clientCertPassword| string   |    false     | Password for the client certificate.                                                                                        |
| clientKey          | string   |    false     | Path to client key file for TLS authentication.                                                                             |
| clientKeyPassword | string   |    false     | Password for the client key.                                                                                                |
| caCert             | string   |    false     | Path to CA certificate file.                                                                                                |
| noSslVerify       | boolean  |    false     | If true, skip server certificate verification. **Warning:** This option should only be used in development or testing environments. Disabling SSL verification poses significant security risks in production as it makes your connection vulnerable to man-in-the-middle attacks. |
| profile             | string   |    false     | Name of the connection profile to apply.                                                                                    |
| queryScanConsistency | integer  |    false     | Query scan consistency. Controls the consistency guarantee for index scanning. Values: 1 for "not_bounded" (fastest option, but results may not include the most recent operations), 2 for "request_plus" (highest consistency level, includes all operations up until the query started, but incurs a performance penalty). If not specified, uses the server's default setting. |
