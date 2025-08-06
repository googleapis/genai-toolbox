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

- [`bigquery-sql`](../tools/bigquery/bigquery-sql.md)  
  Run SQL queries directly against BigQuery datasets.

- [`bigquery-execute-sql`](../tools/bigquery/bigquery-execute-sql.md)  
  Execute structured queries using parameters.

- [`bigquery-get-dataset-info`](../tools/bigquery/bigquery-get-dataset-info.md)  
  Retrieve metadata for a specific dataset.

- [`bigquery-get-table-info`](../tools/bigquery/bigquery-get-table-info.md)  
  Retrieve metadata for a specific table.

- [`bigquery-list-dataset-ids`](../tools/bigquery/bigquery-list-dataset-ids.md)  
  List available dataset IDs.

- [`bigquery-list-table-ids`](../tools/bigquery/bigquery-list-table-ids.md)  
  List tables in a given dataset.

### Pre-built Configurations

- [BigQuery using MCP](https://googleapis.github.io/genai-toolbox/how-to/connect-ide/bigquery_mcp/)  
Connect your IDE to BigQuery using Toolbox.

## Requirements

### IAM Permissions

BigQuery uses [Identity and Access Management (IAM)][iam-overview] to control
user and group access to BigQuery resources like projects, datasets, and tables.
Toolbox will use your [Application Default Credentials (ADC)][adc] to authorize
and authenticate when interacting with [BigQuery][bigquery-docs].

In addition to [setting the ADC for your server][set-adc], you need to ensure
the IAM identity has been given the correct IAM permissions for the queries
you intend to run. Common roles include `roles/bigquery.user` (which includes
permissions to run jobs and read data) or `roles/bigquery.dataViewer`. See
[Introduction to BigQuery IAM][grant-permissions] for more information on
applying IAM permissions and roles to an identity.

[iam-overview]: https://cloud.google.com/bigquery/docs/access-control
[adc]: https://cloud.google.com/docs/authentication#adc
[set-adc]: https://cloud.google.com/docs/authentication/provide-credentials-adc
[grant-permissions]: https://cloud.google.com/bigquery/docs/access-control

## Example

```yaml
sources:
  my-bigquery-source:
    kind: "bigquery"
    project: "my-project-id"
    # location: "US" # Optional: Specifies the location for query jobs.
    # datasets: # Optional: Restricts tool access to a specific list of datasets.
    #   - "my_dataset_1"
    #   - "other_project.my_dataset_2"

```

## Reference

| **field** | **type** | **required** | **description**                                                               |
|-----------|:--------:|:------------:|-------------------------------------------------------------------------------|
| kind      |  string  |     true     | Must be "bigquery".                                                           |
| project   |  string  |     true     | Id of the GCP project that the cluster was created in (e.g. "my-project-id"). |
| location  |  string  |    false     | Specifies the location (e.g., 'us', 'asia-northeast1') in which to run the query job. This location must match the location of any tables referenced in the query. The default behavior is for it to be executed in the US multi-region |
| datasets  | []string |    false     | An optional list of dataset IDs that tools using this source are allowed to access. If provided, any prebuilt tool operation attempting to access a dataset not in this list will be rejected. Note that the toolbox cannot determine which datasets are accessed within an EXECUTE IMMEDIATE statement. Therefore, when datasets restrictions are active, any SQL from the bigquery-execute-sql tool containing EXECUTE IMMEDIATE will be rejected. This restriction does not apply to the bigquery-sql tool. |
