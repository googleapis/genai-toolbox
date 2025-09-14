---
title: "BigQuery"
type: docs
weight: 1
description: >
  BigQuery is Google Cloud's fully managed, petabyte-scale, and cost-effective
  analytics data warehouse that lets you run analytics over vast amounts of 
  data in near real time. With BigQuery, there's no infrastructure to set 
  up or manage, letting you focus on finding meaningful insights using 
  GoogleSQL and taking advantage of flexible pricing models across on-demand 
  and flat-rate options.
---

# BigQuery Source

[BigQuery][bigquery-docs] is Google Cloud's fully managed, petabyte-scale,
and cost-effective analytics data warehouse that lets you run analytics
over vast amounts of data in near real time. With BigQuery, there's no
infrastructure to set up or manage, letting you focus on finding meaningful
insights using GoogleSQL and taking advantage of flexible pricing models
across on-demand and flat-rate options.

If you are new to BigQuery, you can try to
[load and query data with the bq tool][bigquery-quickstart-cli].

BigQuery uses [GoogleSQL][bigquery-googlesql] for querying data. GoogleSQL
is an ANSI-compliant structured query language (SQL) that is also implemented
for other Google Cloud services. SQL queries are handled by cluster nodes
in the same way as NoSQL data requests. Therefore, the same best practices
apply when creating SQL queries to run against your BigQuery data, such as
avoiding full table scans or complex filters.

[bigquery-docs]: https://cloud.google.com/bigquery/docs
[bigquery-quickstart-cli]: https://cloud.google.com/bigquery/docs/quickstarts/quickstart-command-line
[bigquery-googlesql]: https://cloud.google.com/bigquery/docs/reference/standard-sql/

## Available Tools

- [`bigquery-conversational-analytics`](../tools/bigquery/bigquery-conversational-analytics.md)
  Allows conversational interaction with a BigQuery source.

- [`bigquery-execute-sql`](../tools/bigquery/bigquery-execute-sql.md)  
  Execute structured queries using parameters.

- [`bigquery-forecast`](../tools/bigquery/bigquery-forecast.md)
  Forecasts time series data in BigQuery.

- [`bigquery-get-dataset-info`](../tools/bigquery/bigquery-get-dataset-info.md)  
  Retrieve metadata for a specific dataset.

- [`bigquery-get-table-info`](../tools/bigquery/bigquery-get-table-info.md)  
  Retrieve metadata for a specific table.

- [`bigquery-list-dataset-ids`](../tools/bigquery/bigquery-list-dataset-ids.md)  
  List available dataset IDs.

- [`bigquery-list-table-ids`](../tools/bigquery/bigquery-list-table-ids.md)  
  List tables in a given dataset.

- [`bigquery-sql`](../tools/bigquery/bigquery-sql.md)  
  Run SQL queries directly against BigQuery datasets.

### Pre-built Configurations

- [BigQuery using MCP](https://googleapis.github.io/genai-toolbox/how-to/connect-ide/bigquery_mcp/)  
Connect your IDE to BigQuery using Toolbox.

## Requirements

### IAM Permissions

BigQuery uses [Identity and Access Management (IAM)][iam-overview] to control
user and group access to BigQuery resources like projects, datasets, and tables.

### Authentication via Application Default Credentials (ADC)

By **default**, Toolbox will use your [Application Default Credentials (ADC)][adc] to authorize and authenticate when interacting with [BigQuery][bigquery-docs].

When using this method, you need to ensure the IAM identity associated with your
ADC (such as a service account) has the correct permissions for the queries you
intend to run. Common roles include `roles/bigquery.user` (which includes
permissions to run jobs and read data) or `roles/bigbigquery.dataViewer`.
Follow this [guide][set-adc] to set up your ADC.

### Authentication via User's OAuth Access Token

If the `useClientOAuth` parameter is set to `true`, Toolbox will instead use the
OAuth access token for authentication. This token is parsed from the
`Authorization` header passed in with the tool invocation request. This method
allows Toolbox to make queries to [BigQuery][bigquery-docs] on behalf of the
client or the end-user.

When using this on-behalf-of authentication, you must ensure that the
identity used has been granted the correct IAM permissions.

[iam-overview]: <https://cloud.google.com/bigquery/docs/access-control>
[adc]: <https://cloud.google.com/docs/authentication#adc>
[set-adc]: <https://cloud.google.com/docs/authentication/provide-credentials-adc>

## Example

Initialize a BigQuery source that uses ADC:

```yaml
sources:
  my-bigquery-source:
    kind: "bigquery"
    project: "my-project-id"
    # location: "US" # Optional: Specifies the location for query jobs.
    # allowedDatasets: # Optional: Restricts tool access to a specific list of datasets.
    #   - "my_dataset_1"
    #   - "other_project.my_dataset_2"
```

Initialize a BigQuery source that uses the client's access token:

```yaml
sources:
  my-bigquery-client-auth-source:
    kind: "bigquery"
    project: "my-project-id"
    useClientOAuth: true
    # location: "US" # Optional: Specifies the location for query jobs.
    # allowedDatasets: # Optional: Restricts tool access to a specific list of datasets.
    #   - "my_dataset_1"
    #   - "other_project.my_dataset_2"
```

## Reference

| **field**       | **type** | **required** | **description**                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |
|-----------------|:--------:|:------------:|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| kind            |  string  |     true     | Must be "bigquery".                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                 |
| project         |  string  |     true     | Id of the Google Cloud project to use for billing and as the default project for BigQuery resources.                                                                                                                                                                                                                                                                                                                                                                                                                |
| location        |  string  |    false     | Specifies the location (e.g., 'us', 'asia-northeast1') in which to run the query job. This location must match the location of any tables referenced in the query. Defaults to the table's location or 'US' if the location cannot be determined. [Learn More](https://cloud.google.com/bigquery/docs/locations)                                                                                                                                                                                                    |
| allowedDatasets | []string |    false     | An optional list of dataset IDs that tools using this source are allowed to access. If provided, any tool operation attempting to access a dataset not in this list will be rejected. To enforce this, two types of operations are also disallowed: 1) Dataset-level operations (e.g., `CREATE SCHEMA`), and 2) operations where table access cannot be statically analyzed (e.g., `EXECUTE IMMEDIATE`, `CREATE PROCEDURE`). If a single dataset is provided, it will be treated as the default for prebuilt tools. |
| useClientOAuth  |   bool   |    false     | If true, forwards the client's OAuth access token from the "Authorization" header to downstream queries.                                                                                                                                                                                                                                                                                                                                                                                                            |
