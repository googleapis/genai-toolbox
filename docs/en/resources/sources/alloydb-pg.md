---
title: "AlloyDB for PostgreSQL"
linkTitle: "AlloyDB"
type: docs
weight: 1
description: >
  AlloyDB for PostgreSQL is a fully-managed, PostgreSQL-compatible database for
  demanding transactional workloads.

---

## About

[AlloyDB for PostgreSQL][alloydb-docs] is a fully-managed, PostgreSQL-compatible
database for demanding transactional workloads. It provides enterprise-grade
performance and availability while maintaining 100% compatibility with
open-source PostgreSQL.

If you are new to AlloyDB for PostgreSQL, you can [create a free trial
cluster][alloydb-free-trial].

[alloydb-docs]: https://cloud.google.com/alloydb/docs
[alloydb-free-trial]: https://cloud.google.com/alloydb/docs/create-free-trial-cluster

## Available Tools

- [`alloydb-ai-nl`](../tools/alloydbainl/alloydb-ai-nl.md)
  Use natural language queries on AlloyDB, powered by AlloyDB AI.

- [`postgres-sql`](../tools/postgres/postgres-sql.md)
  Execute SQL queries as prepared statements in AlloyDB Postgres.

- [`postgres-execute-sql`](../tools/postgres/postgres-execute-sql.md)
  Run parameterized SQL statements in AlloyDB Postgres.

- [`postgres-list-tables`](../tools/postgres/postgres-list-tables.md)
  List tables in an AlloyDB for PostgreSQL database.

- [`postgres-list-active-queries`](../tools/postgres/postgres-list-active-queries.md)
  List active queries in an AlloyDB for PostgreSQL database.

- [`postgres-list-available-extensions`](../tools/postgres/postgres-list-available-extensions.md)
  List available extensions for installation in a PostgreSQL database.

- [`postgres-list-installed-extensions`](../tools/postgres/postgres-list-installed-extensions.md)
  List installed extensions in a PostgreSQL database.

- [`postgres-list-views`](../tools/postgres/postgres-list-views.md)
  List views in an AlloyDB for PostgreSQL database.

- [`postgres-list-schemas`](../tools/postgres/postgres-list-schemas.md)
  List schemas in an AlloyDB for PostgreSQL database.

- [`postgres-database-overview`](../tools/postgres/postgres-database-overview.md)
  Fetches the current state of the PostgreSQL server.

- [`postgres-list-triggers`](../tools/postgres/postgres-list-triggers.md)
  List triggers in an AlloyDB for PostgreSQL database.

- [`postgres-list-indexes`](../tools/postgres/postgres-list-indexes.md)
  List available user indexes in a PostgreSQL database.

- [`postgres-list-sequences`](../tools/postgres/postgres-list-sequences.md)
  List sequences in a PostgreSQL database.
- [`postgres-long-running-transactions`](../tools/postgres/postgres-long-running-transactions.md)
  List long running transactions in a PostgreSQL database.

- [`postgres-list-locks`](../tools/postgres/postgres-list-locks.md)
  List lock stats in a PostgreSQL database.

- [`postgres-replication-stats`](../tools/postgres/postgres-replication-stats.md)
  List replication stats in a PostgreSQL database.

- [`postgres-list-query-stats`](../tools/postgres/postgres-list-query-stats.md)
  List query statistics in a PostgreSQL database.

- [`postgres-get-column-cardinality`](../tools/postgres/postgres-get-column-cardinality.md)
  List cardinality of columns in a table in a PostgreSQL database.

### Pre-built Configurations

