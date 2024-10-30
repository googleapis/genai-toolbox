# Cloud SQL for PostgreSQL Source 

[Cloud SQL for PostgreSQL][csql-pg-docs] is a fully-managed database service
that helps you set up, maintain, manage, and administer your PostgreSQL
relational databases on Google Cloud Platform.

[csql-pg-docs]: https://cloud.google.com/sql/docs/postgres

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

Currently, this source only supports [connecting over Public IP][public-ip].
Because it uses the Go connector, is uses rotating client certificates to
establish a secure mTLS connection with the instance.

[public-ip]: https://cloud.google.com/sql/docs/postgres/configure-ip

### Database User

Current, this source only uses standard authentication. You will need to [create a
PostreSQL user][cloud-sql-users] to login to the database with. 

[cloud-sql-users]: https://cloud.google.com/sql/docs/postgres/create-manage-users

## Example

```yaml
sources:
    my-cloud-sql-pg-source:
        kind: "cloud-sql-pg-postgres"
        project: "my-project"
        region: "us-central1"
        instance: "my-instance"
        database: "my_db"
        user: "my-user"
        password: "my-password"
```

## Reference

| **field** | **type** | **required** | **description**                                                              |
|-----------|:--------:|:------------:|------------------------------------------------------------------------------|
| kind      |  string  |     true     | Must be "cloud-sql-postgres".                                                |
| project   |  string  |     true     | Name of the GCP project that the cluster was created in (e.g. "my-project"). |
| region    |  string  |     true     | Name of the GCP region that the cluster was created in (e.g. "us-central1"). |
| instance  |  string  |     true     | Name of the Cloud SQL instance within the cluser (e.g. "my-instance").       |
| database  |  string  |     true     | Name of the Postgres database to connect to (e.g. "my_db").                  |
| user      |  string  |     true     | Name of the Postgres user to connect as (e.g. "my-pg-user").                 |
| password  |  string  |     true     | Password of the Postgres user (e.g. "my-password").                          |


