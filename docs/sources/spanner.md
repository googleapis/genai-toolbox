# Spanner Source 

[Spanner][spanner-docs] fully managed, mission-critical database service
that brings together relational, graph, key-value, and search. It offers
transactional consistency at global scale, automatic, synchronous replication
for high availability, and support for two SQL dialects: GoogleSQL (ANSI 2011
with extensions) and PostgreSQL.

If you are new to Spanner, you can try to [create and query a database using
the Google Cloud console][spanner-quickstart].

[spanner-docs]: https://cloud.google.com/spanner/docs
[spanner-quickstart]: https://cloud.google.com/spanner/docs/create-query-database-console

## Requirements 

### IAM Identity
Spanner uses the [OAuth 2.0][oauth2] for API authentication and authorization.
To run your code locally, make sure you run `gcloud auth application-default login` to
set up your local development environment with authentication credentials.

You need to ensure the IAM identity has been given the following IAM permissions:
- `roles/spanner.databaseUser`

[oauth2]: https://datatracker.ietf.org/doc/html/rfc6749

## Example

```yaml
sources:
    my-spanner-source:
        kind: "spanner"
        project: "my-project-name"
        instance: "my-instance"
        dialect: "googlesql"
        database: "my_db"
```

## Reference

| **field** | **type** | **required** | **description**                                                              |
|-----------|:--------:|:------------:|------------------------------------------------------------------------------|
| kind      |  string  |     true     | Must be "spanner".                                                           |
| project   |  string  |     true     | Name of the GCP project that the cluster was created in (e.g. "my-project"). |
| instance  |  string  |     true     | Name of the AlloyDB instance within the cluser (e.g. "my-instance").         |
| dialect   |  string  |     true     | Name of the dialect type of the Spanner database, must be either `googlesql` or `postgresql`. Default: `googlesql`.        |
| database  |  string  |     true     | Name of the Postgres database to connect to (e.g. "my_db").                  |