- [AlloyDB using MCP](https://googleapis.github.io/genai-toolbox/how-to/connect-ide/alloydb_pg_mcp/)
Connect your IDE to AlloyDB using Toolbox.

- [AlloyDB Admin API using MCP](https://googleapis.github.io/genai-toolbox/how-to/connect-ide/alloydb_pg_admin_mcp/)
Create your AlloyDB database with MCP Toolbox.

## Requirements

### IAM Permissions

By default, AlloyDB for PostgreSQL source uses the [AlloyDB Go
Connector][alloydb-go-conn] to authorize and establish mTLS connections to your
AlloyDB instance. The Go connector uses your [Application Default Credentials
(ADC)][adc] to authorize your connection to AlloyDB.

In addition to [setting the ADC for your server][set-adc], you need to ensure
the IAM identity has been given the following IAM roles (or corresponding
permissions):

- `roles/alloydb.client`
- `roles/serviceusage.serviceUsageConsumer`

[alloydb-go-conn]: https://github.com/GoogleCloudPlatform/alloydb-go-connector
[adc]: https://cloud.google.com/docs/authentication#adc
[set-adc]: https://cloud.google.com/docs/authentication/provide-credentials-adc

### Networking

AlloyDB supports connecting over both from external networks via the internet
([public IP][public-ip]), and internal networks ([private IP][private-ip]).
For more information on choosing between the two options, see the AlloyDB page
[Connection overview][conn-overview].

You can configure the `ipType` parameter in your source configuration to
`public` or `private` to match your cluster's configuration. Regardless of which
you choose, all connections use IAM-based authorization and are encrypted with
mTLS.

[private-ip]: https://cloud.google.com/alloydb/docs/private-ip
[public-ip]: https://cloud.google.com/alloydb/docs/connect-public-ip
[conn-overview]: https://cloud.google.com/alloydb/docs/connection-overview

### Authentication

This source supports both password-based authentication and IAM
authentication (using your [Application Default Credentials][adc]).

#### Standard Authentication

To connect using user/password, [create
a PostgreSQL user][alloydb-users] and input your credentials in the `user` and
`password` fields.

```yaml
user: ${USER_NAME}
password: ${PASSWORD}
```

#### IAM Authentication

To connect using IAM authentication:

1. Prepare your database instance and user following this [guide][iam-guide].
2. You could choose one of the two ways to log in:
    - Specify your IAM email as the `user`.
    - Leave your `user` field blank. Toolbox will fetch the [ADC][adc]
      automatically and log in using the email associated with it.
3. Leave the `password` field blank.

[iam-guide]: https://cloud.google.com/alloydb/docs/database-users/manage-iam-auth
[alloydb-users]: https://cloud.google.com/alloydb/docs/database-users/about

## Example

```yaml
sources:
    my-alloydb-pg-source:
        kind: alloydb-postgres
        project: my-project-id
        region: us-central1
        cluster: my-cluster
        instance: my-instance
        database: my_db
        user: ${USER_NAME}
        password: ${PASSWORD}
        # ipType: "public"
```

{{< notice tip >}}
Use environment variable replacement with the format ${ENV_NAME}
instead of hardcoding your secrets into the configuration file.
{{< /notice >}}

## Reference

| **field** | **type** | **required** | **description**                                                                                                          |
|-----------|:--------:|:------------:|--------------------------------------------------------------------------------------------------------------------------|
| kind      |  string  |     true     | Must be "alloydb-postgres".                                                                                              |
| project   |  string  |     true     | Id of the GCP project that the cluster was created in (e.g. "my-project-id").                                            |
| region    |  string  |     true     | Name of the GCP region that the cluster was created in (e.g. "us-central1").                                             |
| cluster   |  string  |     true     | Name of the AlloyDB cluster (e.g. "my-cluster").                                                                         |
| instance  |  string  |     true     | Name of the AlloyDB instance within the cluster (e.g. "my-instance").                                                    |
| database  |  string  |     true     | Name of the Postgres database to connect to (e.g. "my_db").                                                              |
| user      |  string  |    false     | Name of the Postgres user to connect as (e.g. "my-pg-user"). Defaults to IAM auth using [ADC][adc] email if unspecified. |
| password  |  string  |    false     | Password of the Postgres user (e.g. "my-password"). Defaults to attempting IAM authentication if unspecified.            |
| ipType    |  string  |    false     | IP Type of the AlloyDB instance; must be one of `public` or `private`. Default: `public`.                                |
