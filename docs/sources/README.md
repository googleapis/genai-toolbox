# Sources

A Source represents a data sources that a tool can interact with. You can define
Sources as a map in the `sources` section of your `tools.yaml` file. Typically,
a source configuration will contain any information needed to connect with and
interact with the database.

```yaml
sources:
    my-cloud-sql-source:
        kind: cloud-sql-postgres
        project: my-project-name
        region: us-central1
        instance: my-instance-name
        database: my_db
        user: my-user
        password: my-password
```

In implementation, each source is a different connection pool or client that used
to connect to the database and execute the tool. 

## Kinds of Sources

We currently support the following types of kinds of sources:

* [alloydb-postgres](./alloydb-pg.md) - Connect to an AlloyDB for PostgreSQL
  cluster.
* [cloud-sql-postgres](./cloud-sql-pg.md) - Connect to a Cloud SQL for
  PostgreSQL instance.
* [postgres](./postgres.md) - Connect to any PostgreSQL compatible database.
* [spanner](./spanner.md) - Connect to a Spanner database.
