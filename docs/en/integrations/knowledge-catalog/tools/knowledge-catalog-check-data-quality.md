---
title: "dataplex-check-data-quality"
type: docs
weight: 1
description: >
  Creates a new Dataplex Data Quality scan template for a specified BigQuery table and triggers the initial asynchronous execution run using custom defined quality rules.
aliases:
  - /integrations/dataplex/tools/dataplex-check-data-quality/
---

## About

A `dataplex-check-data-quality` tool triggers a new Data Quality scan to evaluate rules (e.g. non-null, value range limits, custom SQL assertions) against table rows.

Since scan template creation is asynchronous, this tool returns an LRO name. You must poll `dataplex-get-operation` with this ID until it is done, extract the `scanId`, and poll `dataplex-get-run-status` with the `scanId` until the job is `SUCCEEDED` before calling `dataplex-get-data-quality-results` to fetch results.


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

The `dataplex-check-data-quality` tool accepts the following parameters:

| **field** | **type** | **required** | **description** |
| --------- | :------: | :----------: | --------------- |
| resourcePath | string | true | The resource path of the target BigQuery table (format: `projects/{project}/datasets/{dataset}/tables/{table}`). |
| location | string | true | The Google Cloud region where the scan should be executed (e.g. `us-central1`). |
| publish | boolean | false | If true, publishes the quality results directly to the Dataplex Universal Catalog. Defaults to false. |
| specJSON | string | true | A raw JSON string defining the quality checks rules (e.g. `{"rules": [{"column": "age", "nonNullExpectation": {}}]}`, maps directly to `dataplexpb.DataQualitySpec`). |

## Example

```yaml
kind: tool
name: check_data_quality
type: dataplex-check-data-quality
source: my-dataplex-source
description: Trigger a new data quality scan.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| type        |  string  |     true     | Must be "dataplex-check-data-quality".                   |
| source      |  string  |     true     | Name of the source the tool should execute on.     |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
