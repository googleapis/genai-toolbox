---
title: "Cloud SQL for MySQL"
linkTitle: "Cloud SQL (MySQL)"
type: docs
weight: 1
description: >
  Cloud SQL for MySQL is a fully-managed database service for MySQL.

---

## About

[Cloud SQL for MySQL][csql-mysql-docs] is a fully-managed database service
that helps you set up, maintain, manage, and administer your MySQL
relational databases on Google Cloud Platform.

If you are new to Cloud SQL for MySQL, you can try [creating and connecting
to a database by following these instructions][csql-mysql-quickstart].

[csql-mysql-docs]: https://cloud.google.com/sql/docs/mysql
[csql-mysql-quickstart]: https://cloud.google.com/sql/docs/mysql/connect-instance-local-computer

## Available Tools

- [`mysql-sql`](../tools/mysql/mysql-sql.md)
  Execute pre-defined prepared SQL queries in Cloud SQL for MySQL.

- [`mysql-execute-sql`](../tools/mysql/mysql-execute-sql.md)
  Run parameterized SQL queries in Cloud SQL for MySQL.

- [`mysql-list-active-queries`](../tools/mysql/mysql-list-active-queries.md)
  List active queries in Cloud SQL for MySQL.

- [`mysql-list-tables`](../tools/mysql/mysql-list-tables.md)
  List tables in a Cloud SQL for MySQL database.

- [`mysql-list-tables-missing-unique-indexes`](../tools/mysql/mysql-list-tables-missing-unique-indexes.md)
  List tables in a Cloud SQL for MySQL database that do not have primary or unique indices.

- [`mysql-list-table-fragmentation`](../tools/mysql/mysql-list-table-fragmentation.md)
  List table fragmentation in Cloud SQL for MySQL tables.

### Pre-built Configurations

- [Cloud SQL for MySQL using
  MCP](https://googleapis.github.io/genai-toolbox/how-to/connect-ide/cloud_sql_mysql_mcp/)
  Connect your IDE to Cloud SQL for MySQL using Toolbox.

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

[csql-go-conn]: https://github.com/GoogleCloudPlatform/cloud-sql-go-connector
[adc]: https://cloud.google.com/docs/authentication#adc
[set-adc]: https://cloud.google.com/docs/authentication/provide-credentials-adc

### Networking

Cloud SQL supports connecting over both from external networks via the internet
([public IP][public-ip]), and internal networks ([private IP][private-ip]).
For more information on choosing between the two options, see the Cloud SQL page
[Connection overview][conn-overview].

You can configure the `ipType` parameter in your source configuration to
`public` or `private` to match your cluster's configuration. Regardless of which
you choose, all connections use IAM-based authorization and are encrypted with
mTLS.

[private-ip]: https://cloud.google.com/sql/docs/mysql/configure-private-ip
[public-ip]: https://cloud.google.com/sql/docs/mysql/configure-ip
[conn-overview]: https://cloud.google.com/sql/docs/mysql/connect-overview

### Authentication

This source supports both password-based authentication and IAM
authentication (using your [Application Default Credentials][adc]).

#### Standard Authentication

To connect using user/password, [create
a MySQL user][cloud-sql-users] and input your credentials in the `user` and
`password` fields.

```yaml
user: ${USER_NAME}
password: ${PASSWORD}
```

[cloud-sql-users]: https://cloud.google.com/sql/docs/mysql/create-manage-users

#### IAM Authentication

To connect using IAM authentication:

1. Prepare your database instance and user following this [guide][iam-guide].
2. You could choose one of the two ways to log in:
    - Specify your IAM email as the `user`.
    - Leave your `user` field blank. Toolbox will fetch the [ADC][adc]
      automatically and log in using the email associated with it.

3. Leave the `password` field blank.

[iam-guide]: https://cloud.google.com/sql/docs/mysql/iam-logins
[cloudsql-users]: https://cloud.google.com/sql/docs/mysql/create-manage-users


## Example

```yaml
sources:
    my-cloud-sql-mysql-source:
        kind: cloud-sql-mysql
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

| **field** | **type** | **required** | **description**                                                                                      |
|-----------|:--------:|:------------:|------------------------------------------------------------------------------------------------------|
| kind      |  string  |     true     | Must be "cloud-sql-mysql".                                                                           |
| project   |  string  |     true     | Id of the GCP project that the cluster was created in (e.g. "my-project-id").                        |
| region    |  string  |     true     | Name of the GCP region that the cluster was created in (e.g. "us-central1").                         |
| instance  |  string  |     true     | Name of the Cloud SQL instance within the cluster (e.g. "my-instance").                              |
| database  |  string  |     true     | Name of the MySQL database to connect to (e.g. "my_db").                                             |
| user      |  string  |     false     | Name of the MySQL user to connect as (e.g "my-mysql-user"). Defaults to IAM auth using [ADC][adc] email if unspecified.                                            |
| password  |  string  |     false     | Password of the MySQL user (e.g. "my-password"). Defaults to attempting IAM authentication if unspecified.                                                    |
| ipType    |  string  |    false     | IP Type of the Cloud SQL instance, must be either `public`,  `private`, or `psc`. Default: `public`. |
