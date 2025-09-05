---
title: "Bigtable"
type: docs
weight: 1
description: >
  Bigtable is a low-latency NoSQL database service for machine learning, operational analytics, and user-facing operations. It's a wide-column, key-value store that can scale to billions of rows and thousands of columns. With Bigtable, you can replicate your data to regions across the world for high availability and data resiliency.

---

# Bigtable Source

[Bigtable][bigtable-docs] is a low-latency NoSQL database service for machine
learning, operational analytics, and user-facing operations. It's a wide-column,
key-value store that can scale to billions of rows and thousands of columns.
With Bigtable, you can replicate your data to regions across the world for high
availability and data resiliency.

If you are new to Bigtable, you can try to [create an instance and write data
with the cbt CLI][bigtable-quickstart-with-cli].

You can use [GoogleSQL statements][bigtable-googlesql] to query your Bigtable
data. GoogleSQL is an ANSI-compliant structured query language (SQL) that is
also implemented for other Google Cloud services. SQL queries are handled by
cluster nodes in the same way as NoSQL data requests. Therefore, the same best
practices apply when creating SQL queries to run against your Bigtable data,
such as avoiding full table scans or complex filters.

[bigtable-docs]: https://cloud.google.com/bigtable/docs
[bigtable-quickstart-with-cli]:
    https://cloud.google.com/bigtable/docs/create-instance-write-data-cbt-cli

[bigtable-googlesql]:
    https://cloud.google.com/bigtable/docs/googlesql-overview

## Available Tools

- [`bigtable-sql`](../tools/bigtable/bigtable-sql.md)
  Run SQL-like queries over Bigtable rows.

## Requirements

### IAM Permissions

Bigtable uses [Identity and Access Management (IAM)][iam-overview] to control
user and group access to Bigtable resources at the project, instance, table, and
backup level. Toolbox will use your [Application Default Credentials (ADC)][adc]
to authorize and authenticate when interacting with [Bigtable][bigtable-docs].

In addition to [setting the ADC for your server][set-adc], you need to ensure
the IAM identity has been given the correct IAM permissions for the query
provided. See [Apply IAM roles][grant-permissions] for more information on
applying IAM permissions and roles to an identity.

[iam-overview]: https://cloud.google.com/bigtable/docs/access-control
[adc]: https://cloud.google.com/docs/authentication#adc
[set-adc]: https://cloud.google.com/docs/authentication/provide-credentials-adc
[grant-permissions]: https://cloud.google.com/bigtable/docs/access-control#iam-management-instance

## Example

```yaml
sources:
  my-bigtable-source:
    kind: "bigtable"
    project: "my-project-id"
    instance: "test-instance"
```

## Reference

| **field** | **type** | **required** | **description**                                                               |
|-----------|:--------:|:------------:|-------------------------------------------------------------------------------|
| type      |  string  |     true     | Must be "bigtable".                                                           |
| project   |  string  |     true     | Id of the GCP project that the cluster was created in (e.g. "my-project-id"). |
| instance  |  string  |     true     | Name of the Bigtable instance.                                                |
