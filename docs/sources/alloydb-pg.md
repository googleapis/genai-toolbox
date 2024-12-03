# AlloyDB for PostgreSQL Source 

[AlloyDB for PostgreSQL][alloydb-docs] is a fully-managed, PostgreSQL-compatible
database for demanding transactional workloads. It provides enterprise-grade
performance and availability while maintaining 100% compatibility with
open-source PostgreSQL.

If you are new to AlloyDB for PostgreSQL, you can try [creating and connecting
to a database by following these instructions][alloydb-quickstart].

[alloydb-docs]: https://cloud.google.com/alloydb/docs
[alloydb-quickstart]: https://cloud.google.com/alloydb/docs/quickstart/create-and-connect

## Requirements 

### IAM Identity
By default, AlloyDB for PostgreSQL source uses the [AlloyDB Go
Connector][alloydb-go-conn] to authorize and establish mTLS connections to your
AlloyDB instance. The Go connector uses your [Application Default Credentials
(ADC)][adc] to authorize your connection to AlloyDB. 

In addition to [setting the ADC for your server][set-adc], you need to ensure the
IAM identity has been given the following IAM permissions:
- `roles/alloydb.client`
- `roles/serviceusage.serviceUsageConsumer`

[alloydb-go-conn]: https://github.com/GoogleCloudPlatform/alloydb-go-connector
[adc]: https://cloud.google.com/docs/authentication#adc
[set-adc]: https://cloud.google.com/docs/authentication/provide-credentials-adc

### Network Path

Currently, this source only supports [connecting over Private
IP][private-ip]. Most notably, this means
you need to connect from a VPC that AlloyDB has been connected to. 

[private-ip]: https://cloud.google.com/alloydb/docs/private-ip

### Database User

Current, this source only uses standard authentication. You will need to [create a
PostreSQL user][alloydb-users] to login to the database with. 

[alloydb-users]: https://cloud.google.com/alloydb/docs/database-users/about

## Example

```yaml
sources:
    my-alloydb-pg-source:
        kind: "alloydb-postgres"
        project: "my-project-name"
        region: "us-central1"
        cluster: "my-cluster"
        instance: "my-instance"
        database: "my_db"
        user: "my-user"
        password: "my-password"
```

## Reference

| **field** | **type** | **required** | **description**                                                              |
|-----------|:--------:|:------------:|------------------------------------------------------------------------------|
| kind      |  string  |     true     | Must be "alloydb-postgres".                                                  |
| project   |  string  |     true     | Name of the GCP project that the cluster was created in (e.g. "my-project"). |
| region    |  string  |     true     | Name of the GCP region that the cluster was created in (e.g. "us-central1"). |
| cluster   |  string  |     true     | Name of the AlloyDB cluster (e.g. "my-cluster").                             |
| instance  |  string  |     true     | Name of the AlloyDB instance within the cluser (e.g. "my-instance").         |
| database  |  string  |     true     | Name of the Postgres database to connect to (e.g. "my_db").                  |
| user      |  string  |     true     | Name of the Postgres user to connect as (e.g. "my-pg-user").                 |
| password  |  string  |     true     | Password of the Postgres user (e.g. "my-password").                          |


