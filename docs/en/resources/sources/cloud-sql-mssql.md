---
title: "Cloud SQL for SQL Server"
linkTitle: "Cloud SQL (SQL Server)"
type: docs
weight: 1
description: >
  Cloud SQL for SQL Server is a fully-managed database service for SQL Server.
---

## About

[Cloud SQL for SQL Server][csql-mssql-docs] is a managed database service that
helps you set up, maintain, manage, and administer your SQL Server databases on
Google Cloud.

If you are new to Cloud SQL for SQL Server, you can try [creating and connecting
to a database by following these instructions][csql-mssql-connect].

[csql-mssql-docs]: https://cloud.google.com/sql/docs/sqlserver
[csql-mssql-connect]: https://cloud.google.com/sql/docs/sqlserver/connect-overview

## Available Tools

- [`mssql-sql`](../tools/mssql/mssql-sql.md)  
  Execute pre-defined SQL Server queries with placeholder parameters.

- [`mssql-execute-sql`](../tools/mssql/mssql-execute-sql.md)  
  Run parameterized SQL Server queries in Cloud SQL for SQL Server.

- [`mssql-list-tables`](../tools/mssql/mssql-list-tables.md)  
  List tables in a Cloud SQL for SQL Server database.

### Pre-built Configurations

- [Cloud SQL for SQL Server using MCP](https://googleapis.github.io/genai-toolbox/how-to/connect-ide/cloud_sql_mssql_mcp/)  
Connect your IDE to Cloud SQL for SQL Server using Toolbox.

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

[private-ip]: https://cloud.google.com/sql/docs/sqlserver/configure-private-ip
[public-ip]: https://cloud.google.com/sql/docs/sqlserver/configure-ip
[conn-overview]: https://cloud.google.com/sql/docs/sqlserver/connect-overview

### Database User

Currently, this source only uses standard authentication. You will need to
[create a SQL Server user][cloud-sql-users] to login to the database with.

[cloud-sql-users]: https://cloud.google.com/sql/docs/sqlserver/create-manage-users

## Example

```yaml
sources:
    my-cloud-sql-mssql-instance:
     kind: cloud-sql-mssql
     project: my-project
     region: my-region
     instance: my-instance
     database: my_db
     user: ${USER_NAME}
     password: ${PASSWORD}
     # ipType: private
```

{{< notice tip >}}
Use environment variable replacement with the format ${ENV_NAME}
instead of hardcoding your secrets into the configuration file.
{{< /notice >}}

## Reference

| **field** | **type** | **required** | **description**                                                                                      |
|-----------|:--------:|:------------:|------------------------------------------------------------------------------------------------------|
| kind      |  string  |     true     | Must be "cloud-sql-mssql".                                                                           |
| project   |  string  |     true     | Id of the GCP project that the cluster was created in (e.g. "my-project-id").                        |
| region    |  string  |     true     | Name of the GCP region that the cluster was created in (e.g. "us-central1").                         |
| instance  |  string  |     true     | Name of the Cloud SQL instance within the cluster (e.g. "my-instance").                              |
| database  |  string  |     true     | Name of the Cloud SQL database to connect to (e.g. "my_db").                                         |
| user      |  string  |     true     | Name of the SQL Server user to connect as (e.g. "my-pg-user").                                       |
| password  |  string  |     true     | Password of the SQL Server user (e.g. "my-password").                                                |
| ipType    |  string  |    false     | IP Type of the Cloud SQL instance, must be either `public`,  `private`, or `psc`. Default: `public`. |
