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
[csql-pg-quickstart]: https://cloud.google.com/sql/docs/postgres/connect-instance-local-compaduter

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

> **_NOTE:_**
> If you are connecting from Compute Engine, make sure your VM also has the 
> [proper scope][gce-access-scopes] to connect using the Cloud SQL Admin API.

[csql-go-conn]: https://github.com/GoogleCloudPlatform/cloud-sql-go-connector
[adc]: https://cloud.google.com/docs/authentication#adc
[set-adc]: https://cloud.google.com/docs/authentication/provide-credentials-adc
[gce-access-scopes]: https://cloud.google.com/compute/docs/access/service-accounts#accesscopesiam

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

### Database User

Current, this source only uses standard authentication. You will need to [create
a PostreSQL user][cloud-sql-users] to login to the database with.

[cloud-sql-users]: https://cloud.google.com/sql/docs/postgres/create-manage-users

## Example

```yaml
sources:
    my-cloud-sql-pg-source:
        kind: "cloud-sql-postgres"
        project: "my-project-id"
        region: "us-central1"
        instance: "my-instance"
        database: "my_db"
        user: "my-user"
        password: "my-password"
        # ipType: "public"
```

## Reference

| **field** | **type** | **required** | **description**                                                                             |
|-----------|:--------:|:------------:|---------------------------------------------------------------------------------------------|
| kind      |  string  |     true     | Must be "cloud-sql-postgres".                                                               |
| project   |  string  |     true     | Id of the GCP project that the cluster was created in (e.g. "my-project-id").               |
| region    |  string  |     true     | Name of the GCP region that the cluster was created in (e.g. "us-central1").                |
| instance  |  string  |     true     | Name of the Cloud SQL instance within the cluster (e.g. "my-instance").                     |
| database  |  string  |     true     | Name of the Postgres database to connect to (e.g. "my_db").                                 |
| user      |  string  |     true     | Name of the Postgres user to connect as (e.g. "my-pg-user").                                |
| password  |  string  |     true     | Password of the Postgres user (e.g. "my-password").                                         |
| ipType    |  string  |    false     | IP Type of the Cloud SQL instance; must be one of `public` or `private`. Default: `public`. |
