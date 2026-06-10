---
title: "dataplex-generate-data-insights"
type: docs
weight: 1
description: >
  Creates a new Dataplex Data Insights (documentation) scan template and triggers the initial asynchronous execution run to generate descriptions, relationships, and sample SQL queries for a table.
aliases:
  - /integrations/dataplex/tools/dataplex-generate-data-insights/
---

## About

A `dataplex-generate-data-insights` tool triggers the creation and run of a Dataplex Data Insights scan on a BigQuery table.

Since the scan template creation is asynchronous, this tool returns a Long-Running Operation (LRO) resource name (format: `projects/{project}/locations/{location}/operations/{operation_id}`).
To orchestrate this workflow, you must:
1. Capture the `operation_id` from this tool's response.
2. Poll the `dataplex-get-operation` tool with this ID until `done` is true.
3. Extract the created scan ID (`scanId`) from the completed operation's response.
4. Poll `dataplex-get-run-status` with the `scanId` until the job state is `SUCCEEDED`.
5. Call `dataplex-get-data-insights` with the `scanId` to fetch the final results.


## Compatible Sources

{{< compatible-sources >}}

## Requirements

### IAM Permissions

Knowledge Catalog uses [Identity and Access Management (IAM)][iam-overview] to control
user and group access to Knowledge Catalog resources. Toolbox will use your
[Application Default Credentials (ADC)][adc] to authorize and authenticate when
interacting with [Knowledge Catalog][dataplex-docs].

In addition to [setting the ADC for your server][set-adc], you need to ensure
the IAM identity has been given the correct IAM permissions for the tasks you
intend to perform. See [Knowledge Catalog IAM permissions][iam-permissions]
and [Knowledge Catalog IAM roles][iam-roles] for more information on
applying IAM permissions and roles to an identity.

[iam-overview]: https://cloud.google.com/dataplex/docs/iam-and-access-control
[adc]: https://cloud.google.com/docs/authentication#adc
[set-adc]: https://cloud.google.com/docs/authentication/provide-credentials-adc
[iam-permissions]: https://cloud.google.com/dataplex/docs/iam-permissions
[iam-roles]: https://cloud.google.com/dataplex/docs/iam-roles

## Parameters

The `dataplex-generate-data-insights` tool accepts the following parameters:

| **field** | **type** | **required** | **description** |
| --------- | :------: | :----------: | --------------- |
| resourcePath | string | true | The resource path of the target BigQuery table (format: `projects/{project}/datasets/{dataset}/tables/{table}`). |
| location | string | true | The Google Cloud region where the scan should be executed (e.g. `us-central1`). |
| publish | boolean | false | If true, publishes the generated insights directly to the Dataplex Universal Catalog. Defaults to false. |

## Example

```yaml
kind: tool
name: generate_data_insights
type: dataplex-generate-data-insights
source: my-dataplex-source
description: Trigger a new data insights scan.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| type        |  string  |     true     | Must be "dataplex-generate-data-insights".                   |
| source      |  string  |     true     | Name of the source the tool should execute on.     |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
