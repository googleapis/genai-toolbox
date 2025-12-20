---
title: "Spanner"
type: docs
weight: 1
description: >
  Spanner is a fully managed database service from Google Cloud that combines 
  relational, key-value, graph, and search capabilities.

---

# Spanner Source

[Spanner][spanner-docs] is a fully managed, mission-critical database service
that brings together relational, graph, key-value, and search. It offers
transactional consistency at global scale, automatic, synchronous replication
for high availability, and support for two SQL dialects: GoogleSQL (ANSI 2011
with extensions) and PostgreSQL.

If you are new to Spanner, you can try to [create and query a database using
the Google Cloud console][spanner-quickstart].

[spanner-docs]: https://cloud.google.com/spanner/docs
[spanner-quickstart]:
    https://cloud.google.com/spanner/docs/create-query-database-console

## Available Tools

- [`spanner-sql`](../tools/spanner/spanner-sql.md)  
  Execute SQL on Google Cloud Spanner.

- [`spanner-execute-sql`](../tools/spanner/spanner-execute-sql.md)  
  Run structured and parameterized queries on Spanner.

- [`spanner-list-tables`](../tools/spanner/spanner-list-tables.md)  
  Retrieve schema information about tables in a Spanner database.

- [`spanner-list-graphs`](../tools/spanner/spanner-list-graphs.md)  
  Retrieve schema information about graphs in a Spanner database.

### Pre-built Configurations

- [Spanner using MCP](https://googleapis.github.io/genai-toolbox/how-to/connect-ide/spanner_mcp/)  
Connect your IDE to Spanner using Toolbox.

## Requirements

### IAM Permissions

Spanner uses [Identity and Access Management (IAM)][iam-overview] to control
user and group access to Spanner resources at the project, Spanner instance, and
Spanner database levels. Toolbox will use your [Application Default Credentials
(ADC)][adc] to authorize and authenticate when interacting with Spanner.

In addition to [setting the ADC for your server][set-adc], you need to ensure
the IAM identity has been given the correct IAM permissions for the query
provided. See [Apply IAM roles][grant-permissions] for more information on
applying IAM permissions and roles to an identity.

[iam-overview]: https://cloud.google.com/spanner/docs/iam
[adc]: https://cloud.google.com/docs/authentication#adc
[set-adc]: https://cloud.google.com/docs/authentication/provide-credentials-adc
[grant-permissions]: https://cloud.google.com/spanner/docs/grant-permissions

## Example

```yaml
sources:
    my-spanner-source:
        kind: "spanner"
        project: "my-project-id"
        instance: "my-instance"
        database: "my_db"
        # dialect: "googlesql"
```

## Reference

| **field** | **type** | **required** | **description**                                                                                                     |
|-----------|:--------:|:------------:|---------------------------------------------------------------------------------------------------------------------|
| kind      |  string  |     true     | Must be "spanner".                                                                                                  |
| project   |  string  |     true     | Id of the GCP project that the cluster was created in (e.g. "my-project-id").                                       |
| instance  |  string  |     true     | Name of the Spanner instance.                                                                                       |
| database  |  string  |     true     | Name of the database on the Spanner instance                                                                        |
| dialect   |  string  |    false     | Name of the dialect type of the Spanner database, must be either `googlesql` or `postgresql`. Default: `googlesql`. |
