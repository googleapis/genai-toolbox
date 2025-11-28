---
title: "Cloud SQL for PostgreSQL"
linkTitle: "Cloud SQL (Postgres)"
type: docs
weight: 1
description: >
  Cloud SQL for PostgreSQL is a fully-managed database service for Postgres.

---

## About

[Cloud SQL for PostgreSQL][csql-pg-docs] is a fully-managed database service
that helps you set up, maintain, manage, and administer your PostgreSQL
relational databases on Google Cloud Platform.

If you are new to Cloud SQL for PostgreSQL, you can try [creating and connecting
to a database by following these instructions][csql-pg-quickstart].

[csql-pg-docs]: https://cloud.google.com/sql/docs/postgres
[csql-pg-quickstart]:
    https://cloud.google.com/sql/docs/postgres/connect-instance-local-computer

## Available Tools

- [`postgres-sql`](../tools/postgres/postgres-sql.md)
  Execute SQL queries as prepared statements in PostgreSQL.

- [`postgres-execute-sql`](../tools/postgres/postgres-execute-sql.md)
  Run parameterized SQL statements in PostgreSQL.

- [`postgres-list-tables`](../tools/postgres/postgres-list-tables.md)
  List tables in a PostgreSQL database.

- [`postgres-list-active-queries`](../tools/postgres/postgres-list-active-queries.md)
  List active queries in a PostgreSQL database.

- [`postgres-list-available-extensions`](../tools/postgres/postgres-list-available-extensions.md)
  List available extensions for installation in a PostgreSQL database.

- [`postgres-list-installed-extensions`](../tools/postgres/postgres-list-installed-extensions.md)
  List installed extensions in a PostgreSQL database.

- [`postgres-list-views`](../tools/postgres/postgres-list-views.md)
  List views in a PostgreSQL database.

- [`postgres-list-schemas`](../tools/postgres/postgres-list-schemas.md)
  List schemas in a PostgreSQL database.

- [`postgres-database-overview`](../tools/postgres/postgres-database-overview.md)
  Fetches the current state of the PostgreSQL server.

- [`postgres-list-triggers`](../tools/postgres/postgres-list-triggers.md)
  List triggers in a PostgreSQL database.

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

- [Cloud SQL for Postgres using
  MCP](https://googleapis.github.io/genai-toolbox/how-to/connect-ide/cloud_sql_pg_mcp/)
Connect your IDE to Cloud SQL for Postgres using Toolbox.

## Requirements

### IAM Permissions

By default, this source uses the [Cloud SQL Go Connector][csql-go-conn] to
authorize and establish mTLS connections to your Cloud SQL instance. The Go
connector uses your [Application Default Credentials (ADC)][adc] to authorize
your connection to Cloud SQL.

In addition to [setting the ADC for your server][set-adc], you need to ensure
the IAM identity has been given the following IAM roles (or corresponding
permissions):

- `roles/cloudsql.client`

{{< notice tip >}}
If you are connecting from Compute Engine, make sure your VM
also has the [proper
scope](https://cloud.google.com/compute/docs/access/service-accounts#accesscopesiam)
to connect using the Cloud SQL Admin API.
{{< /notice >}}

[csql-go-conn]: <https://github.com/GoogleCloudPlatform/cloud-sql-go-connector>
[adc]: <https://cloud.google.com/docs/authentication#adc>
[set-adc]: <https://cloud.google.com/docs/authentication/provide-credentials-adc>

### Networking

Cloud SQL supports connecting over both from external networks via the internet
([public IP][public-ip]), and internal networks ([private IP][private-ip]).
For more information on choosing between the two options, see the Cloud SQL page
[Connection overview][conn-overview].

You can configure the `ipType` parameter in your source configuration to
`public` or `private` to match your cluster's configuration. Regardless of which
you choose, all connections use IAM-based authorization and are encrypted with
mTLS.

[private-ip]: https://cloud.google.com/sql/docs/postgres/configure-private-ip
[public-ip]: https://cloud.google.com/sql/docs/postgres/configure-ip
[conn-overview]: https://cloud.google.com/sql/docs/postgres/connect-overview

### Authentication

This source supports both password-based authentication and IAM
authentication (using your [Application Default Credentials][adc]).

#### Standard Authentication

To connect using user/password, [create
a PostgreSQL user][cloudsql-users] and input your credentials in the `user` and
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

[iam-guide]: https://cloud.google.com/sql/docs/postgres/iam-logins
[cloudsql-users]: https://cloud.google.com/sql/docs/postgres/create-manage-users

## Example

```yaml
sources:
    my-cloud-sql-pg-source:
        kind: cloud-sql-postgres
        project: my-project-id
        region: us-central1
        instance: my-instance
        database: my_db
        user: ${USER_NAME}
        password: ${PASSWORD}
        # ipType: "private"
```

{{< notice tip >}}
Use environment variable replacement with the format ${ENV_NAME}
instead of hardcoding your secrets into the configuration file.
{{< /notice >}}

## Reference

| **field** | **type** | **required** | **description**                                                                                                          |
|-----------|:--------:|:------------:|--------------------------------------------------------------------------------------------------------------------------|
| kind      |  string  |     true     | Must be "cloud-sql-postgres".                                                                                            |
| project   |  string  |     true     | Id of the GCP project that the cluster was created in (e.g. "my-project-id").                                            |
| region    |  string  |     true     | Name of the GCP region that the cluster was created in (e.g. "us-central1").                                             |
| instance  |  string  |     true     | Name of the Cloud SQL instance within the cluster (e.g. "my-instance").                                                  |
| database  |  string  |     true     | Name of the Postgres database to connect to (e.g. "my_db").                                                              |
| user      |  string  |    false     | Name of the Postgres user to connect as (e.g. "my-pg-user"). Defaults to IAM auth using [ADC][adc] email if unspecified. |
| password  |  string  |    false     | Password of the Postgres user (e.g. "my-password"). Defaults to attempting IAM authentication if unspecified.            |
| ipType    |  string  |    false     | IP Type of the Cloud SQL instance; must be one of `public`, `private`, or `psc`. Default: `public`.                      |
