# Cloud SQL for SQL Server Source

[Cloud SQL for SQL Server][csql-mssql-docs] is a managed database service that
helps you set up, maintain, manage, and administer your SQL Server databases on
Google Cloud.

If you are new to Cloud SQL for SQL Server, you can try [creating and connecting
to a database by following these instructions][csql-mssql-connect].

[csql-mssql-docs]: https://cloud.google.com/sql/docs/sqlserver
[csql-mssql-connect]: https://cloud.google.com/sql/docs/sqlserver/connect-overview

## Requirements

### IAM Identity

By default, this source uses the [Cloud SQL Go Connector][csql-go-conn] to
authorize and establish mTLS connections to your Cloud SQL instance. The Go
connector uses your [Application Default Credentials (ADC)][adc] to authorize
your connection to Cloud SQL.

In addition to [setting the ADC for your server][set-adc], you need to ensure the
IAM identity has been given the following IAM roles:

- `roles/cloudsql.client`

[csql-go-conn]: https://github.com/GoogleCloudPlatform/cloud-sql-go-connector
[adc]: https://cloud.google.com/docs/authentication#adc
[set-adc]: https://cloud.google.com/docs/authentication/provide-credentials-adc

### Network Path

Currently, Cloud SQL for SQL Server supports connection over both [private IP][private-ip] and
[public IP][public-ip]. Set the `ipType` parameter in your source
configuration to `public` or `private`.

[private-ip]: https://cloud.google.com/sql/docs/sqlserver/configure-private-ip
[public-ip]: https://cloud.google.com/sql/docs/sqlserver/configure-ip

### Database User

Currently, this source only uses standard authentication. You will need to [create a
SQL Server user][cloud-sql-users] to login to the database with.

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
     ipAddress: localhost
     ipType: public
```

## Reference

| **field** | **type** | **required** | **description**                                                              |
|-----------|:--------:|:------------:|------------------------------------------------------------------------------|
| kind      |  string  |     true     | Must be "cloud-sql-postgres".                                                |
| project   |  string  |     true     | Id of the GCP project that the cluster was created in (e.g. "my-project-id"). |
| region    |  string  |     true     | Name of the GCP region that the cluster was created in (e.g. "us-central1"). |
| instance  |  string  |     true     | Name of the Cloud SQL instance within the cluser (e.g. "my-instance").       |
| database  |  string  |     true     | Name of the Cloud SQL database to connect to (e.g. "my_db").                  |
| ipAddress |  string  |     true     | IP address of the Cloud SQL instance to connect to.|
| ipType   |  string  |      true     | IP Type of the Cloud SQL instance, must be either `public` or `private`. Default: `public`. |
| user      |  string  |     true     | Name of the Postgres user to connect as (e.g. "my-pg-user").                 |
| password  |  string  |     true     | Password of the Postgres user (e.g. "my-password").
