---
title: "dataplex-generate-data-profile"
type: docs
weight: 1
description: >
  Creates a new Dataplex Data Profile scan template for a specified BigQuery table and triggers the initial asynchronous execution run.
aliases:
  - /integrations/dataplex/tools/dataplex-generate-data-profile/
---

## About

A `dataplex-generate-data-profile` tool triggers a new Data Profile scan to compute statistical profiles (min, max, mean, null ratios, distinct ratios, quantiles, etc.) on table columns.

Since scan template creation is asynchronous, this tool returns an LRO name. You must poll `dataplex-get-operation` with this ID until it is done, extract the `scanId`, and poll `dataplex-get-run-status` with the `scanId` until the job is `SUCCEEDED` before calling `dataplex-get-data-profile` to fetch results.


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

The `dataplex-generate-data-profile` tool accepts the following parameters:

| **field** | **type** | **required** | **description** |
| --------- | :------: | :----------: | --------------- |
| resourcePath | string | true | The resource path of the target BigQuery table (format: `projects/{project}/datasets/{dataset}/tables/{table}`). |
| location | string | true | The Google Cloud region where the scan should be executed (e.g. `us-central1`). |
| publish | boolean | false | If true, publishes the profile results directly to the Dataplex Universal Catalog. Defaults to false. |

## Example

```yaml
kind: tool
name: generate_data_profile
type: dataplex-generate-data-profile
source: my-dataplex-source
description: Trigger a new data profiling scan.
```

## Reference

| **field**   | **type** | **required** | **description**                                    |
|-------------|:--------:|:------------:|----------------------------------------------------|
| type        |  string  |     true     | Must be "dataplex-generate-data-profile".                   |
| source      |  string  |     true     | Name of the source the tool should execute on.     |
| description |  string  |     true     | Description of the tool that is passed to the LLM. |
